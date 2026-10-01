package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor moves funds into the EVM Custody vault on behalf of the depositor
// whose key the supplied sign.Signer holds. It implements core.VaultDepositor.
type Depositor struct {
	client      *ethclient.Client
	custody     *Custody
	custodyAddr common.Address
	signer      sign.Signer
	assets      blockchain.AssetResolver
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor binds the Custody vault at custodyAddr over client; signer is the
// depositor's secp256k1 identity (it pays and, for ERC-20, approves).
func NewDepositor(client *ethclient.Client, custodyAddr common.Address, signer sign.Signer, assets blockchain.AssetResolver) (*Depositor, error) {
	if assets == nil {
		return nil, fmt.Errorf("evm: asset resolver is required")
	}
	custody, err := NewCustody(custodyAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load custody: %w", err)
	}
	return &Depositor{client: client, custody: custody, custodyAddr: custodyAddr, signer: signer, assets: assets}, nil
}

// DepositSubmitError is returned by SubmitDeposit/SubmitDepositWithKey for any
// failure after the nonce has been read, so Key, Nonce and DepositID are always
// set. Step names the failed step; TxHash is set when that step's transaction
// was signed, empty when nothing was broadcast.
//
// Never retry blindly: a blind retry reads the next nonce and can create a
// second real deposit. Resolve the outcome first:
//   - Step "deposit" with a TxHash: call ChainDepositStatus(TxHash, DepositID,
//     minConf). DepositConfirmed or DepositPending means the deposit exists;
//     DepositAbsent means that transaction did not produce it.
//   - No TxHash, or Step "approve": this call did not broadcast a deposit. Read
//     the vault's current nonce with the Custody binding
//     (Custody.GetNonce(depositor, Key)) at `latest`. If it is greater than
//     Nonce, a deposit with that nonce has landed (this one or another from
//     the same depositor and key); otherwise the nonce is unused and calling
//     SubmitDeposit again is safe.
type DepositSubmitError struct {
	// Step is "approve" or "deposit".
	Step      string
	Key       *big.Int
	Nonce     *big.Int
	DepositID string
	// TxHash is the transaction hash, when it could be determined even though
	// the step ultimately failed (e.g. broadcast or mining failed, but signing
	// succeeded so the hash was already known).
	TxHash string
	Err    error
}

func (e *DepositSubmitError) Error() string {
	if e.TxHash != "" {
		return fmt.Sprintf("evm: %s step failed (key=%s nonce=%s depositId=%s txHash=%s): %v",
			e.Step, e.Key, e.Nonce, e.DepositID, e.TxHash, e.Err)
	}
	return fmt.Sprintf("evm: %s step failed (key=%s nonce=%s depositId=%s): %v",
		e.Step, e.Key, e.Nonce, e.DepositID, e.Err)
}

func (e *DepositSubmitError) Unwrap() error { return e.Err }

// ErrStaleNonce indicates the nonce read at the start of SubmitDeposit was
// already consumed by submission time (for example a load-balanced RPC node
// lagging behind another). Liveness only: a fresh call reads the current nonce.
// See DepositSubmitError on retrying safely.
var ErrStaleNonce = errors.New("evm: nonce already consumed by another deposit (stale read)")

// ErrDepositEventNotFound indicates the deposit transaction mined successfully
// but its receipt has no Deposited log, emitted by the vault, whose depositor
// and nonce reproduce the expected DepositID (for example the vault address
// or the chain ID does not match the contract that ran). The returned
// DepositSubmitError carries TxHash and DepositID; no receipt will be signed
// for that DepositID.
var ErrDepositEventNotFound = errors.New("evm: deposited event not found")

// SubmitDeposit credits dest.Account with amount of assetAddress, using nonce
// key 0. Equivalent to SubmitDepositWithKey(..., big.NewInt(0)).
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination) (core.SubmitDepositResult, error) {
	return d.SubmitDepositWithKey(ctx, assetAddress, amount, dest, big.NewInt(0))
}

// SubmitDepositWithKey selects the upper 192 bits of the 2D nonce. Each key has
// its own consecutive sequence, so one depositor address can keep several
// independent deposit streams (IDeposit.sol recommends key =
// uint192(uint160(user)) when depositing for many users).
//
// It reads getNonce itself, so it is for an account that submits its own
// deposits. A contract depositing for users must take the sequence from the
// user's signed input, never from chain state (double credit after a deep
// reorg).
//
// ERC-20: approves the vault for exactly amount, then calls Custody.deposit;
// the approve is skipped when the allowance already covers amount. Native
// marker: sends ETH with msg.value == amount. Blocks until the deposit mines,
// then requires the vault's Deposited log for the returned DepositID
// (ErrDepositEventNotFound otherwise).
//
// Do not call concurrently for the same (depositor, key): both race one
// on-chain counter and one reverts. After an error return, see
// DepositSubmitError before retrying.
func (d *Depositor) SubmitDepositWithKey(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination, key *big.Int) (core.SubmitDepositResult, error) {
	assetAddress = normalizeDepositAssetAddress(assetAddress)
	if err := d.assets.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return core.SubmitDepositResult{}, err
	}
	if amount.Sign() <= 0 {
		return core.SubmitDepositResult{}, fmt.Errorf("evm: amount must be positive")
	}
	decimals, err := d.assets.AssetDecimals(ctx, assetAddress)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	amt, err := blockchain.DecimalToBaseUnits(amount, decimals)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("evm: amount: %w", err)
	}
	assetAddr := depositAssetAddress(assetAddress)
	accountAddr, err := parseClearnetAccount(dest.Account)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	if key == nil {
		key = big.NewInt(0)
	}

	depositorAddr, err := sign.EthAddress(d.signer)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("evm: depositor address: %w", err)
	}
	chainID, err := d.client.ChainID(ctx)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("evm: chain id: %w", err)
	}
	// Read at `latest`, never `pending` (that would turn a blind retry into a
	// second deposit), and before the approve so both steps share one identity.
	nonce, err := d.custody.GetNonce(&bind.CallOpts{Context: ctx}, depositorAddr, key)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("evm: get nonce: %w", err)
	}
	depositID := DepositID(chainID, d.custodyAddr, depositorAddr, nonce)

	if assetAddr != (common.Address{}) {
		// ERC-20: approve exactly amt unless the allowance already covers it.
		// Known gap: a nonzero allowance below amt still sends a
		// nonzero-to-nonzero approve, which USDT-style tokens reject; reset
		// the allowance to zero first.
		token, err := NewMockERC20(assetAddr, d.client)
		if err != nil {
			return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, Err: fmt.Errorf("ERC20 bind %s: %w", assetAddr.Hex(), err)}
		}
		current, err := token.Allowance(&bind.CallOpts{Context: ctx}, depositorAddr, d.custodyAddr)
		if err != nil {
			return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, Err: fmt.Errorf("read allowance: %w", err)}
		}
		if current.Cmp(amt) < 0 {
			approveOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
			if err != nil {
				return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, Err: err}
			}
			approveOpts.NoSend = true
			approveTx, err := token.Approve(approveOpts, d.custodyAddr, amt)
			if err != nil {
				return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, Err: fmt.Errorf("ERC20 approve: %w", err)}
			}
			approveHash := approveTx.Hash().Hex()
			if err := d.client.SendTransaction(ctx, approveTx); err != nil {
				return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, TxHash: approveHash, Err: fmt.Errorf("send ERC20 approve: %w", err)}
			}
			if err := waitMined(ctx, d.client, approveTx); err != nil {
				return core.SubmitDepositResult{}, &DepositSubmitError{Step: "approve", Key: key, Nonce: nonce, DepositID: depositID, TxHash: approveHash, Err: fmt.Errorf("ERC20 approve wait: %w", err)}
			}
		}
	}

	depositOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
	if err != nil {
		return core.SubmitDepositResult{}, &DepositSubmitError{Step: "deposit", Key: key, Nonce: nonce, DepositID: depositID, Err: err}
	}
	if assetAddr == (common.Address{}) {
		depositOpts.Value = amt
	}
	depositOpts.NoSend = true
	tx, err := d.custody.Deposit(depositOpts, accountAddr, assetAddr, amt, dest.Ref, nonce)
	if err != nil {
		return core.SubmitDepositResult{}, &DepositSubmitError{
			Step: "deposit", Key: key, Nonce: nonce, DepositID: depositID,
			Err: d.classifyDepositRevert(ctx, fmt.Errorf("deposit: %w", err), depositorAddr, key, nonce),
		}
	}
	txHash := tx.Hash().Hex()
	if err := d.client.SendTransaction(ctx, tx); err != nil {
		return core.SubmitDepositResult{}, &DepositSubmitError{Step: "deposit", Key: key, Nonce: nonce, DepositID: depositID, TxHash: txHash, Err: fmt.Errorf("send deposit: %w", err)}
	}
	receipt, err := waitMinedReceipt(ctx, d.client, tx)
	if err != nil {
		return core.SubmitDepositResult{}, &DepositSubmitError{
			Step: "deposit", Key: key, Nonce: nonce, DepositID: depositID, TxHash: txHash,
			Err: d.classifyDepositRevert(ctx, err, depositorAddr, key, nonce),
		}
	}
	if !d.hasDepositLog(receipt, chainID, depositID) {
		return core.SubmitDepositResult{}, &DepositSubmitError{
			Step: "deposit", Key: key, Nonce: nonce, DepositID: depositID, TxHash: txHash,
			Err: ErrDepositEventNotFound,
		}
	}
	return core.SubmitDepositResult{TxHash: txHash, DepositID: depositID}, nil
}

// classifyDepositRevert re-labels err as ErrStaleNonce when it can tell the
// deposit call above failed because the nonce it used was no longer next:
// either the revert reason names Custody's own check directly, or —
// for a mined revert with no reason — a fresh getNonce read shows the chain
// has already moved past the nonce this call used.
func (d *Depositor) classifyDepositRevert(ctx context.Context, err error, depositor common.Address, key, nonce *big.Int) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "Invalid nonce") {
		return fmt.Errorf("%w: %v", ErrStaleNonce, err)
	}
	current, gerr := d.custody.GetNonce(&bind.CallOpts{Context: ctx}, depositor, key)
	if gerr == nil && current.Cmp(nonce) > 0 {
		return fmt.Errorf("%w: %v", ErrStaleNonce, err)
	}
	return err
}

func normalizeDepositAssetAddress(assetAddress string) string {
	if strings.TrimSpace(assetAddress) == "" {
		return nativeAssetAddress
	}
	return assetAddress
}

// parseClearnetAccount decodes dest.Account into an EVM address from a
// bare hex, an optional case-insensitive "0x" prefix, or a yellow://.../user/<hex>
// URI's last segment, and surrounding whitespace.
func parseClearnetAccount(account string) (common.Address, error) {
	acct, err := core.ParseClearnetAccount(account)
	if err != nil {
		return common.Address{}, fmt.Errorf("evm: %w", err)
	}
	return common.Address(acct), nil
}

// ChainDepositStatus reports the on-chain status of a deposit identified by
// txHash and depositId (both returned by SubmitDeposit), by fetching the
// receipt at txHash and finding the vault's Deposited log whose depositor and
// nonce reproduce depositId. It is a pure on-chain read: it does not apply
// custody's crediting rules, so DepositConfirmed does not guarantee a credit.
func (d *Depositor) ChainDepositStatus(ctx context.Context, txHash, depositId string, minConf uint64) (core.DepositStatus, error) {
	if _, err := ParseDepositID(depositId); err != nil {
		return core.DepositAbsent, err
	}
	hash, err := parseEVMTxHash(txHash)
	if err != nil {
		return core.DepositAbsent, err
	}
	receipt, err := d.client.TransactionReceipt(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// No receipt — maybe still pending in the mempool.
			if _, isPending, perr := d.client.TransactionByHash(ctx, hash); perr == nil && isPending {
				return core.DepositPending, nil
			}
			return core.DepositAbsent, nil
		}
		return core.DepositAbsent, fmt.Errorf("evm: tx receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return core.DepositAbsent, nil
	}
	chainID, err := d.client.ChainID(ctx)
	if err != nil {
		return core.DepositPending, fmt.Errorf("evm: chain id: %w", err)
	}
	if !d.hasDepositLog(receipt, chainID, depositId) {
		return core.DepositAbsent, nil
	}
	head, err := d.client.BlockNumber(ctx)
	if err != nil {
		return core.DepositPending, fmt.Errorf("evm: block number: %w", err)
	}
	var confs uint64
	if bn := receipt.BlockNumber.Uint64(); head >= bn {
		confs = head - bn + 1
	}
	if confs >= minConf {
		return core.DepositConfirmed, nil
	}
	return core.DepositPending, nil
}

// hasDepositLog reports whether receipt contains a Deposited log, emitted by
// the vault itself, whose depositor and nonce reproduce wantDepositID.
func (d *Depositor) hasDepositLog(receipt *types.Receipt, chainID *big.Int, wantDepositID string) bool {
	for _, raw := range receipt.Logs {
		if raw.Address != d.custodyAddr {
			continue
		}
		event, err := d.custody.ParseDeposited(*raw)
		if err != nil {
			continue
		}
		if DepositID(chainID, d.custodyAddr, event.Depositor, event.Nonce) == wantDepositID {
			return true
		}
	}
	return false
}

// parseEVMTxHash validates that txHash is a transaction hash. A
// "txHash/logIndex" combined form is rejected: the hash and DepositID are
// always passed separately.
func parseEVMTxHash(txHash string) (common.Hash, error) {
	txHash = strings.TrimSpace(txHash)
	if !common.IsHexHash(txHash) {
		return common.Hash{}, fmt.Errorf("evm: txHash must be a transaction hash")
	}
	return common.HexToHash(txHash), nil
}

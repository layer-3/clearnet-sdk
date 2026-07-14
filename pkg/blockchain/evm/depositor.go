package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
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

// SubmitDeposit credits dest.Account with amount of assetAddress. For an ERC-20
// (assetAddress is a non-zero hex address) it approves the vault then calls
// Custody.deposit; for the native marker it sends ETH with msg.value == amount.
// Blocks until the deposit tx mines.
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination) (string, error) {
	assetAddress = normalizeDepositAssetAddress(assetAddress)
	if err := d.assets.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return "", err
	}
	if amount.Sign() <= 0 {
		return "", fmt.Errorf("evm: amount must be positive")
	}
	decimals, err := d.assets.AssetDecimals(ctx, assetAddress)
	if err != nil {
		return "", err
	}
	amt, err := blockchain.DecimalToBaseUnits(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("evm: amount: %w", err)
	}
	assetAddr := depositAssetAddress(assetAddress)
	accountAddr := common.HexToAddress(dest.Account)

	if assetAddr == (common.Address{}) {
		opts, _, err := signerTransactOpts(ctx, d.client, d.signer)
		if err != nil {
			return "", err
		}
		opts.Value = amt
		tx, err := d.custody.Deposit(opts, accountAddr, common.Address{}, amt, dest.Ref)
		if err != nil {
			return "", fmt.Errorf("ETH deposit: %w", err)
		}
		receipt, err := waitMinedReceipt(ctx, d.client, tx)
		if err != nil {
			return "", err
		}
		return d.depositTxID(receipt, accountAddr, common.Address{}, amt, dest.Ref)
	}

	// ERC-20: approve the vault, then deposit.
	token, err := NewMockERC20(assetAddr, d.client)
	if err != nil {
		return "", fmt.Errorf("ERC20 bind %s: %w", assetAddr.Hex(), err)
	}
	approveOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
	if err != nil {
		return "", err
	}
	approveTx, err := token.Approve(approveOpts, d.custodyAddr, amt)
	if err != nil {
		return "", fmt.Errorf("ERC20 approve: %w", err)
	}
	if err := waitMined(ctx, d.client, approveTx); err != nil {
		return "", fmt.Errorf("ERC20 approve wait: %w", err)
	}

	depositOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
	if err != nil {
		return "", err
	}
	tx, err := d.custody.Deposit(depositOpts, accountAddr, assetAddr, amt, dest.Ref)
	if err != nil {
		return "", fmt.Errorf("ERC20 deposit: %w", err)
	}
	receipt, err := waitMinedReceipt(ctx, d.client, tx)
	if err != nil {
		return "", err
	}
	return d.depositTxID(receipt, accountAddr, assetAddr, amt, dest.Ref)
}

func normalizeDepositAssetAddress(assetAddress string) string {
	if strings.TrimSpace(assetAddress) == "" {
		return nativeAssetAddress
	}
	return assetAddress
}

// VerifyDeposit reports the on-chain status of the deposit txID. EVM deposit
// txIDs are event-level IDs of the form txHash/logIndex. A raw txHash is
// accepted for transaction-level status checks.
func (d *Depositor) VerifyDeposit(ctx context.Context, txID string, minConf uint64) (core.DepositStatus, error) {
	hash, logIndex, err := parseEVMDepositTxID(txID)
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
	if logIndex != nil && !d.hasDepositLog(receipt, *logIndex) {
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

func (d *Depositor) depositTxID(receipt *types.Receipt, account common.Address, asset common.Address, amount *big.Int, ref [32]byte) (string, error) {
	for _, raw := range receipt.Logs {
		if raw.Address != d.custodyAddr {
			continue
		}
		event, err := d.custody.ParseDeposited(*raw)
		if err != nil {
			continue
		}
		if event.Account == account && event.DepositReference == ref && event.Asset == asset && event.Amount.Cmp(amount) == 0 {
			return fmt.Sprintf("%s/%d", raw.TxHash.Hex(), raw.Index), nil
		}
	}
	return "", fmt.Errorf("evm: deposited event not found in tx %s", receipt.TxHash.Hex())
}

func (d *Depositor) hasDepositLog(receipt *types.Receipt, logIndex uint) bool {
	for _, raw := range receipt.Logs {
		if raw.Index != logIndex || raw.Address != d.custodyAddr {
			continue
		}
		if _, err := d.custody.ParseDeposited(*raw); err == nil {
			return true
		}
	}
	return false
}

func parseEVMDepositTxID(txID string) (common.Hash, *uint, error) {
	txID = strings.TrimSpace(txID)
	hashText, indexText, hasIndex := strings.Cut(txID, "/")
	if !common.IsHexHash(hashText) {
		return common.Hash{}, nil, fmt.Errorf("evm: txID must be a transaction hash or txHash/logIndex")
	}
	hash := common.HexToHash(hashText)
	if !hasIndex {
		return hash, nil, nil
	}
	index64, err := strconv.ParseUint(indexText, 10, 32)
	if err != nil {
		return common.Hash{}, nil, fmt.Errorf("evm: bad txID log index %q: %w", indexText, err)
	}
	index := uint(index64)
	return hash, &index, nil
}

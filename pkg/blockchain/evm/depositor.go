package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor moves funds into the EVM Custody vault on behalf of the depositor
// whose key the supplied sign.Signer holds. It implements core.VaultDepositor.
type Depositor struct {
	client      *ethclient.Client
	custody     *Custody
	custodyAddr common.Address
	signer      sign.Signer
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor binds the Custody vault at custodyAddr over client; signer is the
// depositor's secp256k1 identity (it pays and, for ERC-20, approves).
func NewDepositor(client *ethclient.Client, custodyAddr common.Address, signer sign.Signer) (*Depositor, error) {
	custody, err := NewCustody(custodyAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load custody: %w", err)
	}
	return &Depositor{client: client, custody: custody, custodyAddr: custodyAddr, signer: signer}, nil
}

// SubmitDeposit credits dest.Account with amount base units of assetAddress. For
// an ERC-20 (assetAddress is a non-zero hex address) it approves the vault then
// calls Custody.deposit; for the zero address it sends native ETH with
// msg.value == amount. Blocks until the deposit tx mines.
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount *big.Int, dest core.DepositDestination) (core.TxRef, error) {
	assetAddr, err := depositAssetAddress(assetAddress)
	if err != nil {
		return core.TxRef{}, err
	}
	if amount == nil || amount.Sign() <= 0 {
		return core.TxRef{}, fmt.Errorf("evm: amount must be positive")
	}
	amt := new(big.Int).Set(amount)
	accountAddr := common.HexToAddress(dest.Account)

	if assetAddr == (common.Address{}) {
		opts, _, err := signerTransactOpts(ctx, d.client, d.signer)
		if err != nil {
			return core.TxRef{}, err
		}
		opts.Value = amt
		tx, err := d.custody.Deposit(opts, accountAddr, common.Address{}, amt, dest.Ref)
		if err != nil {
			return core.TxRef{}, fmt.Errorf("ETH deposit: %w", err)
		}
		if err := waitMined(ctx, d.client, tx); err != nil {
			return core.TxRef{}, err
		}
		return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
	}

	// ERC-20: approve the vault, then deposit.
	token, err := NewMockERC20(assetAddr, d.client)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("ERC20 bind %s: %w", assetAddr.Hex(), err)
	}
	approveOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	approveTx, err := token.Approve(approveOpts, d.custodyAddr, amt)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("ERC20 approve: %w", err)
	}
	if err := waitMined(ctx, d.client, approveTx); err != nil {
		return core.TxRef{}, fmt.Errorf("ERC20 approve wait: %w", err)
	}

	depositOpts, _, err := signerTransactOpts(ctx, d.client, d.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	tx, err := d.custody.Deposit(depositOpts, accountAddr, assetAddr, amt, dest.Ref)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("ERC20 deposit: %w", err)
	}
	if err := waitMined(ctx, d.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

// VerifyDeposit reports the on-chain status of the deposit tx in ref. A reverted
// deposit (receipt status != 1) reads as DepositAbsent since it credited
// nothing. Confirmations are counted inclusively (head - blockNumber + 1).
func (d *Depositor) VerifyDeposit(ctx context.Context, ref core.TxRef, minConf uint64) (core.DepositStatus, error) {
	hash := common.Hash(ref.Hash)
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

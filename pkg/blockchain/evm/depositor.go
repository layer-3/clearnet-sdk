package evm

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

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

// Deposit credits `account` with `amount` of `asset`. For an ERC-20 (asset is a
// non-zero hex address) it approves the vault then calls
// Custody.deposit(account, asset, amount); for the zero address it sends native
// ETH with msg.value == amount. Blocks until the deposit tx mines.
func (d *Depositor) Deposit(ctx context.Context, asset string, amount decimal.Decimal, account string) (core.TxRef, error) {
	assetAddr, err := depositAssetAddress(asset)
	if err != nil {
		return core.TxRef{}, err
	}
	amt := amount.BigInt()
	accountAddr := common.HexToAddress(account)

	if assetAddr == (common.Address{}) {
		opts, _, err := signerTransactOpts(ctx, d.client, d.signer)
		if err != nil {
			return core.TxRef{}, err
		}
		opts.Value = amt
		tx, err := d.custody.Deposit(opts, accountAddr, common.Address{}, amt)
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
	tx, err := d.custody.Deposit(depositOpts, accountAddr, assetAddr, amt)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("ERC20 deposit: %w", err)
	}
	if err := waitMined(ctx, d.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

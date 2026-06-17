package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// FaucetAdapter wraps the Faucet binding for testnet token drips and parameter
// reads. It implements core.FaucetWriter and core.FaucetReader.
type FaucetAdapter struct {
	client *ethclient.Client
	faucet *Faucet
	auth   *bind.TransactOpts
}

var (
	_ core.FaucetWriter = (*FaucetAdapter)(nil)
	_ core.FaucetReader = (*FaucetAdapter)(nil)
)

// NewFaucetAdapter binds the Faucet at faucetAddr over client with a
// transactor for the given key (needed for the drip writes).
func NewFaucetAdapter(ctx context.Context, client *ethclient.Client, faucetAddr common.Address, key *ecdsa.PrivateKey) (*FaucetAdapter, error) {
	faucet, err := NewFaucet(faucetAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load faucet: %w", err)
	}
	auth, err := newTransactor(ctx, client, key)
	if err != nil {
		return nil, err
	}
	return &FaucetAdapter{client: client, faucet: faucet, auth: auth}, nil
}

// ─── Write ───────────────────────────────────────────────────────────────────

// Drip claims tokens to the transactor's own address.
func (a *FaucetAdapter) Drip(ctx context.Context) error {
	tx, err := a.faucet.Drip(txOpts(a.auth, ctx))
	if err != nil {
		return fmt.Errorf("faucet drip: %w", err)
	}
	return waitMined(ctx, a.client, tx)
}

// DripTo claims tokens to recipient.
func (a *FaucetAdapter) DripTo(ctx context.Context, recipient common.Address) error {
	tx, err := a.faucet.DripTo(txOpts(a.auth, ctx), recipient)
	if err != nil {
		return fmt.Errorf("faucet drip-to: %w", err)
	}
	return waitMined(ctx, a.client, tx)
}

// ─── Read ────────────────────────────────────────────────────────────────────

func (a *FaucetAdapter) DripAmount(ctx context.Context) (*big.Int, error) {
	return a.faucet.DripAmount(&bind.CallOpts{Context: ctx})
}

func (a *FaucetAdapter) Cooldown(ctx context.Context) (*big.Int, error) {
	return a.faucet.Cooldown(&bind.CallOpts{Context: ctx})
}

func (a *FaucetAdapter) Owner(ctx context.Context) (common.Address, error) {
	return a.faucet.Owner(&bind.CallOpts{Context: ctx})
}

func (a *FaucetAdapter) LastDrip(ctx context.Context, addr common.Address) (*big.Int, error) {
	return a.faucet.LastDrip(&bind.CallOpts{Context: ctx}, addr)
}

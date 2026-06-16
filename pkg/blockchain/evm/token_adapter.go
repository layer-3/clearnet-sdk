package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// TokenAdapter reads ERC-20 balances. It binds the token contract per call
// from the address passed to BalanceOf, so one adapter serves any token. It
// implements core.TokenReader.
type TokenAdapter struct {
	client *ethclient.Client
}

var _ core.TokenReader = (*TokenAdapter)(nil)

// NewTokenAdapter creates a read-only token adapter over client.
func NewTokenAdapter(client *ethclient.Client) *TokenAdapter {
	return &TokenAdapter{client: client}
}

// BalanceOf returns the ERC-20 balance of account (hex) for the token contract
// at token (hex).
func (a *TokenAdapter) BalanceOf(ctx context.Context, token string, account string) (*big.Int, error) {
	if !common.IsHexAddress(token) {
		return nil, fmt.Errorf("invalid token address %q", token)
	}
	if !common.IsHexAddress(account) {
		return nil, fmt.Errorf("invalid account address %q", account)
	}
	erc20, err := NewMockERC20(common.HexToAddress(token), a.client)
	if err != nil {
		return nil, fmt.Errorf("bind token %s: %w", token, err)
	}
	return erc20.BalanceOf(&bind.CallOpts{Context: ctx}, common.HexToAddress(account))
}

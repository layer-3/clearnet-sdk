// Package evm implements the chain-agnostic adapter interfaces (see pkg/core)
// against an EVM chain, one focused type per concern: Depositor and
// WithdrawalFinalizer (the vault money path), plus RegistryAdapter,
// TokenAdapter, FraudAdapter, and FaucetAdapter. Each wraps the relevant
// generated binding(s) over a caller-supplied *ethclient.Client; write-capable
// types additionally take a sign.Signer (the registry/token/faucet/fraud
// adapters take a raw key). The package never dials on its own behalf beyond
// resolving the chain ID for the transactor.
package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// signerTransactOpts builds TransactOpts whose tx signature is produced by a
// sign.Signer (so KMS keys work), routing through SignEthDigest. Returns the
// signer's Ethereum address alongside. The caller sets per-tx fields (Value,
// gas) on the returned opts.
func signerTransactOpts(ctx context.Context, client *ethclient.Client, s sign.Signer) (*bind.TransactOpts, common.Address, error) {
	addr, err := sign.EthAddress(s)
	if err != nil {
		return nil, common.Address{}, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("get chain ID: %w", err)
	}
	txSigner := gethtypes.LatestSignerForChainID(chainID)
	opts := &bind.TransactOpts{
		From:    addr,
		Context: ctx,
		Signer: func(from common.Address, tx *gethtypes.Transaction) (*gethtypes.Transaction, error) {
			if from != addr {
				return nil, bind.ErrNotAuthorized
			}
			sig, err := sign.SignEthDigest(ctx, s, txSigner.Hash(tx).Bytes(), addr)
			if err != nil {
				return nil, fmt.Errorf("evm: sign tx: %w", err)
			}
			return tx.WithSignature(txSigner, sig)
		},
	}
	return opts, addr, nil
}

// newTransactor builds a chain-ID-bound keyed transactor for write-capable
// adapters. The chain ID is read once from the client at construction.
func newTransactor(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, fmt.Errorf("create transactor: %w", err)
	}
	return auth, nil
}

// txOpts clones the stored transactor with a per-call context (and no value).
func txOpts(auth *bind.TransactOpts, ctx context.Context) *bind.TransactOpts {
	o := *auth
	o.Context = ctx
	o.Value = nil
	return &o
}

// waitMinedReceipt blocks until tx is mined and reverts on a failed receipt.
func waitMinedReceipt(ctx context.Context, client *ethclient.Client, tx *gethtypes.Transaction) (*gethtypes.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, fmt.Errorf("wait mined: %w", err)
	}
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction reverted (tx=%s)", tx.Hash().Hex())
	}
	return receipt, nil
}

// waitMined blocks until tx is mined and reverts on a failed receipt.
func waitMined(ctx context.Context, client *ethclient.Client, tx *gethtypes.Transaction) error {
	_, err := waitMinedReceipt(ctx, client, tx)
	return err
}

// depositAssetAddress converts a validated protocol asset address to the
// contract address; "0" denotes native ETH.
func depositAssetAddress(asset string) common.Address {
	if asset == nativeAssetAddress {
		return common.Address{}
	}
	return common.HexToAddress(asset)
}

// convertNodeRecord maps a Registry NodeRecord binding struct into a core.Slot.
func convertNodeRecord(n NodeRecord) *core.Slot {
	slot := &core.Slot{
		Index:         uint64(n.Index),
		TokenID:       new(big.Int).SetUint64(uint64(n.TokenId)),
		ActivatedAt:   n.ActivatedAt,
		DeactivatedAt: n.DeactivatedAt,
		Collateral:    new(big.Int).Add(n.OperatorCollateral, n.SponsorCollateral),
	}
	// Encode G2 pubkey if present (zero point means slot has no BLS key yet).
	if !isZeroG2(n.BlsPubkeyG2) {
		var pk [128]byte
		copyG2ToBytes(pk[:], n.BlsPubkeyG2)
		slot.BLSPubKey = pk[:]
	}
	return slot
}

func isZeroG2(g2 [4]*big.Int) bool {
	for _, c := range g2 {
		if c != nil && c.Sign() != 0 {
			return false
		}
	}
	return true
}

func copyG2ToBytes(dst []byte, g2 [4]*big.Int) {
	for i, c := range g2 {
		if c == nil {
			continue
		}
		b := c.Bytes()
		// Right-align into 32-byte slots.
		copy(dst[i*32+(32-len(b)):(i+1)*32], b)
	}
}

// parseRegistryActivationFromReceipt extracts (tokenId, nodeId) from the
// NodeActivated event in a register() receipt.
func parseRegistryActivationFromReceipt(registry *Registry, receipt *gethtypes.Receipt) (uint32, [32]byte, error) {
	for _, log := range receipt.Logs {
		event, err := registry.ParseNodeActivated(*log)
		if err != nil {
			continue
		}
		return event.TokenId, event.NodeId, nil
	}
	return 0, [32]byte{}, fmt.Errorf("NodeActivated event not found in receipt")
}

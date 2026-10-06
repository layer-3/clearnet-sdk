package evm

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/bls"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// RegistryAdapter wraps the ClearnetRegistryProtocol binding (plus the
// staking-token binding for collateral approvals) for node onboarding and
// registry queries. It implements core.RegistryReader and core.RegistryWriter.
type RegistryAdapter struct {
	client       *ethclient.Client
	registry     *ClearnetRegistryProtocol
	registryAddr common.Address
	token        *MockERC20
	auth         *bind.TransactOpts
}

var (
	_ core.RegistryReader = (*RegistryAdapter)(nil)
	_ core.RegistryWriter = (*RegistryAdapter)(nil)
)

// ErrInvalidRegistryProof is returned before approval for a malformed or invalid PoP.
var ErrInvalidRegistryProof = errors.New("invalid Registry proof of possession")

// NewRegistryAdapter binds the registry at registryAddr through
// IClearnetRegistryProtocol, and the staking token at tokenAddr, over client,
// with a transactor for the given key. The token is needed because Lock/Fund
// approve collateral before the registry call.
func NewRegistryAdapter(ctx context.Context, client *ethclient.Client, registryAddr, tokenAddr common.Address, key *ecdsa.PrivateKey) (*RegistryAdapter, error) {
	registry, err := NewClearnetRegistryProtocol(registryAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}
	token, err := NewMockERC20(tokenAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load staking token: %w", err)
	}
	auth, err := newTransactor(ctx, client, key)
	if err != nil {
		return nil, err
	}
	return &RegistryAdapter{
		client:       client,
		registry:     registry,
		registryAddr: registryAddr,
		token:        token,
		auth:         auth,
	}, nil
}

// ─── Write ───────────────────────────────────────────────────────────────────

// Lock approves collateral and calls Registry.register, which mints a NodeID
// into escrow and activates it. It returns the token ID.
// popSignature signs bls.RegistryProofOfPossessionDigest and is verified on chain.
// The Registry must support the possessionSignature registration argument.
// Registries with the old three-argument register ABI are unsupported. Local
// proof validation precedes approval; registration gas is estimated after
// approval. An on-chain failure can still leave allowance: approval and register
// are separate transactions, not an atomic operation.
func (a *RegistryAdapter) Lock(ctx context.Context, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, popSignature [2]*big.Int, maxPrice *big.Int) (uint32, error) {
	chainID, err := a.client.ChainID(ctx)
	if err != nil {
		return 0, fmt.Errorf("registration chain ID: %w", err)
	}
	if err := validateRegistryProof(chainID, a.registryAddr, a.auth.From, blsPubkeyG1, blsPubkeyG2, popSignature); err != nil {
		return 0, err
	}
	floor, err := a.registry.FloorPrice(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("floor price: %w", err)
	}
	if maxPrice != nil && floor.Cmp(maxPrice) > 0 {
		return 0, fmt.Errorf("price exceeds max")
	}
	collateral := new(big.Int).Mul(floor, big.NewInt(2))

	tokenApproveTx, err := a.token.Approve(txOpts(a.auth, ctx), a.registryAddr, collateral)
	if err != nil {
		return 0, fmt.Errorf("approve staking token: %w", err)
	}
	if err := waitMined(ctx, a.client, tokenApproveTx); err != nil {
		return 0, fmt.Errorf("approve staking token wait: %w", err)
	}

	registerOpts := txOpts(a.auth, ctx)
	registerTx, err := a.registry.Register(registerOpts, blsPubkeyG1, blsPubkeyG2, popSignature, collateral)
	if err != nil {
		return 0, fmt.Errorf("register: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, a.client, registerTx)
	if err != nil {
		return 0, fmt.Errorf("register wait: %w", err)
	}
	if receipt.Status == 0 {
		return 0, fmt.Errorf("register reverted")
	}
	tokenId, _, err := parseRegistryActivationFromReceipt(a.registry, receipt)
	if err != nil {
		return 0, fmt.Errorf("parse NodeActivated: %w", err)
	}
	return tokenId, nil
}

func validateRegistryProof(chainID *big.Int, registry, operator common.Address, g1 [2]*big.Int, g2 [4]*big.Int, proof [2]*big.Int) error {
	digest, err := bls.RegistryProofOfPossessionDigest(chainID, registry, operator, g1, g2)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRegistryProof, err)
	}
	var encoded [64]byte
	for i, c := range proof {
		if c == nil || c.Sign() < 0 || c.BitLen() > 256 {
			return fmt.Errorf("%w: malformed signature", ErrInvalidRegistryProof)
		}
		c.FillBytes(encoded[i*32 : (i+1)*32])
	}
	sigma, err := bls.DeserializeG1(encoded[:])
	if err != nil || sigma.IsInfinity() {
		return fmt.Errorf("%w: invalid signature point", ErrInvalidRegistryProof)
	}
	var keyBytes [128]byte
	for i, c := range g2 {
		c.FillBytes(keyBytes[i*32 : (i+1)*32])
	}
	key, err := bls.DeserializeG2(keyBytes[:])
	if err != nil || key.IsInfinity() {
		return fmt.Errorf("%w: invalid public key", ErrInvalidRegistryProof)
	}
	valid, err := bls.Verify(sigma, key, digest)
	if err != nil || !valid {
		return fmt.Errorf("%w: signature does not verify", ErrInvalidRegistryProof)
	}
	// The proof binds both keys but must not authorize an inconsistent G1 key.
	var g1Bytes [64]byte
	for i, c := range g1 {
		c.FillBytes(g1Bytes[i*32 : (i+1)*32])
	}
	pubG1, err := bls.DeserializeG1(g1Bytes[:])
	if err != nil || pubG1.IsInfinity() {
		return fmt.Errorf("%w: invalid G1 public key", ErrInvalidRegistryProof)
	}
	_, _, genG1, genG2 := bn254.Generators()
	genG1.Neg(&genG1)
	consistent, err := bn254.PairingCheck([]bn254.G1Affine{pubG1, genG1}, []bn254.G2Affine{genG2, key})
	if err != nil || !consistent {
		return fmt.Errorf("%w: inconsistent G1/G2 key", ErrInvalidRegistryProof)
	}
	return nil
}

func (a *RegistryAdapter) Unlock(ctx context.Context, tokenId uint32) error {
	tx, err := a.registry.Unlock(txOpts(a.auth, ctx), tokenId)
	if err != nil {
		return err
	}
	return waitMined(ctx, a.client, tx)
}

func (a *RegistryAdapter) Release(ctx context.Context, tokenId uint32) error {
	tx, err := a.registry.Release(txOpts(a.auth, ctx), tokenId)
	if err != nil {
		return err
	}
	return waitMined(ctx, a.client, tx)
}

func (a *RegistryAdapter) Fund(ctx context.Context, tokenId uint32, amount *big.Int) error {
	tokenApproveTx, err := a.token.Approve(txOpts(a.auth, ctx), a.registryAddr, amount)
	if err != nil {
		return fmt.Errorf("approve staking token for funding: %w", err)
	}
	if err := waitMined(ctx, a.client, tokenApproveTx); err != nil {
		return fmt.Errorf("approve staking token wait for funding: %w", err)
	}

	tx, err := a.registry.Fund(txOpts(a.auth, ctx), tokenId, amount)
	if err != nil {
		return err
	}
	return waitMined(ctx, a.client, tx)
}

// ─── Read ────────────────────────────────────────────────────────────────────

func (a *RegistryAdapter) GetNodeByID(ctx context.Context, nodeID [32]byte) (*core.Slot, error) {
	n, err := a.registry.GetNodeById(&bind.CallOpts{Context: ctx}, nodeID)
	if err != nil {
		return nil, err
	}
	slot := convertNodeRecord(n)
	slot.ID = nodeID
	return slot, nil
}

func (a *RegistryAdapter) FloorPrice(ctx context.Context) (*big.Int, error) {
	return a.registry.FloorPrice(&bind.CallOpts{Context: ctx})
}

func (a *RegistryAdapter) UnbondingPeriod(ctx context.Context) (uint64, error) {
	return a.registry.UNBONDINGPERIOD(&bind.CallOpts{Context: ctx})
}

func (a *RegistryAdapter) GetNodeId(ctx context.Context, tokenId uint32) ([32]byte, error) {
	return a.registry.GetNodeId(&bind.CallOpts{Context: ctx}, tokenId)
}

func (a *RegistryAdapter) GetNodes(ctx context.Context, offset, limit *big.Int) ([]*core.Slot, error) {
	nodes, err := a.registry.GetNodes(&bind.CallOpts{Context: ctx}, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*core.Slot, len(nodes))
	for i, n := range nodes {
		slot := convertNodeRecord(n)
		id, err := a.registry.GetNodeId(&bind.CallOpts{Context: ctx}, n.TokenId)
		if err == nil {
			slot.ID = id
		}
		result[i] = slot
	}
	return result, nil
}

func (a *RegistryAdapter) GetActiveNodes(ctx context.Context, offset, limit *big.Int) ([]*core.Slot, error) {
	nodes, err := a.registry.GetNodes(&bind.CallOpts{Context: ctx}, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*core.Slot, 0, len(nodes))
	for _, n := range nodes {
		if n.DeactivatedAt != 0 {
			continue
		}
		slot := convertNodeRecord(n)
		id, err := a.registry.GetNodeId(&bind.CallOpts{Context: ctx}, n.TokenId)
		if err == nil {
			slot.ID = id
		}
		result = append(result, slot)
	}
	return result, nil
}

func (a *RegistryAdapter) TotalNodes(ctx context.Context) (*big.Int, error) {
	return a.registry.TotalNodes(&bind.CallOpts{Context: ctx})
}

func (a *RegistryAdapter) ActiveCount(ctx context.Context) (uint32, error) {
	return a.registry.ActiveCount(&bind.CallOpts{Context: ctx})
}

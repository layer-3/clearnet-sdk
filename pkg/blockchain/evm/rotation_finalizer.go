package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// rotationLookupWindow bounds the eth_getLogs range when resolving the tx hash
// of an already-applied rotation.
const rotationLookupWindow = uint64(50_000)

// RotationFinalizer rotates the Custody vault's signer set via updateSigners,
// authorized by the current (outgoing) k-of-n quorum. It is the rotation
// analogue of WithdrawalFinalizer and implements core.SignerRotationFinalizer.
// It owns the node's signer (signs the rotation digest and submits) and the
// vault address + chain id supplied at construction.
type RotationFinalizer struct {
	client     *ethclient.Client
	custody    *Custody
	vaultAddr  common.Address
	chainID    uint64
	signer     sign.Signer
	signerAddr common.Address
	fees       FeeConfig
}

var _ core.SignerRotationFinalizer = (*RotationFinalizer)(nil)

// NewRotationFinalizer binds the Custody vault at vaultAddr and reads the chain
// id from client. signer is this node's secp256k1 identity.
func NewRotationFinalizer(ctx context.Context, client *ethclient.Client, vaultAddr common.Address, signer sign.Signer, fees FeeConfig) (*RotationFinalizer, error) {
	custody, err := NewCustody(vaultAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load custody: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	return &RotationFinalizer{
		client:     client,
		custody:    custody,
		vaultAddr:  vaultAddr,
		chainID:    chainID.Uint64(),
		signer:     signer,
		signerAddr: addr,
		fees:       fees,
	}, nil
}

// evmRotPacked is the canonical rotation payload: the new signer set (ascending)
// + threshold, and the signer nonce the digest is bound to.
type evmRotPacked struct {
	NewSigners   []string `json:"newSigners"` // ascending hex addresses
	NewThreshold int      `json:"newThreshold"`
	SignerNonce  string   `json:"signerNonce"` // decimal
}

// Pack reads the live signer nonce and returns the canonical JSON for rotating
// to newSigners / newThreshold (signers sorted ascending, as Custody requires).
// opID is ignored: EVM binds rotation replay to the on-chain Custody signerNonce,
// so the operation identity is not embedded in the payload.
func (f *RotationFinalizer) Pack(ctx context.Context, _ [32]byte, newSigners []string, newThreshold int) ([]byte, error) {
	addrs, err := parseSignerAddresses(newSigners)
	if err != nil {
		return nil, err
	}
	if newThreshold <= 0 || newThreshold > len(addrs) {
		return nil, fmt.Errorf("evm: threshold %d out of range for %d signers", newThreshold, len(addrs))
	}
	nonce, err := f.custody.SignerNonce(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("read signer nonce: %w", err)
	}
	p := evmRotPacked{NewSigners: addrsToHex(addrs), NewThreshold: newThreshold, SignerNonce: nonce.String()}
	return json.Marshal(p)
}

// Validate re-derives the rotation target from newSigners / newThreshold and
// asserts the packed payload matches, including a re-read of the live nonce to
// reject a packer that bound a stale or wrong signer nonce.
func (f *RotationFinalizer) Validate(ctx context.Context, _ [32]byte, packed []byte, newSigners []string, newThreshold int) error {
	var got evmRotPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(newSigners)
	if err != nil {
		return err
	}
	want := evmRotPacked{NewSigners: addrsToHex(addrs), NewThreshold: newThreshold}
	if got.NewThreshold != want.NewThreshold || !equalStrings(got.NewSigners, want.NewSigners) {
		return fmt.Errorf("packed rotation does not match request")
	}
	nonce, err := f.custody.SignerNonce(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("read signer nonce: %w", err)
	}
	if got.SignerNonce != nonce.String() {
		return fmt.Errorf("packed signer nonce %s != live %s", got.SignerNonce, nonce)
	}
	return nil
}

// Sign produces this node's 65-byte ECDSA signature (V ∈ {0,1}) over the
// rotation digest derived from the packed bytes.
func (f *RotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.signer, digest[:], f.signerAddr)
}

// Submit merges the collected signatures against the live (outgoing) signer set
// and broadcasts updateSigners. Idempotent: if the rotation already applied it
// returns the prior tx hash without re-submitting.
func (f *RotationFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (core.TxRef, error) {
	var p evmRotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return core.TxRef{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(p.NewSigners)
	if err != nil {
		return core.TxRef{}, err
	}
	if txHash, done, err := f.VerifyRotation(ctx, p.NewSigners, p.NewThreshold); err != nil {
		return core.TxRef{}, err
	} else if done {
		return core.TxRef{Hash: txHash, Raw: common.Hash(txHash).Hex()}, nil
	}

	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return core.TxRef{}, err
	}
	liveSigners, liveThreshold, err := fetchLiveQuorum(ctx, f.custody)
	if err != nil {
		return core.TxRef{}, err
	}
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, liveSigners, liveThreshold)
	if err != nil {
		return core.TxRef{}, err
	}

	opts, _, err := signerTransactOpts(ctx, f.client, f.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	if err := applyFees(ctx, f.client, f.fees, opts); err != nil {
		return core.TxRef{}, err
	}
	tx, err := f.custody.UpdateSigners(opts, addrs, big.NewInt(int64(p.NewThreshold)), sigs)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("updateSigners: %w", err)
	}
	if err := waitMined(ctx, f.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

// VerifyRotation reports whether the on-chain signer set now equals newSigners
// with the given threshold. When set, it resolves the SignersUpdated event's tx
// hash within the lookback window.
func (f *RotationFinalizer) VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) ([32]byte, bool, error) {
	addrs, err := parseSignerAddresses(newSigners)
	if err != nil {
		return [32]byte{}, false, err
	}
	live, err := f.custody.Signers(&bind.CallOpts{Context: ctx})
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read signers: %w", err)
	}
	thr, err := f.custody.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || int(thr.Int64()) != newThreshold || !addrSetEqual(live, addrs) {
		return [32]byte{}, false, nil
	}
	return f.lookupRotationTxHash(ctx, addrs), true, nil
}

// --- helpers ---

func (f *RotationFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p evmRotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(p.NewSigners)
	if err != nil {
		return [32]byte{}, err
	}
	nonce, ok := new(big.Int).SetString(p.SignerNonce, 10)
	if !ok {
		return [32]byte{}, fmt.Errorf("bad signer nonce %q", p.SignerNonce)
	}
	return ComputeRotationDigest(f.chainID, f.vaultAddr, addrs, big.NewInt(int64(p.NewThreshold)), nonce), nil
}

// lookupRotationTxHash finds the SignersUpdated event matching addrs within the
// lookback window; a zero hash (with done already established) is acceptable.
func (f *RotationFinalizer) lookupRotationTxHash(ctx context.Context, addrs []common.Address) [32]byte {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		return [32]byte{}
	}
	var from uint64
	if head > rotationLookupWindow {
		from = head - rotationLookupWindow
	}
	it, err := f.custody.FilterSignersUpdated(&bind.FilterOpts{Context: ctx, Start: from, End: &head})
	if err != nil {
		return [32]byte{}
	}
	defer it.Close()
	var last [32]byte
	for it.Next() {
		if addrSetEqual(it.Event.NewSigners, addrs) {
			last = it.Event.Raw.TxHash // keep the most recent match
		}
	}
	return last
}

// parseSignerAddresses validates and sorts the incoming hex addresses ascending
// (Custody requires ascending, no duplicates) — the order the digest binds.
func parseSignerAddresses(newSigners []string) ([]common.Address, error) {
	if len(newSigners) == 0 {
		return nil, fmt.Errorf("evm: empty new signer set")
	}
	out := make([]common.Address, 0, len(newSigners))
	seen := make(map[common.Address]struct{}, len(newSigners))
	for _, s := range newSigners {
		if !common.IsHexAddress(s) {
			return nil, fmt.Errorf("evm: signer %q is not a hex address", s)
		}
		a := common.HexToAddress(s)
		if _, dup := seen[a]; dup {
			return nil, fmt.Errorf("evm: duplicate signer %s", a)
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i][:], out[j][:]) < 0 })
	return out, nil
}

func addrsToHex(addrs []common.Address) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.Hex()
	}
	return out
}

// addrSetEqual reports whether two address slices hold the same set (order
// independent). Both are deduplicated by construction here.
func addrSetEqual(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[common.Address]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	for _, x := range b {
		if _, ok := set[x]; !ok {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

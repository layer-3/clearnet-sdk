package sol

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// RotationFinalizer rotates the custody program's signer set via update_signers,
// authorized by the current (outgoing) ed25519 quorum and verified on-chain via
// the Ed25519 precompile — the rotation analogue of WithdrawalFinalizer. The
// node's signer contributes one share; a separate fee-payer pays + submits. It
// implements core.SignerRotationFinalizer.
type RotationFinalizer struct {
	client      *rpc.Client
	programID   solana.PublicKey
	chainID     uint64
	configPDA   solana.PublicKey
	eventAuth   solana.PublicKey
	cuLimit     uint32
	cuPrice     uint64
	commitment  rpc.CommitmentType
	signer      sign.Signer
	nodePub     solana.PublicKey
	feePayer    sign.Signer
	feePayerPub solana.PublicKey
}

var _ core.SignerRotationFinalizer = (*RotationFinalizer)(nil)

// NewRotationFinalizer builds the finalizer. signer is this node's ed25519
// custody key (one quorum share); feePayer pays for and submits the
// update_signers transaction. cfg reuses the withdrawal Config (chain id,
// compute budget, commitment).
func NewRotationFinalizer(rpcURL string, programID solana.PublicKey, signer, feePayer sign.Signer, cfg Config) (*RotationFinalizer, error) {
	nodePub, err := solanaPub(signer)
	if err != nil {
		return nil, err
	}
	payerPub, err := solanaPub(feePayer)
	if err != nil {
		return nil, fmt.Errorf("sol: fee payer: %w", err)
	}
	limit := cfg.ComputeUnitLimit
	if limit == 0 {
		limit = defaultComputeUnitLimit
	}
	commitment := cfg.Commitment
	if commitment == "" {
		commitment = rpc.CommitmentFinalized
	}
	return &RotationFinalizer{
		client:      rpc.New(rpcURL),
		programID:   programID,
		chainID:     cfg.ChainID,
		configPDA:   ConfigPDA(programID),
		eventAuth:   eventAuthorityPDA(programID),
		cuLimit:     limit,
		cuPrice:     cfg.ComputeUnitPrice,
		commitment:  commitment,
		signer:      signer,
		nodePub:     nodePub,
		feePayer:    feePayer,
		feePayerPub: payerPub,
	}, nil
}

// rotPacked is the canonical rotation payload: the new signer set + threshold
// and the signer nonce the digest is bound to.
type rotPacked struct {
	NewSigners   []string `json:"newSigners"` // base58, ascending
	NewThreshold uint8    `json:"newThreshold"`
	SignerNonce  uint64   `json:"signerNonce"`
}

// Pack reads the live signer nonce and returns the canonical JSON for rotating
// to newSigners / newThreshold. opID is ignored: Solana binds rotation replay to
// the on-chain program signer nonce, so the operation identity is not embedded
// in the payload.
func (f *RotationFinalizer) Pack(ctx context.Context, _ [32]byte, newSigners []string, newThreshold int) ([]byte, error) {
	pubs, err := parseRotationSigners(newSigners)
	if err != nil {
		return nil, err
	}
	thr, err := checkThreshold(newThreshold, len(pubs))
	if err != nil {
		return nil, err
	}
	cfg, err := fetchConfig(ctx, f.client, f.programID, f.commitment)
	if err != nil {
		return nil, err
	}
	p := rotPacked{NewSigners: make([]string, len(pubs)), NewThreshold: thr, SignerNonce: cfg.SignerNonce}
	for i, pk := range pubs {
		p.NewSigners[i] = pk.String()
	}
	return json.Marshal(p)
}

// Validate re-derives the rotation target from newSigners / newThreshold,
// asserts the packed payload matches it, and re-reads the live nonce to reject a
// packer that bound a stale or wrong signer nonce.
func (f *RotationFinalizer) Validate(ctx context.Context, _ [32]byte, packed []byte, newSigners []string, newThreshold int) error {
	var got rotPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("sol: decode packed: %w", err)
	}
	pubs, err := parseRotationSigners(newSigners)
	if err != nil {
		return err
	}
	thr, err := checkThreshold(newThreshold, len(pubs))
	if err != nil {
		return err
	}
	want := make([]string, len(pubs))
	for i, pk := range pubs {
		want[i] = pk.String()
	}
	if got.NewThreshold != thr || !equalStrings(got.NewSigners, want) {
		return fmt.Errorf("sol: packed rotation does not match request")
	}
	cfg, err := fetchConfig(ctx, f.client, f.programID, f.commitment)
	if err != nil {
		return err
	}
	if got.SignerNonce != cfg.SignerNonce {
		return fmt.Errorf("sol: packed signer nonce %d != live %d", got.SignerNonce, cfg.SignerNonce)
	}
	return nil
}

// Sign returns this node's share: nodePubkey(32) ‖ ed25519 signature(64) over
// the rotation digest.
func (f *RotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	sig, err := f.signer.Sign(ctx, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sol: sign rotation digest: %w", err)
	}
	if len(sig) != 64 {
		return nil, fmt.Errorf("sol: ed25519 signature must be 64 bytes, got %d", len(sig))
	}
	share := make([]byte, shareLen)
	copy(share[:32], f.nodePub[:])
	copy(share[32:], sig)
	return share, nil
}

// Submit filters the collected shares against the live (outgoing) signer set,
// assembles the Ed25519-precompile + update_signers transaction, and broadcasts
// it (fee-payer signed). Idempotent: if the rotation already applied it returns
// without re-submitting.
func (f *RotationFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (core.TxRef, error) {
	var p rotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return core.TxRef{}, fmt.Errorf("sol: decode packed: %w", err)
	}
	newPubs, err := parseRotationSigners(p.NewSigners)
	if err != nil {
		return core.TxRef{}, err
	}
	if _, done, _ := f.VerifyRotation(ctx, p.NewSigners, int(p.NewThreshold)); done {
		return core.TxRef{}, nil
	}

	cfg, err := fetchConfig(ctx, f.client, f.programID, f.commitment)
	if err != nil {
		return core.TxRef{}, err
	}
	pubkeys, sigs, err := assembleQuorum(shares, cfg.Signers, int(cfg.Threshold))
	if err != nil {
		return core.TxRef{}, err
	}

	commitment := SignersCommitment(newPubs, p.NewThreshold)
	digest := RotateDigest(f.chainID, f.programID, f.configPDA, commitment, p.SignerNonce)
	ed25519Ix, err := BuildEd25519Instruction(pubkeys, sigs, digest[:])
	if err != nil {
		return core.TxRef{}, err
	}
	leading := []solana.Instruction{
		computebudget.NewSetComputeUnitLimitInstruction(f.cuLimit).Build(),
		computebudget.NewSetComputeUnitPriceInstruction(f.cuPrice).Build(),
	}
	sigIxIndex := uint8(len(leading))
	updateIx, err := custody.NewUpdateSignersInstruction(
		newPubs, p.NewThreshold, sigIxIndex,
		f.configPDA, solana.SysVarInstructionsPubkey, f.eventAuth, f.programID,
	)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("sol: build update_signers ix: %w", err)
	}
	instructions := append(leading, ed25519Ix, updateIx)

	sig, err := signAndSend(ctx, f.client, instructions, f.feePayerPub, f.feePayer, f.commitment)
	if err != nil {
		if _, done, verr := f.VerifyRotation(ctx, p.NewSigners, int(p.NewThreshold)); verr == nil && done {
			return core.TxRef{}, nil
		}
		return core.TxRef{}, err
	}
	// Block until the Config reflects the new set, so the returned ref
	// corresponds to an applied rotation (mirrors the withdrawal finalizer).
	if err := f.waitRotated(ctx, p.NewSigners, int(p.NewThreshold)); err != nil {
		return core.TxRef{}, err
	}
	return txRef(sig), nil
}

func (f *RotationFinalizer) waitRotated(ctx context.Context, newSigners []string, newThreshold int) error {
	deadline := time.Now().Add(confirmTimeout)
	for {
		if _, done, _ := f.VerifyRotation(ctx, newSigners, newThreshold); done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("sol: rotation not applied within %s", confirmTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(confirmPollInterval):
		}
	}
}

// VerifyRotation reports whether the on-chain Config now holds exactly the
// requested signer set + threshold. Binary — no tx hash is recoverable from the
// Config account, so a zero hash is returned with done=true.
func (f *RotationFinalizer) VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) ([32]byte, bool, error) {
	pubs, err := parseRotationSigners(newSigners)
	if err != nil {
		return [32]byte{}, false, err
	}
	thr, err := checkThreshold(newThreshold, len(pubs))
	if err != nil {
		return [32]byte{}, false, err
	}
	cfg, err := fetchConfig(ctx, f.client, f.programID, f.commitment)
	if err != nil {
		return [32]byte{}, false, err
	}
	if cfg.Threshold != thr || len(cfg.Signers) != len(pubs) {
		return [32]byte{}, false, nil
	}
	for i := range pubs {
		if cfg.Signers[i] != pubs[i] {
			return [32]byte{}, false, nil
		}
	}
	return [32]byte{}, true, nil
}

// --- helpers ---

func (f *RotationFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p rotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("sol: decode packed: %w", err)
	}
	pubs, err := parseRotationSigners(p.NewSigners)
	if err != nil {
		return [32]byte{}, err
	}
	commitment := SignersCommitment(pubs, p.NewThreshold)
	return RotateDigest(f.chainID, f.programID, f.configPDA, commitment, p.SignerNonce), nil
}

// parseRotationSigners decodes the incoming signer set (base58 or 32-byte hex)
// into solana pubkeys sorted ascending — the order the program stores and the
// commitment binds. Rejects duplicates.
func parseRotationSigners(newSigners []string) ([]solana.PublicKey, error) {
	if len(newSigners) == 0 {
		return nil, fmt.Errorf("sol: empty new signer set")
	}
	out := make([]solana.PublicKey, 0, len(newSigners))
	seen := make(map[solana.PublicKey]struct{}, len(newSigners))
	for _, s := range newSigners {
		pk, err := parsePubkey(s)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[pk]; dup {
			return nil, fmt.Errorf("sol: duplicate signer %s", pk)
		}
		seen[pk] = struct{}{}
		out = append(out, pk)
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i][:], out[j][:]) < 0 })
	return out, nil
}

// parsePubkey accepts a 32-byte hex string or a base58 pubkey.
func parsePubkey(s string) (solana.PublicKey, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return solana.PublicKeyFromBytes(b), nil
	}
	pk, err := solana.PublicKeyFromBase58(s)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("sol: signer %q is neither 32-byte hex nor base58: %w", s, err)
	}
	return pk, nil
}

func checkThreshold(newThreshold, n int) (uint8, error) {
	if newThreshold <= 0 || newThreshold > n {
		return 0, fmt.Errorf("sol: threshold %d out of range for %d signers", newThreshold, n)
	}
	if n > 255 {
		return 0, fmt.Errorf("sol: too many signers (%d)", n)
	}
	return uint8(newThreshold), nil
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

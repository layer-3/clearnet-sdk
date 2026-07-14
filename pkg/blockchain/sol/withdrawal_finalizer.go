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
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// shareLen is a per-signer share: ed25519 pubkey (32) ‖ signature (64).
const shareLen = 96

const (
	defaultComputeUnitLimit = uint32(200_000)
	confirmPollInterval     = 1 * time.Second
	confirmTimeout          = 30 * time.Second
)

// Config tunes the submit transaction.
type Config struct {
	ChainID          uint64
	ComputeUnitLimit uint32 // 0 → 200k
	ComputeUnitPrice uint64 // micro-lamports per CU (priority fee)
	// Commitment is the level at which on-chain reads (Config, the Withdrawal
	// PDA) are observed. Empty → CommitmentFinalized.
	//
	// NOTE: production should use CommitmentFinalized — a withdrawal is only
	// truly settled once finalized (no rollback). CommitmentConfirmed is a
	// devnet/test speed tradeoff: it observes results in ~1-2 slots instead of
	// waiting ~32 for finality, cutting the local flow from ~16s to a few.
	Commitment rpc.CommitmentType
	// AddressLookupTable, when set, makes the submit emit a v0 transaction using
	// this ALT. Required for large quorums: the Ed25519 instruction grows ~112
	// bytes per signer, so beyond ~8-9 signers a legacy transaction exceeds the
	// 1232-byte packet limit. Zero → legacy transaction.
	AddressLookupTable solana.PublicKey
}

// WithdrawalFinalizer executes a withdrawal against the custody Anchor program:
// a digest signed by the ed25519 quorum, verified on-chain via the Ed25519
// precompile. The node's signer contributes one share; a separate fee-payer
// signer pays + submits. It implements core.VaultWithdrawalFinalizer, for both
// native SOL and SPL tokens (the SPL path adds the recipient-ATA creation and
// the token remaining-accounts the program's execute expects).
type WithdrawalFinalizer struct {
	client      *rpc.Client
	programID   solana.PublicKey
	chainID     uint64
	vaultPDA    solana.PublicKey
	configPDA   solana.PublicKey
	eventAuth   solana.PublicKey
	cuLimit     uint32
	cuPrice     uint64
	commitment  rpc.CommitmentType
	alt         solana.PublicKey
	signer      sign.Signer
	nodePub     solana.PublicKey
	feePayer    sign.Signer
	feePayerPub solana.PublicKey
	assets      blockchain.AssetResolver
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer builds the finalizer. signer is this node's ed25519
// custody key (contributes a quorum share); feePayer is a distinct ed25519 key
// that pays for and submits the execute transaction.
func NewWithdrawalFinalizer(rpcURL string, programID solana.PublicKey, signer, feePayer sign.Signer, cfg Config, assets blockchain.AssetResolver) (*WithdrawalFinalizer, error) {
	if assets == nil {
		return nil, fmt.Errorf("sol: asset resolver is required")
	}
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
	return &WithdrawalFinalizer{
		client:      rpc.New(rpcURL),
		programID:   programID,
		chainID:     cfg.ChainID,
		vaultPDA:    VaultPDA(programID),
		configPDA:   ConfigPDA(programID),
		eventAuth:   eventAuthorityPDA(programID),
		cuLimit:     limit,
		cuPrice:     cfg.ComputeUnitPrice,
		commitment:  commitment,
		alt:         cfg.AddressLookupTable,
		signer:      signer,
		nodePub:     nodePub,
		feePayer:    feePayer,
		feePayerPub: payerPub,
		assets:      assets,
	}, nil
}

type solPacked struct {
	To           string `json:"to"`           // recipient (base58)
	Mint         string `json:"mint"`         // mint (base58); zero pubkey = native SOL
	Amount       uint64 `json:"amount"`       // base units / lamports
	WithdrawalID string `json:"withdrawalId"` // 32-byte hex
	Deadline     int64  `json:"deadline"`     // unix seconds; authorization void past this
}

// Pack resolves the withdrawal target and returns the canonical JSON.
func (f *WithdrawalFinalizer) Pack(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte, deadline int64) ([]byte, error) {
	p, err := f.packedFromOp(ctx, op, withdrawalID, deadline)
	if err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

// Validate re-derives the canonical payload from the op and the caller-supplied
// deadline and asserts a match — a peer packing a different deadline than the
// quorum agreed is rejected here, before Sign.
func (f *WithdrawalFinalizer) Validate(ctx context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte, deadline int64) error {
	var got solPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("sol: decode packed: %w", err)
	}
	want, err := f.packedFromOp(ctx, op, withdrawalID, deadline)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("sol: packed withdrawal does not match op: got %+v want %+v", got, want)
	}
	return nil
}

// Sign returns this node's share: nodePubkey(32) ‖ ed25519 signature(64) over
// the withdrawal digest.
func (f *WithdrawalFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	sig, err := f.signer.Sign(ctx, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sol: sign digest: %w", err)
	}
	if len(sig) != 64 {
		return nil, fmt.Errorf("sol: ed25519 signature must be 64 bytes, got %d", len(sig))
	}
	share := make([]byte, shareLen)
	copy(share[:32], f.nodePub[:])
	copy(share[32:], sig)
	return share, nil
}

// merge filters the shares against the live on-chain signer set and orders +
// trims them to the quorum, returning the parallel ed25519 pubkeys / signatures.
func (f *WithdrawalFinalizer) merge(ctx context.Context, shares [][]byte) (pubkeys, sigs [][]byte, err error) {
	cfg, err := fetchConfig(ctx, f.client, f.programID, f.commitment)
	if err != nil {
		return nil, nil, err
	}
	return assembleQuorum(shares, cfg.Signers, int(cfg.Threshold))
}

// Submit filters + orders the collected shares against the live signer set,
// assembles the Ed25519-precompile + execute transaction, and broadcasts it
// (fee-payer signed), then waits for the Withdrawal PDA to appear.
func (f *WithdrawalFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (core.TxRef, error) {
	var p solPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return core.TxRef{}, fmt.Errorf("sol: decode packed: %w", err)
	}
	to, mint, amount, wid, deadline, err := decodePacked(p)
	if err != nil {
		return core.TxRef{}, err
	}

	pubkeys, sigs, err := f.merge(ctx, shares)
	if err != nil {
		return core.TxRef{}, err
	}

	digest := WithdrawDigest(f.chainID, f.programID, f.vaultPDA, to, mint, amount, wid, deadline)
	ed25519Ix, err := BuildEd25519Instruction(pubkeys, sigs, digest[:])
	if err != nil {
		return core.TxRef{}, err
	}
	// Leading instructions before the Ed25519 companion: the two compute-budget
	// instructions, plus (SPL only) an idempotent recipient-ATA creation paid by
	// the fee payer (the recipient may not have a token account yet). sigIxIndex
	// — which execute introspects to find its Ed25519 companion — is the count of
	// these leading instructions.
	leading := []solana.Instruction{
		computebudget.NewSetComputeUnitLimitInstruction(f.cuLimit).Build(),
		computebudget.NewSetComputeUnitPriceInstruction(f.cuPrice).Build(),
	}
	if !mint.IsZero() {
		leading = append(leading,
			associatedtokenaccount.NewCreateIdempotentInstruction(f.feePayerPub, to, mint).Build())
	}
	sigIxIndex := uint8(len(leading))
	execIx, err := f.buildExecuteIx(to, mint, amount, wid, sigIxIndex, deadline)
	if err != nil {
		return core.TxRef{}, err
	}
	instructions := append(leading, ed25519Ix, execIx)

	sig, err := signAndSend(ctx, f.client, instructions, f.feePayerPub, f.feePayer, f.commitment, f.alt)
	if err != nil {
		// A peer may have already landed it.
		if h, executed, verr := f.VerifyExecution(ctx, wid); verr == nil && executed {
			return core.TxRef{Hash: h}, nil
		}
		return core.TxRef{}, err
	}
	if err := f.waitExecuted(ctx, wid); err != nil {
		return core.TxRef{}, err
	}
	return txRef(sig), nil
}

// buildExecuteIx builds the execute instruction. The native account list comes
// from the generated binding; for an SPL withdrawal it appends the token
// remaining-accounts the program's execute expects (token program, vault ATA,
// recipient ATA), reusing the binding's data encoding so the discriminator and
// arguments stay byte-exact.
func (f *WithdrawalFinalizer) buildExecuteIx(to, mint solana.PublicKey, amount uint64, wid [32]byte, sigIxIndex uint8, deadline int64) (solana.Instruction, error) {
	execIx, err := custody.NewExecuteInstruction(
		to, mint, amount, wid, sigIxIndex, deadline,
		f.feePayerPub, f.configPDA, f.vaultPDA, WithdrawalPDA(f.programID, wid),
		to, solana.SysVarInstructionsPubkey, solana.SystemProgramID, f.eventAuth, f.programID,
	)
	if err != nil {
		return nil, fmt.Errorf("sol: build execute ix: %w", err)
	}
	if mint.IsZero() {
		return execIx, nil
	}
	vaultATA, _, err := solana.FindAssociatedTokenAddress(f.vaultPDA, mint)
	if err != nil {
		return nil, fmt.Errorf("sol: vault ATA: %w", err)
	}
	recipientATA, _, err := solana.FindAssociatedTokenAddress(to, mint)
	if err != nil {
		return nil, fmt.Errorf("sol: recipient ATA: %w", err)
	}
	data, err := execIx.Data()
	if err != nil {
		return nil, fmt.Errorf("sol: execute data: %w", err)
	}
	metas := append(execIx.Accounts(),
		solana.NewAccountMeta(solana.TokenProgramID, false, false),
		solana.NewAccountMeta(vaultATA, true, false),
		solana.NewAccountMeta(recipientATA, true, false),
	)
	return solana.NewInstruction(f.programID, metas, data), nil
}

// VerifyExecution reports whether the Withdrawal PDA exists (the on-chain
// executed flag). The tx hash is not recoverable from the PDA alone, so a zero
// hash is returned with executed=true.
func (f *WithdrawalFinalizer) VerifyExecution(ctx context.Context, withdrawalID [32]byte) ([32]byte, bool, error) {
	info, err := f.client.GetAccountInfoWithOpts(ctx, WithdrawalPDA(f.programID, withdrawalID), &rpc.GetAccountInfoOpts{Commitment: f.commitment})
	if err != nil {
		// solana-go returns an error for a missing account; treat as not-found.
		if err == rpc.ErrNotFound {
			return [32]byte{}, false, nil
		}
		return [32]byte{}, false, nil
	}
	if info == nil || info.Value == nil {
		return [32]byte{}, false, nil
	}
	return [32]byte{}, true, nil
}

func (f *WithdrawalFinalizer) waitExecuted(ctx context.Context, withdrawalID [32]byte) error {
	deadline := time.Now().Add(confirmTimeout)
	for {
		if _, executed, _ := f.VerifyExecution(ctx, withdrawalID); executed {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("sol: withdrawal %x not executed within %s", withdrawalID, confirmTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(confirmPollInterval):
		}
	}
}

// --- helpers ---

func (f *WithdrawalFinalizer) packedFromOp(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte, deadline int64) (solPacked, error) {
	to, err := solana.PublicKeyFromBase58(op.Recipient)
	if err != nil {
		return solPacked{}, fmt.Errorf("sol: recipient %q not base58: %w", op.Recipient, err)
	}
	id, err := blockchain.AssetIDFromURI(op.AssetURI)
	if err != nil {
		return solPacked{}, fmt.Errorf("sol: asset URI: %w", err)
	}
	if id.Family != blockchain.ChainFamilySOL {
		return solPacked{}, fmt.Errorf("sol: asset family %q does not match %q", id.Family, blockchain.ChainFamilySOL)
	}
	if id.ChainID != 0 {
		return solPacked{}, fmt.Errorf("sol: asset chain id must be 0, got %d", id.ChainID)
	}
	if err := f.assets.ValidateAssetAddress(ctx, id.AssetAddress); err != nil {
		return solPacked{}, err
	}
	decimals, err := f.assets.AssetDecimals(ctx, id.AssetAddress)
	if err != nil {
		return solPacked{}, err
	}
	amt, err := blockchain.DecimalToBaseUnits(op.Amount, decimals)
	if err != nil {
		return solPacked{}, fmt.Errorf("sol: amount: %w", err)
	}
	if !amt.IsUint64() || amt.Sign() <= 0 {
		return solPacked{}, fmt.Errorf("sol: amount %s not a positive uint64 base-unit value", op.Amount.String())
	}
	mint, err := resolveMint(id.AssetAddress)
	if err != nil {
		return solPacked{}, err
	}
	return solPacked{
		To:           to.String(),
		Mint:         mint.String(),
		Amount:       amt.Uint64(),
		WithdrawalID: hex.EncodeToString(withdrawalID[:]),
		Deadline:     deadline,
	}, nil
}

func (f *WithdrawalFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p solPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("sol: decode packed: %w", err)
	}
	to, mint, amount, wid, deadline, err := decodePacked(p)
	if err != nil {
		return [32]byte{}, err
	}
	return WithdrawDigest(f.chainID, f.programID, f.vaultPDA, to, mint, amount, wid, deadline), nil
}

func decodePacked(p solPacked) (to, mint solana.PublicKey, amount uint64, wid [32]byte, deadline int64, err error) {
	if to, err = solana.PublicKeyFromBase58(p.To); err != nil {
		return
	}
	if mint, err = solana.PublicKeyFromBase58(p.Mint); err != nil {
		return
	}
	b, e := hex.DecodeString(p.WithdrawalID)
	if e != nil || len(b) != 32 {
		err = fmt.Errorf("sol: bad withdrawalID %q", p.WithdrawalID)
		return
	}
	copy(wid[:], b)
	amount = p.Amount
	deadline = p.Deadline
	return
}

// assembleQuorum filters shares to the authorized signer set, dedups, orders by
// pubkey ascending (the program's verifier walks them in order), and trims to
// the threshold.
func assembleQuorum(shares [][]byte, authorized []solana.PublicKey, threshold int) (pubkeys, sigs [][]byte, err error) {
	auth := make(map[solana.PublicKey]struct{}, len(authorized))
	for _, s := range authorized {
		auth[s] = struct{}{}
	}
	type item struct {
		pub solana.PublicKey
		sig []byte
	}
	seen := make(map[solana.PublicKey]struct{})
	var items []item
	for _, sh := range shares {
		if len(sh) != shareLen {
			continue
		}
		var pub solana.PublicKey
		copy(pub[:], sh[:32])
		if _, ok := auth[pub]; !ok {
			continue
		}
		if _, dup := seen[pub]; dup {
			continue
		}
		seen[pub] = struct{}{}
		items = append(items, item{pub: pub, sig: append([]byte(nil), sh[32:96]...)})
	}
	if len(items) < threshold {
		return nil, nil, fmt.Errorf("sol: only %d of %d authorized shares", len(items), threshold)
	}
	sort.Slice(items, func(i, j int) bool { return bytes.Compare(items[i].pub[:], items[j].pub[:]) < 0 })
	items = items[:threshold]
	for _, it := range items {
		pk := it.pub
		pubkeys = append(pubkeys, append([]byte(nil), pk[:]...))
		sigs = append(sigs, it.sig)
	}
	return pubkeys, sigs, nil
}

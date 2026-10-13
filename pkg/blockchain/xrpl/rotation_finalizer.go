package xrpl

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Peersyst/xrpl-go/xrpl/queries/account"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// RotationFinalizer rotates the vault account's SignerList via a multi-signed
// SignerListSet — the rotation analogue of WithdrawalFinalizer. It owns the
// node's signer; the quorum's blobs are merged off-mesh by the caller. It
// implements core.SignerRotationFinalizer.
//
// Replay defense is the vault account's own Sequence (autofilled into the
// SignerListSet), so there is no separate nonce: a given Sequence applies once.
type RotationFinalizer struct {
	rpcURL string
	// client holds the cached NetworkID; each operation's requests go through
	// a context-bound client from rpc (see rpcCall).
	client       *rpc.Client
	vaultAddress string
	threshold    int // current SignerQuorum — sizes the multi-sign fee
	signer       sign.Signer
	id           Identity

	// resolveThreshold, when set, supplies the live SignerQuorum used to size the
	// multi-sign fee autofill in Pack and the multisign fee floor in Validate.
	// Lets a quorum-raising rotation pay a fee sized against the vault's current
	// (outgoing) quorum rather than a boot-time threshold. It must read the
	// validated live SignerList. nil makes Pack fall back to threshold and
	// Validate read the validated SignerList itself.
	resolveThreshold func(context.Context) (int, error)

	// resolveBaseFee reads the validated ledger's base fee for the multisign fee
	// floor in Validate. Defaults to validated server_state over live RPC.
	resolveBaseFee BaseFeeResolver

	// resolveLedger reads the current validated LedgerState, used to set (Pack) and
	// bound (Validate) LastLedgerSequence so a signed-but-unbroadcast rotation blob
	// dies at the ledger level. Rotations have no withdrawal deadline, so
	// the budget is ledger-based only. Defaults to a live-RPC resolver.
	resolveLedger LedgerStateResolver

	// standalone marks a test-only standalone rippled (manual ledger_accept): LLS
	// is omitted / required-absent. TEST-ONLY.
	standalone bool
}

// SetThresholdResolver installs a hook that resolves the live SignerQuorum used
// to size the fee autofill and the multisign fee floor. It must read the
// validated live SignerList. Optional; unset uses the static construction-time
// threshold for autofill and a validated SignerList read for the floor.
func (f *RotationFinalizer) SetThresholdResolver(fn func(context.Context) (int, error)) {
	f.resolveThreshold = fn
}

// SetBaseFeeResolver overrides the validated base-fee read used for the
// multisign fee floor. Optional; unset reads validated server_state over RPC.
func (f *RotationFinalizer) SetBaseFeeResolver(fn BaseFeeResolver) {
	f.resolveBaseFee = fn
}

// SetLedgerStateResolver overrides the current-ledger resolver used to set and
// bound LastLedgerSequence. Optional; unset uses a live-RPC resolver.
func (f *RotationFinalizer) SetLedgerStateResolver(fn LedgerStateResolver) {
	f.resolveLedger = fn
}

// SetStandaloneLedgerMode toggles standalone-rippled behavior: LastLedgerSequence
// is omitted on build and required-absent on validate. TEST-ONLY.
func (f *RotationFinalizer) SetStandaloneLedgerMode(v bool) {
	f.standalone = v
}

// rpc returns a client whose requests run under ctx (see rpcCall).
func (f *RotationFinalizer) rpc(ctx context.Context) (*rpc.Client, error) {
	return rpcCall(ctx, f.rpcURL, f.client)
}

// ledgerState resolves the current validated LedgerState, using the injected
// resolver or a validated live-RPC default.
func (f *RotationFinalizer) ledgerState(ctx context.Context) (LedgerState, error) {
	if f.resolveLedger != nil {
		return f.resolveLedger(ctx)
	}
	call, err := f.rpc(ctx)
	if err != nil {
		return LedgerState{}, err
	}
	return validatedLedgerState(call)
}

// LiveQuorum returns the SignerQuorum of the vault's SignerList in the latest
// validated ledger. Callers wire it as the ThresholdResolver (and reuse it for
// the ceremony collect count) so a quorum-raising rotation sizes the fee, the
// multisign fee floor and the quorum against validated live state rather than
// the boot-time threshold. A failed or unvalidated read is an error; a rotation
// that has not yet validated is not reflected.
func (f *RotationFinalizer) LiveQuorum(ctx context.Context) (int, error) {
	call, err := f.rpc(ctx)
	if err != nil {
		return 0, err
	}
	return fetchValidatedSignerQuorum(call, f.vaultAddress)
}

var _ core.SignerRotationFinalizer = (*RotationFinalizer)(nil)

// NewRotationFinalizer builds the XRPL rotation finalizer. threshold is the
// current SignerQuorum (used to size the multi-sign fee and trim the quorum);
// signer is one of the current SignerList members.
func NewRotationFinalizer(rpcURL, vaultAddress string, threshold int, signer sign.Signer) (*RotationFinalizer, error) {
	client, err := newRPCClient(rpcURL)
	if err != nil {
		return nil, err
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &RotationFinalizer{
		rpcURL:       rpcURL,
		client:       client,
		vaultAddress: vaultAddress,
		threshold:    threshold,
		signer:       signer,
		id:           id,
	}, nil
}

// Pack builds the autofilled multi-sign SignerListSet installing newSigners /
// newThreshold (each member weight 1), returning its sorted-key JSON. opID is
// ignored: XRPL binds rotation replay to the account Sequence (autofilled here),
// so the operation identity is not embedded in the payload.
func (f *RotationFinalizer) Pack(ctx context.Context, _ [32]byte, newSigners []string, newThreshold int) ([]byte, error) {
	entries, err := signerEntries(newSigners, newThreshold)
	if err != nil {
		return nil, err
	}
	flatTx := transaction.FlatTransaction{
		"TransactionType": "SignerListSet",
		"Account":         f.vaultAddress,
		"SignerQuorum":    uint32(newThreshold),
		"SignerEntries":   entries,
	}
	quorum := f.threshold
	if f.resolveThreshold != nil {
		quorum, err = f.resolveThreshold(ctx)
		if err != nil {
			return nil, fmt.Errorf("xrpl: resolve live quorum: %w", err)
		}
	}
	call, err := f.rpc(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := resolveNetworkIDPolicy(f.client, call); err != nil {
		return nil, err
	}
	if err := call.AutofillMultisigned(&flatTx, uint64(quorum)); err != nil {
		return nil, fmt.Errorf("xrpl: autofill: %w", err)
	}
	// Replace autofill's LastLedgerSequence with a ledger-budget bound (or drop it
	// in standalone mode). No withdrawal deadline applies to a rotation, so the
	// budget is ledger-based only.
	if f.standalone {
		delete(flatTx, "LastLedgerSequence")
	} else {
		state, err := f.ledgerState(ctx)
		if err != nil {
			return nil, err
		}
		lls, err := buildLLS(state, 0)
		if err != nil {
			return nil, err
		}
		flatTx["LastLedgerSequence"] = lls
	}
	return CanonicalJSON(flatTx)
}

// Validate asserts the packed SignerListSet rotates to exactly the requested set,
// that its LastLedgerSequence is inside the follower's ledger band, and that
// its Fee is at least the multisign floor. The CURRENT SignerList authorizes a
// SignerListSet, so the floor uses the current validated live quorum, not
// newThreshold. A failed floor lookup is an operational error, not
// ErrCandidateRejected.
func (f *RotationFinalizer) Validate(ctx context.Context, _ [32]byte, packed []byte, newSigners []string, newThreshold int) error {
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		return candidateRejected(fmt.Errorf("xrpl: decode packed: %w", err))
	}
	policy := llsPolicy{standalone: f.standalone}
	if !f.standalone {
		state, err := f.ledgerState(ctx)
		if err != nil {
			return err
		}
		policy.current = state
	}
	call, err := f.rpc(ctx)
	if err != nil {
		return err
	}
	minFee, err := resolveFeeFloor(ctx, call, f.vaultAddress, f.resolveThreshold, f.resolveBaseFee)
	if err != nil {
		return err
	}
	networkPolicy, err := resolveNetworkIDPolicy(f.client, call)
	if err != nil {
		return err
	}
	if err := ValidateNetworkID(flat, networkPolicy); err != nil {
		return candidateRejected(err)
	}
	if err := validateCanonicalRotation(flat, newSigners, newThreshold, f.vaultAddress, policy, minFee); err != nil {
		return candidateRejected(err)
	}
	return nil
}

// Sign multi-signs the packed SignerListSet and returns this node's blob.
func (f *RotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		return nil, fmt.Errorf("xrpl: decode packed: %w", err)
	}
	blob, err := signMultisig(ctx, f.signer, f.id, flat)
	if err != nil {
		return nil, err
	}
	return []byte(blob), nil
}

// Submit combines the collected multi-sign blobs (trimmed to the current quorum)
// and broadcasts the SignerListSet, returning the txID.
func (f *RotationFinalizer) Submit(ctx context.Context, _ []byte, signatures [][]byte) (string, error) {
	call, err := f.rpc(ctx)
	if err != nil {
		return "", err
	}
	merged, err := combineLive(call, f.vaultAddress, signatures)
	if err != nil {
		return "", err
	}
	result, err := call.SubmitMultisigned(merged, false)
	if err != nil {
		return "", fmt.Errorf("xrpl: submit_multisigned: %w", err)
	}
	switch result.EngineResult {
	case "tesSUCCESS", "terQUEUED":
		hash, err := computeTxHash(result.TxBlob)
		if err != nil {
			return "", err
		}
		return hashHex(hash), nil
	default:
		return "", fmt.Errorf("xrpl: rotation rejected: %s - %s", result.EngineResult, result.EngineResultMessage)
	}
}

// VerifyRotation reads the vault's on-chain SignerList and reports whether it now
// holds exactly newSigners with SignerQuorum == newThreshold. Binary: no txID is
// recoverable from the SignerList object, so an empty txID is returned with
// done=true.
func (f *RotationFinalizer) VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) (string, bool, error) {
	call, err := f.rpc(ctx)
	if err != nil {
		return "", false, err
	}
	resp, err := call.GetAccountObjects(&account.ObjectsRequest{
		Account: types.Address(f.vaultAddress),
		Type:    account.SignerListObject,
	})
	if err != nil {
		return "", false, fmt.Errorf("xrpl rotation verify: account_objects: %w", err)
	}
	want := make(map[string]struct{}, len(newSigners))
	for _, s := range newSigners {
		want[s] = struct{}{}
	}
	for _, obj := range resp.AccountObjects {
		if asString(obj["LedgerEntryType"]) != "SignerList" {
			continue
		}
		quorum, ok := uint32Field(obj["SignerQuorum"])
		if !ok || int(quorum) != newThreshold {
			return "", false, nil
		}
		got, err := signerEntryAccounts(obj["SignerEntries"])
		if err != nil || len(got) != len(want) {
			return "", false, nil
		}
		for a := range got {
			if _, ok := want[a]; !ok {
				return "", false, nil
			}
		}
		return "", true, nil
	}
	return "", false, nil
}

// signerEntries builds the SignerListSet SignerEntries (each member weight 1),
// sorted by account ascending for a deterministic canonical payload. Validates
// the set is non-empty, duplicate-free, and structurally quorum-consistent.
// Policy beyond 1 <= k <= n belongs to the caller.
func signerEntries(newSigners []string, newThreshold int) ([]any, error) {
	if len(newSigners) == 0 {
		return nil, fmt.Errorf("xrpl: empty new signer set")
	}
	if newThreshold <= 0 || newThreshold > len(newSigners) {
		return nil, fmt.Errorf("xrpl: threshold %d out of range for %d signers", newThreshold, len(newSigners))
	}
	seen := make(map[string]struct{}, len(newSigners))
	sorted := make([]string, 0, len(newSigners))
	for _, s := range newSigners {
		if _, dup := seen[s]; dup {
			return nil, fmt.Errorf("xrpl: duplicate signer %s", s)
		}
		seen[s] = struct{}{}
		sorted = append(sorted, s)
	}
	sort.Strings(sorted)
	entries := make([]any, len(sorted))
	for i, a := range sorted {
		entries[i] = map[string]any{"SignerEntry": map[string]any{"Account": a, "SignerWeight": 1}}
	}
	return entries, nil
}

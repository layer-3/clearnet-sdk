package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/queries/account"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// executionScanPages bounds how many account_tx pages VerifyExecution reads.
const executionScanPages = 5

// TicketProvider supplies the Ticket sequence that authorizes a withdrawal.
// Custody backs this with its ticket pool/store; tests and simpler clients back
// it with a fixed or create-then-return ticket.
type TicketProvider interface {
	TicketFor(ctx context.Context, withdrawalID [32]byte) (uint32, error)
}

// WithdrawalFinalizer is the XRPL multi-sign vault withdrawal path. It owns the
// node's signer and a TicketProvider; the quorum's blobs are merged off-mesh by
// the caller. It implements core.VaultWithdrawalFinalizer.
type WithdrawalFinalizer struct {
	rpcURL string
	// networkID caches the live NetworkID across this finalizer's concurrent
	// operations; each operation's requests go through a context-bound client
	// from rpc (see rpcCall).
	networkID    networkIDCache
	vaultAddress string
	threshold    int
	signer       sign.Signer
	id           Identity
	tickets      TicketProvider
	assets       blockchain.AssetResolver

	// resolveThreshold, when set, supplies the live SignerQuorum used to size the
	// multi-sign fee autofill in Pack and the multisign fee floor in Validate. It
	// lets a quorum-raising rotation take effect without a fleet restart: the fee
	// is sized against the vault's current quorum rather than the boot-time
	// threshold. It must read the validated live SignerList. nil makes Pack fall
	// back to threshold and Validate read the validated SignerList itself.
	resolveThreshold func(context.Context) (int, error)

	// resolveBaseFee reads the validated ledger's base fee for the multisign fee
	// floor in Validate. Defaults to validated server_state over live RPC.
	resolveBaseFee BaseFeeResolver

	// resolveLedger reads the current validated LedgerState, used to set (Pack)
	// and bound (Validate) the withdrawal's LastLedgerSequence. Defaults
	// to a live-RPC resolver; injectable for followers/tests.
	resolveLedger LedgerStateResolver

	// standalone marks a test-only standalone rippled (manual ledger_accept):
	// ledgers do not advance on their own, so LLS-based expiry is meaningless and
	// the field is omitted / required-absent instead.
	standalone bool
}

// SetThresholdResolver installs a hook that resolves the live SignerQuorum used
// to size the fee autofill and the multisign fee floor (see resolveThreshold).
// It must read the validated live SignerList. Optional; callers that leave it
// unset get the static threshold for autofill and a validated SignerList read
// for the floor.
func (f *WithdrawalFinalizer) SetThresholdResolver(fn func(context.Context) (int, error)) {
	f.resolveThreshold = fn
}

// SetBaseFeeResolver overrides the validated base-fee read used for the
// multisign fee floor. Optional; unset reads validated server_state over RPC.
func (f *WithdrawalFinalizer) SetBaseFeeResolver(fn BaseFeeResolver) {
	f.resolveBaseFee = fn
}

// SetLedgerStateResolver overrides the current-ledger resolver used to set and
// bound LastLedgerSequence. Optional; unset uses a live-RPC resolver.
func (f *WithdrawalFinalizer) SetLedgerStateResolver(fn LedgerStateResolver) {
	f.resolveLedger = fn
}

// SetStandaloneLedgerMode toggles standalone-rippled behavior: LastLedgerSequence
// is omitted on build and required-absent on validate. TEST-ONLY — enabling it in
// production removes XRPL withdrawal expiry.
func (f *WithdrawalFinalizer) SetStandaloneLedgerMode(v bool) {
	f.standalone = v
}

// rpc returns a client whose requests run under ctx (see rpcCall).
func (f *WithdrawalFinalizer) rpc(ctx context.Context) (*rpc.Client, error) {
	return rpcCall(ctx, f.rpcURL, &f.networkID)
}

// ledgerState resolves the current validated LedgerState, using the injected
// resolver or a validated live-RPC default.
func (f *WithdrawalFinalizer) ledgerState(ctx context.Context) (LedgerState, error) {
	if f.resolveLedger != nil {
		return f.resolveLedger(ctx)
	}
	call, err := f.rpc(ctx)
	if err != nil {
		return LedgerState{}, err
	}
	return validatedLedgerState(call)
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer builds the XRPL vault finalizer. threshold is the
// SignerQuorum; tickets authorizes each withdrawal's TicketSequence.
func NewWithdrawalFinalizer(rpcURL, vaultAddress string, threshold int, signer sign.Signer, tickets TicketProvider, assets blockchain.AssetResolver) (*WithdrawalFinalizer, error) {
	if assets == nil {
		return nil, fmt.Errorf("xrpl: asset resolver is required")
	}
	if _, err := newRPCClient(rpcURL); err != nil {
		return nil, err
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalFinalizer{
		rpcURL:       rpcURL,
		vaultAddress: vaultAddress,
		threshold:    threshold,
		signer:       signer,
		id:           id,
		tickets:      tickets,
		assets:       assets,
	}, nil
}

// LiveQuorum returns the SignerQuorum of the vault's SignerList in the latest
// validated ledger. Callers wire it as the ThresholdResolver (and reuse it for
// the ceremony collect count) so a quorum-raising rotation sizes the fee, the
// multisign fee floor and the quorum against validated live state rather than
// the boot-time threshold. A failed or unvalidated read is an error; a rotation
// that has not yet validated is not reflected.
func (f *WithdrawalFinalizer) LiveQuorum(ctx context.Context) (int, error) {
	call, err := f.rpc(ctx)
	if err != nil {
		return 0, err
	}
	return fetchValidatedSignerQuorum(call, f.vaultAddress)
}

// feeQuorum returns the SignerQuorum used to size the multi-sign fee: the live
// value from resolveThreshold when set, else the static threshold.
func (f *WithdrawalFinalizer) feeQuorum(ctx context.Context) (int, error) {
	if f.resolveThreshold == nil {
		return f.threshold, nil
	}
	q, err := f.resolveThreshold(ctx)
	if err != nil {
		return 0, fmt.Errorf("xrpl: resolve live quorum: %w", err)
	}
	return q, nil
}

// Pack binds a Ticket and builds the autofilled multi-sign Payment, returning
// its sorted-key JSON. It sets LastLedgerSequence to current + a per-attempt
// budget clamped so the tx's estimated close is at or before ValidUntil
// ; if too little budget remains it fails and the withdrawal parks.
func (f *WithdrawalFinalizer) Pack(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte, bounds core.WithdrawalTimeBounds) ([]byte, error) {
	if err := bounds.Validate(); err != nil {
		return nil, err
	}
	amount, err := BuildAmount(ctx, f.assets, op)
	if err != nil {
		return nil, err
	}
	ticket, err := f.tickets.TicketFor(ctx, withdrawalID)
	if err != nil {
		return nil, fmt.Errorf("xrpl: ticket: %w", err)
	}
	payment := transaction.Payment{
		BaseTx: transaction.BaseTx{
			Account:        types.Address(f.vaultAddress),
			Sequence:       0,
			TicketSequence: ticket,
		},
		Destination: types.Address(op.Recipient),
		Amount:      amount,
		InvoiceID:   types.Hash256(strings.ToUpper(hex.EncodeToString(withdrawalID[:]))),
	}
	flatTx := payment.Flatten()
	flatTx["Sequence"] = uint32(0)
	quorum, err := f.feeQuorum(ctx)
	if err != nil {
		return nil, err
	}
	call, err := f.rpc(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := resolveNetworkIDPolicy(&f.networkID, call); err != nil {
		return nil, err
	}
	if err := call.AutofillMultisigned(&flatTx, uint64(quorum)); err != nil {
		return nil, fmt.Errorf("xrpl: autofill: %w", err)
	}
	// xrpl-go permits up to 2 XRP by default, while every follower refuses
	// a canonical fee above our 1 XRP ceiling. Never emit an unsignable body.
	if err := checkFee("packed payment", flatTx["Fee"], 1); err != nil {
		return nil, err
	}
	flatTx["Sequence"] = uint32(0)
	// Autofill sets its own LastLedgerSequence; replace it with our ValidUntil-bound
	// budget (or drop it entirely in standalone mode).
	if f.standalone {
		delete(flatTx, "LastLedgerSequence")
	} else {
		state, err := f.ledgerState(ctx)
		if err != nil {
			return nil, err
		}
		lls, err := buildLLS(state, bounds.ValidUntil)
		if err != nil {
			return nil, err
		}
		flatTx["LastLedgerSequence"] = lls
	}
	return CanonicalJSON(flatTx)
}

// Validate re-derives the trust-bound shape from the op and asserts the packed
// flatTx matches, including that LastLedgerSequence is inside this follower's
// ValidUntil-bound expiry band and that Fee is at least the multisign floor
// for the validated live quorum. A failed floor lookup is an operational error,
// not ErrCandidateRejected.
func (f *WithdrawalFinalizer) Validate(ctx context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte, bounds core.WithdrawalTimeBounds) error {
	if err := bounds.Validate(); err != nil {
		return err
	}
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		return candidateRejected(fmt.Errorf("xrpl: decode packed: %w", err))
	}
	if err := validatePaymentShape(flat, op, withdrawalID, f.vaultAddress); err != nil {
		return candidateRejected(err)
	}
	policy := llsPolicy{standalone: f.standalone, validUntil: bounds.ValidUntil}
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
	networkPolicy, err := resolveNetworkIDPolicy(&f.networkID, call)
	if err != nil {
		return err
	}
	if err := ValidateNetworkID(flat, networkPolicy); err != nil {
		return candidateRejected(err)
	}
	if err := ValidateCanonical(ctx, f.assets, flat, op, withdrawalID, f.vaultAddress, policy, minFee); err != nil {
		return candidateRejected(err)
	}
	return nil
}

// Sign multi-signs the packed Payment and returns this node's blob.
func (f *WithdrawalFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
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

// Submit combines the collected multi-sign blobs (trimmed to the quorum) and
// broadcasts the result, returning the txID.
func (f *WithdrawalFinalizer) Submit(ctx context.Context, _ []byte, signatures [][]byte) (string, error) {
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
		return "", fmt.Errorf("xrpl: submit rejected: %s - %s", result.EngineResult, result.EngineResultMessage)
	}
}

// VerifyExecution scans the vault's recent account_tx for a validated,
// successful outgoing Payment whose InvoiceID equals the withdrawalID. An
// InvoiceID on a tec-failed Payment is not proof of recipient credit. Custody
// uses its own watcher-ledger verifier; this bounded scan is for SDK consumers
// and returns an error if its horizon is exhausted instead of claiming absence.
func (f *WithdrawalFinalizer) VerifyExecution(ctx context.Context, withdrawalID [32]byte) (string, bool, error) {
	want := strings.ToUpper(hex.EncodeToString(withdrawalID[:]))
	call, err := f.rpc(ctx)
	if err != nil {
		return "", false, err
	}
	var marker any
	for page := 0; page < executionScanPages; page++ {
		resp, err := call.GetAccountTransactions(&account.TransactionsRequest{
			Account: types.Address(f.vaultAddress),
			Limit:   100,
			Marker:  marker,
		})
		if err != nil {
			return "", false, fmt.Errorf("xrpl verify: account_tx: %w", err)
		}
		for _, tx := range resp.Transactions {
			if !strings.EqualFold(asString(tx.Tx["InvoiceID"]), want) ||
				!tx.Validated || tx.Meta.TransactionResult != "tesSUCCESS" ||
				asString(tx.Tx["TransactionType"]) != "Payment" ||
				asString(tx.Tx["Account"]) != f.vaultAddress {
				continue
			}
			txID := strings.ToUpper(string(tx.Hash))
			if _, err := hex.DecodeString(txID); err != nil || len(txID) != 64 {
				return "", false, fmt.Errorf("xrpl verify: validated Payment has invalid transaction hash %q", tx.Hash)
			}
			return txID, true, nil
		}
		if resp.Marker == nil {
			return "", false, nil // Full validated account history was searched.
		}
		marker = resp.Marker
	}
	return "", false, fmt.Errorf("xrpl verify: account_tx scan exceeded %d pages; execution outcome is unknown", executionScanPages)
}

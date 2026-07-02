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
	client       *rpc.Client
	vaultAddress string
	threshold    int
	signer       sign.Signer
	id           Identity
	tickets      TicketProvider

	// resolveThreshold, when set, supplies the live SignerQuorum used to size the
	// multi-sign fee autofill in Pack. It lets a quorum-raising rotation take
	// effect without a fleet restart: the fee is sized against the vault's current
	// quorum rather than the boot-time threshold. nil falls back to threshold.
	resolveThreshold func(context.Context) (int, error)

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
// to size the fee autofill (see resolveThreshold). Optional; callers that leave
// it unset get the static threshold passed at construction.
func (f *WithdrawalFinalizer) SetThresholdResolver(fn func(context.Context) (int, error)) {
	f.resolveThreshold = fn
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

// ledgerState resolves the current validated LedgerState, using the injected
// resolver or a live-RPC default.
func (f *WithdrawalFinalizer) ledgerState(ctx context.Context) (LedgerState, error) {
	if f.resolveLedger != nil {
		return f.resolveLedger(ctx)
	}
	return clientLedgerState(f.client)(ctx)
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer builds the XRPL vault finalizer. threshold is the
// SignerQuorum; tickets authorizes each withdrawal's TicketSequence.
func NewWithdrawalFinalizer(rpcURL, vaultAddress string, threshold int, signer sign.Signer, tickets TicketProvider) (*WithdrawalFinalizer, error) {
	client, err := newRPCClient(rpcURL)
	if err != nil {
		return nil, err
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalFinalizer{
		client:       client,
		vaultAddress: vaultAddress,
		threshold:    threshold,
		signer:       signer,
		id:           id,
		tickets:      tickets,
	}, nil
}

// LiveQuorum returns the vault's current on-chain SignerQuorum. Callers wire it
// as the ThresholdResolver (and reuse it for the ceremony collect count) so a
// quorum-raising rotation sizes the fee and quorum against live state rather
// than the boot-time threshold.
func (f *WithdrawalFinalizer) LiveQuorum(_ context.Context) (int, error) {
	_, q, err := fetchLiveSignerList(f.client, f.vaultAddress)
	return q, err
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
// budget clamped so the tx's estimated close is at or before the deadline
//; if too little budget remains it fails and the withdrawal parks.
func (f *WithdrawalFinalizer) Pack(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte, deadline int64) ([]byte, error) {
	amount, err := BuildAmount(op)
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
	if err := ensureNetworkID(f.client); err != nil {
		return nil, err
	}
	if err := f.client.AutofillMultisigned(&flatTx, uint64(quorum)); err != nil {
		return nil, fmt.Errorf("xrpl: autofill: %w", err)
	}
	flatTx["Sequence"] = uint32(0)
	// Autofill sets its own LastLedgerSequence; replace it with our deadline-bound
	// budget (or drop it entirely in standalone mode).
	if f.standalone {
		delete(flatTx, "LastLedgerSequence")
	} else {
		state, err := f.ledgerState(ctx)
		if err != nil {
			return nil, err
		}
		lls, err := buildLLS(state, deadline)
		if err != nil {
			return nil, err
		}
		flatTx["LastLedgerSequence"] = lls
	}
	return CanonicalJSON(flatTx)
}

// Validate re-derives the trust-bound shape from the op and asserts the packed
// flatTx matches, including that LastLedgerSequence is inside this follower's
// deadline-bound expiry band.
func (f *WithdrawalFinalizer) Validate(ctx context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte, deadline int64) error {
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		return fmt.Errorf("xrpl: decode packed: %w", err)
	}
	policy := llsPolicy{standalone: f.standalone, deadline: deadline}
	if !f.standalone {
		state, err := f.ledgerState(ctx)
		if err != nil {
			return err
		}
		policy.current = state
	}
	return ValidateCanonical(flat, op, withdrawalID, f.vaultAddress, policy)
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
// broadcasts the result, returning the tx reference.
func (f *WithdrawalFinalizer) Submit(_ context.Context, _ []byte, signatures [][]byte) (core.TxRef, error) {
	merged, err := combineLive(f.client, f.vaultAddress, signatures)
	if err != nil {
		return core.TxRef{}, err
	}
	result, err := f.client.SubmitMultisigned(merged, false)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("xrpl: submit_multisigned: %w", err)
	}
	switch result.EngineResult {
	case "tesSUCCESS", "terQUEUED":
		hash, err := computeTxHash(result.TxBlob)
		if err != nil {
			return core.TxRef{}, err
		}
		return core.TxRef{Hash: hash, Raw: hashHex(hash)}, nil
	default:
		return core.TxRef{}, fmt.Errorf("xrpl: submit rejected: %s - %s", result.EngineResult, result.EngineResultMessage)
	}
}

// VerifyExecution scans the vault's recent account_tx for a Payment whose
// InvoiceID equals the withdrawalID, returning its tx hash + true.
func (f *WithdrawalFinalizer) VerifyExecution(ctx context.Context, withdrawalID [32]byte) ([32]byte, bool, error) {
	want := strings.ToUpper(hex.EncodeToString(withdrawalID[:]))
	var marker any
	for page := 0; page < executionScanPages; page++ {
		resp, err := f.client.GetAccountTransactions(&account.TransactionsRequest{
			Account: types.Address(f.vaultAddress),
			Limit:   100,
			Marker:  marker,
		})
		if err != nil {
			return [32]byte{}, false, fmt.Errorf("xrpl verify: account_tx: %w", err)
		}
		for _, tx := range resp.Transactions {
			if strings.EqualFold(asString(tx.Tx["InvoiceID"]), want) {
				h, err := hex.DecodeString(string(tx.Hash))
				if err != nil || len(h) != 32 {
					return [32]byte{}, true, nil // executed; hash unparseable
				}
				var out [32]byte
				copy(out[:], h)
				return out, true, nil
			}
		}
		if resp.Marker == nil {
			break
		}
		marker = resp.Marker
	}
	return [32]byte{}, false, nil
}

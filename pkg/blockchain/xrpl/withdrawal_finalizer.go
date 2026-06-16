package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl"
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
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer builds the XRPL vault finalizer. threshold is the
// SignerQuorum; tickets authorizes each withdrawal's TicketSequence.
func NewWithdrawalFinalizer(rpcURL, vaultAddress string, threshold int, signer sign.Signer, tickets TicketProvider) (*WithdrawalFinalizer, error) {
	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalFinalizer{
		client:       rpc.NewClient(cfg),
		vaultAddress: vaultAddress,
		threshold:    threshold,
		signer:       signer,
		id:           id,
		tickets:      tickets,
	}, nil
}

// Pack binds a Ticket and builds the autofilled multi-sign Payment, returning
// its sorted-key JSON.
func (f *WithdrawalFinalizer) Pack(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte) ([]byte, error) {
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
	if err := f.client.AutofillMultisigned(&flatTx, uint64(f.threshold)); err != nil {
		return nil, fmt.Errorf("xrpl: autofill: %w", err)
	}
	flatTx["Sequence"] = uint32(0)
	delete(flatTx, "LastLedgerSequence")
	return CanonicalJSON(flatTx)
}

// Validate re-derives the trust-bound shape from the op and asserts the packed
// flatTx matches.
func (f *WithdrawalFinalizer) Validate(_ context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte) error {
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		return fmt.Errorf("xrpl: decode packed: %w", err)
	}
	return ValidateCanonical(flat, op, withdrawalID, f.vaultAddress)
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

// merge combines the collected multi-sign blobs into one submittable blob.
// Exactly `threshold` signatures are included: Pack autofilled the multi-sign
// fee for that count (base × (1 + threshold)), so including extras would
// under-pay (telINSUF_FEE_P) and waste fee. Any threshold of the SignerList's
// members satisfies the quorum.
func (f *WithdrawalFinalizer) merge(signatures [][]byte) (string, error) {
	if len(signatures) < f.threshold {
		return "", fmt.Errorf("xrpl: have %d signatures, need %d", len(signatures), f.threshold)
	}
	blobs := make([]string, 0, f.threshold)
	for _, s := range signatures[:f.threshold] {
		blobs = append(blobs, string(s))
	}
	final, err := xrpl.Multisign(blobs...)
	if err != nil {
		return "", fmt.Errorf("xrpl: combine signatures: %w", err)
	}
	return final, nil
}

// Submit combines the collected multi-sign blobs and broadcasts the result,
// returning the tx reference.
func (f *WithdrawalFinalizer) Submit(_ context.Context, _ []byte, signatures [][]byte) (core.TxRef, error) {
	merged, err := f.merge(signatures)
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

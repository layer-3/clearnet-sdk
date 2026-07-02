// Package xrpl implements the XRP Ledger custody vault via Peersyst/xrpl-go:
// a depositor that sends tagged Payments, and a multi-sign withdrawal finalizer
// over a SignerList-configured vault account. Both take a sign.Signer; neither
// holds persistence or a mesh.
package xrpl

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
	binarycodec "github.com/Peersyst/xrpl-go/binary-codec"
	xrplcrypto "github.com/Peersyst/xrpl-go/pkg/crypto"
	"github.com/Peersyst/xrpl-go/xrpl"
	"github.com/Peersyst/xrpl-go/xrpl/queries/account"
	"github.com/Peersyst/xrpl-go/xrpl/queries/common"
	"github.com/Peersyst/xrpl-go/xrpl/queries/ledger"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// maxAcceptableFeeDrops caps the Fee a node will sign on a canonical Payment.
const maxAcceptableFeeDrops uint64 = 1_000_000

// LastLedgerSequence budget. XRPL has no per-tx timestamp expiry, so
// a signed-but-unbroadcast Payment would live as a standing bearer claim. Every
// canonical withdrawal instead carries a LastLedgerSequence: rippled drops the
// tx once the network passes that ledger, so each blob dies at the ledger level.
//
// The critical correctness property: the budget is per-attempt
// and small — derived from ExecutionMargin, NOT the full validity window — so
// every blob a retry can produce dies before SealedAt + MaxBlockAge + margin =
// deadline. A full-window budget would let a blob signed near the freshness
// cutoff outlive clearnet's re-credit and double-spend.
const (
	// LedgerBudget is the number of ledgers of headroom the builder targets when
	// setting LastLedgerSequence (~2min at ~4s/ledger).
	LedgerBudget = uint32(30)
	// MaxLedgerBudget is the upper bound a follower accepts on (LLS - current):
	// slack over LedgerBudget for the ledgers that close between build and
	// validation, still far under the deadline.
	MaxLedgerBudget = uint32(120)
	// minLedgerBudget is the floor: if fewer than this many ledgers fit before
	// the deadline, Pack fails and the withdrawal parks and expires cleanly
	// rather than being signed with a near-dead LLS.
	minLedgerBudget = uint32(5)
	// assumedLedgerCloseSec is the conservative average ledger close interval
	// used to translate a ledger budget into a wall-clock close-time estimate.
	assumedLedgerCloseSec = int64(4)
	// rippleEpochOffset converts a Ripple-epoch close time (seconds since
	// 2000-01-01 UTC) to a unix timestamp.
	rippleEpochOffset = int64(946684800)
)

// canonicalAllowedFields is the allowlist of top-level keys a node accepts on a
// canonical Payment flatTx before signing.
var canonicalAllowedFields = map[string]struct{}{
	"TransactionType": {}, "Account": {}, "Destination": {}, "Amount": {},
	"InvoiceID": {}, "TicketSequence": {}, "Sequence": {}, "Fee": {},
	"SigningPubKey": {}, "Flags": {}, "NetworkID": {}, "LastLedgerSequence": {},
}

// LedgerState is a snapshot of the network's current validated ledger: its index
// and close time (as a unix timestamp). Used to set and to bound a withdrawal's
// LastLedgerSequence against its deadline.
type LedgerState struct {
	ValidatedIndex uint32
	CloseUnix      int64
}

// LedgerStateResolver reads the current validated LedgerState. Injectable so
// followers/tests can supply a deterministic snapshot instead of a live RPC.
type LedgerStateResolver func(ctx context.Context) (LedgerState, error)

// llsPolicy encodes the LastLedgerSequence band a follower enforces before
// signing. In standalone mode (test-only rippled with manual
// ledger_accept) ledgers do not advance on their own, so LLS-based expiry is
// meaningless and the field must be ABSENT; otherwise it must fall inside the
// two-sided band and close before the deadline.
type llsPolicy struct {
	standalone bool
	current    LedgerState
	deadline   int64 // 0 = no deadline clamp (rotations)
}

// estimatedCloseUnix estimates when ledger `index` will close, from the current
// validated ledger and the assumed close interval.
func (p llsPolicy) estimatedCloseUnix(index uint32) int64 {
	return p.current.CloseUnix + int64(index-p.current.ValidatedIndex)*assumedLedgerCloseSec
}

// buildLLS computes the LastLedgerSequence a builder sets: current + budget,
// where budget is min(LedgerBudget, ledgers that fit before the deadline). It
// fails if fewer than minLedgerBudget ledgers fit before the deadline — the
// withdrawal parks and expires cleanly rather than being signed with a near-dead
// LLS. deadline == 0 (rotations) skips the deadline clamp.
func buildLLS(state LedgerState, deadline int64) (uint32, error) {
	budget := LedgerBudget
	if deadline != 0 {
		if deadline <= state.CloseUnix {
			return 0, fmt.Errorf("xrpl: deadline %d not after current close %d", deadline, state.CloseUnix)
		}
		if fit := (deadline - state.CloseUnix) / assumedLedgerCloseSec; fit < int64(budget) {
			budget = uint32(fit)
		}
	}
	if budget < minLedgerBudget {
		return 0, fmt.Errorf("xrpl: only %d ledgers of budget before deadline, need %d — withdrawal parks", budget, minLedgerBudget)
	}
	return state.ValidatedIndex + budget, nil
}

// clientLedgerState resolves the current validated LedgerState from a live
// rippled via the ledger query. The default resolver for the finalizers.
func clientLedgerState(client *rpc.Client) LedgerStateResolver {
	return func(_ context.Context) (LedgerState, error) {
		resp, err := client.GetLedger(&ledger.Request{LedgerIndex: common.Validated})
		if err != nil {
			return LedgerState{}, fmt.Errorf("xrpl: get validated ledger: %w", err)
		}
		return LedgerState{
			ValidatedIndex: uint32(resp.LedgerIndex),
			CloseUnix:      int64(resp.Ledger.CloseTime) + rippleEpochOffset,
		}, nil
	}
}

// checkField enforces the LastLedgerSequence band on a flatTx field value.
func (p llsPolicy) checkField(raw any, present bool) error {
	if p.standalone {
		if present {
			return fmt.Errorf("xrpl canonical: LastLedgerSequence must be absent in standalone mode")
		}
		return nil
	}
	if !present {
		return fmt.Errorf("xrpl canonical: missing LastLedgerSequence")
	}
	lls, ok := uint32Field(raw)
	if !ok {
		return fmt.Errorf("xrpl canonical: invalid LastLedgerSequence %v", raw)
	}
	if lls <= p.current.ValidatedIndex {
		return fmt.Errorf("xrpl canonical: LastLedgerSequence %d not ahead of current ledger %d", lls, p.current.ValidatedIndex)
	}
	if lls > p.current.ValidatedIndex+MaxLedgerBudget {
		return fmt.Errorf("xrpl canonical: LastLedgerSequence %d exceeds current %d + budget %d", lls, p.current.ValidatedIndex, MaxLedgerBudget)
	}
	if p.deadline != 0 && p.estimatedCloseUnix(lls) > p.deadline {
		return fmt.Errorf("xrpl canonical: LastLedgerSequence %d estimated close %d past deadline %d", lls, p.estimatedCloseUnix(lls), p.deadline)
	}
	return nil
}

// Identity is a signer's XRPL classic address + signing pubkey hex.
type Identity struct {
	ClassicAddress   string
	SigningPubKeyHex string
}

// DeriveIdentity maps a sign.Signer's public key to its XRPL identity.
func DeriveIdentity(s sign.Signer) (Identity, error) {
	pub := s.PublicKey()
	var xrplPub []byte
	switch s.Algorithm() {
	case sign.AlgSecp256k1:
		if len(pub) != 33 {
			return Identity{}, fmt.Errorf("xrpl: secp256k1 pubkey must be 33-byte compressed, got %d", len(pub))
		}
		xrplPub = pub
	case sign.AlgEd25519:
		if len(pub) != 32 {
			return Identity{}, fmt.Errorf("xrpl: ed25519 pubkey must be 32 bytes, got %d", len(pub))
		}
		// XRPL ed25519 pubkeys take a 0xED prefix.
		xrplPub = append([]byte{0xED}, pub...)
	default:
		return Identity{}, fmt.Errorf("xrpl: unsupported signer algorithm %q", s.Algorithm())
	}
	pubHex := strings.ToUpper(hex.EncodeToString(xrplPub))
	addr, err := addresscodec.EncodeClassicAddressFromPublicKeyHex(pubHex)
	if err != nil {
		return Identity{}, fmt.Errorf("xrpl: derive classic address: %w", err)
	}
	return Identity{ClassicAddress: addr, SigningPubKeyHex: pubHex}, nil
}

// signDigest runs the algorithm-specific signing primitive over the
// codec-encoded tx bytes.
func signDigest(ctx context.Context, s sign.Signer, encodedHex string) ([]byte, error) {
	raw, err := hex.DecodeString(encodedHex)
	if err != nil {
		return nil, fmt.Errorf("xrpl: decode encoded tx: %w", err)
	}
	switch s.Algorithm() {
	case sign.AlgSecp256k1:
		return s.Sign(ctx, xrplcrypto.Sha512Half(raw))
	case sign.AlgEd25519:
		return s.Sign(ctx, raw)
	default:
		return nil, fmt.Errorf("xrpl: unsupported algorithm %q", s.Algorithm())
	}
}

// signMultisig produces this node's multi-sign blob for tx.
func signMultisig(ctx context.Context, s sign.Signer, id Identity, tx transaction.FlatTransaction) (string, error) {
	tx["SigningPubKey"] = ""
	encoded, err := binarycodec.EncodeForMultisigning(tx, id.ClassicAddress)
	if err != nil {
		return "", fmt.Errorf("xrpl: EncodeForMultisigning: %w", err)
	}
	sigBytes, err := signDigest(ctx, s, encoded)
	if err != nil {
		return "", fmt.Errorf("xrpl: sign: %w", err)
	}
	inner := types.Signer{SignerData: types.SignerData{
		Account:       types.Address(id.ClassicAddress),
		TxnSignature:  strings.ToUpper(hex.EncodeToString(sigBytes)),
		SigningPubKey: id.SigningPubKeyHex,
	}}
	tx["Signers"] = []any{inner.Flatten()}
	return binarycodec.Encode(tx)
}

// signSingle signs tx as a single-signer transaction and returns the submittable blob.
func signSingle(ctx context.Context, s sign.Signer, id Identity, tx transaction.FlatTransaction) (string, error) {
	tx["SigningPubKey"] = id.SigningPubKeyHex
	encoded, err := binarycodec.EncodeForSigning(tx)
	if err != nil {
		return "", fmt.Errorf("xrpl: EncodeForSigning: %w", err)
	}
	sigBytes, err := signDigest(ctx, s, encoded)
	if err != nil {
		return "", fmt.Errorf("xrpl: sign: %w", err)
	}
	tx["TxnSignature"] = strings.ToUpper(hex.EncodeToString(sigBytes))
	return binarycodec.Encode(tx)
}

// BuildAmount converts a WithdrawalOp into an XRPL CurrencyAmount.
func BuildAmount(op *core.WithdrawalOp) (types.CurrencyAmount, error) {
	return currencyAmount(op.L1Asset, op.Amount)
}

// currencyAmount maps an asset key + decimal amount to an XRPL CurrencyAmount:
//
//	"" / "XRP"            — native XRP; amount is drops (integer).
//	"CUR.rIssuer" / "CUR:rIssuer" — issued currency; amount is a decimal value.
func currencyAmount(asset string, amount decimal.Decimal) (types.CurrencyAmount, error) {
	l1 := strings.TrimSpace(asset)
	if l1 == "" || strings.EqualFold(l1, "XRP") {
		drops := amount.BigInt()
		if !drops.IsUint64() {
			return nil, fmt.Errorf("xrpl: xrp amount %s overflows uint64 drops", drops.String())
		}
		return types.XRPCurrencyAmount(drops.Uint64()), nil
	}
	var currency, issuer string
	for _, sep := range []string{".", ":"} {
		if i := strings.Index(l1, sep); i > 0 {
			currency, issuer = l1[:i], l1[i+1:]
			break
		}
	}
	if currency == "" || issuer == "" {
		return nil, fmt.Errorf("xrpl: invalid asset %q: expected \"XRP\" or \"CUR.rIssuer\"", l1)
	}
	return types.IssuedCurrencyAmount{Issuer: types.Address(issuer), Currency: currency, Value: amount.String()}, nil
}

// ValidateCanonical asserts the canonical flatTx matches the op and that its
// LastLedgerSequence falls inside the follower's expiry band.
func ValidateCanonical(flat transaction.FlatTransaction, op *core.WithdrawalOp, withdrawalID [32]byte, vault string, policy llsPolicy) error {
	if asString(flat["TransactionType"]) != "Payment" {
		return fmt.Errorf("xrpl canonical: wrong TransactionType %v", flat["TransactionType"])
	}
	if !strings.EqualFold(asString(flat["Account"]), vault) {
		return fmt.Errorf("xrpl canonical: Account %v != vault %s", flat["Account"], vault)
	}
	if !strings.EqualFold(asString(flat["Destination"]), op.Recipient) {
		return fmt.Errorf("xrpl canonical: Destination %v != op.Recipient %s", flat["Destination"], op.Recipient)
	}
	wantAmount, err := BuildAmount(op)
	if err != nil {
		return fmt.Errorf("xrpl canonical: build expected Amount: %w", err)
	}
	if err := amountEqual(flat["Amount"], wantAmount); err != nil {
		return fmt.Errorf("xrpl canonical: Amount mismatch: %w", err)
	}
	wantInvoice := strings.ToUpper(hex.EncodeToString(withdrawalID[:]))
	if !strings.EqualFold(asString(flat["InvoiceID"]), wantInvoice) {
		return fmt.Errorf("xrpl canonical: InvoiceID %v != withdrawalID %s", flat["InvoiceID"], wantInvoice)
	}
	if _, ok := uint32Field(flat["TicketSequence"]); !ok {
		return fmt.Errorf("xrpl canonical: missing or invalid TicketSequence %v", flat["TicketSequence"])
	}
	if seq, ok := uint32Field(flat["Sequence"]); !ok || seq != 0 {
		return fmt.Errorf("xrpl canonical: Sequence must be 0 on Tickets path, got %v", flat["Sequence"])
	}
	fee, ok := uint32Field(flat["Fee"])
	if !ok {
		return fmt.Errorf("xrpl canonical: missing or invalid Fee %v", flat["Fee"])
	}
	if uint64(fee) > maxAcceptableFeeDrops {
		return fmt.Errorf("xrpl canonical: Fee %d drops exceeds ceiling %d", fee, maxAcceptableFeeDrops)
	}
	// Flags is allowlisted (honest bodies omit it) but its value must be
	// constrained: a non-zero Flags can carry tfPartialPayment, which on an
	// issued-currency withdrawal lets the submitter deliver less than Amount
	// while the ceremony still reports success (ISS-002). Reject any present
	// Flags that is non-zero or non-numeric.
	if raw, ok := flat["Flags"]; ok {
		if v, ok := uint32Field(raw); !ok || v != 0 {
			return fmt.Errorf("xrpl canonical: non-zero or invalid Flags not permitted: %v", raw)
		}
	}
	llsRaw, llsPresent := flat["LastLedgerSequence"]
	if err := policy.checkField(llsRaw, llsPresent); err != nil {
		return err
	}
	for k := range flat {
		if _, ok := canonicalAllowedFields[k]; !ok {
			return fmt.Errorf("xrpl canonical: unexpected field %q", k)
		}
	}
	return nil
}

// rotationAllowedFields is the allowlist of top-level keys a node accepts on a
// canonical SignerListSet flatTx before signing.
var rotationAllowedFields = map[string]struct{}{
	"TransactionType": {}, "Account": {}, "SignerQuorum": {}, "SignerEntries": {},
	"Sequence": {}, "Fee": {}, "SigningPubKey": {}, "Flags": {}, "NetworkID": {},
	"LastLedgerSequence": {},
}

// validateCanonicalRotation asserts the canonical SignerListSet flatTx rotates
// the vault to exactly newSigners / newThreshold (each entry weight 1, quorum ==
// newThreshold), with a fee within the ceiling and no unexpected fields.
func validateCanonicalRotation(flat transaction.FlatTransaction, newSigners []string, newThreshold int, vault string, policy llsPolicy) error {
	if asString(flat["TransactionType"]) != "SignerListSet" {
		return fmt.Errorf("xrpl rotation: wrong TransactionType %v", flat["TransactionType"])
	}
	if !strings.EqualFold(asString(flat["Account"]), vault) {
		return fmt.Errorf("xrpl rotation: Account %v != vault %s", flat["Account"], vault)
	}
	quorum, ok := uint32Field(flat["SignerQuorum"])
	if !ok || int(quorum) != newThreshold {
		return fmt.Errorf("xrpl rotation: SignerQuorum %v != newThreshold %d", flat["SignerQuorum"], newThreshold)
	}
	gotSet, err := signerEntryAccounts(flat["SignerEntries"])
	if err != nil {
		return err
	}
	wantSet := make(map[string]struct{}, len(newSigners))
	for _, s := range newSigners {
		wantSet[s] = struct{}{}
	}
	if len(gotSet) != len(wantSet) {
		return fmt.Errorf("xrpl rotation: %d signer entries != %d new signers", len(gotSet), len(wantSet))
	}
	for a := range gotSet {
		if _, ok := wantSet[a]; !ok {
			return fmt.Errorf("xrpl rotation: unexpected signer entry %s", a)
		}
	}
	fee, ok := uint32Field(flat["Fee"])
	if !ok {
		return fmt.Errorf("xrpl rotation: missing or invalid Fee %v", flat["Fee"])
	}
	if uint64(fee) > maxAcceptableFeeDrops {
		return fmt.Errorf("xrpl rotation: Fee %d drops exceeds ceiling %d", fee, maxAcceptableFeeDrops)
	}
	// Flags is allowlisted but its value must be constrained; reject any present
	// Flags that is non-zero or non-numeric (ISS-002).
	if raw, ok := flat["Flags"]; ok {
		if v, ok := uint32Field(raw); !ok || v != 0 {
			return fmt.Errorf("xrpl rotation: non-zero or invalid Flags not permitted: %v", raw)
		}
	}
	llsRaw, llsPresent := flat["LastLedgerSequence"]
	if err := policy.checkField(llsRaw, llsPresent); err != nil {
		return err
	}
	for k := range flat {
		if _, ok := rotationAllowedFields[k]; !ok {
			return fmt.Errorf("xrpl rotation: unexpected field %q", k)
		}
	}
	return nil
}

// signerEntryAccounts extracts the set of member accounts from a SignerEntries
// value (each element is {"SignerEntry": {"Account": ..., "SignerWeight": ...}}),
// rejecting weights other than 1 and duplicate accounts.
func signerEntryAccounts(raw any) (map[string]struct{}, error) {
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("xrpl rotation: SignerEntries not an array (%T)", raw)
	}
	out := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		wrapper, ok := e.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xrpl rotation: signer entry not an object")
		}
		inner, ok := wrapper["SignerEntry"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xrpl rotation: missing SignerEntry object")
		}
		acct := asString(inner["Account"])
		if acct == "" {
			return nil, fmt.Errorf("xrpl rotation: signer entry missing Account")
		}
		if w, ok := uint32Field(inner["SignerWeight"]); !ok || w != 1 {
			return nil, fmt.Errorf("xrpl rotation: signer entry %s weight %v != 1", acct, inner["SignerWeight"])
		}
		if _, dup := out[acct]; dup {
			return nil, fmt.Errorf("xrpl rotation: duplicate signer entry %s", acct)
		}
		out[acct] = struct{}{}
	}
	return out, nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func amountEqual(got any, want types.CurrencyAmount) error {
	switch w := want.(type) {
	case types.XRPCurrencyAmount:
		if asString(got) != w.String() {
			return fmt.Errorf("XRP amount %v != %s", got, w.String())
		}
		return nil
	case types.IssuedCurrencyAmount:
		gotMap, ok := got.(map[string]any)
		if !ok {
			return fmt.Errorf("issued amount must be an object, got %v (%T)", got, got)
		}
		if !strings.EqualFold(asString(gotMap["issuer"]), string(w.Issuer)) {
			return fmt.Errorf("issuer %v != %s", gotMap["issuer"], w.Issuer)
		}
		if !strings.EqualFold(asString(gotMap["currency"]), w.Currency) {
			return fmt.Errorf("currency %v != %s", gotMap["currency"], w.Currency)
		}
		if asString(gotMap["value"]) != w.Value {
			return fmt.Errorf("value %v != %s", gotMap["value"], w.Value)
		}
		return nil
	default:
		return fmt.Errorf("unsupported expected amount type %T", want)
	}
}

func uint32Field(raw any) (uint32, bool) {
	switch v := raw.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < 0 || uint64(n) > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(n), true
	case float64:
		if v < 0 || v > float64(^uint32(0)) {
			return 0, false
		}
		return uint32(v), true
	case int:
		if v < 0 || uint64(v) > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(v), true
	case uint32:
		return v, true
	case uint64:
		if v > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(v), true
	case string:
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, false
		}
		return uint32(n), true
	default:
		return 0, false
	}
}

// CanonicalJSON encodes a FlatTransaction with sorted keys.
func CanonicalJSON(flatTx transaction.FlatTransaction) ([]byte, error) {
	keys := make([]string, 0, len(flatTx))
	for k := range flatTx {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := json.Marshal(flatTx[k])
		if err != nil {
			return nil, fmt.Errorf("xrpl: encode key %q: %w", k, err)
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// combineLive merges the collected multi-sign blobs, filtering them against the
// vault's live SignerList and trimming to the live SignerQuorum rather than a
// boot-time threshold. A blob signed by a peer whose key just rotated off (or
// hasn't rotated on yet) would make rippled reject the assembled tx
// (tefBAD_SIGNATURE), and a stale quorum would under- or over-fill the Signers
// array; reading the live list keeps both correct across a rotation. Pack
// autofilled the multi-sign fee for the quorum count, so trimming to exactly
// liveQuorum keeps the fee right. Shared by the withdrawal and rotation submits.
func combineLive(client *rpc.Client, vault string, signatures [][]byte) (string, error) {
	blobs := make([]string, 0, len(signatures))
	for _, s := range signatures {
		blobs = append(blobs, string(s))
	}
	authorized, liveQuorum, err := fetchLiveSignerList(client, vault)
	if err != nil {
		return "", err
	}
	blobs, err = filterBlobsByAuthorized(blobs, authorized)
	if err != nil {
		return "", err
	}
	if len(blobs) < liveQuorum {
		return "", fmt.Errorf("xrpl: only %d authorized signatures after live-SignerList filter, need %d", len(blobs), liveQuorum)
	}
	blobs = blobs[:liveQuorum]
	final, err := xrpl.Multisign(blobs...)
	if err != nil {
		return "", fmt.Errorf("xrpl: combine signatures: %w", err)
	}
	return final, nil
}

// fetchLiveSignerList reads the vault's live SignerList via account_info and
// returns the currently-authorized r-addresses plus the live SignerQuorum.
func fetchLiveSignerList(client *rpc.Client, vault string) (map[string]struct{}, int, error) {
	resp, err := client.GetAccountInfo(&account.InfoRequest{
		Account:     types.Address(vault),
		SignerLists: true,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("xrpl: account_info: %w", err)
	}
	if len(resp.SignerLists) == 0 {
		return nil, 0, fmt.Errorf("xrpl: vault has no SignerList configured")
	}
	live := resp.SignerLists[0]
	authorized := make(map[string]struct{}, len(live.SignerEntries))
	for _, e := range live.SignerEntries {
		authorized[string(e.SignerEntry.Account)] = struct{}{}
	}
	quorum := int(live.SignerQuorum)
	if quorum <= 0 || quorum > len(live.SignerEntries) {
		return nil, 0, fmt.Errorf("xrpl: live SignerQuorum %d out of range for %d entries", quorum, len(live.SignerEntries))
	}
	return authorized, quorum, nil
}

// filterBlobsByAuthorized drops blobs whose inner signer is not in authorized,
// de-duping by signer account.
func filterBlobsByAuthorized(blobs []string, authorized map[string]struct{}) ([]string, error) {
	out := make([]string, 0, len(blobs))
	seen := make(map[string]struct{}, len(blobs))
	for i, b := range blobs {
		acct, err := blobSignerAccount(b)
		if err != nil {
			return nil, fmt.Errorf("xrpl: decode blob %d: %w", i, err)
		}
		if _, dup := seen[acct]; dup {
			continue
		}
		if _, ok := authorized[acct]; !ok {
			continue
		}
		seen[acct] = struct{}{}
		out = append(out, b)
	}
	return out, nil
}

// blobSignerAccount decodes a multi-sign blob (one signer's contribution) and
// returns the classic r-address of its inner Signer entry.
func blobSignerAccount(blob string) (string, error) {
	decoded, err := binarycodec.Decode(blob)
	if err != nil {
		return "", err
	}
	signersAny, ok := decoded["Signers"]
	if !ok {
		return "", fmt.Errorf("blob missing Signers field")
	}
	signers, ok := signersAny.([]any)
	if !ok || len(signers) == 0 {
		return "", fmt.Errorf("blob Signers field not a non-empty array")
	}
	first, ok := signers[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("blob Signers[0] not an object")
	}
	inner, ok := first["Signer"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("blob Signers[0].Signer not an object")
	}
	acct, ok := inner["Account"].(string)
	if !ok || acct == "" {
		return "", fmt.Errorf("blob Signers[0].Signer.Account missing")
	}
	return acct, nil
}

// hashHex is the uppercase hex of a 32-byte tx hash (XRPL's display form).
func hashHex(h [32]byte) string { return strings.ToUpper(hex.EncodeToString(h[:])) }

// computeTxHash computes the XRPL transaction hash from a tx blob hex string.
func computeTxHash(txBlobHex string) ([32]byte, error) {
	blobBytes, err := hex.DecodeString(txBlobHex)
	if err != nil {
		return [32]byte{}, fmt.Errorf("xrpl: decode tx blob: %w", err)
	}
	buf := append([]byte{0x54, 0x58, 0x4E, 0x00}, blobBytes...)
	var hash [32]byte
	copy(hash[:], xrplcrypto.Sha512Half(buf))
	return hash, nil
}

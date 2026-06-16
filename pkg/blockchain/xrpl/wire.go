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
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// maxAcceptableFeeDrops caps the Fee a node will sign on a canonical Payment.
const maxAcceptableFeeDrops uint64 = 1_000_000

// canonicalAllowedFields is the allowlist of top-level keys a node accepts on a
// canonical Payment flatTx before signing.
var canonicalAllowedFields = map[string]struct{}{
	"TransactionType": {}, "Account": {}, "Destination": {}, "Amount": {},
	"InvoiceID": {}, "TicketSequence": {}, "Sequence": {}, "Fee": {},
	"SigningPubKey": {}, "Flags": {},
}

// parseDepositTag extracts the XRPL DestinationTag from a crediting account.
//
// The custody deposit watcher derives the credited account FROM the tag —
// `core.UserURI("xrpl-" + tag)` — so the tag is the primary identifier, not a
// hash of anything. The account therefore must be of the form `xrpl-<tag>`
// (optionally as the last segment of a yellow:// URI); this reverses that
// mapping to recover the uint32 tag the depositor must set.
func parseDepositTag(account string) (uint32, error) {
	seg := account
	if i := strings.LastIndex(seg, "/"); i >= 0 {
		seg = seg[i+1:]
	}
	rest, ok := strings.CutPrefix(strings.ToLower(seg), "xrpl-")
	if !ok {
		return 0, fmt.Errorf("xrpl: account %q must be of the form xrpl-<tag> (or yellow://.../user/xrpl-<tag>)", account)
	}
	n, err := strconv.ParseUint(rest, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("xrpl: bad deposit tag in account %q: %w", account, err)
	}
	return uint32(n), nil
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

// ValidateCanonical asserts the canonical flatTx matches the op.
func ValidateCanonical(flat transaction.FlatTransaction, op *core.WithdrawalOp, withdrawalID [32]byte, vault string) error {
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
	"Sequence": {}, "Fee": {}, "SigningPubKey": {}, "Flags": {},
}

// validateCanonicalRotation asserts the canonical SignerListSet flatTx rotates
// the vault to exactly newSigners / newThreshold (each entry weight 1, quorum ==
// newThreshold), with a fee within the ceiling and no unexpected fields.
func validateCanonicalRotation(flat transaction.FlatTransaction, newSigners []string, newThreshold int, vault string) error {
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

// combineMultisign trims the collected multi-sign blobs to exactly `threshold`
// and combines them into one submittable blob. Pack autofilled the multi-sign
// fee for that count (base × (1 + threshold)), so including extras would
// under-pay (telINSUF_FEE_P) and waste fee; any threshold of the SignerList's
// members satisfies the quorum. Shared by the withdrawal and rotation submits.
func combineMultisign(signatures [][]byte, threshold int) (string, error) {
	if len(signatures) < threshold {
		return "", fmt.Errorf("xrpl: have %d signatures, need %d", len(signatures), threshold)
	}
	blobs := make([]string, 0, threshold)
	for _, s := range signatures[:threshold] {
		blobs = append(blobs, string(s))
	}
	final, err := xrpl.Multisign(blobs...)
	if err != nil {
		return "", fmt.Errorf("xrpl: combine signatures: %w", err)
	}
	return final, nil
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

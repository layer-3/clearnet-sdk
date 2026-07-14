package xrpl

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Peersyst/xrpl-go/xrpl/transaction"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// roundTrip marshals then unmarshals a flatTx, reproducing the numeric shape
// (float64) the real Validate path sees after decoding the packed bytes.
func roundTrip(t *testing.T, flat transaction.FlatTransaction) transaction.FlatTransaction {
	t.Helper()
	b, err := json.Marshal(flat)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out transaction.FlatTransaction
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// TestValidateCanonical_FlagsRejected covers the ISS-002 guard: a present Flags
// must be numeric zero; a non-zero or non-numeric value (tfPartialPayment is
// 131072) is rejected so an issued-currency withdrawal cannot underdeliver.
func TestValidateCanonical_FlagsRejected(t *testing.T) {
	const vault = "rVaULtAdd1111111111111111111111111"
	ctx := context.Background()
	assets := NewAssetResolver(AssetResolverConfig{})
	op := &core.WithdrawalOp{
		Recipient: "rDeST1111111111111111111111111111",
		AssetURI:  "yellow://ynet/asset/custody/xrpl/0/0",
		Amount:    decimal.NewFromInt(1),
	}
	amt, err := BuildAmount(ctx, assets, op)
	if err != nil {
		t.Fatalf("BuildAmount: %v", err)
	}
	var wid [32]byte
	wid[0], wid[31] = 0xAB, 0xCD

	base := func() transaction.FlatTransaction {
		return transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         vault,
			"Destination":     op.Recipient,
			"Amount":          amt,
			"InvoiceID":       strings.ToUpper(hex.EncodeToString(wid[:])),
			"TicketSequence":  uint32(5),
			"Sequence":        uint32(0),
			"Fee":             uint32(100),
		}
	}

	if err := ValidateCanonical(ctx, assets, roundTrip(t, base()), op, wid, vault, llsPolicy{standalone: true}); err != nil {
		t.Fatalf("valid canonical rejected: %v", err)
	}

	zero := base()
	zero["Flags"] = uint32(0)
	if err := ValidateCanonical(ctx, assets, roundTrip(t, zero), op, wid, vault, llsPolicy{standalone: true}); err != nil {
		t.Errorf("Flags=0 rejected: %v", err)
	}

	partial := base()
	partial["Flags"] = uint32(131072) // tfPartialPayment
	if err := ValidateCanonical(ctx, assets, roundTrip(t, partial), op, wid, vault, llsPolicy{standalone: true}); err == nil {
		t.Error("tfPartialPayment Flags accepted")
	}

	nonNumeric := base()
	nonNumeric["Flags"] = "deadbeef"
	if err := ValidateCanonical(ctx, assets, roundTrip(t, nonNumeric), op, wid, vault, llsPolicy{standalone: true}); err == nil {
		t.Error("non-numeric Flags accepted")
	}
}

// TestValidateCanonical_LLSBand covers the LastLedgerSequence band a
// follower enforces: present, ahead of current, within MaxLedgerBudget, and
// estimated to close at or before the deadline.
func TestValidateCanonical_LLSBand(t *testing.T) {
	const vault = "rVaULtAdd1111111111111111111111111"
	ctx := context.Background()
	assets := NewAssetResolver(AssetResolverConfig{})
	op := &core.WithdrawalOp{
		Recipient: "rDeST1111111111111111111111111111",
		AssetURI:  "yellow://ynet/asset/custody/xrpl/0/0",
		Amount:    decimal.NewFromInt(1),
	}
	amt, err := BuildAmount(ctx, assets, op)
	if err != nil {
		t.Fatalf("BuildAmount: %v", err)
	}
	var wid [32]byte
	wid[0], wid[31] = 0xAB, 0xCD

	// current ledger 1000 closing at t=10000; deadline far in the future.
	current := LedgerState{ValidatedIndex: 1000, CloseUnix: 10_000}
	deadline := int64(10_000 + 10_000) // ~2500 ledgers of headroom
	policy := llsPolicy{current: current, deadline: deadline}

	withLLS := func(lls uint32) transaction.FlatTransaction {
		return roundTrip(t, transaction.FlatTransaction{
			"TransactionType":    "Payment",
			"Account":            vault,
			"Destination":        op.Recipient,
			"Amount":             amt,
			"InvoiceID":          strings.ToUpper(hex.EncodeToString(wid[:])),
			"TicketSequence":     uint32(5),
			"Sequence":           uint32(0),
			"Fee":                uint32(100),
			"LastLedgerSequence": lls,
		})
	}

	// In-band: current + LedgerBudget.
	if err := ValidateCanonical(ctx, assets, withLLS(current.ValidatedIndex+LedgerBudget), op, wid, vault, policy); err != nil {
		t.Errorf("in-band LLS rejected: %v", err)
	}
	// Missing LLS is rejected in non-standalone mode.
	missing := withLLS(0)
	delete(missing, "LastLedgerSequence")
	if err := ValidateCanonical(ctx, assets, missing, op, wid, vault, policy); err == nil {
		t.Error("missing LastLedgerSequence accepted")
	}
	// Not ahead of current.
	if err := ValidateCanonical(ctx, assets, withLLS(current.ValidatedIndex), op, wid, vault, policy); err == nil {
		t.Error("LLS == current accepted")
	}
	// Beyond MaxLedgerBudget.
	if err := ValidateCanonical(ctx, assets, withLLS(current.ValidatedIndex+MaxLedgerBudget+1), op, wid, vault, policy); err == nil {
		t.Error("LLS beyond MaxLedgerBudget accepted")
	}
	// Estimated close past the deadline: tight deadline of only 4 ledgers.
	tight := llsPolicy{current: current, deadline: current.CloseUnix + 4*assumedLedgerCloseSec}
	if err := ValidateCanonical(ctx, assets, withLLS(current.ValidatedIndex+LedgerBudget), op, wid, vault, tight); err == nil {
		t.Error("LLS closing past deadline accepted")
	}
}

// TestBuildLLS covers the builder's deadline clamp and the park-on-too-little
// budget behavior.
func TestBuildLLS(t *testing.T) {
	current := LedgerState{ValidatedIndex: 1000, CloseUnix: 10_000}

	// No deadline (rotation): current + full LedgerBudget.
	if lls, err := buildLLS(current, 0); err != nil || lls != current.ValidatedIndex+LedgerBudget {
		t.Fatalf("buildLLS(no deadline) = %d, %v; want %d", lls, err, current.ValidatedIndex+LedgerBudget)
	}
	// Generous deadline: clamp does not bite.
	if lls, err := buildLLS(current, current.CloseUnix+10_000); err != nil || lls != current.ValidatedIndex+LedgerBudget {
		t.Fatalf("buildLLS(generous) = %d, %v; want %d", lls, err, current.ValidatedIndex+LedgerBudget)
	}
	// Tight-but-viable deadline: clamps below LedgerBudget but above the floor.
	tight := current.CloseUnix + int64(10)*assumedLedgerCloseSec
	if lls, err := buildLLS(current, tight); err != nil || lls != current.ValidatedIndex+10 {
		t.Fatalf("buildLLS(tight) = %d, %v; want %d", lls, err, current.ValidatedIndex+10)
	}
	// Too little budget: parks (error).
	if _, err := buildLLS(current, current.CloseUnix+int64(minLedgerBudget-1)*assumedLedgerCloseSec); err == nil {
		t.Error("buildLLS accepted a deadline with less than minLedgerBudget")
	}
	// Deadline already passed.
	if _, err := buildLLS(current, current.CloseUnix-1); err == nil {
		t.Error("buildLLS accepted a deadline before current close")
	}
}

// TestValidateCanonicalRotation_FlagsRejected is the SignerListSet analogue.
func TestValidateCanonicalRotation_FlagsRejected(t *testing.T) {
	const vault = "rVaULtAdd1111111111111111111111111"
	newSigners := []string{"rAAA1111111111111111111111111111aa", "rBBB1111111111111111111111111111bb"}
	entries, err := signerEntries(newSigners, 2)
	if err != nil {
		t.Fatalf("signerEntries: %v", err)
	}

	base := func() transaction.FlatTransaction {
		return transaction.FlatTransaction{
			"TransactionType": "SignerListSet",
			"Account":         vault,
			"SignerQuorum":    uint32(2),
			"SignerEntries":   entries,
			"Sequence":        uint32(1),
			"Fee":             uint32(100),
		}
	}

	if err := validateCanonicalRotation(roundTrip(t, base()), newSigners, 2, vault, llsPolicy{standalone: true}); err != nil {
		t.Fatalf("valid rotation rejected: %v", err)
	}

	partial := base()
	partial["Flags"] = uint32(131072)
	if err := validateCanonicalRotation(roundTrip(t, partial), newSigners, 2, vault, llsPolicy{standalone: true}); err == nil {
		t.Error("non-zero Flags accepted on rotation")
	}
}

// TestFilterBlobsByAuthorized covers the live-SignerList filter that combineLive
// applies: blobs from non-authorized signers are dropped, and a duplicate signer
// is kept once. Uses real multi-sign blobs (no devnet).
func TestFilterBlobsByAuthorized(t *testing.T) {
	ctx := context.Background()
	signerA, idA := newXRPLSigner(t)
	signerB, idB := newXRPLSigner(t)

	blobA := multisignBlob(t, ctx, signerA, idA, idB.ClassicAddress)
	blobB := multisignBlob(t, ctx, signerB, idB, idA.ClassicAddress)

	// blobSignerAccount recovers the inner signer's account.
	if got, err := blobSignerAccount(blobA); err != nil || got != idA.ClassicAddress {
		t.Fatalf("blobSignerAccount(blobA) = %q, %v; want %s", got, err, idA.ClassicAddress)
	}

	// Only A authorized: B dropped.
	authorized := map[string]struct{}{idA.ClassicAddress: {}}
	out, err := filterBlobsByAuthorized([]string{blobA, blobB}, authorized)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(out) != 1 || out[0] != blobA {
		t.Fatalf("filter kept %d blobs, want only blobA", len(out))
	}

	// Duplicate A kept once.
	out, err = filterBlobsByAuthorized([]string{blobA, blobA}, map[string]struct{}{idA.ClassicAddress: {}})
	if err != nil {
		t.Fatalf("filter dup: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("dup signer kept %d times, want 1", len(out))
	}
}

func newXRPLSigner(t *testing.T) (sign.Signer, Identity) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	s, err := sign.NewKeySignerFromEd25519(priv)
	if err != nil {
		t.Fatalf("ed25519 signer: %v", err)
	}
	id, err := DeriveIdentity(s)
	if err != nil {
		t.Fatalf("DeriveIdentity: %v", err)
	}
	return s, id
}

func multisignBlob(t *testing.T, ctx context.Context, s sign.Signer, id Identity, dest string) string {
	t.Helper()
	flat := transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         id.ClassicAddress,
		"Destination":     dest,
		"Amount":          "1000000",
		"Fee":             "100",
		"Sequence":        uint32(1),
	}
	blob, err := signMultisig(ctx, s, id, flat)
	if err != nil {
		t.Fatalf("signMultisig: %v", err)
	}
	return blob
}

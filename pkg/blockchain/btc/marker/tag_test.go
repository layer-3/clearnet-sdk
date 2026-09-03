package marker

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestGenericTag pins GenericDepositTagHex against a fresh SHA256 of
// GenericDepositTagPreimage. A one-character typo in the preimage would
// silently produce a different tag - and therefore a different generic
// address.
func TestGenericTag(t *testing.T) {
	want, err := hex.DecodeString(GenericDepositTagHex)
	if err != nil {
		t.Fatalf("GenericDepositTagHex is not valid hex: %v", err)
	}
	got := sha256.Sum256([]byte(GenericDepositTagPreimage))
	if hex.EncodeToString(got[:]) != GenericDepositTagHex {
		t.Fatalf("SHA256(preimage) = %x, want %s", got, GenericDepositTagHex)
	}
	if fn := GenericDepositTag(); hex.EncodeToString(fn[:]) != GenericDepositTagHex {
		t.Fatalf("GenericDepositTag() = %x, want %s", fn, GenericDepositTagHex)
	}
	if len(want) != 32 {
		t.Fatalf("GenericDepositTagHex decodes to %d bytes, want 32", len(want))
	}
}

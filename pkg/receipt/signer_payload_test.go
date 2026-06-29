package receipt

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSignerPayload_RoundTrip(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}
	b, err := MarshalSignerPayload(signers, 2)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, thr, err := ParseSignerPayload(b)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if thr != 2 {
		t.Fatalf("threshold: got %d want 2", thr)
	}
	// Parsed set is ascending regardless of input order.
	want := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("signer[%d]: got %s want %s", i, got[i].Hex(), want[i].Hex())
		}
	}
}

// TestSignerPayload_Deterministic asserts content-addressing: the same intent in
// a different input order yields byte-identical output (so the checksum agrees).
func TestSignerPayload_Deterministic(t *testing.T) {
	a := []common.Address{
		common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		common.HexToAddress("0x00000000000000000000000000000000000000bb"),
	}
	b := []common.Address{a[1], a[0]}
	pa, err := MarshalSignerPayload(a, 1)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := MarshalSignerPayload(b, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pa, pb) {
		t.Fatalf("payload not order-independent:\n%s\n%s", pa, pb)
	}
}

func TestSignerPayload_Rejections(t *testing.T) {
	one := []common.Address{common.HexToAddress("0x01")}
	two := []common.Address{
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
	}
	if _, err := MarshalSignerPayload(nil, 1); err == nil {
		t.Error("expected error for empty signer set")
	}
	if _, err := MarshalSignerPayload(one, 2); err == nil {
		t.Error("expected error for threshold > signers")
	}
	if _, err := MarshalSignerPayload(two, 0); err == nil {
		t.Error("expected error for zero threshold")
	}
	dup := []common.Address{one[0], one[0]}
	if _, err := MarshalSignerPayload(dup, 1); err == nil {
		t.Error("expected error for duplicate signer")
	}

	// Version skew is rejected on parse.
	bad := []byte(`{"v":2,"threshold":1,"signers":["0x0000000000000000000000000000000000000001"]}`)
	if _, _, err := ParseSignerPayload(bad); err == nil {
		t.Error("expected error for unsupported version")
	}
	// Non-ascending order is rejected.
	unsorted := []byte(`{"v":1,"threshold":1,"signers":["0x0000000000000000000000000000000000000002","0x0000000000000000000000000000000000000001"]}`)
	if _, _, err := ParseSignerPayload(unsorted); err == nil {
		t.Error("expected error for non-ascending signers")
	}
}

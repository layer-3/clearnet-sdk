package receipt

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type fakeChecksumSource struct {
	checksum [32]byte
	ok       bool
}

func (f fakeChecksumSource) LatestChecksum(_ [32]byte) ([32]byte, bool) {
	return f.checksum, f.ok
}

type fakeStore map[[32]byte][]byte

func (s fakeStore) Payload(_ context.Context, checksum [32]byte) ([]byte, error) {
	p, ok := s[checksum]
	if !ok {
		return nil, fmt.Errorf("not held")
	}
	return p, nil
}

func sum(b []byte) [32]byte {
	var out [32]byte
	copy(out[:], crypto.Keccak256(b))
	return out
}

func TestRegistrySignerSource_Load(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	payload, err := MarshalSignerPayload(signers, 2)
	if err != nil {
		t.Fatal(err)
	}
	checksum := sum(payload)

	var key [32]byte
	key[0] = 0xAA
	src, err := NewRegistrySignerSource(key, fakeChecksumSource{checksum: checksum, ok: true}, fakeStore{checksum: payload})
	if err != nil {
		t.Fatal(err)
	}
	got, thr, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if thr != 2 || len(got) != 3 {
		t.Fatalf("unexpected (signers=%d, threshold=%d)", len(got), thr)
	}
}

func TestRegistrySignerSource_NoChecksum(t *testing.T) {
	var key [32]byte
	src, _ := NewRegistrySignerSource(key, fakeChecksumSource{ok: false}, fakeStore{})
	if _, _, err := src.Load(context.Background()); err == nil {
		t.Fatal("expected error when no checksum is confirmed")
	}
}

// TestRegistrySignerSource_ChecksumMismatch is the safety property: a payload
// store that returns bytes not matching the on-chain checksum cannot inject a
// signer set — Load re-derives keccak256 and rejects.
func TestRegistrySignerSource_ChecksumMismatch(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}
	payload, _ := MarshalSignerPayload(signers, 1)

	// On-chain checksum claims something else.
	var wrong [32]byte
	wrong[0] = 0x99

	var key [32]byte
	src, _ := NewRegistrySignerSource(key, fakeChecksumSource{checksum: wrong, ok: true}, fakeStore{wrong: payload})
	if _, _, err := src.Load(context.Background()); err == nil {
		t.Fatal("expected checksum-mismatch rejection")
	}
}

func TestRegistrySignerSource_MissingPayload(t *testing.T) {
	var key, checksum [32]byte
	checksum[0] = 0x01
	src, _ := NewRegistrySignerSource(key, fakeChecksumSource{checksum: checksum, ok: true}, fakeStore{})
	if _, _, err := src.Load(context.Background()); err == nil {
		t.Fatal("expected error when payload is not held")
	}
}

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
	// The zero address is a valid hex address but never a valid signer.
	zero := []byte(`{"v":1,"threshold":1,"signers":["0x0000000000000000000000000000000000000000"]}`)
	if _, _, err := ParseSignerPayload(zero); err == nil {
		t.Error("expected error for zero-address signer")
	}
}

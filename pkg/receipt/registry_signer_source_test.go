package receipt

import (
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

package receipt

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

type fakeRegistryEventReader struct {
	event core.ConfigRegistryEvent
	ok    bool
	err   error

	wantRegistry common.Address
	wantIssuerID common.Address
	wantKey      [32]byte
}

func (f fakeRegistryEventReader) LatestConfigRegistryEvent(_ context.Context, registry common.Address, issuerID common.Address, key [32]byte) (core.ConfigRegistryEvent, bool, error) {
	if f.err != nil {
		return core.ConfigRegistryEvent{}, false, f.err
	}
	if f.wantRegistry != (common.Address{}) && registry != f.wantRegistry {
		return core.ConfigRegistryEvent{}, false, fmt.Errorf("registry = %s", registry.Hex())
	}
	if f.wantIssuerID != (common.Address{}) && issuerID != f.wantIssuerID {
		return core.ConfigRegistryEvent{}, false, fmt.Errorf("issuer = %s", issuerID.Hex())
	}
	if f.wantKey != ([32]byte{}) && key != f.wantKey {
		return core.ConfigRegistryEvent{}, false, fmt.Errorf("key = 0x%s", common.Bytes2Hex(key[:]))
	}
	return f.event, f.ok, nil
}

func checksum(b []byte) [32]byte {
	return [32]byte(crypto.Keccak256Hash(b))
}

func TestRegistrySignerSource_Load(t *testing.T) {
	registry := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	issuerID := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	payload, err := MarshalSignerPayload(signers, 2)
	if err != nil {
		t.Fatal(err)
	}
	src, err := NewRegistrySignerSource(registry, fakeRegistryEventReader{
		event: core.ConfigRegistryEvent{
			Registry: registry,
			IssuerID: issuerID,
			Key:      ConfigRegistrySignersKey,
			Checksum: checksum(payload),
			HasData:  true,
			Data:     payload,
		},
		ok:           true,
		wantRegistry: registry,
		wantIssuerID: issuerID,
		wantKey:      ConfigRegistrySignersKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := src.LoadReceiptSigners(context.Background(), issuerID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Threshold != 2 || len(got.Signers) != 3 {
		t.Fatalf("unexpected (signers=%d, threshold=%d)", len(got.Signers), got.Threshold)
	}
}

func TestRegistrySignerSource_NoEvent(t *testing.T) {
	registry := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	issuerID := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	src, _ := NewRegistrySignerSource(registry, fakeRegistryEventReader{})
	if _, err := src.LoadReceiptSigners(context.Background(), issuerID); err == nil {
		t.Fatal("expected error when no signer event is confirmed")
	}
}

func TestRegistrySignerSource_ChecksumMismatch(t *testing.T) {
	registry := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	issuerID := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	payload, _ := MarshalSignerPayload([]common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}, 1)
	src, _ := NewRegistrySignerSource(registry, fakeRegistryEventReader{
		event: core.ConfigRegistryEvent{
			Registry: registry,
			IssuerID: issuerID,
			Key:      ConfigRegistrySignersKey,
			Checksum: [32]byte{0x99},
			HasData:  true,
			Data:     payload,
		},
		ok: true,
	})
	if _, err := src.LoadReceiptSigners(context.Background(), issuerID); err == nil {
		t.Fatal("expected checksum-mismatch rejection")
	}
}

func TestRegistrySignerSource_Rejections(t *testing.T) {
	registry := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	issuerID := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	payload, _ := MarshalSignerPayload([]common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
	}, 1)
	base := core.ConfigRegistryEvent{
		Registry: registry,
		IssuerID: issuerID,
		Key:      ConfigRegistrySignersKey,
		Checksum: checksum(payload),
		HasData:  true,
		Data:     payload,
	}
	cases := []struct {
		name string
		mut  func(*core.ConfigRegistryEvent)
	}{
		{"wrong registry", func(ev *core.ConfigRegistryEvent) {
			ev.Registry = common.HexToAddress("0x00000000000000000000000000000000000000cc")
		}},
		{"wrong issuer", func(ev *core.ConfigRegistryEvent) {
			ev.IssuerID = common.HexToAddress("0x00000000000000000000000000000000000000dd")
		}},
		{"wrong key", func(ev *core.ConfigRegistryEvent) { ev.Key = [32]byte{0x01} }},
		{"no data", func(ev *core.ConfigRegistryEvent) { ev.HasData = false }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := base
			tc.mut(&ev)
			src, _ := NewRegistrySignerSource(registry, fakeRegistryEventReader{event: ev, ok: true})
			if _, err := src.LoadReceiptSigners(context.Background(), issuerID); err == nil {
				t.Fatal("expected load to fail")
			}
		})
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
	got, err := ParseSignerPayload(b)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Threshold != 2 {
		t.Fatalf("threshold: got %d want 2", got.Threshold)
	}
	want := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	if len(got.Signers) != len(want) {
		t.Fatalf("len: got %d want %d", len(got.Signers), len(want))
	}
	for i := range want {
		if got.Signers[i] != want[i] {
			t.Fatalf("signer[%d]: got %s want %s", i, got.Signers[i].Hex(), want[i].Hex())
		}
	}
}

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

	bad := []byte(`{"v":2,"threshold":1,"signers":["0x0000000000000000000000000000000000000001"]}`)
	if _, err := ParseSignerPayload(bad); err == nil {
		t.Error("expected error for unsupported version")
	}
	unsorted := []byte(`{"v":1,"threshold":1,"signers":["0x0000000000000000000000000000000000000002","0x0000000000000000000000000000000000000001"]}`)
	if _, err := ParseSignerPayload(unsorted); err == nil {
		t.Error("expected error for non-ascending signers")
	}
	zero := []byte(`{"v":1,"threshold":1,"signers":["0x0000000000000000000000000000000000000000"]}`)
	if _, err := ParseSignerPayload(zero); err == nil {
		t.Error("expected error for zero-address signer")
	}
}

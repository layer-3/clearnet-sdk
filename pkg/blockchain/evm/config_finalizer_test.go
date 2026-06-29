package evm

import (
	"encoding/json"
	"testing"
)

func TestParseBytes32_RoundTrip(t *testing.T) {
	var b [32]byte
	b[0] = 0xDE
	b[31] = 0xAD
	s := hexBytes32(b)
	got, err := parseBytes32(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got != b {
		t.Fatalf("round trip mismatch: %x != %x", got, b)
	}
	// Accepts upper-case 0X and bare hex too.
	if _, err := parseBytes32("0X" + "00"); err == nil {
		t.Fatal("expected length error for short hex")
	}
	if _, err := parseBytes32("zz"); err == nil {
		t.Fatal("expected error for non-hex")
	}
}

func TestEvmCommitPacked_JSONStable(t *testing.T) {
	p := evmCommitPacked{Key: "0x01", Checksum: "0x02", ExpectedEpoch: 7}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var got evmCommitPacked
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Fatalf("round trip mismatch: %+v != %+v", got, p)
	}
}

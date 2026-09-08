package evm

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestParseClearnetAccount pins the exact accepted/rejected input set for
// parseClearnetAccount (delegating to core.ParseClearnetAccount): bare hex,
// an optional case-insensitive "0x" prefix, a yellow://.../user/<hex> URI's
// last segment, and surrounding whitespace — matching BTC/SOL/XRPL. It also
// pins that an ADR-015 sub-account URI
// (yellow://.../user/<addr>/tag/<32-byte-ref>) is rejected rather than
// silently parsed as the trailing 32-byte reference.
func TestParseClearnetAccount(t *testing.T) {
	const addrHex = "000102030405060708090a0b0c0d0e0f10111213"
	const refHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	wantAddr := common.HexToAddress("0x" + addrHex)

	accept := []string{
		addrHex,
		"0x" + addrHex,
		"0X" + addrHex,
		strings.ToUpper(addrHex),
		"  " + addrHex + "  ",
		"\t" + addrHex + "\n",
		"yellow://ynet/user/" + addrHex,
		"  yellow://ynet/user/" + addrHex + "  ",
	}
	for _, in := range accept {
		t.Run("accept/"+in, func(t *testing.T) {
			got, err := parseClearnetAccount(in)
			if err != nil {
				t.Fatalf("parseClearnetAccount(%q) error = %v, want nil", in, err)
			}
			if got != wantAddr {
				t.Fatalf("parseClearnetAccount(%q) = %s, want %s", in, got.Hex(), wantAddr.Hex())
			}
		})
	}

	reject := []string{
		"",
		"not-hex",
		addrHex[:38],   // 19 bytes
		addrHex + "00", // 21 bytes
		"yellow://ynet/user/" + addrHex + "/tag/" + refHex, // ADR-015 sub-account URI: last segment is the 32-byte reference, not the address.
	}
	for _, in := range reject {
		t.Run("reject/"+in, func(t *testing.T) {
			if _, err := parseClearnetAccount(in); err == nil {
				t.Fatalf("parseClearnetAccount(%q) error = nil, want error", in)
			}
		})
	}
}

// TestParseClearnetAccount_RegressionAgainstSilentCorruption verifies that
// an oversized (e.g. 32-byte) value is now rejected instead of silently
// truncated to its last 20 bytes, because a joined user URI resolves to the
// real embedded address instead of silently becoming the zero address.
func TestParseClearnetAccount_RegressionAgainstSilentCorruption(t *testing.T) {
	const addrHex = "d8da6bf26964af9d7eed9e03e53415d37aa96045"
	joinedURI := "yellow://ynet/user/0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	wantAddr := common.HexToAddress("0x" + addrHex)
	if zero := common.HexToAddress(joinedURI); zero != (common.Address{}) {
		t.Fatalf("test assumption broken: common.HexToAddress(%q) = %s, want zero address (verify against the pinned go-ethereum version)", joinedURI, zero.Hex())
	}
	got, err := parseClearnetAccount(joinedURI)
	if err != nil {
		t.Fatalf("parseClearnetAccount(%q) error = %v, want the embedded address", joinedURI, err)
	}
	if got != wantAddr {
		t.Fatalf("parseClearnetAccount(%q) = %s, want %s (the embedded address, not the zero address)", joinedURI, got.Hex(), wantAddr.Hex())
	}

	// A 32-byte hex value (e.g. an ADR-015 reference mistakenly passed as the
	// account): common.HexToAddress silently keeps only the last 20 bytes
	// instead of erroring on the length mismatch.
	ref32 := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	if got, want := common.HexToAddress(ref32), common.HexToAddress(ref32[len(ref32)-40:]); got != want {
		t.Fatalf("test assumption broken: common.HexToAddress(%q) = %s, want truncation to the last 20 bytes %s (verify against the pinned go-ethereum version)", ref32, got.Hex(), want.Hex())
	}
	if _, err := parseClearnetAccount(ref32); err == nil {
		t.Fatalf("parseClearnetAccount(%q) accepted a 32-byte value instead of requiring exactly 20 bytes", ref32)
	}
}

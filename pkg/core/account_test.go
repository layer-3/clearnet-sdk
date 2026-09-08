package core

import (
	"encoding/hex"
	"strings"
	"testing"
)

// TestParseClearnetAccount pins the exact accepted/rejected input set shared
// by all four chain depositors (BTC, SOL, XRPL, EVM), in both SDK languages:
// bare hex, an optional case-insensitive "0x" prefix, a yellow://.../user/<hex>
// URI's last segment, and surrounding whitespace. It also pins that an ADR-015
// sub-account URI (yellow://.../user/<addr>/tag/<32-byte-ref>) is rejected
// rather than silently parsed as the trailing 32-byte reference, and that its
// error names the fix.
func TestParseClearnetAccount(t *testing.T) {
	const addrHex = "000102030405060708090a0b0c0d0e0f10111213"
	const refHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	want, err := hex.DecodeString(addrHex)
	if err != nil {
		t.Fatal(err)
	}
	var wantAddr [20]byte
	copy(wantAddr[:], want)

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
			got, err := ParseClearnetAccount(in)
			if err != nil {
				t.Fatalf("ParseClearnetAccount(%q) error = %v, want nil", in, err)
			}
			if got != wantAddr {
				t.Fatalf("ParseClearnetAccount(%q) = %x, want %x", in, got, wantAddr)
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
			if _, err := ParseClearnetAccount(in); err == nil {
				t.Fatalf("ParseClearnetAccount(%q) error = nil, want error", in)
			}
		})
	}
}

// TestParseClearnetAccount_TagHint pins that the error hints at splitting an
// ADR-015 sub-account URI only when the input actually contains "/tag/".
func TestParseClearnetAccount_TagHint(t *testing.T) {
	const addrHex = "000102030405060708090a0b0c0d0e0f10111213"
	const refHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

	_, err := ParseClearnetAccount("yellow://ynet/user/" + addrHex + "/tag/" + refHex)
	if err == nil || !strings.Contains(err.Error(), "/tag/<ref>") {
		t.Fatalf("expected /tag/ split hint, got: %v", err)
	}

	_, err = ParseClearnetAccount("not-hex")
	if err == nil || strings.Contains(err.Error(), "/tag/") {
		t.Fatalf("expected no /tag/ hint for unrelated malformed input, got: %v", err)
	}
}

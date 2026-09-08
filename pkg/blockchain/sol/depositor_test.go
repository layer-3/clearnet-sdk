package sol

import (
	"encoding/hex"
	"strings"
	"testing"
)

// TestParseClearnetAccount pins the exact accepted/rejected input set for
// parseClearnetAccount (delegating to core.ParseClearnetAccount): bare hex,
// an optional case-insensitive "0x" prefix, a yellow://.../user/<hex> URI's
// last segment, and surrounding whitespace. It also pins that an ADR-015
// sub-account URI (yellow://.../user/<addr>/tag/<32-byte-ref>) is rejected
// rather than silently parsed as the trailing 32-byte reference.
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
			got, err := parseClearnetAccount(in)
			if err != nil {
				t.Fatalf("parseClearnetAccount(%q) error = %v, want nil", in, err)
			}
			if got != wantAddr {
				t.Fatalf("parseClearnetAccount(%q) = %x, want %x", in, got, wantAddr)
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

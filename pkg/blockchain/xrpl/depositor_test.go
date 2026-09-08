package xrpl

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// TestAccountMemo verifies the ynet-account memo encodes the 20-byte account
// followed by the 32-byte ADR-015 reference, matching what a deposit watcher
// decodes (MemoType "ynet-account", MemoData = account || reference, both hex).
func TestAccountMemo(t *testing.T) {
	var ref [32]byte
	ref[0], ref[31] = 0xAB, 0xCD
	dest := core.DepositDestination{Account: "0x00000000000000000000000000000000000000a2", Ref: ref}

	mw, err := accountMemo(dest)
	if err != nil {
		t.Fatalf("accountMemo: %v", err)
	}

	if got, want := mw.Memo.MemoType, hex.EncodeToString([]byte("ynet-account")); got != want {
		t.Errorf("MemoType: got %s, want %s", got, want)
	}

	data, err := hex.DecodeString(mw.Memo.MemoData)
	if err != nil {
		t.Fatalf("MemoData not hex: %v", err)
	}
	if len(data) != 52 {
		t.Fatalf("MemoData length: got %d, want 52 (20 account + 32 reference)", len(data))
	}
	wantAccount := [20]byte{18: 0x00, 19: 0xa2}
	if !bytes.Equal(data[:20], wantAccount[:]) {
		t.Errorf("account bytes: got %x", data[:20])
	}
	if !bytes.Equal(data[20:], ref[:]) {
		t.Errorf("reference bytes: got %x, want %x", data[20:], ref[:])
	}
}

// TestAccountMemo_RejectsBadAccount rejects an account that is not 20 bytes.
func TestAccountMemo_RejectsBadAccount(t *testing.T) {
	if _, err := accountMemo(core.DepositDestination{Account: "0xdead"}); err == nil {
		t.Error("short account accepted")
	}
	if _, err := accountMemo(core.DepositDestination{Account: "not-hex"}); err == nil {
		t.Error("non-hex account accepted")
	}
}

// TestAccountMemo_ClearnetAccountInputSet pins the exact accepted/rejected
// input set for accountMemo's account decoding (via
// core.ParseClearnetAccount): bare hex, an optional case-insensitive "0x"
// prefix, a yellow://.../user/<hex> URI's last segment, and surrounding
// whitespace. It also pins that an ADR-015 sub-account URI
// (yellow://.../user/<addr>/tag/<32-byte-ref>) is rejected rather than silently
// parsed as the trailing 32-byte reference.
func TestAccountMemo_ClearnetAccountInputSet(t *testing.T) {
	const addrHex = "000102030405060708090a0b0c0d0e0f10111213"
	const refHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	want, err := hex.DecodeString(addrHex)
	if err != nil {
		t.Fatal(err)
	}
	var wantAccount [20]byte
	copy(wantAccount[:], want)

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
			mw, err := accountMemo(core.DepositDestination{Account: in})
			if err != nil {
				t.Fatalf("accountMemo(%q) error = %v, want nil", in, err)
			}
			data, err := hex.DecodeString(mw.Memo.MemoData)
			if err != nil {
				t.Fatalf("MemoData not hex: %v", err)
			}
			if !bytes.Equal(data[:20], wantAccount[:]) {
				t.Fatalf("accountMemo(%q) account = %x, want %x", in, data[:20], wantAccount)
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
			if _, err := accountMemo(core.DepositDestination{Account: in}); err == nil {
				t.Fatalf("accountMemo(%q) error = nil, want error", in)
			}
		})
	}
}

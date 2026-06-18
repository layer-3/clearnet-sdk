package xrpl

import (
	"bytes"
	"encoding/hex"
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

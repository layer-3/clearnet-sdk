package btc

import (
	"fmt"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/internal/depositidtest"
)

type depositIDVector struct {
	Name string `json:"name"`
	TxID string `json:"txid"`
	Vout uint32 `json:"vout"`
	ID   string `json:"id"`
}

func readDepositIDVectors(t *testing.T) []depositIDVector {
	return depositidtest.Vectors[depositIDVector](t, "btc")
}

// The vectors were generated from custody's deposit watcher; DepositID must
// reproduce them byte for byte, and parseDepositID must invert them.
func TestDepositID_GoldenVectors(t *testing.T) {
	for _, v := range readDepositIDVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			if got := DepositID(v.TxID, v.Vout); got != v.ID {
				t.Fatalf("DepositID() = %s, want %s", got, v.ID)
			}
			vout, err := parseDepositID(v.TxID, v.ID)
			if err != nil || vout != v.Vout {
				t.Fatalf("parseDepositID() = (%d, %v), want (%d, nil)", vout, err, v.Vout)
			}
		})
	}
}

func TestParseDepositIDInputs(t *testing.T) {
	txHash := strings.Repeat("ab", 32)
	notShape := func(id string) string {
		return fmt.Sprintf("btc: deposit ID %q is not <txid>:<vout> for txid %s", id, txHash)
	}
	badIndex := func(id string) string {
		return fmt.Sprintf("btc: deposit ID %q has an invalid output index", id)
	}
	accepted := map[string]uint32{
		txHash + ":0":          0,
		txHash + ":7":          7,
		txHash + ":4294967295": 4294967295,
	}
	for id, want := range accepted {
		if got, err := parseDepositID(txHash, id); err != nil || got != want {
			t.Fatalf("parseDepositID(%q) = (%d, %v), want (%d, nil)", id, got, err, want)
		}
	}
	rejected := map[string]string{
		"":                             notShape(""),
		txHash:                         notShape(txHash),
		strings.ToUpper(txHash) + ":0": notShape(strings.ToUpper(txHash) + ":0"),
		"x" + txHash + ":0":            notShape("x" + txHash + ":0"),
		txHash + ";0":                  notShape(txHash + ";0"),
		txHash + ":":                   badIndex(txHash + ":"),
		txHash + ":4294967296":         badIndex(txHash + ":4294967296"),
		txHash + ":01":                 badIndex(txHash + ":01"),
		txHash + ":+1":                 badIndex(txHash + ":+1"),
		txHash + ":-1":                 badIndex(txHash + ":-1"),
		txHash + ":1 ":                 badIndex(txHash + ":1 "),
		txHash + ":1:2":                badIndex(txHash + ":1:2"),
		txHash + "::1":                 badIndex(txHash + "::1"),
		txHash + ":0x1":                badIndex(txHash + ":0x1"),
		txHash + ":1_0":                badIndex(txHash + ":1_0"),
	}
	for id, want := range rejected {
		if _, err := parseDepositID(txHash, id); err == nil || err.Error() != want {
			t.Fatalf("parseDepositID(%q) error = %v, want %q", id, err, want)
		}
	}
}

func TestPaysDepositScript(t *testing.T) {
	script := []byte{0x00, 0x20, 0xaa}
	tests := []struct {
		name          string
		value         int64
		scriptPubKey  []byte
		depositScript []byte
		want          bool
	}{
		{"pays", 1, []byte{0x00, 0x20, 0xaa}, script, true},
		{"zero value", 0, script, script, false},
		{"negative value", -1, script, script, false},
		{"other script", 1, []byte{0x00, 0x20, 0xab}, script, false},
		{"prefix only", 1, script[:2], script, false},
		{"longer script", 1, append(append([]byte{}, script...), 0), script, false},
		{"empty deposit script", 1, nil, nil, false},
		{"empty output script", 1, nil, script, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PaysDepositScript(tc.value, tc.scriptPubKey, tc.depositScript); got != tc.want {
				t.Fatalf("PaysDepositScript = %v, want %v", got, tc.want)
			}
		})
	}
}

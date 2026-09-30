package evm

import (
	"encoding/json"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// depositIDVector mirrors one entry of the ISS-068 shared golden vectors
// (testdata/deposit_id_vectors.json), also consumed by custody's Foundry
// tests and the TS SDK. Field names match the JSON exactly.
type depositIDVector struct {
	Name      string `json:"name"`
	ChainID   int64  `json:"chainId"`
	Vault     string `json:"vault"`
	Depositor string `json:"depositor"`
	Key       string `json:"key"`
	// Sequence is a decimal string, not a JSON number: the max_nonce
	// vector's sequence is 2^64-1, which round-trips exactly through Go's
	// strconv but would lose precision if parsed as a JSON number in a
	// float64-based decoder (as the TS SDK's test also must account for).
	Sequence   string `json:"sequence"`
	NonceHex   string `json:"nonce"`
	ExpectedID string `json:"id"`
}

func readDepositIDVectors(t *testing.T) []depositIDVector {
	t.Helper()
	raw, err := os.ReadFile("testdata/deposit_id_vectors.json")
	if err != nil {
		t.Fatalf("read golden vectors: %v", err)
	}
	var doc struct {
		Vectors []depositIDVector `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse golden vectors: %v", err)
	}
	if len(doc.Vectors) == 0 {
		t.Fatalf("golden vectors file has no vectors")
	}
	return doc.Vectors
}

// The SDK's DepositID reproduces the exact bytes custody's Foundry tests
// and the TS SDK compute for the same shared vectors.
func TestDepositID_GoldenVectors(t *testing.T) {
	for _, v := range readDepositIDVectors(t) {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			nonce, ok := new(big.Int).SetString(strings.TrimPrefix(v.NonceHex, "0x"), 16)
			if !ok {
				t.Fatalf("bad nonce hex %q", v.NonceHex)
			}
			got := DepositID(big.NewInt(v.ChainID), common.HexToAddress(v.Vault), common.HexToAddress(v.Depositor), nonce)
			if got != v.ExpectedID {
				t.Fatalf("DepositID() = %s, want %s", got, v.ExpectedID)
			}

			// Round-trip ComposeNonce/SplitNonce against the vector's own key/sequence.
			key, ok := new(big.Int).SetString(v.Key, 10)
			if !ok {
				t.Fatalf("bad key %q", v.Key)
			}
			sequence, err := strconv.ParseUint(v.Sequence, 10, 64)
			if err != nil {
				t.Fatalf("bad sequence %q: %v", v.Sequence, err)
			}
			composed := ComposeNonce(key, sequence)
			if composed.Cmp(nonce) != 0 {
				t.Fatalf("ComposeNonce(%s, %d) = %s, want %s", key, sequence, composed, nonce)
			}
			gotKey, gotSeq := SplitNonce(nonce)
			if gotKey.Cmp(key) != 0 || gotSeq != sequence {
				t.Fatalf("SplitNonce(%s) = (%s, %d), want (%s, %d)", nonce, gotKey, gotSeq, key, sequence)
			}
		})
	}
}

// Flipping any single hash input changes the ID.
func TestDepositID_EachInputChangesID(t *testing.T) {
	vault := common.HexToAddress("0x1111111111111111111111111111111111111111")
	depositor := common.HexToAddress("0x2222222222222222222222222222222222222222")
	nonce := ComposeNonce(big.NewInt(7), 3)
	base := DepositID(big.NewInt(1), vault, depositor, nonce)

	cases := map[string]string{
		"chainid":   DepositID(big.NewInt(2), vault, depositor, nonce),
		"vault":     DepositID(big.NewInt(1), common.HexToAddress("0x3333333333333333333333333333333333333333"), depositor, nonce),
		"depositor": DepositID(big.NewInt(1), vault, common.HexToAddress("0x4444444444444444444444444444444444444444"), nonce),
		"nonce":     DepositID(big.NewInt(1), vault, depositor, ComposeNonce(big.NewInt(7), 4)),
	}
	for name, id := range cases {
		if id == base {
			t.Fatalf("changing %s did not change the deposit ID", name)
		}
	}
}

func TestParseDepositID(t *testing.T) {
	good := DepositID(big.NewInt(1), common.HexToAddress("0x11"), common.HexToAddress("0x22"), ComposeNonce(big.NewInt(0), 0))
	if _, err := ParseDepositID(good); err != nil {
		t.Fatalf("ParseDepositID(%q) = %v, want nil", good, err)
	}

	rejected := []string{
		"",
		"0x1234",
		strings.Repeat("a", 64),                // missing 0x
		"0x" + strings.Repeat("A", 64),         // uppercase hex not accepted
		"0x" + strings.Repeat("g", 64),         // non-hex
		"0x" + strings.Repeat("de", 32) + "/3", // old txHash/logIndex shape
	}
	for _, id := range rejected {
		if _, err := ParseDepositID(id); err == nil {
			t.Fatalf("ParseDepositID(%q) = nil error, want ErrInvalidDepositID", id)
		}
	}
}

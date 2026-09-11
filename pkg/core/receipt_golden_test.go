package core

import (
	"encoding/hex"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

var updateReceiptGoldens = flag.Bool("update-receipt-goldens", false, "regenerate receipt CBOR golden fixtures")

func TestGoldens_Receipts(t *testing.T) {
	root := receiptGoldenRoot(t)
	cases := []struct {
		name string
		body interface{ MarshalCBOR(io.Writer) error }
	}{
		{name: "burn", body: &BurnReceipt{
			WithdrawalID:  [32]byte{0x01, 0x02, 0x03},
			BlockEntryRef: BlockEntryRef{BlockHash: [32]byte{0xaa, 0xbb}, EntryIndex: 9},
			TxID:          "0xdeadbeef", Status: WithdrawalExecuted,
			Proof: ReceiptProof{SignerEpoch: 7, Signatures: [][]byte{{0x10, 0x11}, {0x20, 0x21, 0x22}}},
		}},
		{name: "mint", body: &MintReceipt{
			TxID: "0xfeed", Account: "yellow://ynet/user/0xabc",
			AssetURI: "yellow://ynet/asset/0x0000000000000000000000000000000000001234/evm/1/0x1",
			Amount:   decimal.NewFromInt(125),
			Proof:    ReceiptProof{SignerEpoch: 8, Signatures: [][]byte{{0x30, 0x31}}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(root, tc.name+".golden.hex")
			var buf goldenBuffer
			if err := tc.body.MarshalCBOR(&buf); err != nil {
				t.Fatal(err)
			}
			encoded := hex.EncodeToString(buf.bytes)
			if *updateReceiptGoldens {
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(encoded+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v (run with -update-receipt-goldens)", path, err)
			}
			if string(want) != encoded+"\n" {
				t.Fatalf("CBOR drift: got %s, want %s", encoded, string(want))
			}
		})
	}
}

type goldenBuffer struct{ bytes []byte }

func (b *goldenBuffer) Write(p []byte) (int, error) {
	b.bytes = append(b.bytes, p...)
	return len(p), nil
}

func receiptGoldenRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "goldens", "receipts")
}

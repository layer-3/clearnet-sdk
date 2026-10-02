package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain/internal/depositidtest"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

type depositIDVector struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	ID   string `json:"id"`
}

// The vectors were generated from custody's deposit watcher; DepositID must
// reproduce them byte for byte.
func TestDepositID_GoldenVectors(t *testing.T) {
	for _, v := range depositidtest.Vectors[depositIDVector](t, "xrpl") {
		t.Run(v.Name, func(t *testing.T) {
			if got := DepositID(v.Hash); got != v.ID {
				t.Fatalf("DepositID() = %s, want %s", got, v.ID)
			}
		})
	}
}

const statusTestVault = "rVault1111111111111111111111111111"

func statusTestMemo(memoType string, data []byte) string {
	return fmt.Sprintf(`{"Memo":{"MemoType":%q,"MemoData":%q}}`,
		strings.ToUpper(hex.EncodeToString([]byte(memoType))), strings.ToUpper(hex.EncodeToString(data)))
}

func statusTestTx(t *testing.T, validated bool, result, txType, from, to string, memos ...string) *transactions.TxResponse {
	t.Helper()
	raw := fmt.Sprintf(`{"hash":"AB","validated":%t,"meta":{"TransactionResult":%q},"tx_json":{"TransactionType":%q,"Account":%q,"Destination":%q,"Memos":[%s]}}`,
		validated, result, txType, from, to, strings.Join(memos, ","))
	var tx transactions.TxResponse
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatalf("decode tx fixture: %v", err)
	}
	return &tx
}

func TestDepositStatus(t *testing.T) {
	account := make([]byte, 20)
	account[19] = 0xa2
	data := append(append([]byte{}, account...), make([]byte, 32)...)
	good := statusTestMemo(accountMemoType, data)
	zeroAccount := statusTestMemo(accountMemoType, make([]byte, 52))
	short := statusTestMemo(accountMemoType, data[:51])
	otherType := statusTestMemo("other", data)
	const sender = "rSender11111111111111111111111111"

	tests := []struct {
		name string
		tx   *transactions.TxResponse
		want core.DepositStatus
	}{
		{"validated deposit", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault, good), core.DepositConfirmed},
		{"unvalidated deposit", statusTestTx(t, false, "", "Payment", sender, statusTestVault, good), core.DepositPending},
		{"failed tec payment", statusTestTx(t, true, "tecPATH_DRY", "Payment", sender, statusTestVault, good), core.DepositAbsent},
		{"other destination", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, "rOther111111111111111111111111111", good), core.DepositAbsent},
		{"from the vault", statusTestTx(t, true, "tesSUCCESS", "Payment", statusTestVault, statusTestVault, good), core.DepositAbsent},
		{"not a payment", statusTestTx(t, true, "tesSUCCESS", "TrustSet", sender, statusTestVault, good), core.DepositAbsent},
		{"no memo", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault), core.DepositAbsent},
		{"other memo type", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault, otherType), core.DepositAbsent},
		{"short memo data", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault, short), core.DepositAbsent},
		{"zero account", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault, zeroAccount), core.DepositAbsent},
		{"memo after another memo", statusTestTx(t, true, "tesSUCCESS", "Payment", sender, statusTestVault, otherType, good), core.DepositConfirmed},
		{"unvalidated non-deposit", statusTestTx(t, false, "", "Payment", sender, "rOther111111111111111111111111111", good), core.DepositAbsent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := depositStatus(tc.tx, statusTestVault); got != tc.want {
				t.Fatalf("depositStatus() = %v, want %v", got, tc.want)
			}
		})
	}
}

// A deposit ID that is not the lower-cased hash is a caller error, reported
// before any chain lookup.
func TestChainDepositStatusRejectsMismatchedDepositID(t *testing.T) {
	d := &Depositor{vaultAddress: statusTestVault}
	txHash := strings.Repeat("AB", 32)
	for _, id := range []string{txHash, strings.Repeat("cd", 32), ""} {
		if got, err := d.ChainDepositStatus(context.Background(), txHash, id, 1); err == nil || got != core.DepositAbsent {
			t.Fatalf("ChainDepositStatus(%q) = (%v, %v), want (absent, error)", id, got, err)
		}
	}
}

func TestIsVaultDepositPayment(t *testing.T) {
	const vault, sender = "rVault1111111111111111111111111111", "rSender11111111111111111111111111"
	payment := func(edit func(transaction.FlatTransaction)) transaction.FlatTransaction {
		tx := transaction.FlatTransaction{"TransactionType": "Payment", "Account": sender, "Destination": vault}
		edit(tx)
		return tx
	}
	tests := []struct {
		name string
		tx   transaction.FlatTransaction
		want bool
	}{
		{"incoming payment", payment(func(transaction.FlatTransaction) {}), true},
		{"not a payment", payment(func(tx transaction.FlatTransaction) { tx["TransactionType"] = "TrustSet" }), false},
		{"missing type", payment(func(tx transaction.FlatTransaction) { delete(tx, "TransactionType") }), false},
		{"from the vault", payment(func(tx transaction.FlatTransaction) { tx["Account"] = vault }), false},
		{"missing sender", payment(func(tx transaction.FlatTransaction) { delete(tx, "Account") }), true},
		{"other destination", payment(func(tx transaction.FlatTransaction) { tx["Destination"] = sender }), false},
		{"destination case differs", payment(func(tx transaction.FlatTransaction) { tx["Destination"] = "rvault1111111111111111111111111111" }), false},
		{"missing destination", payment(func(tx transaction.FlatTransaction) { delete(tx, "Destination") }), false},
		{"non-string destination", payment(func(tx transaction.FlatTransaction) { tx["Destination"] = 7 }), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsVaultDepositPayment(tc.tx, vault); got != tc.want {
				t.Fatalf("IsVaultDepositPayment() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseDepositMemo(t *testing.T) {
	memo := func(memoType string, data []byte) any {
		return map[string]any{"Memo": map[string]any{
			"MemoType": hex.EncodeToString([]byte(memoType)),
			"MemoData": hex.EncodeToString(data),
		}}
	}
	first := make([]byte, 52)
	first[0], first[51] = 0x11, 0x12
	second := make([]byte, 52)
	second[0] = 0x21

	tests := []struct {
		name  string
		memos any
		want  []byte
	}{
		{"single memo", []any{memo(accountMemoType, first)}, first},
		{"first well-formed wins", []any{memo(accountMemoType, first), memo(accountMemoType, second)}, first},
		{"malformed memos skipped", []any{
			"entry",
			map[string]any{"Memo": 1},
			memo("other", second),
			memo(accountMemoType, second[:51]),
			map[string]any{"Memo": map[string]any{"MemoType": hex.EncodeToString([]byte(accountMemoType)), "MemoData": "zz"}},
			memo(accountMemoType, first),
		}, first},
		{"zero account parses", []any{memo(accountMemoType, make([]byte, 52))}, make([]byte, 52)},
		{"memos not an array", memo(accountMemoType, first), nil},
		{"nil memos", nil, nil},
		{"no deposit memo", []any{memo("other", first)}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			account, reference, ok := ParseDepositMemo(tc.memos)
			if ok != (tc.want != nil) {
				t.Fatalf("ok = %v, want %v", ok, tc.want != nil)
			}
			if !ok {
				return
			}
			if got := append(account[:], reference[:]...); hex.EncodeToString(got) != hex.EncodeToString(tc.want) {
				t.Fatalf("memo = %x, want %x", got, tc.want)
			}
		})
	}
}

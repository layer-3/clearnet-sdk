package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	binarycodec "github.com/Peersyst/xrpl-go/binary-codec"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// rpcFixture answers one rippled request, preserving the same JSON-RPC result
// envelope as a real server. Tests assert the actual finalizer's interpretation.
func rpcFixture(t *testing.T, fn func(method string, params []map[string]any) any) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request struct {
			Method string           `json:"method"`
			Params []map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result := fn(request.Method, request.Params)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestVerifyExecutionRequiresValidatedSuccessfulOutgoingPayment(t *testing.T) {
	wid := [32]byte{1, 2, 3}
	invoice := strings.ToUpper(hex.EncodeToString(wid[:]))
	txID := strings.Repeat("A", 64)
	entry := func(validated bool, result, typ, account string) map[string]any {
		return map[string]any{
			"validated": validated,
			"hash":      txID,
			"meta":      map[string]any{"TransactionResult": result},
			"tx_json":   map[string]any{"InvoiceID": invoice, "TransactionType": typ, "Account": account},
		}
	}
	for _, tc := range []struct {
		name string
		txs  []map[string]any
		want bool
	}{
		{"validated success", []map[string]any{entry(true, "tesSUCCESS", "Payment", floorTestVault)}, true},
		{"tec failed", []map[string]any{entry(true, "tecUNFUNDED_PAYMENT", "Payment", floorTestVault)}, false},
		{"unvalidated", []map[string]any{entry(false, "tesSUCCESS", "Payment", floorTestVault)}, false},
		{"wrong type", []map[string]any{entry(true, "tesSUCCESS", "AccountSet", floorTestVault)}, false},
		{"incoming Payment", []map[string]any{entry(true, "tesSUCCESS", "Payment", "rAnotherAccount")}, false},
		{"failed then successful", []map[string]any{entry(true, "tecUNFUNDED_PAYMENT", "Payment", floorTestVault), entry(true, "tesSUCCESS", "Payment", floorTestVault)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := rpcFixture(t, func(method string, _ []map[string]any) any {
				if method != "account_tx" {
					t.Errorf("unexpected RPC %s", method)
				}
				return map[string]any{"status": "success", "validated": true, "transactions": tc.txs}
			})
			f, _ := newFloorWithdrawalFinalizer(t, url)
			hash, executed, err := f.VerifyExecution(context.Background(), wid)
			if err != nil || executed != tc.want || (executed && hash != txID) || (!executed && hash != "") {
				t.Fatalf("VerifyExecution = %q, %t, %v; want executed=%t", hash, executed, err, tc.want)
			}
		})
	}
}

func TestVerifyExecutionRefusesIncompleteHistoryAsAbsenceProof(t *testing.T) {
	var pages atomic.Int32
	url := rpcFixture(t, func(method string, _ []map[string]any) any {
		if method != "account_tx" {
			t.Errorf("unexpected RPC %s", method)
		}
		return map[string]any{
			"status": "success", "transactions": []any{},
			"marker": map[string]any{"page": pages.Add(1)},
		}
	})
	f, _ := newFloorWithdrawalFinalizer(t, url)
	_, executed, err := f.VerifyExecution(context.Background(), [32]byte{1})
	if err == nil || executed || !strings.Contains(err.Error(), "outcome is unknown") {
		t.Fatalf("incomplete account history = executed:%t err:%v, want unknown error", executed, err)
	}
	if got := pages.Load(); got != executionScanPages {
		t.Fatalf("account_tx pages = %d, want bounded %d", got, executionScanPages)
	}
}

func TestVerifyRotationRequiresValidatedSignerList(t *testing.T) {
	for _, validated := range []bool{false, true} {
		t.Run(map[bool]string{false: "unvalidated", true: "validated"}[validated], func(t *testing.T) {
			url := rpcFixture(t, func(method string, params []map[string]any) any {
				if method != "account_objects" || len(params) != 1 || params[0]["ledger_index"] != "validated" {
					t.Errorf("want account_objects of validated ledger, got %s %+v", method, params)
				}
				entries := make([]any, len(floorNewSigners))
				for i, signer := range floorNewSigners {
					entries[i] = map[string]any{"SignerEntry": map[string]any{"Account": signer, "SignerWeight": 1}}
				}
				return map[string]any{"status": "success", "validated": validated, "account_objects": []any{map[string]any{
					"LedgerEntryType": "SignerList", "SignerQuorum": 2, "SignerEntries": entries,
				}}}
			})
			f, _ := newFloorRotationFinalizer(t, url)
			_, done, err := f.VerifyRotation(context.Background(), floorNewSigners, 2)
			if validated && (err != nil || !done) {
				t.Fatalf("validated rotation = %t, %v", done, err)
			}
			if !validated && (err == nil || done) {
				t.Fatalf("unvalidated rotation = %t, %v; want error and not done", done, err)
			}
		})
	}
}

func TestMalformedCandidatesRejectedBeforeAnyRPC(t *testing.T) {
	var calls atomic.Int32
	url := rpcFixture(t, func(method string, _ []map[string]any) any {
		calls.Add(1)
		t.Errorf("malformed candidate triggered live RPC %s", method)
		return map[string]any{"status": "success"}
	})
	withdrawal, _ := newFloorWithdrawalFinalizer(t, url)
	packed, op, wid := floorWithdrawal(t, floorTestFloor)
	var flat transaction.FlatTransaction
	if err := json.Unmarshal(packed, &flat); err != nil {
		t.Fatal(err)
	}
	flat["TransactionType"] = "AccountSet"
	malformed, err := CanonicalJSON(flat)
	if err != nil {
		t.Fatal(err)
	}
	if err := withdrawal.Validate(context.Background(), malformed, op, wid, floorTestBounds); !errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("withdrawal Validate error = %v, want candidate rejection", err)
	}
	rotation, _ := newFloorRotationFinalizer(t, url)
	var rot transaction.FlatTransaction
	if err := json.Unmarshal(floorRotation(t, floorNewSigners, 2, floorTestFloor), &rot); err != nil {
		t.Fatal(err)
	}
	rot["TransactionType"] = "Payment"
	invalidRotation, err := CanonicalJSON(rot)
	if err != nil {
		t.Fatal(err)
	}
	if err := rotation.Validate(context.Background(), [32]byte{}, invalidRotation, floorNewSigners, 2); !errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("rotation Validate error = %v, want candidate rejection", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("malformed candidates made %d RPCs", calls.Load())
	}
}

func TestMultisignBlobRequiresExactlyOneSigner(t *testing.T) {
	s, id := newXRPLSigner(t)
	blob := multisignBlob(t, context.Background(), s, id, id.ClassicAddress)
	decoded, err := binarycodec.Decode(blob)
	if err != nil {
		t.Fatal(err)
	}
	signers := decoded["Signers"].([]any)
	decoded["Signers"] = append(signers, signers[0])
	bad, err := binarycodec.Encode(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blobSignerAccount(bad); err == nil || !strings.Contains(err.Error(), "exactly one Signer") {
		t.Fatalf("two-signer blob error = %v, want rejection", err)
	}
}

func TestClassicAddressesAreCaseSensitiveInCanonicalBody(t *testing.T) {
	wid := [32]byte{7}
	op := &core.WithdrawalOp{Recipient: "rDeST1111111111111111111111111111"}
	body := transaction.FlatTransaction{
		"TransactionType": "Payment", "Account": floorTestVault, "Destination": op.Recipient,
		"InvoiceID": strings.ToUpper(hex.EncodeToString(wid[:])), "TicketSequence": uint32(5), "Sequence": uint32(0),
	}
	if err := validatePaymentShape(body, op, wid, floorTestVault); err != nil {
		t.Fatalf("matching addresses rejected: %v", err)
	}
	body["Account"] = strings.ToUpper(floorTestVault)
	if err := validatePaymentShape(body, op, wid, floorTestVault); err == nil {
		t.Fatal("case-changed classic Account accepted")
	}
	body["Account"] = floorTestVault
	body["Destination"] = strings.ToUpper(op.Recipient)
	if err := validatePaymentShape(body, op, wid, floorTestVault); err == nil {
		t.Fatal("case-changed classic Destination accepted")
	}
}

func TestEmptySubmittedBlobIsNotATransactionHash(t *testing.T) {
	if _, err := computeTxHash(""); err == nil {
		t.Fatal("empty blob produced an XRPL transaction hash")
	}
}

func TestAbsentNetworkIDPolicyIsCached(t *testing.T) {
	var reads atomic.Int32
	url := rpcFixture(t, func(method string, _ []map[string]any) any {
		if method != "server_info" {
			t.Errorf("unexpected RPC %s", method)
		}
		reads.Add(1)
		return map[string]any{"status": "success", "info": map[string]any{"network_id": 0}}
	})
	var cache networkIDCache
	for i := 0; i < 2; i++ {
		call, err := rpcCall(context.Background(), url, &cache)
		if err != nil {
			t.Fatal(err)
		}
		policy, err := resolveNetworkIDPolicy(&cache, call)
		if err != nil || policy.Required || call.NetworkID != 0 {
			t.Fatalf("policy = %+v, NetworkID = %d, err = %v", policy, call.NetworkID, err)
		}
	}
	if got := reads.Load(); got != 1 {
		t.Fatalf("server_info reads = %d, want 1", got)
	}
}

type safetyTicket uint32

func (t safetyTicket) TicketFor(context.Context, [32]byte) (uint32, error) { return uint32(t), nil }

// xrpl-go caps its autofilled fee at 2 XRP. The node must refuse the body
// immediately when that fee exceeds the 1 XRP ceiling its followers enforce.
func TestPackRejectsAutofilledFeeAboveFollowerCeiling(t *testing.T) {
	url := rpcFixture(t, func(method string, _ []map[string]any) any {
		switch method {
		case "server_info":
			return map[string]any{"status": "success", "info": map[string]any{
				"network_id": 0, "load_factor": 200000,
				"validated_ledger": map[string]any{"base_fee_xrp": 0.00001},
			}}
		case "ledger":
			return map[string]any{"status": "success", "ledger_index": 1000}
		case "account_info":
			return map[string]any{"status": "success", "account_data": map[string]any{"Sequence": 1}}
		default:
			t.Errorf("unexpected RPC %s", method)
			return map[string]any{"status": "error"}
		}
	})
	signer, vault := newXRPLSigner(t)
	_, recipient := newXRPLSigner(t)
	withdrawal, err := NewWithdrawalFinalizer(url, vault.ClassicAddress, 1, signer, safetyTicket(7), NewAssetResolver(AssetResolverConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	withdrawal.SetStandaloneLedgerMode(true)
	op := &core.WithdrawalOp{
		Recipient: recipient.ClassicAddress,
		AssetURI:  "yellow://ynet/asset/0x0000000000000000000000000000000000001234/xrpl/0/0",
		Amount:    decimal.NewFromInt(1),
	}
	if _, err := withdrawal.Pack(context.Background(), op, [32]byte{1}, floorTestBounds); err == nil || !strings.Contains(err.Error(), "exceeds ceiling") {
		t.Fatalf("withdrawal Pack error = %v, want fee ceiling error", err)
	}
	rotation, err := NewRotationFinalizer(url, vault.ClassicAddress, 1, signer)
	if err != nil {
		t.Fatal(err)
	}
	rotation.SetStandaloneLedgerMode(true)
	if _, err := rotation.Pack(context.Background(), [32]byte{2}, []string{vault.ClassicAddress}, 1); err == nil || !strings.Contains(err.Error(), "exceeds ceiling") {
		t.Fatalf("rotation Pack error = %v, want fee ceiling error", err)
	}
}

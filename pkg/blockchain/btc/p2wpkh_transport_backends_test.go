package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func testBackendRawTx(t *testing.T) ([]byte, string) {
	t.Helper()
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(&wire.OutPoint{Hash: chainhash.Hash{1}, Index: 2}, nil, nil))
	tx.AddTxOut(wire.NewTxOut(1_000, []byte{txscript.OP_TRUE}))
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), tx.TxHash().String()
}

func TestCoreP2WPKHBackendScanTransport(t *testing.T) {
	address := "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqdku202"
	txID := strings.Repeat("1", 64)
	script := "0014" + strings.Repeat("00", 20)
	rawTx, broadcastTxID := testBackendRawTx(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		switch req.Method {
		case "scantxoutset":
			if !bytes.Contains(req.Params, []byte("addr("+address+")")) {
				t.Errorf("scantxoutset params = %s", req.Params)
			}
			writeCoreResult(t, w, fmt.Sprintf(`{"success":true,"height":200,"total_amount":1.0,"unspents":[{"txid":%q,"vout":2,"scriptPubKey":%q,"amount":1.0,"height":198}]}`, txID, script))
		case "estimatesmartfee":
			writeCoreResult(t, w, `{"feerate":0.00100001}`)
		case "sendrawtransaction":
			if !bytes.Contains(req.Params, []byte(hex.EncodeToString(rawTx))) {
				t.Errorf("sendrawtransaction params = %s", req.Params)
			}
			writeCoreResult(t, w, fmt.Sprintf("%q", broadcastTxID))
		case "getrawtransaction":
			writeCoreResult(t, w, fmt.Sprintf(`{"txid":%q,"confirmations":3,"vout":[]}`, broadcastTxID))
		default:
			t.Errorf("unexpected RPC method %q", req.Method)
		}
	}))
	defer server.Close()

	backend, err := NewCoreP2WPKHBackend(
		NewClient(server.URL, "", "", "", WithHTTPClient(server.Client())),
		CoreP2WPKHBackendConfig{UTXOMode: CoreScanTxOutSetUTXOs},
	)
	if err != nil {
		t.Fatal(err)
	}
	utxos, err := backend.ListUnspent(context.Background(), address, 3)
	if err != nil || len(utxos) != 1 {
		t.Fatalf("ListUnspent = %+v, %v", utxos, err)
	}
	if got := utxos[0]; got.TxID != txID || got.Vout != 2 || got.AmountSats != 100_000_000 || got.Confirmations != 3 || hex.EncodeToString(got.ScriptPubKey) != script {
		t.Fatalf("UTXO = %+v", got)
	}
	if rate, err := backend.FeeRateSatPerVByte(context.Background(), 6); err != nil || rate != 101 {
		t.Fatalf("FeeRateSatPerVByte = %d, %v; want 101", rate, err)
	}
	if got, err := backend.Broadcast(context.Background(), rawTx); err != nil || got != broadcastTxID {
		t.Fatalf("Broadcast = %q, %v", got, err)
	}
	if confirmations, known, err := backend.GetTransactionConfirmations(context.Background(), broadcastTxID); err != nil || !known || confirmations != 3 {
		t.Fatalf("GetTransactionConfirmations = %d, %v, %v", confirmations, known, err)
	}
}

func TestCoreP2WPKHBackendWalletModeAndFeeChoices(t *testing.T) {
	var walletRequests, feeRequests int
	server := newCoreTestServer(t, func(w http.ResponseWriter, r *http.Request, method string) {
		switch method {
		case "listunspent":
			walletRequests++
			if r.URL.EscapedPath() != "/wallet/payer" {
				t.Errorf("wallet path = %q", r.URL.EscapedPath())
			}
			writeCoreResult(t, w, `[]`)
		case "estimatesmartfee":
			feeRequests++
			writeCoreResult(t, w, `{"errors":["Insufficient data or no feerate found"]}`)
		default:
			t.Errorf("unexpected method %q", method)
		}
	})
	defer server.Close()
	client := NewClient(server.URL, "", "", "payer", WithHTTPClient(server.Client()))

	wallet, err := NewCoreP2WPKHBackend(client, CoreP2WPKHBackendConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wallet.ListUnspent(context.Background(), "address", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := wallet.FeeRateSatPerVByte(context.Background(), 6); !errors.Is(err, ErrP2WPKHFeeEstimateUnavailable) {
		t.Fatalf("fee error = %v", err)
	}

	fixed, err := NewCoreP2WPKHBackend(client, CoreP2WPKHBackendConfig{FixedFeeRateSatPerVByte: 7})
	if err != nil {
		t.Fatal(err)
	}
	if rate, err := fixed.FeeRateSatPerVByte(context.Background(), 6); err != nil || rate != 7 {
		t.Fatalf("fixed fee = %d, %v", rate, err)
	}
	if walletRequests != 1 || feeRequests != 1 {
		t.Fatalf("wallet requests = %d, fee requests = %d", walletRequests, feeRequests)
	}
}

func TestCoreP2WPKHBackendConstructorValidation(t *testing.T) {
	client := NewClient("http://unused.invalid", "", "", "")
	for _, tc := range []struct {
		name   string
		client *Client
		cfg    CoreP2WPKHBackendConfig
	}{
		{name: "nil client"},
		{name: "invalid mode", client: client, cfg: CoreP2WPKHBackendConfig{UTXOMode: CoreUTXOMode(99)}},
		{name: "negative fixed fee", client: client, cfg: CoreP2WPKHBackendConfig{FixedFeeRateSatPerVByte: -1}},
		{name: "negative request timeout", client: client, cfg: CoreP2WPKHBackendConfig{RequestTimeout: -1}},
		{name: "negative scan timeout", client: client, cfg: CoreP2WPKHBackendConfig{ScanTimeout: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewCoreP2WPKHBackend(tc.client, tc.cfg); err == nil {
				t.Fatal("constructor unexpectedly succeeded")
			}
		})
	}
}

func TestCoreP2WPKHBackendConfirmationTxIDValidation(t *testing.T) {
	requestedTxID := strings.Repeat("a", 64)
	returnedTxID := strings.Repeat("b", 64)
	requests := 0
	server := newCoreTestServer(t, func(w http.ResponseWriter, _ *http.Request, method string) {
		requests++
		if method != "getrawtransaction" {
			t.Errorf("unexpected method %q", method)
		}
		writeCoreResult(t, w, fmt.Sprintf(`{"txid":%q,"confirmations":1,"vout":[]}`, returnedTxID))
	})
	defer server.Close()
	backend, err := NewCoreP2WPKHBackend(
		NewClient(server.URL, "", "", "", WithHTTPClient(server.Client())),
		CoreP2WPKHBackendConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := backend.GetTransactionConfirmations(context.Background(), "not-a-txid"); err == nil {
		t.Fatal("invalid query transaction ID unexpectedly succeeded")
	}
	if requests != 0 {
		t.Fatalf("invalid query made %d RPC requests", requests)
	}
	if _, _, err := backend.GetTransactionConfirmations(context.Background(), requestedTxID); err == nil || !strings.Contains(err.Error(), "transaction ID mismatch") {
		t.Fatalf("mismatched response error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("mismatched query made %d RPC requests", requests)
	}
}

func testEsploraAddress(t *testing.T) (string, string) {
	t.Helper()
	hash := make([]byte, 20)
	hash[0] = 1
	address, err := btcutil.NewAddressWitnessPubKeyHash(hash, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		t.Fatal(err)
	}
	return address.EncodeAddress(), hex.EncodeToString(script)
}

func TestEsploraP2WPKHBackendTransport(t *testing.T) {
	address, script := testEsploraAddress(t)
	txID := strings.Repeat("2", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/address/" + address + "/utxo":
			fmt.Fprintf(w, `[{"txid":%q,"vout":0,"value":5000,"status":{"confirmed":true,"block_height":198}}]`, txID)
		case "/api/blocks/tip/height":
			fmt.Fprint(w, "200")
		case "/api/tx/" + txID:
			fmt.Fprintf(w, `{"vout":[{"scriptpubkey":%q,"value":5000}]}`, script)
		case "/api/tx/" + txID + "/status":
			fmt.Fprint(w, `{"confirmed":true,"block_height":198}`)
		case "/api/fee-estimates":
			fmt.Fprint(w, `{"6":2.01}`)
		case "/api/tx":
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "text/plain" {
				t.Errorf("broadcast request = %s, content-type %q", r.Method, r.Header.Get("Content-Type"))
			}
			fmt.Fprint(w, txID)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	backend, err := NewEsploraP2WPKHBackend(EsploraP2WPKHBackendConfig{
		BaseURL: server.URL + "/api/", Network: &chaincfg.TestNet3Params, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	utxos, err := backend.ListUnspent(context.Background(), address, 3)
	if err != nil || len(utxos) != 1 {
		t.Fatalf("ListUnspent = %+v, %v", utxos, err)
	}
	if got := utxos[0]; got.TxID != txID || got.AmountSats != 5_000 || got.Confirmations != 3 || hex.EncodeToString(got.ScriptPubKey) != script {
		t.Fatalf("UTXO = %+v", got)
	}
	if rate, err := backend.FeeRateSatPerVByte(context.Background(), 6); err != nil || rate != 3 {
		t.Fatalf("FeeRateSatPerVByte = %d, %v", rate, err)
	}
	if got, err := backend.Broadcast(context.Background(), []byte{1}); err != nil || got != txID {
		t.Fatalf("Broadcast = %q, %v", got, err)
	}
	if confirmations, known, err := backend.GetTransactionConfirmations(context.Background(), txID); err != nil || !known || confirmations != 3 {
		t.Fatalf("GetTransactionConfirmations = %d, %v, %v", confirmations, known, err)
	}
}

func TestEsploraP2WPKHBackendStatusAndFeeFallback(t *testing.T) {
	txID := strings.Repeat("3", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tx/" + txID + "/status":
			http.NotFound(w, r)
		case "/fee-estimates":
			fmt.Fprint(w, `{"2":1}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	backend, err := NewEsploraP2WPKHBackend(EsploraP2WPKHBackendConfig{
		BaseURL: server.URL, Network: &chaincfg.TestNet3Params, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirmations, known, err := backend.GetTransactionConfirmations(context.Background(), txID); err != nil || known || confirmations != 0 {
		t.Fatalf("unknown confirmations = %d, %v, %v", confirmations, known, err)
	}
	if _, err := backend.FeeRateSatPerVByte(context.Background(), 6); !errors.Is(err, ErrP2WPKHFeeEstimateUnavailable) {
		t.Fatalf("fee error = %v", err)
	}

	fixed, err := NewEsploraP2WPKHBackend(EsploraP2WPKHBackendConfig{
		BaseURL: server.URL, Network: &chaincfg.TestNet3Params, HTTPClient: server.Client(), FixedFeeRateSatPerVByte: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rate, err := fixed.FeeRateSatPerVByte(context.Background(), 6); err != nil || rate != 9 {
		t.Fatalf("fixed fee = %d, %v", rate, err)
	}
}

func TestEsploraP2WPKHBackendConstructorValidation(t *testing.T) {
	for _, cfg := range []EsploraP2WPKHBackendConfig{
		{},
		{BaseURL: "https://example.test"},
		{BaseURL: "://bad", Network: &chaincfg.TestNet3Params},
		{BaseURL: "ftp://example.test", Network: &chaincfg.TestNet3Params},
		{BaseURL: "https://example.test?query=1", Network: &chaincfg.TestNet3Params},
		{BaseURL: "https://example.test", Network: &chaincfg.TestNet3Params, FixedFeeRateSatPerVByte: -1},
		{BaseURL: "https://example.test", Network: &chaincfg.TestNet3Params, RequestTimeout: -1},
	} {
		if _, err := NewEsploraP2WPKHBackend(cfg); err == nil {
			t.Fatalf("constructor accepted %+v", cfg)
		}
	}
}

func TestEsploraP2WPKHBackendRejectsExcessiveUTXORows(t *testing.T) {
	address, _ := testEsploraAddress(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/address/"+address+"/utxo" {
			t.Errorf("unexpected follow-up request %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		rows := make([]map[string]any, maxEsploraUTXORows+1)
		for i := range rows {
			rows[i] = map[string]any{
				"txid":   strings.Repeat("1", 64),
				"vout":   i,
				"value":  1,
				"status": map[string]any{"confirmed": true, "block_height": 1},
			}
		}
		if err := json.NewEncoder(w).Encode(rows); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()
	backend, err := NewEsploraP2WPKHBackend(EsploraP2WPKHBackendConfig{
		BaseURL: server.URL, Network: &chaincfg.TestNet3Params, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := backend.ListUnspent(context.Background(), address, 1); err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("excessive UTXO error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("excessive UTXO response triggered %d HTTP requests; want 1", requests)
	}
}

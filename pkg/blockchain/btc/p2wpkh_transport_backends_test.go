package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	rawTx, _ := testBackendRawTx(t)
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
	if got, err := backend.Broadcast(context.Background(), rawTx); err != nil || got != txID {
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
		{BaseURL: "https://example.test", Network: &chaincfg.TestNet3Params, MaxUTXORows: -1},
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
		rows := make([]map[string]any, defaultEsploraMaxUTXORows+1)
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

// esploraFundedServer serves one address funded by a single transaction with
// outputs many outputs, so the row count of discovery can be varied
// independently of how many inputs a payment actually needs.
func esploraFundedServer(t *testing.T, address, script, fundingTxID string, values []int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/address/"+address+"/utxo":
			rows := make([]map[string]any, len(values))
			for i, value := range values {
				rows[i] = map[string]any{
					"txid":   fundingTxID,
					"vout":   i,
					"value":  value,
					"status": map[string]any{"confirmed": true, "block_height": 199},
				}
			}
			if err := json.NewEncoder(w).Encode(rows); err != nil {
				t.Errorf("encode UTXO rows: %v", err)
			}
		case r.URL.Path == "/blocks/tip/height":
			fmt.Fprint(w, "200")
		case r.URL.Path == "/tx/"+fundingTxID:
			vouts := make([]map[string]any, len(values))
			for i, value := range values {
				vouts[i] = map[string]any{"scriptpubkey": script, "value": value}
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"txid": fundingTxID, "vout": vouts}); err != nil {
				t.Errorf("encode funding transaction: %v", err)
			}
		case r.URL.Path == "/tx" && r.Method == http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read broadcast body: %v", err)
				return
			}
			raw, err := hex.DecodeString(strings.TrimSpace(string(body)))
			if err != nil {
				t.Errorf("decode broadcast body: %v", err)
				return
			}
			tx, err := deserializeTx(raw)
			if err != nil {
				t.Errorf("deserialize broadcast transaction: %v", err)
				return
			}
			fmt.Fprint(w, tx.TxHash().String())
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

// TestEsploraP2WPKHBackendUTXORowBoundIsNotAnInputBound proves that discovery
// bounds transport cost rather than coin selection: an address holding far more
// outputs than the default bound is spendable from a single sufficient output
// once MaxUTXORows admits the wallet, while P2WPKHConfig.MaxInputs keeps
// bounding the inputs the transaction actually spends.
func TestEsploraP2WPKHBackendUTXORowBoundIsNotAnInputBound(t *testing.T) {
	signer := p2wpkhTestKeySigner(t, 1)
	senderAddress, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(signer.PublicKey()), &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	senderScript, err := txscript.PayToAddrScript(senderAddress)
	if err != nil {
		t.Fatal(err)
	}
	address := senderAddress.EncodeAddress()
	script := hex.EncodeToString(senderScript)
	fundingTxID := strings.Repeat("4", 64)

	// One output alone covers the payment; the rest are dust-sized noise that
	// pushes the row count past the default bound.
	values := make([]int64, defaultEsploraMaxUTXORows+50)
	values[0] = 1_000_000
	for i := 1; i < len(values); i++ {
		values[i] = 1_000
	}
	server := esploraFundedServer(t, address, script, fundingTxID, values)
	defer server.Close()

	cfg := EsploraP2WPKHBackendConfig{
		BaseURL:                 server.URL,
		Network:                 &chaincfg.TestNet3Params,
		HTTPClient:              server.Client(),
		FixedFeeRateSatPerVByte: 1,
	}
	bounded, err := NewEsploraP2WPKHBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bounded.ListUnspent(context.Background(), address, 1); err == nil ||
		!strings.Contains(err.Error(), fmt.Sprintf("exceeds maximum %d", defaultEsploraMaxUTXORows)) {
		t.Fatalf("default bound error = %v", err)
	}

	cfg.MaxUTXORows = len(values)
	backend, err := NewEsploraP2WPKHBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	utxos, err := backend.ListUnspent(context.Background(), address, 1)
	if err != nil || len(utxos) != len(values) {
		t.Fatalf("raised bound ListUnspent = %d UTXOs, %v", len(utxos), err)
	}

	senderCfg := p2wpkhTestConfig()
	senderCfg.MinConfirmations = 1
	senderCfg.MaxInputs = defaultEsploraMaxUTXORows
	sender, err := NewP2WPKHSender(&chaincfg.TestNet3Params, backend, signer, senderCfg)
	if err != nil {
		t.Fatal(err)
	}
	if sender.Address() != address {
		t.Fatalf("sender address = %q, want %q", sender.Address(), address)
	}
	txID, err := sender.Send(context.Background(), p2wpkhTestRecipient(t, &chaincfg.TestNet3Params), 500_000)
	if err != nil {
		t.Fatalf("Send over a %d-output wallet: %v", len(values), err)
	}
	if err := validateHashString(txID); err != nil {
		t.Fatalf("Send returned invalid txid %q: %v", txID, err)
	}
}

func esploraRejectBroadcast(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, `sendrawtransaction RPC error: {"code":-27,"message":"Transaction already in block chain"}`)
}

// TestEsploraP2WPKHBackendBroadcastIdempotency proves that an ambiguous
// broadcast failure is resolved by an exact lookup instead of being surfaced to
// a caller who would retry with freshly selected inputs, and that a broadcast
// Esplora genuinely did not accept still fails.
func TestEsploraP2WPKHBackendBroadcastIdempotency(t *testing.T) {
	rawTx, txID := testBackendRawTx(t)
	otherTxID := strings.Repeat("7", 64)

	for _, tc := range []struct {
		name          string
		broadcast     http.HandlerFunc
		lookup        http.HandlerFunc
		wantTxID      string
		wantErrSubstr string
	}{
		{
			name:      "rejected broadcast of a mempool transaction is idempotent",
			broadcast: esploraRejectBroadcast,
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"txid":%q,"status":{"confirmed":false}}`, txID)
			},
			wantTxID: txID,
		},
		{
			name:      "rejected broadcast of a confirmed transaction is idempotent",
			broadcast: esploraRejectBroadcast,
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"txid":%q,"status":{"confirmed":true,"block_height":100}}`, txID)
			},
			wantTxID: txID,
		},
		{
			name:      "lost reply for an accepted transaction is idempotent",
			broadcast: hangUpEsploraBroadcast(t),
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"txid":%q}`, txID)
			},
			wantTxID: txID,
		},
		{
			name:          "genuinely rejected broadcast remains an error",
			broadcast:     esploraRejectBroadcast,
			lookup:        http.NotFound,
			wantErrSubstr: "Esplora HTTP 400",
		},
		{
			name:      "lookup returning a different transaction is not accepted",
			broadcast: esploraRejectBroadcast,
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"txid":%q}`, otherTxID)
			},
			wantErrSubstr: "Esplora HTTP 400",
		},
		{
			name:      "failing lookup leaves the broadcast a failure",
			broadcast: esploraRejectBroadcast,
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErrSubstr: "Esplora HTTP 400",
		},
		{
			name:          "lost reply for a transaction Esplora never saw remains an error",
			broadcast:     hangUpEsploraBroadcast(t),
			lookup:        http.NotFound,
			wantErrSubstr: "Esplora broadcast",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var broadcasts, lookups int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/tx":
					broadcasts++
					tc.broadcast(w, r)
				case r.Method == http.MethodGet && r.URL.Path == "/tx/"+txID:
					lookups++
					tc.lookup(w, r)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
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

			got, err := backend.Broadcast(context.Background(), rawTx)
			if tc.wantErrSubstr == "" {
				if err != nil || got != tc.wantTxID {
					t.Fatalf("Broadcast = %q, %v; want %q", got, err, tc.wantTxID)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Fatalf("Broadcast = %q, %v; want error containing %q", got, err, tc.wantErrSubstr)
			}
			if broadcasts != 1 {
				t.Fatalf("broadcast requests = %d, want 1", broadcasts)
			}
			if lookups != 1 {
				t.Fatalf("lookup requests = %d, want 1", lookups)
			}
		})
	}
}

// hangUpEsploraBroadcast drops the connection after the request is delivered,
// reproducing an accepted broadcast whose reply never reaches the caller.
func hangUpEsploraBroadcast(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("test server does not support hijacking")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack broadcast connection: %v", err)
			return
		}
		conn.Close()
	}
}

func TestEsploraP2WPKHBackendRejectsUndecodableBroadcast(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Errorf("undecodable transaction reached the network as %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	backend, err := NewEsploraP2WPKHBackend(EsploraP2WPKHBackendConfig{
		BaseURL: server.URL, Network: &chaincfg.TestNet3Params, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Broadcast(context.Background(), []byte{1}); err == nil ||
		!strings.Contains(err.Error(), "decode transaction before Esplora broadcast") {
		t.Fatalf("undecodable broadcast error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("undecodable broadcast made %d HTTP requests; want 0", requests)
	}
}

// TestEsploraP2WPKHBackendUnconfirmedStatusRequiresLookup pins the endpoint
// semantics the idempotency check depends on: Esplora answers 200
// {"confirmed":false} for a transaction it has never seen, so only the
// transaction endpoint distinguishes a mempool transaction from an absent one.
func TestEsploraP2WPKHBackendUnconfirmedStatusRequiresLookup(t *testing.T) {
	txID := strings.Repeat("5", 64)
	for _, tc := range []struct {
		name      string
		lookup    http.HandlerFunc
		wantKnown bool
	}{
		{
			name:      "absent transaction",
			lookup:    http.NotFound,
			wantKnown: false,
		},
		{
			name: "mempool transaction",
			lookup: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"txid":%q,"status":{"confirmed":false}}`, txID)
			},
			wantKnown: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/tx/" + txID + "/status":
					fmt.Fprint(w, `{"confirmed":false}`)
				case "/tx/" + txID:
					tc.lookup(w, r)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
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
			confirmations, known, err := backend.GetTransactionConfirmations(context.Background(), txID)
			if err != nil || confirmations != 0 || known != tc.wantKnown {
				t.Fatalf("GetTransactionConfirmations = %d, %v, %v; want 0, %v, nil", confirmations, known, err, tc.wantKnown)
			}
		})
	}
}

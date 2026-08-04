package btc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newCoreTestServer(t *testing.T, handler func(http.ResponseWriter, *http.Request, string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handler(w, r, request.Method)
	}))
}

func writeCoreResult(t *testing.T, w http.ResponseWriter, result string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintf(w, `{"result":%s,"error":null,"id":"sdk"}`, result)
	if err != nil {
		t.Errorf("write response: %v", err)
	}
}

func TestClientAuthenticatedRootAndWalletRequests(t *testing.T) {
	var sawRoot, sawWallet bool
	server := newCoreTestServer(t, func(w http.ResponseWriter, r *http.Request, method string) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "alice" || pass != "secret" {
			t.Errorf("BasicAuth = %q, %q, %v", user, pass, ok)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		switch method {
		case "getblockcount":
			sawRoot = true
			if r.URL.EscapedPath() != "/" {
				t.Errorf("root path = %q", r.URL.EscapedPath())
			}
			writeCoreResult(t, w, "123")
		case "listunspent":
			sawWallet = true
			if r.URL.EscapedPath() != "/wallet/wallet%20one" {
				t.Errorf("wallet path = %q", r.URL.EscapedPath())
			}
			writeCoreResult(t, w, "[]")
		default:
			t.Errorf("unexpected method %q", method)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "alice", "secret", "wallet one")
	if _, err := client.GetBlockCount(context.Background()); err != nil {
		t.Fatalf("GetBlockCount: %v", err)
	}
	if _, err := client.ListUnspent(context.Background(), 2, []string{"address"}); err != nil {
		t.Fatalf("ListUnspent: %v", err)
	}
	if !sawRoot || !sawWallet {
		t.Fatalf("sawRoot=%v sawWallet=%v", sawRoot, sawWallet)
	}
}

func TestClientTransportFailures(t *testing.T) {
	t.Run("HTTP status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "getblockcount") || !strings.Contains(err.Error(), "HTTP status 503") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("typed RPC error on non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"result":null,"error":{"code":-27,"message":"already in block chain"},"id":"sdk"}`)
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").SendRawTransaction(context.Background(), "00")
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -27 || !strings.Contains(err.Error(), "sendrawtransaction") {
			t.Fatalf("error = %v, RPC error = %#v", err, rpcErr)
		}
	})

	t.Run("typed RPC error on 2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"result":null,"error":{"code":-5,"message":"not found"},"id":"sdk"}`)
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").GetTxOut(context.Background(), "00", 0, true)
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -5 || !strings.Contains(err.Error(), "gettxout") {
			t.Fatalf("error = %v, RPC error = %#v", err, rpcErr)
		}
	})

	t.Run("oversized response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", int(maxRPCResponseBytes)+1))
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "exceeds") || !strings.Contains(err.Error(), "read response") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("response exactly at limit", func(t *testing.T) {
		prefix := `{"result":123,"error":null,"id":"sdk"}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, prefix+strings.Repeat(" ", int(maxRPCResponseBytes)-len(prefix)))
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "", "", "").GetBlockCount(context.Background())
		if err != nil || got != 123 {
			t.Fatalf("GetBlockCount = %d, %v", got, err)
		}
	})

	t.Run("malformed envelope", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{not-json`)
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "decode response envelope") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("malformed result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCoreResult(t, w, `"not-a-height"`)
		}))
		defer server.Close()
		_, err := NewClient(server.URL, "", "", "").GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "decode result") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("request marshal", func(t *testing.T) {
		client := NewClient("http://unused.invalid", "", "", "")
		err := client.call(context.Background(), "badparams", []any{func() {}}, nil)
		if err == nil || !strings.Contains(err.Error(), "badparams: marshal request") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("request construction", func(t *testing.T) {
		client := NewClient("://invalid", "", "", "")
		err := client.call(context.Background(), "brokenurl", []any{}, nil)
		if err == nil || !strings.Contains(err.Error(), "brokenurl: construct request") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("response read", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       readErrorCloser{},
			}, nil
		})}
		_, err := NewClient("http://bitcoind.invalid", "", "", "", WithHTTPClient(httpClient)).GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "getblockcount: read response") {
			t.Fatalf("error = %v", err)
		}
	})
}

type readErrorCloser struct{}

func (readErrorCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (readErrorCloser) Close() error             { return nil }

func TestClientContextCancellationAndHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
			writeCoreResult(t, w, "1")
		}
	}))
	defer server.Close()

	t.Run("caller cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewClient(server.URL, "", "", "").GetBlockCount(ctx)
		if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "transport") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("injected HTTP timeout", func(t *testing.T) {
		httpClient := &http.Client{Timeout: time.Millisecond}
		_, err := NewClient(server.URL, "", "", "", WithHTTPClient(httpClient)).GetBlockCount(context.Background())
		if err == nil || !strings.Contains(err.Error(), "transport") {
			t.Fatalf("error = %v", err)
		}
		var netErr interface{ Timeout() bool }
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("error does not preserve timeout: %v", err)
		}
	})
}

func TestClientGetBlockchainInfoSupportedChains(t *testing.T) {
	chains := []CoreChain{CoreChainMain, CoreChainTest, CoreChainTestnet4, CoreChainSignet, CoreChainRegtest}
	for _, chain := range chains {
		t.Run(string(chain), func(t *testing.T) {
			server := newCoreTestServer(t, func(w http.ResponseWriter, _ *http.Request, method string) {
				if method != "getblockchaininfo" {
					t.Errorf("method = %q", method)
				}
				writeCoreResult(t, w, fmt.Sprintf(`{"chain":%q}`, chain))
			})
			defer server.Close()
			got, err := NewClient(server.URL, "", "", "").GetBlockchainInfo(context.Background())
			if err != nil || got != chain {
				t.Fatalf("GetBlockchainInfo = %q, %v", got, err)
			}
		})
	}

	server := newCoreTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ string) {
		writeCoreResult(t, w, `{"chain":"unknown"}`)
	})
	defer server.Close()
	if _, err := NewClient(server.URL, "", "", "").GetBlockchainInfo(context.Background()); err == nil || !strings.Contains(err.Error(), "unsupported chain") {
		t.Fatalf("unsupported chain error = %v", err)
	}
}

func TestClientScanTxOutSetTypedExactResult(t *testing.T) {
	txid := strings.Repeat("1", 64)
	bestBlock := strings.Repeat("2", 64)
	server := newCoreTestServer(t, func(w http.ResponseWriter, _ *http.Request, method string) {
		if method != "scantxoutset" {
			t.Errorf("method = %q", method)
		}
		result := fmt.Sprintf(`{"success":true,"txouts":17,"height":110,"bestblock":%q,"total_amount":0.00000001,"unspents":[{"txid":%q,"vout":3,"scriptPubKey":"0014aabb","desc":"addr(example)#sum","amount":0.00000001,"height":100,"coinbase":true}]}`, bestBlock, txid)
		writeCoreResult(t, w, result)
	})
	defer server.Close()

	result, err := NewClient(server.URL, "", "", "").ScanTxOutSet(context.Background(), []string{"addr(example)"})
	if err != nil {
		t.Fatalf("ScanTxOutSet: %v", err)
	}
	if !result.Success || result.TxOuts != 17 || result.Height != 110 || result.BestBlock != bestBlock || result.TotalAmountSats != 1 {
		t.Fatalf("result metadata = %#v", result)
	}
	if len(result.Unspents) != 1 {
		t.Fatalf("unspents = %#v", result.Unspents)
	}
	u := result.Unspents[0]
	if u.TxID != txid || u.Vout != 3 || fmt.Sprintf("%x", u.ScriptPubKey) != "0014aabb" || u.Descriptor != "addr(example)#sum" || u.AmountSats != 1 || u.Height != 100 || u.Confirmations != 11 || !u.Coinbase {
		t.Fatalf("unspent = %#v", u)
	}
}

func TestClientScanTxOutSetRequestShape(t *testing.T) {
	var gotParams []json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Method != "scantxoutset" {
			t.Errorf("method = %q", request.Method)
		}
		gotParams = request.Params
		writeCoreResult(t, w, `{"success":true,"txouts":0,"height":0,"bestblock":"","total_amount":0,"unspents":[]}`)
	}))
	defer server.Close()
	if _, err := NewClient(server.URL, "", "", "").ScanTxOutSet(context.Background(), []string{"addr(one)", "addr(two)"}); err != nil {
		t.Fatalf("ScanTxOutSet: %v", err)
	}
	if len(gotParams) != 2 || string(gotParams[0]) != `"start"` || string(gotParams[1]) != `["addr(one)","addr(two)"]` {
		t.Fatalf("params = %s", gotParams)
	}
}

func TestClientScanTxOutSetAmountValidation(t *testing.T) {
	txid := strings.Repeat("a", 64)
	tests := []struct {
		name     string
		amount   string
		want     int64
		wantText string
	}{
		{name: "one satoshi", amount: "0.00000001", want: 1},
		{name: "maximum int64", amount: "92233720368.54775807", want: int64(^uint64(0) >> 1)},
		{name: "excessive precision", amount: "0.000000001", wantText: "sub-satoshi precision"},
		{name: "negative", amount: "-0.00000001", wantText: "negative"},
		{name: "overflow", amount: "92233720368.54775808", wantText: "outside the int64 range"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				result := fmt.Sprintf(`{"success":true,"txouts":1,"height":1,"bestblock":"","total_amount":%s,"unspents":[{"txid":%q,"vout":0,"scriptPubKey":"00","amount":%s,"height":1}]}`, test.amount, txid, test.amount)
				writeCoreResult(t, w, result)
			}))
			defer server.Close()
			result, err := NewClient(server.URL, "", "", "").ScanTxOutSet(context.Background(), []string{"addr(example)"})
			if test.wantText != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantText) {
					t.Fatalf("error = %v, want %q", err, test.wantText)
				}
				return
			}
			if err != nil || result.TotalAmountSats != test.want || result.Unspents[0].AmountSats != test.want {
				t.Fatalf("result = %#v, error = %v", result, err)
			}
		})
	}
}

func TestClientScanTxOutSetValidatesEachUnspentFieldIndependently(t *testing.T) {
	txid := strings.Repeat("b", 64)
	tests := []struct {
		name     string
		amount   string
		script   string
		wantText string
	}{
		{name: "invalid unspent amount with valid total", amount: "0.000000001", script: "00", wantText: "unspent 0 amount"},
		{name: "invalid unspent script with valid amount", amount: "0.00000001", script: "xyz", wantText: "unspent 0 scriptPubKey"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				result := fmt.Sprintf(`{"success":true,"txouts":1,"height":1,"bestblock":"","total_amount":0.00000001,"unspents":[{"txid":%q,"vout":0,"scriptPubKey":%q,"amount":%s,"height":1}]}`, txid, test.script, test.amount)
				writeCoreResult(t, w, result)
			}))
			defer server.Close()
			_, err := NewClient(server.URL, "", "", "").ScanTxOutSet(context.Background(), []string{"addr(example)"})
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error = %v, want %q", err, test.wantText)
			}
		})
	}
}

func TestClientScanAlreadyInProgressClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"result":null,"error":{"code":-8,"message":"Scan already in progress, use action abort or status"},"id":"sdk"}`)
	}))
	defer server.Close()
	_, err := NewClient(server.URL, "", "", "").ScanTxOutSet(context.Background(), []string{"addr(example)"})
	var rpcErr *RPCError
	if !errors.Is(err, ErrScanAlreadyInProgress) || !errors.As(err, &rpcErr) || rpcErr.Code != -8 {
		t.Fatalf("error = %v, RPC error = %#v", err, rpcErr)
	}
}

func TestClientEstimateSmartFeeExactCeiling(t *testing.T) {
	tests := []struct {
		name     string
		feeRate  string
		want     int64
		wantText string
	}{
		{name: "exact", feeRate: "0.00001000", want: 1},
		{name: "fraction rounds up", feeRate: "0.00001001", want: 2},
		{name: "sub sat per vbyte rounds up", feeRate: "0.00000001", want: 1},
		{name: "nonpositive", feeRate: "0", wantText: "non-positive"},
		{name: "negative", feeRate: "-0.00001000", wantText: "negative"},
		{name: "overflow", feeRate: "92233720368547.75808", wantText: "outside the int64 range"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCoreResult(t, w, `{"feerate":`+test.feeRate+`}`)
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, 9)
			if test.wantText != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantText) {
					t.Fatalf("error = %v, want %q", err, test.wantText)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("estimate = %d, %v; want %d", got, err, test.want)
			}
		})
	}
}

func TestClientEstimateUnavailableVersusFailure(t *testing.T) {
	t.Run("unavailable uses fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCoreResult(t, w, `{"errors":["Insufficient data or no feerate found"]}`)
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, 17)
		if err != nil || got != 17 {
			t.Fatalf("estimate = %d, %v", got, err)
		}
	})

	for _, fallback := range []int64{0, -1} {
		t.Run(fmt.Sprintf("unavailable rejects fallback %d", fallback), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCoreResult(t, w, `{"errors":["Insufficient data or no feerate found"]}`)
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, fallback)
			if err == nil || got != 0 || !strings.Contains(err.Error(), "non-positive fallback") {
				t.Fatalf("estimate = %d, %v", got, err)
			}
		})
	}

	for _, result := range []string{`{}`, `{"feerate":null}`, `{"feerate":null,"errors":[]}`, `{"errors":[""]}`} {
		t.Run("missing estimate is malformed "+result, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCoreResult(t, w, result)
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, 17)
			if err == nil || got != 0 || !strings.Contains(err.Error(), "decode result") {
				t.Fatalf("estimate = %d, %v", got, err)
			}
		})
	}

	t.Run("RPC failure does not use fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"result":null,"error":{"code":-1,"message":"failure"},"id":"sdk"}`)
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, 17)
		if err == nil || got != 0 {
			t.Fatalf("estimate = %d, %v", got, err)
		}
	})

	t.Run("malformed result does not use fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCoreResult(t, w, `{"feerate":"not-a-number"}`)
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "", "", "").EstimateSmartFeeSatPerVByte(context.Background(), 6, 17)
		if err == nil || got != 0 {
			t.Fatalf("estimate = %d, %v", got, err)
		}
	})
}

func TestClientSendRawTransactionValidatesResult(t *testing.T) {
	valid := strings.Repeat("c", 64)
	tests := []struct {
		name    string
		result  string
		want    string
		wantErr bool
	}{
		{name: "valid", result: `"` + valid + `"`, want: valid},
		{name: "missing", result: "__missing__", wantErr: true},
		{name: "null", result: `null`, wantErr: true},
		{name: "empty", result: `""`, wantErr: true},
		{name: "short", result: `"abcd"`, wantErr: true},
		{name: "nonhex", result: `"` + strings.Repeat("z", 64) + `"`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.result == "__missing__" {
					_, _ = io.WriteString(w, `{"error":null,"id":"sdk"}`)
					return
				}
				writeCoreResult(t, w, test.result)
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "", "", "").SendRawTransaction(context.Background(), "00")
			if test.wantErr {
				if err == nil || got != "" || !strings.Contains(err.Error(), "sendrawtransaction: decode result") {
					t.Fatalf("SendRawTransaction = %q, %v", got, err)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("SendRawTransaction = %q, %v", got, err)
			}
		})
	}
}

func TestClientExistingMonetaryResultsAreExact(t *testing.T) {
	server := newCoreTestServer(t, func(w http.ResponseWriter, _ *http.Request, method string) {
		switch method {
		case "listunspent":
			writeCoreResult(t, w, `[{"txid":"id","vout":0,"amount":92233720368.54775807,"confirmations":2,"scriptPubKey":"00"}]`)
		case "gettxout":
			writeCoreResult(t, w, `{"confirmations":3,"value":0.00000001,"scriptPubKey":{"hex":"00"}}`)
		case "getrawtransaction":
			writeCoreResult(t, w, `{"txid":"id","confirmations":4,"vout":[{"value":0.00000001,"scriptPubKey":{"hex":"00"}}]}`)
		default:
			t.Errorf("method = %q", method)
		}
	})
	defer server.Close()
	client := NewClient(server.URL, "", "", "wallet")
	unspents, err := client.ListUnspent(context.Background(), 1, nil)
	if err != nil || len(unspents) != 1 || unspents[0].AmountSats != int64(^uint64(0)>>1) {
		t.Fatalf("ListUnspent = %#v, %v", unspents, err)
	}
	txout, err := client.GetTxOut(context.Background(), "id", 0, true)
	if err != nil || txout.AmountSats != 1 {
		t.Fatalf("GetTxOut = %#v, %v", txout, err)
	}
	raw, err := client.GetRawTransaction(context.Background(), "id")
	if err != nil || raw.Vouts[0].ValueSats != 1 {
		t.Fatalf("GetRawTransaction = %#v, %v", raw, err)
	}
}

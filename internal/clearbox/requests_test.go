package clearbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func publicRequest(s *Server, path, peer, forwarded string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "https://sandbox.example"+path, nil)
	r.RemoteAddr = peer
	r.Header.Set("X-Forwarded-For", forwarded)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestPublicServingAndRateLimit(t *testing.T) {
	s, _, _ := fixture(t, Config{PublicURL: "https://sandbox.example/", RequestsPerSecond: 1})
	for range 2 {
		w := publicRequest(s, "/v1/gateway/info", "192.0.2.1:1234", "")
		var info client.NetworkInfo
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &info) != nil || info.RPCURL != "https://sandbox.example/rpc" {
			t.Fatal("public RPC URL was not advertised correctly", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Header().Get("X-Clearbox-Simulated") != "true" || strings.Contains(w.Body.String(), s.SeedPrivateKey) {
			t.Fatal("public browser headers or seed-key isolation failed")
		}
	}
	if w := publicRequest(s, "/v1/gateway/info", "192.0.2.1:1234", "203.0.113.10"); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("spoofed forwarded IP bypassed the limiter", w.Code)
	}
	for range 110 {
		if w := publicRequest(s, "/v1/pools", "192.0.2.1:1234", ""); w.Code != 429 {
			t.Fatal("rejected client exceeded its burst", w.Code)
		}
	}
	if w := publicRequest(s, "/healthz", "192.0.2.1:1234", ""); w.Code != 200 {
		t.Fatal("health probe was rate limited", w.Code)
	}
	if w := publicRequest(s, "/v1/gateway/info", "192.0.2.2:1234", ""); w.Code != 200 {
		t.Fatal("one client's limit affected another", w.Code)
	}
	r := httptest.NewRequest("OPTIONS", "https://sandbox.example/rpc", nil)
	r.Header.Set("Origin", "https://developer.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("public browser preflight failed", w.Code)
	}
}

func TestTrustedProxyBoundaryAndBoundedVisitors(t *testing.T) {
	s, _, _ := fixture(t, Config{PublicURL: "https://sandbox.example", RequestsPerSecond: 1, TrustedProxies: []string{"127.0.0.0/8"}})
	for range 2 {
		if w := publicRequest(s, "/v1/pools", "127.0.0.1:1234", "198.51.100.1"); w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if w := publicRequest(s, "/v1/pools", "127.0.0.1:1234", "198.51.100.1"); w.Code != 429 {
		t.Fatal("proxy client limit missing", w.Code)
	}
	if w := publicRequest(s, "/v1/pools", "127.0.0.1:1234", "198.51.100.2"); w.Code != 200 {
		t.Fatal("proxy clients share one bucket", w.Code)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 198.51.100.1")
	if s.traffic.clientIP(r).String() != "198.51.100.1" {
		t.Fatal("untrusted leftmost forwarding hop was accepted")
	}
	// A stream of distinct clients cannot grow the visitor table indefinitely.
	tokens, _ := configure(&Config{})
	for i := 0; i < 2048; i++ {
		ip := netip.AddrFrom4([4]byte{10, byte(i >> 8), byte(i), 1})
		tokens.clients[ip] = &visitor{seen: time.Now()}
	}
	if tokens.allow(r) || len(tokens.clients) != 2048 {
		t.Fatal("visitor table was not bounded")
	}
}

func TestPublicGlobalRequestAndTransferLimits(t *testing.T) {
	s, _, c := fixture(t, Config{PublicURL: "https://sandbox.example", RequestsPerSecond: 1000})
	s.traffic.global.SetLimit(0) // Freeze refill so burst checks do not depend on CPU speed.
	for i := 0; i < 100; i++ {
		if w := publicRequest(s, "/v1/pools", fmt.Sprintf("192.0.2.%d:1", i), ""); w.Code != 200 {
			t.Fatal("global burst ended early", i, w.Code)
		}
	}
	if w := publicRequest(s, "/v1/pools", "198.51.100.1:1", ""); w.Code != 429 {
		t.Fatal("global request limit missing", w.Code)
	}
	// Use a fresh server so the admission check is independent of read limits.
	s, _, c = fixture(t, Config{PublicURL: "https://sandbox.example", RequestsPerSecond: 1000})
	s.traffic.writes.SetLimit(0)
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	for range 4 {
		signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "1")
		if _, err := c.SubmitTransfer(context.Background(), signed); err != nil {
			t.Fatal(err)
		}
	}
	before := getBalance(t, c, s.SeedAddress, "YELLOW")
	signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "1")
	if _, err := c.SubmitTransfer(context.Background(), signed); err == nil {
		t.Fatal("unbounded write admission")
	}
	if getBalance(t, c, s.SeedAddress, "YELLOW") != before || len(s.ops) != 4 {
		t.Fatal("throttled transfer mutated state")
	}
}

func TestReadsAndRejectedTransfersDoNotCreateAccounts(t *testing.T) {
	s, _, c := fixture(t, Config{})
	for i := 1; i <= 20; i++ {
		address := fmt.Sprintf("0x%040x", i)
		if getBalance(t, c, address, "YELLOW") != "0" {
			t.Fatal("unknown address got funds")
		}
	}
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, bobKey, s.SeedAddress, []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "1")
	if _, err := c.SubmitTransfer(context.Background(), signed); err == nil {
		t.Fatal("unfunded sender admitted")
	}
	if _, err := c.Transfers(context.Background(), bob.Hex(), client.TransferPageOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(s.accounts) != 1 || len(s.ops) != 0 || s.dirty {
		t.Fatal("read/rejected request grew the ledger")
	}
}

func TestCapacityPreservesExistingOperations(t *testing.T) {
	s, _, c := fixture(t, Config{MaxOperations: 1})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	first := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1")
	sealed, err := c.SubmitTransfer(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	before := getBalance(t, c, s.SeedAddress, "YELLOW")
	second := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "1")
	if _, err := c.SubmitTransfer(context.Background(), second); err == nil {
		t.Fatal("capacity did not stop new operations")
	}
	replay, err := c.SubmitTransfer(context.Background(), first)
	if err != nil || *replay.Receipt != *sealed.Receipt {
		t.Fatal("capacity broke idempotent replay", err)
	}
	if getBalance(t, c, s.SeedAddress, "YELLOW") != before {
		t.Fatal("capacity rejection changed balance")
	}
	if history, err := c.Transfers(context.Background(), bob.Hex(), client.TransferPageOptions{}); err != nil || len(history.Items) != 1 {
		t.Fatal("history unavailable at capacity", err)
	}
}

func TestPublicSealDelayPacesNewWritesWithoutDelayingReadsOrReplay(t *testing.T) {
	s, server, c := fixture(t, Config{PublicURL: "https://sandbox.example", RequestsPerSecond: 1000, SealDelay: time.Minute})
	now := time.Now()
	s.now = func() time.Time { return now }
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	first := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1")
	pending, err := c.SubmitTransfer(context.Background(), first)
	if err != nil || pending.Status != "pending" {
		t.Fatal("delayed write was not admitted as pending", pending, err)
	}
	second := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "1")
	if _, err := c.SubmitTransfer(context.Background(), second); err == nil {
		t.Fatal("seal delay did not pace new public writes")
	}
	if status, err := c.Operation(context.Background(), first.OperationID); err != nil || status.Status != "pending" {
		t.Fatal("reads were blocked by write pacing", err)
	}
	if replay, err := c.SubmitTransfer(context.Background(), first); err != nil || replay.Status != "pending" {
		t.Fatal("pending replay was rate limited", err)
	}
	s.mu.Lock()
	now = now.Add(time.Minute)
	s.mu.Unlock()
	if status, err := c.Operation(context.Background(), first.OperationID); err != nil || status.Status != "sealed" {
		t.Fatal("configured seal did not execute", err)
	}
	if replay, err := c.SubmitTransfer(context.Background(), first); err != nil || replay.Status != "sealed" {
		t.Fatal("sealed replay was rate limited", err)
	}
	// The browser can obtain the actual retry interval from the response header.
	request := struct {
		To        string                 `json:"to"`
		Assets    []client.TransferAsset `json:"assets"`
		MaxFee    string                 `json:"maxFee"`
		Nonce     uint64                 `json:"nonce"`
		Signature string                 `json:"signature"`
	}{second.Intent.To, second.Intent.Assets, second.Intent.MaxFee, second.Intent.Nonce, second.Signature}
	raw, _ := json.Marshal(request)
	response, err := http.Post(server.URL+"/v1/transfers", "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 429 || response.Header.Get("Retry-After") != "60" {
		t.Fatal("write retry interval missing", response.StatusCode, response.Header)
	}
}

type pausedBody struct{ started, release chan struct{} }

func (b *pausedBody) Read([]byte) (int, error) { close(b.started); <-b.release; return 0, io.EOF }
func (b *pausedBody) Close() error             { return nil }

func TestSlowUploadDoesNotLockLedgerAndQueueIsBounded(t *testing.T) {
	s, _, _ := fixture(t, Config{})
	body := &pausedBody{make(chan struct{}), make(chan struct{})}
	defer close(body.release)
	r := httptest.NewRequest("POST", "http://localhost/rpc", nil)
	r.Body = body
	done := make(chan struct{})
	go func() { defer close(done); s.ServeHTTP(httptest.NewRecorder(), r) }()
	t.Cleanup(func() { <-done })
	<-body.started
	finished := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/v1/gateway/info", nil))
		finished <- w.Code
	}()
	select {
	case status := <-finished:
		if status != 200 {
			t.Fatal(status)
		}
	case <-time.After(time.Second):
		t.Fatal("slow upload locked the ledger")
	}
	for len(s.inflight) < cap(s.inflight) {
		s.inflight <- struct{}{}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/v1/pools", nil))
	if w.Code != 429 {
		t.Fatal("unbounded in-flight queue", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/healthz", nil))
	if w.Code != 200 {
		t.Fatal("health blocked by slow uploads")
	}
	for len(s.inflight) > 1 {
		<-s.inflight
	}
}

func TestInvalidPublicConfiguration(t *testing.T) {
	for _, cfg := range []Config{
		{PublicURL: "http://sandbox.example"}, {PublicURL: "https://user:pass@sandbox.example"}, {PublicURL: "https://sandbox.example/path"},
		{TrustedProxies: []string{"not-a-cidr"}}, {RequestsPerSecond: -1}, {MaxOperations: -1},
	} {
		if s, err := New(cfg); err == nil {
			s.Close()
			t.Fatal("invalid server configuration accepted")
		}
	}
}

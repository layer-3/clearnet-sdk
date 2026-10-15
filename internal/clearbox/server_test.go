package clearbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

const bobKey = "0000000000000000000000000000000000000000000000000000000000000002"

// getJSON fetches a gateway REST resource and decodes it into out.
func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("%s status %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T, cfg Config) (*Server, *httptest.Server, *client.Client) {
	t.Helper()
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(s)
	t.Cleanup(func() { httpServer.Close(); s.Close() })
	c, err := client.New(httpServer.URL, NetworkID)
	if err != nil {
		t.Fatal(err)
	}
	return s, httpServer, c
}

func testSigner(t *testing.T, key string) sign.Signer {
	t.Helper()
	k, err := crypto.HexToECDSA(key)
	if err != nil {
		t.Fatal(err)
	}
	return sign.NewKeySignerFromECDSA(k)
}

func prepared(t *testing.T, c *client.Client, key, to string, legs []client.TransferAmount, maxFee string) *client.SignedTransfer {
	t.Helper()
	signer := testSigner(t, key)
	intent, err := c.PrepareTransfer(context.Background(), signer, to, legs, maxFee)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := c.SignTransfer(context.Background(), signer, intent)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func getBalance(t *testing.T, c *client.Client, address, asset string) string {
	t.Helper()
	b, err := c.Balance(context.Background(), address)
	if err != nil {
		t.Fatal(err)
	}
	for _, leg := range b.Balances {
		if leg.Asset == asset {
			return leg.ClearnetBalance
		}
	}
	t.Fatalf("missing balance %s", asset)
	return ""
}

func TestSDKRoundTripAndBlockCodecs(t *testing.T) {
	s, _, c := fixture(t, Config{})
	ctx := context.Background()
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	seed, _ := sign.EthAddress(testSigner(t, s.SeedPrivateKey))
	if seed.Hex() != s.SeedAddress || getBalance(t, c, bob.Hex(), "YELLOW") != "0" {
		t.Fatal("only the generated seed identity may receive launch funding")
	}
	signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "20"}}, "1")
	result, err := c.SubmitTransfer(ctx, signed)
	if err != nil || result.Status != "sealed" {
		t.Fatalf("submit: %+v, %v", result, err)
	}
	if got := getBalance(t, c, bob.Hex(), "YELLOW"); got != "20000000000000000000" {
		t.Fatal(got)
	}
	if got := getBalance(t, c, bob.Hex(), "USDC"); got != "0" {
		t.Fatal(got)
	}
	// Fee = 0.020 YELLOW.
	if got := getBalance(t, c, s.SeedAddress, "YELLOW"); got != "999979980000000000000000" {
		t.Fatalf("sender amount and fee: %s", got)
	}
	replay, err := c.SubmitTransfer(ctx, signed)
	if err != nil || *replay.Receipt != *result.Receipt {
		t.Fatalf("replay changed the receipt: %+v %v", replay, err)
	}
	status, err := c.Operation(ctx, signed.OperationID)
	if err != nil || status.Status != "sealed" || *status.Receipt != *result.Receipt {
		t.Fatalf("operation: %+v %v", status, err)
	}
	history, err := c.Transfers(ctx, bob.Hex(), client.TransferPageOptions{})
	if err != nil || len(history.Items) != 1 || history.Items[0].Direction != "in" || history.Items[0].Delivery != "not_verified" {
		t.Fatalf("incoming history: %+v %v", history, err)
	}
	s.mu.Lock()
	block := s.blocks[0]
	op, err := block.Entries[0].DecodeTransferOp()
	s.mu.Unlock()
	if err != nil || op.TxID != signed.OperationID || len(op.Assets) != 1 {
		t.Fatalf("SDK block codec: %+v %v", op, err)
	}
	back := prepared(t, c, bobKey, s.SeedAddress, []client.TransferAmount{{Asset: "YELLOW", Amount: "2"}}, "0.01")
	if _, err := c.SubmitTransfer(ctx, back); err != nil {
		t.Fatal(err)
	}
	if got := getBalance(t, c, bob.Hex(), "YELLOW"); got != "17998000000000000000" {
		t.Fatalf("recipient cannot send received funds: %s", got)
	}
}

func TestPendingReservationsDeliveryAndConflicts(t *testing.T) {
	s, _, c := fixture(t, Config{SealDelay: time.Second, DeliveryDelay: time.Second})
	now := time.Unix(1_800_000_000, 0)
	s.now = func() time.Time { return now }
	tick := func(d time.Duration) { s.mu.Lock(); now = now.Add(d); s.mu.Unlock() }
	ctx := context.Background()
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "600000"}}, "1000")
	conflict := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "20"}}, "1")
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := c.SubmitTransfer(ctx, signed)
			if err != nil {
				errs <- err
			} else if result.Status != "pending" {
				errs <- fmt.Errorf("expected pending: %+v", result)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if _, err := c.SubmitTransfer(ctx, conflict); err == nil {
		t.Fatal("a different intent at the reserved nonce was accepted")
	}
	balance, _ := c.Balance(ctx, s.SeedAddress)
	if balance.Nonce != 1 || getBalance(t, c, s.SeedAddress, "YELLOW") != "399400000000000000000000" {
		t.Fatalf("duplicate submissions mutated state: %+v", balance)
	}
	overspend := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "400000"}}, "1000")
	if _, err := c.SubmitTransfer(ctx, overspend); err == nil {
		t.Fatal("pending funds were spent twice")
	}
	tick(time.Second)
	status, err := c.Operation(ctx, signed.OperationID)
	if err != nil || status.Status != "sealed" {
		t.Fatalf("seal: %+v %v", status, err)
	}
	if getBalance(t, c, bob.Hex(), "YELLOW") != "0" {
		t.Fatal("sender seal incorrectly implied recipient credit")
	}
	tick(time.Second)
	if getBalance(t, c, bob.Hex(), "YELLOW") != "600000000000000000000000" {
		t.Fatal("recipient delivery not applied")
	}
	s.mu.Lock()
	if s.account(s.SeedAddress).Clearing["YELLOW"].Sign() != 0 {
		t.Error("clearing balance was not released after delivery")
	}
	s.mu.Unlock()
}

func TestRejectionsDoNotConsumeNonceOrFunds(t *testing.T) {
	s, _, c := fixture(t, Config{})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	before := getBalance(t, c, s.SeedAddress, "YELLOW")
	cases := []struct{ key, amount, fee string }{
		{s.SeedPrivateKey, "1", "0"},
		{s.SeedPrivateKey, "999000.999001", "2000"}, // Would leave less than 10 after fees.
		{s.SeedPrivateKey, "1000001", "2000"},
		{bobKey, "1", "1"}, // Unknown/unfunded accounts cannot send.
	}
	for _, tc := range cases {
		signed := prepared(t, c, tc.key, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: tc.amount}}, tc.fee)
		if _, err := c.SubmitTransfer(context.Background(), signed); err == nil {
			t.Errorf("accepted %+v", tc)
		}
	}
	if got := getBalance(t, c, s.SeedAddress, "YELLOW"); got != before {
		t.Fatal("rejection changed funds")
	}
	balance, _ := c.Balance(context.Background(), s.SeedAddress)
	if balance.Nonce != 0 {
		t.Fatal("rejection consumed a nonce")
	}
}

func TestFeeFloorsAndHistoryRetention(t *testing.T) {
	if got := feeFor([]eip712.TransferAsset{{Asset: "ETH", Amount: integer("1")}, {Asset: "YELLOW", Amount: integer("1")}}); got.String() != "200000000000000" {
		t.Fatalf("dust floor must apply separately to each leg: %s", got)
	}
	s, _, c := fixture(t, Config{})
	now := time.Now()
	s.now = func() time.Time { return now }
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "20"}}, "1")
	if _, err := c.SubmitTransfer(context.Background(), signed); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	now = now.Add(31 * 24 * time.Hour)
	s.mu.Unlock()
	history, err := c.Transfers(context.Background(), bob.Hex(), client.TransferPageOptions{})
	if err != nil || len(history.Items) != 0 {
		t.Fatalf("30-day history retention: %+v %v", history, err)
	}
}

func TestHistoryCursorIsAccountBoundAndStable(t *testing.T) {
	s, httpServer, c := fixture(t, Config{})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	for range 3 {
		signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "0.01")
		if _, err := c.SubmitTransfer(context.Background(), signed); err != nil {
			t.Fatal(err)
		}
	}
	first, err := c.Transfers(context.Background(), s.SeedAddress, client.TransferPageOptions{Limit: 1})
	if err != nil || first.NextCursor == nil || first.Items[0].Direction != "out" || first.Items[0].Delivery != "sender_sealed" {
		t.Fatalf("page: %+v %v", first, err)
	}
	second, err := c.Transfers(context.Background(), s.SeedAddress, client.TransferPageOptions{Limit: 2, Cursor: *first.NextCursor})
	if err != nil || len(second.Items) != 2 || second.NextCursor != nil || second.Items[0].OperationID == first.Items[0].OperationID {
		t.Fatalf("pagination duplicated/skipped: %+v %v", second, err)
	}
	if _, err := c.Transfers(context.Background(), bob.Hex(), client.TransferPageOptions{Cursor: *first.NextCursor}); err == nil {
		t.Fatal("cursor leaked across accounts")
	}
	for _, query := range []string{"limit=0", "limit=101", "limit=no", "cursor=bad"} {
		resp, err := http.Get(httpServer.URL + "/v1/accounts/" + s.SeedAddress + "/transfers?" + query)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("%s status %d", query, resp.StatusCode)
		}
	}
}

func TestRESTAndRPCBoundaries(t *testing.T) {
	s, server, _ := fixture(t, Config{})
	cases := []struct {
		path, body string
		status     int
		code       int
	}{
		{"/v1/transfers", "null", 400, -32600},
		{"/v1/transfers", "{}", 400, -32600},
		{"/v1/transfers", "{} {}", 400, -32600},
		{"/v1/transfers", `{"nonce":0,"signature":"zz"}`, 400, -32001},
		{"/v1/transfers", `{"nonce":0,"signature":"0x","assets":[{"asset":"YELLOW","amount":"-1"}]}`, 400, -32602},
		{"/v1/transfers", `{"nonce":0,"signature":"0x","assets":[{"asset":"YELLOW","amount":"1"}],"to":"` + s.SeedAddress + `","maxFee":"0"}`, 401, -32001},
		{"/v1/transfers", strings.Repeat(" ", 65537), 400, -32600},
		{"/rpc", "{", 200, -32700},
		{"/rpc", "[]", 200, -32600},
		{"/rpc", "null", 200, -32600},
		{"/rpc", `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":[]}`, 200, -32601},
		{"/rpc", `{"jsonrpc":"2.0","id":1,"method":"eth_getBalance","params":["invalid","latest"]}`, 200, -32602},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%d/%s", tc.path, tc.code, tc.body[:min(len(tc.body), 24)]), func(t *testing.T) {
			resp, err := http.Post(server.URL+tc.path, "application/json", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			var body struct {
				Error rpcError `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || resp.StatusCode != tc.status || body.Error.Code != tc.code {
				t.Fatalf("response: %d %+v %v", resp.StatusCode, body, err)
			}
		})
	}
	resp, err := http.Post(server.URL+"/rpc", "application/json", strings.NewReader(`[{"jsonrpc":"2.0","id":1,"method":"eth_chainId"},{"jsonrpc":"2.0","method":"net_version"},{"jsonrpc":"2.0","id":"wallet","method":"eth_getBlockByNumber","params":["`+s.SeedAddress+`","latest",false]}]`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var batch []map[string]any
	if json.NewDecoder(resp.Body).Decode(&batch) != nil || len(batch) != 2 || batch[0]["result"] != "0x59444556" {
		t.Fatalf("batch/notification support: %+v", batch)
	}
	resp, err = http.Post(server.URL+"/rpc", "application/json", strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatal("notification received a response")
	}
}

// The gateway metadata payload must match the gateway's own projection: pool
// IDs are the canonical lowercase base asset, and unset contract addresses are
// the zero address rather than an empty string.
func TestGatewayMetadataPayloads(t *testing.T) {
	_, server, _ := fixture(t, Config{})
	var info struct {
		VirtualRouter string            `json:"virtualRouter"`
		Contracts     map[string]string `json:"contracts"`
	}
	getJSON(t, server.URL+"/v1/gateway/info", &info)
	if info.VirtualRouter != eip712.RouterHex {
		t.Fatalf("virtual router %q is not the EIP-712 verifying contract", info.VirtualRouter)
	}
	for field, want := range map[string]string{
		"custody": zeroAddress, "registry": zeroAddress, "faucet": zeroAddress,
		"token": catalog[2].Address,
	} {
		if got := info.Contracts[field]; got != want {
			t.Fatalf("contracts.%s = %q, want %q", field, got, want)
		}
	}
	var pools struct {
		Items []struct{ PoolID, TokenA, TokenB string }
	}
	getJSON(t, server.URL+"/v1/pools", &pools)
	if len(pools.Items) != 2 {
		t.Fatalf("pool fixtures: %+v", pools.Items)
	}
	for _, pool := range pools.Items {
		if pool.PoolID != strings.ToLower(pool.TokenA) || pool.TokenB != "YELLOW" {
			t.Fatalf("pool %q is not the canonical base-asset ID for %s/%s", pool.PoolID, pool.TokenA, pool.TokenB)
		}
	}
}

func TestLocalOnly(t *testing.T) {
	for _, address := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "example.com:8080", "192.168.1.1:8080"} {
		if listener, err := LoopbackListen(address); err == nil {
			listener.Close()
			t.Errorf("nonlocal bind allowed: %s", address)
		}
	}
	_, server, _ := fixture(t, Config{})
	for _, tc := range []struct {
		host, origin string
		status       int
	}{
		{"evil.example", "", 403},
		{"", "https://evil.example", 403},
		{"", "null", 403},
		{"", "http://localhost:3000", 200},
		{"", "http://127.0.0.1:3000", 200},
	} {
		req, _ := http.NewRequest("GET", server.URL+"/healthz", nil)
		if tc.host != "" {
			req.Host = tc.host
		}
		req.Header.Set("Origin", tc.origin)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%+v: %d", tc, resp.StatusCode)
		}
		if tc.status == 200 && resp.Header.Get("Access-Control-Allow-Origin") != tc.origin {
			t.Fatal("local CORS missing")
		}
	}
}

package clearbox

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// rpcCall invokes the wallet RPC surface with JSON-marshalled params, the shape
// the HTTP handler receives.
func rpcCall(t *testing.T, s *Server, method string, values ...any) (any, *rpcError) {
	t.Helper()
	params := make([]json.RawMessage, len(values))
	for i, value := range values {
		params[i], _ = json.Marshal(value)
	}
	return s.rpc(method, params)
}

func TestCanonicalUserURITransfer(t *testing.T) {
	s, _, c := fixture(t, Config{})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, s.SeedPrivateKey, core.UserURI(bob.Hex()), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1")
	result, err := c.SubmitTransfer(context.Background(), signed)
	if err != nil || result.Status != "sealed" {
		t.Fatalf("canonical user URI transfer: %+v %v", result, err)
	}
	if got := getBalance(t, c, bob.Hex(), "YELLOW"); got != "25000000000000000000" {
		t.Fatal("URI recipient was not credited", got)
	}
	replay, err := c.SubmitTransfer(context.Background(), signed)
	if err != nil || *replay.Receipt != *result.Receipt {
		t.Fatalf("URI replay changed the receipt: %+v %v", replay, err)
	}
}

func TestRPCAccountScopedBlocks(t *testing.T) {
	s, _, c := fixture(t, Config{})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	call := func(method string, values ...any) (any, *rpcError) { return rpcCall(t, s, method, values...) }
	for _, method := range []string{"eth_blockNumber", "eth_getBlockByNumber"} {
		if _, err := call(method); err == nil || err.Code != -32602 {
			t.Fatal("account is required by Clearnet block RPC", method, err)
		}
	}
	if got, err := call("eth_getBlockByNumber", s.SeedAddress, "latest", false); err != nil || got != nil {
		t.Fatal("empty account has no synthetic genesis block", got, err)
	}
	first, err := c.SubmitTransfer(context.Background(), prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.SubmitTransfer(context.Background(), prepared(t, c, bobKey, s.SeedAddress, []client.TransferAmount{{Asset: "YELLOW", Amount: "2"}}, "1")); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{s.SeedAddress, core.UserURI(bob.Hex())} {
		if got, err := call("eth_blockNumber", address); err != nil || got != "0x1" {
			t.Fatal("block count included another account", got, err)
		}
	}
	for _, tag := range []string{"earliest", "latest", "pending", "0x1"} {
		got, err := call("eth_getBlockByNumber", s.SeedAddress, tag, false)
		if err != nil {
			t.Fatal(err)
		}
		block, ok := got.(map[string]any)
		if !ok || block["hash"] != first.Receipt.BlockHash || block["number"] != "0x1" {
			t.Fatal("wrong account-scoped block", got)
		}
		if block["parentHash"] != "0x"+accountID(s.SeedAddress) {
			t.Fatal("Clearnet block parentHash must project its anchor", block)
		}
		transactions, ok := block["transactions"].([]any)
		if !ok || len(transactions) != 1 {
			t.Fatal("block omitted its real entry", block)
		}
	}
	if got, err := call("eth_getBlockByHash", first.Receipt.BlockHash, false); err != nil || got == nil {
		t.Fatal("sealed block hash cannot be resolved", got, err)
	}
	if got, err := call("eth_getBlockByNumber", s.SeedAddress, "0x2", false); err != nil || got != nil {
		t.Fatal("missing account block must return null", got, err)
	}
}

// Wallet clients probe the zero address before a call: with no explicit `from`,
// cast sends eth_getTransactionCount(0x0, "latest") and eth_getBalance(0x0) as a
// preflight. Those reads must resolve to an empty account so the caller reaches
// the intended method instead of a misleading address error.
func TestRPCZeroAddressReads(t *testing.T) {
	s, _, _ := fixture(t, Config{})
	const zero = "0x0000000000000000000000000000000000000000"
	for method, want := range map[string]string{
		"eth_getBalance":          "0x0",
		"eth_getTransactionCount": "0x0",
		"eth_getCode":             "0x",
	} {
		if got, err := rpcCall(t, s, method, zero, "latest"); err != nil || got != want {
			t.Fatalf("%s(%s) = %v, %v; want %s", method, zero, got, err, want)
		}
	}
	seedBalance := "0x" + new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).Text(16)
	if got, err := rpcCall(t, s, "eth_getBalance", s.SeedAddress, "latest"); err != nil || got != seedBalance {
		t.Fatalf("funded account balance = %v, %v; want %s", got, err, seedBalance)
	}
	// An account with no state reads as empty rather than erroring.
	if got, err := rpcCall(t, s, "eth_getBalance", "0x000000000000000000000000000000000000bEEF", "latest"); err != nil || got != "0x0" {
		t.Fatalf("unfunded account balance = %v, %v; want 0x0", got, err)
	}
	// The zero address is still not an account for block queries.
	if _, err := rpcCall(t, s, "eth_blockNumber", zero); err == nil || err.Code != -32602 || !strings.Contains(err.Message, "nonzero") {
		t.Fatalf("eth_blockNumber must reject the zero address as an account: %v", err)
	}
	if _, err := rpcCall(t, s, "eth_getBlockByNumber", zero, "latest", false); err == nil || err.Code != -32602 || !strings.Contains(err.Message, "nonzero") {
		t.Fatalf("eth_getBlockByNumber must reject the zero address as an account: %v", err)
	}
	// Historical state stays unsupported, and is reported as a tag problem.
	if _, err := rpcCall(t, s, "eth_getBalance", s.SeedAddress, "0x0"); err == nil || err.Code != -32602 || !strings.Contains(err.Message, "latest/pending") {
		t.Fatalf("historical tag must be rejected by tag: %v", err)
	}
}

// The preflight must not mask the real limitation: a cast-style batch (a
// zero-address nonce probe followed by eth_call) answers the probe and then
// reports eth_call itself as unsupported.
func TestRPCWalletPreflightOverHTTP(t *testing.T) {
	_, server, _ := fixture(t, Config{})
	body := `[{"jsonrpc":"2.0","id":1,"method":"eth_getTransactionCount","params":["0x0000000000000000000000000000000000000000","latest"]},` +
		`{"jsonrpc":"2.0","id":2,"method":"eth_call","params":[{"to":"` + eip712.RouterHex + `","data":"0x57e871e7"},"latest"]}]`
	resp, err := http.Post(server.URL+"/rpc", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var results []struct {
		ID     any       `json:"id"`
		Result any       `json:"result"`
		Error  *rpcError `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("batch returned %d results", len(results))
	}
	if results[0].Error != nil || results[0].Result != "0x0" {
		t.Fatalf("zero-address nonce probe failed: %+v", results[0])
	}
	if results[1].Error == nil || results[1].Error.Code != -32601 {
		t.Fatalf("eth_call must report unsupported, got %+v", results[1])
	}
}

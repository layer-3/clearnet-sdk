package clearbox

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

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
	call := func(method string, values ...any) (any, *rpcError) {
		t.Helper()
		params := make([]json.RawMessage, len(values))
		for i, value := range values {
			params[i], _ = json.Marshal(value)
		}
		return s.rpc(method, params)
	}
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

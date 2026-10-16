package yellow

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/internal/clearbox"
	"github.com/layer-3/clearnet-sdk/pkg/client"
)

// Exercise the actual CLI command handlers, encrypted accounts, trust checks,
// signing and operation journal against the independent public Go simulator.
func TestCLIWithClearbox(t *testing.T) {
	sim, err := clearbox.New(clearbox.Config{SealDelay: 10 * time.Millisecond, PublicURL: "https://sandbox.example", RequestsPerSecond: 1000})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(sim)
	defer sim.Close()
	defer server.Close()
	dir := t.TempDir()
	run := func(prompt prompter, args ...string) string {
		t.Helper()
		a, out, diag := newTestApp(dir, prompt)
		if code := a.execute(context.Background(), append([]string{"--json"}, args...)); code != 0 {
			t.Fatalf("%v: exit %d, %s %s", args, code, out, diag)
		}
		assertOneJSON(t, out.String())
		return out.String()
	}
	prompt := &scriptedPrompt{approve: true}
	run(prompt, "network", "add", "local", "--gateway", server.URL, "--network-id", "YDEV")
	// Generated LOCAL TEST KEY; never use this account on a real network.
	run(&scriptedPrompt{approve: true, lines: []string{"key"}, secrets: []string{sim.SeedPrivateKey, testPass, testPass}}, "account", "import", "seed")
	run(prompt, "account", "create", "recipient", "--agent")
	recipient, err := (store{dir}).account("recipient")
	if err != nil {
		t.Fatal(err)
	}
	first := run(prompt, "--account", "seed", "transfer", "send", recipient.Address, "--asset", "YELLOW=25", "--max-fee", "1", "--wait", "5s", "--idempotency-key", "local-first-transfer")
	if !strings.Contains(first, `"state":"sealed"`) {
		t.Fatal("CLI did not reconcile pending operation", first)
	}
	// Same client idempotency key must resolve the saved operation, not sign again.
	run(prompt, "--account", "seed", "transfer", "send", recipient.Address, "--asset", "YELLOW=25", "--max-fee", "1", "--wait", "5s", "--idempotency-key", "local-first-transfer")
	run(prompt, "--account", "recipient", "transfer", "send", sim.SeedAddress, "--asset", "YELLOW=2", "--max-fee", "0.1", "--wait", "5s")
	if history := run(prompt, "--account", "recipient", "transfer", "history"); !strings.Contains(history, `"direction":"in"`) || !strings.Contains(history, `"direction":"out"`) {
		t.Fatal("CLI history did not show both directions", history)
	}
	c, _ := client.New(server.URL, "YDEV")
	seed, _ := c.Balance(context.Background(), sim.SeedAddress)
	balance, _ := c.Balance(context.Background(), recipient.Address)
	if seed.Nonce != 1 || balance.Nonce != 1 {
		t.Fatalf("idempotent CLI replay consumed a nonce: %+v %+v", seed, balance)
	}
	if raw := run(prompt, "--account", "recipient", "balance"); !strings.Contains(raw, "22998000000000000000") {
		t.Fatal("CLI balance did not show received and spent funds", raw)
	}
	found := false
	for _, leg := range balance.Balances {
		if leg.Asset == "YELLOW" && leg.ClearnetBalance == "22998000000000000000" {
			found = true
		}
	}
	if !found {
		t.Fatalf("CLI transfer did not affect live recipient balance: %+v", balance)
	}
}

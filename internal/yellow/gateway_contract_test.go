package yellow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
)

// The companion gateway contract harness supplies real HTTP handlers and a
// local operator control endpoint; no production service or user wallet is used.
func TestProductionGatewayOperatorRecovery(t *testing.T) {
	url := os.Getenv("YELLOW_GATEWAY_CONTRACT_URL")
	operator := os.Getenv("YELLOW_GATEWAY_OPERATOR_URL")
	if url == "" || operator == "" {
		t.Skip("run scripts/test-gateway-contract.sh with the companion gateway checkout")
	}
	dir := t.TempDir()
	api, err := client.New(url, "YDEV")
	if err != nil {
		t.Fatal(err)
	}
	info, err := api.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := client.NetworkFingerprint(*info)
	if err != nil {
		t.Fatal(err)
	}
	run := func(want int, args ...string) map[string]any {
		t.Helper()
		code, result, diag := runUnattended(t, dir, args...)
		if code != want {
			t.Fatalf("%v: exit %d, want %d: %v %s", args, code, want, result, diag)
		}
		if value, ok := result["result"].(map[string]any); ok {
			return value
		}
		return result
	}
	run(0, "network", "add", "contract", "--gateway", url, "--network-id", "YDEV", "--fingerprint", fingerprint)
	account := run(0, "account", "create", "bot", "--agent")
	// The operator harness funds only this generated test account.
	resp, err := http.Post(operator+"/fund?address="+account["address"].(string), "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("test funding failed", resp.StatusCode)
	}
	send := []string{"--account", "bot", "transfer", "send", "0x00000000000000000000000000000000000000bb", "--asset", "YELLOW=0.000000000000001", "--max-fee", "0.0000000000000001"}
	initial := run(ExitUnknown, send...)
	id := initial["operationId"].(string)
	ac, err := (store{dir}).account("bot")
	if err != nil {
		t.Fatal(err)
	}
	retained, _ := json.Marshal(ac.Operations[0].Payload)
	run(ExitUnknown, send...)
	run(ExitUnknown, "operation", "retry", id)
	// Repair connectivity and let the original routing window finish.
	resp, err = http.Post(operator+"/repair?operation="+id, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("operator repair failed", resp.StatusCode)
	}
	status, err := api.Operation(context.Background(), id)
	if err != nil || status.PendingUntil <= 0 {
		t.Fatalf("missing admission evidence: %+v %v", status, err)
	}
	if delay := time.Until(time.Unix(status.PendingUntil+1, 0)); delay > 0 {
		if delay > 5*time.Second {
			t.Fatal("test window was not shortened")
		}
		time.Sleep(delay)
	}
	if result := run(0, "operation", "retry", id); result["state"] != "sealed" {
		t.Fatal(result)
	}
	if result := run(0, "operation", "show", id); result["state"] != "sealed" {
		t.Fatal(result)
	}
	ac, err = (store{dir}).account("bot")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(ac.Operations[0].Payload)
	if !bytes.Equal(retained, after) {
		t.Fatal("operator recovery changed the retained authorization")
	}
	if result := run(0, send...); result["state"] != "sealed" {
		t.Fatal("new sends stayed blocked", result)
	}
	ac, err = (store{dir}).account("bot")
	if err != nil || len(ac.Operations) != 2 || ac.Operations[0].ID != id {
		t.Fatal("journal was not retained", err)
	}
}

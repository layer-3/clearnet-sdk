package yellow

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
)

// runUnattended executes like an agent's tool call: stdin is not a terminal.
func runUnattended(t *testing.T, dir string, args ...string) (int, map[string]any, string) {
	t.Helper()
	in, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, diag := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute(context.Background(), append([]string{"--json"}, args...), in, out, diag, dir, "test")
	assertOneJSON(t, out.String())
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return code, envelope, diag.String()
}

// setupAgent stores the fixture key as an agent account named "bot".
func setupAgent(t *testing.T, g *gatewayFixture) (string, []byte) {
	t.Helper()
	dir, _ := setup(t, g)
	s := store{dir}
	cfg, _ := s.config()
	cfg.Account = ""
	if err := s.saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "accounts", "alice.json")); err != nil {
		t.Fatal(err)
	}
	pass, err := newAgentPass()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.saveAgentPass("bot", pass); err != nil {
		t.Fatal(err)
	}
	address := crypto.PubkeyToAddress(mustTestPrivate(t).PublicKey).Hex()
	env, key, err := encryptSecret(pass, address, secret{"key", testKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.saveAccount("bot", &accountFile{Version: 1, Address: address, Agent: true, Secret: env, Operations: []operation{}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clear(key) })
	return dir, key
}

func agentSend(extra ...string) []string {
	return append([]string{"--account", "bot", "transfer", "send", recipient, "--asset", "YELLOW=1.25", "--max-fee", "0"}, extra...)
}

func TestAgentAccountCreateIsUnattended(t *testing.T) {
	dir := t.TempDir()
	code, env, diag := runUnattended(t, dir, "account", "create", "bot", "--agent")
	if code != 0 {
		t.Fatal(code, env, diag)
	}
	result := env["result"].(map[string]any)
	if result["agent"] != true || !strings.HasPrefix(result["address"].(string), "0x") {
		t.Fatal("bad result", result)
	}
	for _, path := range []string{filepath.Join(dir, "accounts", "bot.json"), filepath.Join(dir, "accounts", "bot.unlock")} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
			t.Fatal("permissions too broad", path, st.Mode())
		}
	}
	cfg, _ := (store{dir}).config()
	if cfg.Account != "" {
		t.Fatal("agent account became the default")
	}
	code, env, _ = runUnattended(t, dir, "account", "list")
	list := env["result"].([]any)
	if code != 0 || len(list) != 1 || list[0].(map[string]any)["agent"] != true {
		t.Fatal("agent flag missing from list", env)
	}
	// Personal accounts still need a person.
	if code, _, _ := runUnattended(t, dir, "account", "create", "personal"); code != 2 {
		t.Fatal("personal account created without a terminal", code)
	}
}

func TestAgentAccountSendsAndRecoversUnattended(t *testing.T) {
	g := newGateway(t)
	g.postCode = 504
	dir, _ := setupAgent(t, g)
	code, env, diag := runUnattended(t, dir, agentSend()...)
	if code != 4 || len(g.posted) != 1 {
		t.Fatal("agent send", code, env, diag)
	}
	if !strings.Contains(diag, "signing without a prompt") || !strings.Contains(diag, "Review transfer") {
		t.Fatal("agent signing not reported on stderr", diag)
	}
	id := env["result"].(map[string]any)["operationId"].(string)
	// The unresolved guard still applies to agents.
	if code, _, _ := runUnattended(t, dir, agentSend()...); code != 4 || len(g.posted) != 1 {
		t.Fatal("agent escaped the unresolved guard", code)
	}
	g.status = client.OperationStatus{Status: "unknown", PendingUntil: time.Now().Unix()}
	g.postCode = 200
	code, env, diag = runUnattended(t, dir, "operation", "retry", id)
	if code != 0 || len(g.posted) != 2 || !bytes.Equal(g.posted[0], g.posted[1]) {
		t.Fatal("agent replay", code, env, diag)
	}
}

func TestAgentBackupRestoresAsPersonal(t *testing.T) {
	g := newGateway(t)
	g.postCode = 504
	dir, key := setupAgent(t, g)
	if code, _, _ := runUnattended(t, dir, agentSend()...); code != 4 {
		t.Fatal("send")
	}
	backup := filepath.Join(t.TempDir(), "bot.backup.json")
	if code, _, _ := runUnattended(t, dir, "account", "backup", "bot", "--out", backup); code != 2 {
		t.Fatal("agent backup ran without a person choosing its passphrase")
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true, secrets: []string{"backup passphrase", "backup passphrase"}})
	if code := a.execute(context.Background(), []string{"--json", "account", "backup", "bot", "--out", backup}); code != 0 {
		t.Fatal(code, out)
	}
	raw, _ := os.ReadFile(backup)
	var restored accountFile
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Agent || len(restored.Operations) != 1 {
		t.Fatal("backup kept the agent flag or lost the journal")
	}
	sec, newKey, err := decryptSecret([]byte("backup passphrase"), &restored)
	if err != nil || sec.Value != testKey {
		t.Fatal("backup secret", err)
	}
	defer clear(newKey)
	ac, _ := (store{dir}).account("bot")
	original, err := decodeOperation(key, ac.Operations[0])
	if err != nil {
		t.Fatal(err)
	}
	copied, err := decodeOperation(newKey, restored.Operations[0])
	if err != nil {
		t.Fatal(err)
	}
	if original.Signature != copied.Signature {
		t.Fatal("backup changed the authorization")
	}
}

func TestDryRunNeverUnlocksOrSends(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	// alice is a personal account; a dry run still needs no terminal.
	code, env, diag := runUnattended(t, dir, "transfer", "send", recipient, "--asset", "YELLOW=1.25", "--max-fee", "0.5", "--dry-run")
	if code != 0 || len(g.posted) != 0 {
		t.Fatal(code, env, diag)
	}
	result := env["result"].(map[string]any)
	assets := result["assets"].([]any)
	fee := result["maxFee"].(map[string]any)
	if result["dryRun"] != true || result["nonce"] != float64(7) || assets[0].(map[string]any)["baseUnits"] != "1250000000000000000" || fee["baseUnits"] != "500000000000000000" || fee["amount"] != "0.5" {
		t.Fatal("bad preview", result)
	}
	ac, _ := (store{dir}).account("alice")
	if len(ac.Operations) != 0 {
		t.Fatal("dry run saved an operation")
	}
}

func TestIdempotencyKeyNeverSendsTwice(t *testing.T) {
	g := newGateway(t)
	dir, _ := setupAgent(t, g)
	code, env, _ := runUnattended(t, dir, agentSend("--idempotency-key", "invoice-42")...)
	if code != 0 || len(g.posted) != 1 {
		t.Fatal("first send", code, env)
	}
	first := env["result"].(map[string]any)
	if first["idempotencyKey"] != "invoice-42" {
		t.Fatal("key not reported", first)
	}
	code, env, diag := runUnattended(t, dir, agentSend("--idempotency-key", "invoice-42", "--wait", "5s")...)
	if code != 0 || len(g.posted) != 1 || env["result"].(map[string]any)["operationId"] != first["operationId"] {
		t.Fatal("repeated key sent again", code, env, diag)
	}
	if !strings.Contains(diag, "nothing new was signed or sent") {
		t.Fatal("repeat not explained", diag)
	}
	code, env, _ = runUnattended(t, dir, "--account", "bot", "transfer", "send", recipient, "--asset", "YELLOW=2", "--max-fee", "0", "--idempotency-key", "invoice-42")
	if code != 2 || len(g.posted) != 1 || !strings.Contains(env["error"].(string), "different transfer") {
		t.Fatal("key reused for another transfer", code, env)
	}
	// Equal amounts written differently are the same request.
	if code, _, _ := runUnattended(t, dir, "--account", "bot", "transfer", "send", "yellow://ynet/user/"+recipient, "--asset", "YELLOW=1.250", "--max-fee", "0.0", "--idempotency-key", "invoice-42"); code != 0 || len(g.posted) != 1 {
		t.Fatal("equivalent request not matched", code)
	}
	if code, _, _ := runUnattended(t, dir, agentSend("--idempotency-key", "bad key")...); code != 2 {
		t.Fatal("invalid key accepted", code)
	}
}

func TestNetworkFingerprintPinning(t *testing.T) {
	g := newGateway(t)
	_, fp, err := reviewed(&g.info)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	add := []string{"network", "add", "dev", "--gateway", g.server.URL, "--network-id", "YDEV"}
	code, env, _ := runUnattended(t, dir, add...)
	if code != 2 || !strings.Contains(env["error"].(string), "--fingerprint "+fp) {
		t.Fatal("unpinned add without a terminal", code, env)
	}
	if code, _, _ := runUnattended(t, dir, append(add, "--fingerprint", strings.Repeat("0", 64))...); code != 6 {
		t.Fatal("wrong fingerprint accepted", code)
	}
	cfg, _ := (store{dir}).config()
	if len(cfg.Networks) != 0 {
		t.Fatal("rejected network was saved")
	}
	if code, env, _ := runUnattended(t, dir, append(add, "--fingerprint", "0x"+strings.ToUpper(fp))...); code != 0 {
		t.Fatal("pinned add", code, env)
	}
	g.info.Assets[0].Decimals = 9
	_, next, _ := reviewed(&g.info)
	if code, _, _ := runUnattended(t, dir, "network", "trust-update", "dev", "--fingerprint", fp); code != 6 {
		t.Fatal("stale pin accepted a changed catalogue", code)
	}
	if code, env, _ := runUnattended(t, dir, "network", "trust-update", "dev", "--fingerprint", next); code != 0 {
		t.Fatal("pinned trust update", code, env)
	}
	cfg, _ = (store{dir}).config()
	if cfg.Networks["dev"].Fingerprint != next {
		t.Fatal("trust update not saved")
	}
}

func TestSchemaDescribesCommands(t *testing.T) {
	code, env, _ := runUnattended(t, t.TempDir(), "schema")
	if code != 0 {
		t.Fatal(code, env)
	}
	result := env["result"].(map[string]any)
	var send map[string]any
	for _, c := range result["commands"].([]any) {
		if c.(map[string]any)["command"] == "yellow transfer send" {
			send = c.(map[string]any)
		}
	}
	if send == nil || !strings.Contains(send["terminal"].(string), "agent accounts") {
		t.Fatal("transfer send missing", result)
	}
	flags := map[string]map[string]any{}
	for _, f := range send["flags"].([]any) {
		flags[f.(map[string]any)["name"].(string)] = f.(map[string]any)
	}
	if flags["--asset"]["required"] != true || flags["--asset"]["repeatable"] != true || flags["--dry-run"] == nil || flags["--idempotency-key"] == nil {
		t.Fatal("send flags", flags)
	}
	if result["exitCodes"].(map[string]any)["4"] == nil {
		t.Fatal("exit codes missing")
	}
}

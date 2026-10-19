package yellow

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

const testPass = "a strong test passphrase"
const recipient = "0x00000000000000000000000000000000000000bb"

var testKey = strings.Repeat("0", 63) + "1"

type scriptedPrompt struct {
	secrets, lines  []string
	approve         bool
	words           string
	exportedKey     string
	exportedName    string
	exportedAddress string
	exportedBytes   []byte
}

func (p *scriptedPrompt) secret(string) ([]byte, error) {
	if len(p.secrets) == 0 {
		return []byte(testPass), nil
	}
	v := p.secrets[0]
	p.secrets = p.secrets[1:]
	return []byte(v), nil
}
func (p *scriptedPrompt) line(string) (string, error) {
	if len(p.lines) == 0 {
		return "", fmt.Errorf("missing test response")
	}
	v := p.lines[0]
	p.lines = p.lines[1:]
	return v, nil
}
func (p *scriptedPrompt) confirm(string) error {
	if !p.approve {
		return fail(ExitCancelled, "cancelled")
	}
	return nil
}
func (p *scriptedPrompt) recovery(words string) error { p.words = words; return p.confirm("") }
func (p *scriptedPrompt) privateKey(name, address string, key []byte) error {
	if err := p.confirm(""); err != nil {
		return err
	}
	p.exportedName, p.exportedAddress, p.exportedKey = name, address, string(key)
	p.exportedBytes = key
	return nil
}
func newTestApp(dir string, prompt prompter) (*app, *bytes.Buffer, *bytes.Buffer) {
	out, diag := new(bytes.Buffer), new(bytes.Buffer)
	a := &app{store: store{dir}, out: out, diag: diag, prompt: prompt, version: "test"}
	return a, out, diag
}

type gatewayFixture struct {
	mu           sync.Mutex
	info         client.NetworkInfo
	server       *httptest.Server
	postCode     int
	status       client.OperationStatus
	posted       [][]byte
	infoHook     func(int)
	postHook     func()
	statusHook   func()
	infoCalls    int
	delay        time.Duration
	historyCode  int
	historyPages map[string]client.TransferPage
	historyCalls int
	nonce        uint64
	cursor       string
}

func newGateway(t *testing.T) *gatewayFixture {
	t.Helper()
	g := &gatewayFixture{nonce: 7, postCode: 200, info: client.NetworkInfo{NetworkID: "YDEV", ChainID: eip712.NetworkChainID("YDEV").Uint64(), VirtualRouter: eip712.RouterHex, Capabilities: []string{client.CapabilityTransferStatus, client.CapabilityRecentTransfers}, Assets: []client.AssetInfo{{Symbol: "YELLOW", Decimals: 18}, {Symbol: "ETH", Decimals: 6}}}}
	g.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		switch {
		case r.URL.Path == "/v1/gateway/info":
			g.infoCalls++
			if g.infoHook != nil {
				g.infoHook(g.infoCalls)
			}
			_ = json.NewEncoder(w).Encode(g.info)
		case strings.HasSuffix(r.URL.Path, "/balances"):
			fmt.Fprintf(w, `{"address":%q,"nonce":%d,"balances":[{"symbol":"YELLOW","clearnetBalance":"1250000000000000000"}]}`, crypto.PubkeyToAddress(mustTestPrivate(t).PublicKey).Hex(), g.nonce)
		case r.Method == "POST" && r.URL.Path == "/v1/transfers":
			raw, _ := io.ReadAll(r.Body)
			g.posted = append(g.posted, raw)
			if g.postHook != nil {
				g.postHook()
			}
			if g.delay > 0 {
				time.Sleep(g.delay)
			}
			var body struct {
				To     string
				Assets []client.TransferAsset
				MaxFee string
				Nonce  uint64
			}
			_ = json.Unmarshal(raw, &body)
			// Obtain the operation ID from the already committed journal in postHook,
			// or compute it through the public SDK's domain helper for fixture responses.
			assets := []eip712.TransferAsset{}
			for _, v := range body.Assets {
				amount, _ := client.AmountInBaseUnits(v.Amount, 0)
				assets = append(assets, eip712.TransferAsset{Asset: v.Asset, Amount: amount})
			}
			fee, _ := client.AmountInBaseUnits(body.MaxFee, 0)
			id, _ := eip712.TransferOperationID(eip712.NetworkChainID("YDEV"), crypto.PubkeyToAddress(mustTestPrivate(t).PublicKey), body.To, assets, fee, body.Nonce)
			if g.postCode == 307 {
				w.Header().Set("Location", g.server.URL+"/v1/transfers")
			}
			w.WriteHeader(g.postCode)
			switch g.postCode {
			case 200:
				_ = json.NewEncoder(w).Encode(client.Receipt{OperationID: id, BlockHash: "0x" + strings.Repeat("ab", 32), StateRoot: "0x" + strings.Repeat("cd", 32), EntryIndex: 4, SealedAt: 42})
			case 202:
				_ = json.NewEncoder(w).Encode(client.OperationStatus{OperationID: id, Kind: "transfer", Status: "pending"})
			case 400:
				fmt.Fprint(w, `{"error":{"code":-32602,"message":"invalid transfer"}}`)
			case 401:
				fmt.Fprint(w, `not JSON`)
			case 307:
				w.Header().Set("Location", g.server.URL+"/v1/transfers")
			default:
				fmt.Fprint(w, `{"error":{"code":-32006,"message":"outcome unknown"}}`)
			}
		case strings.HasPrefix(r.URL.Path, "/v1/operations/"):
			if g.statusHook != nil {
				g.statusHook()
			}
			status := g.status
			status.OperationID = strings.TrimPrefix(r.URL.Path, "/v1/operations/")
			status.Kind = "transfer"
			if status.Status == "" {
				status.Status = "unknown"
			}
			if status.Receipt != nil {
				copy := *status.Receipt
				copy.OperationID = status.OperationID
				status.Receipt = &copy
			}
			_ = json.NewEncoder(w).Encode(status)
		case strings.HasSuffix(r.URL.Path, "/transfers"):
			g.historyCalls++
			if g.historyCode != 0 {
				w.WriteHeader(g.historyCode)
				fmt.Fprint(w, `{"error":{"code":-32001,"message":"backfilling"}}`)
				return
			}
			g.cursor = r.URL.Query().Get("cursor")
			if g.historyPages != nil {
				_ = json.NewEncoder(w).Encode(g.historyPages[g.cursor])
				return
			}
			next := "next-page"
			_ = json.NewEncoder(w).Encode(client.TransferPage{RetentionDays: 30, NextCursor: &next, Items: []client.TransferActivity{{Direction: "incoming", From: recipient, To: recipient, Assets: []client.TransferAsset{{Asset: "YELLOW", Amount: "100"}}, BlockHash: "block"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(g.server.Close)
	return g
}
func mustTestPrivate(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func setup(t *testing.T, g *gatewayFixture) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	s := store{dir}
	if err := s.init(); err != nil {
		t.Fatal(err)
	}
	id, fp, err := reviewed(&g.info)
	if err != nil {
		t.Fatal(err)
	}
	p := profile{Name: "dev", Gateway: g.server.URL, NetworkID: "YDEV", Identity: id, Fingerprint: fp}
	if err := s.saveConfig(config{Version: 1, Network: "dev", Account: "alice", Networks: map[string]profile{"dev": p}}); err != nil {
		t.Fatal(err)
	}
	address := crypto.PubkeyToAddress(mustTestPrivate(t).PublicKey).Hex()
	env, key, err := encryptSecret([]byte(testPass), address, secret{"key", testKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.saveAccount("alice", &accountFile{Version: 1, Address: address, Secret: env, Operations: []operation{}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clear(key) })
	return dir, key
}
func sendArgs(extra ...string) []string {
	return append([]string{"--json", "transfer", "send", recipient, "--asset", "YELLOW=1.25", "--max-fee", "0"}, extra...)
}
func assertOneJSON(t *testing.T, out string) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var v map[string]any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("invalid JSON: %s (%v)", out, err)
	}
	if v["version"] != float64(1) {
		t.Fatal("missing JSON version")
	}
	if err := dec.Decode(&v); err != io.EOF {
		t.Fatal("more than one JSON object")
	}
}
func TestTransferOutcomesPersistBeforePOST(t *testing.T) {
	for _, tt := range []struct {
		status, exit int
		state        string
	}{{200, 0, "sealed"}, {202, 3, "pending"}, {504, 4, "unresolved"}, {400, 5, "rejected"}, {401, 4, "unresolved"}, {307, 4, "unresolved"}} {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			g := newGateway(t)
			g.postCode = tt.status
			dir, key := setup(t, g)
			g.postHook = func() {
				ac, err := (store{dir}).account("alice")
				if err != nil {
					t.Error(err)
					return
				}
				if len(ac.Operations) != 1 || ac.Operations[0].State != "unresolved" || ac.Operations[0].Submitted == nil {
					t.Error("POST preceded durable reservation")
				}
				if _, err := decodeOperation(key, ac.Operations[0]); err != nil {
					t.Error(err)
				}
			}
			a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			if code := a.execute(context.Background(), sendArgs()); code != tt.exit {
				t.Fatalf("exit %d want %d: %s", code, tt.exit, out)
			}
			assertOneJSON(t, out.String())
			ac, err := a.store.account("alice")
			if err != nil {
				t.Fatal(err)
			}
			if ac.Operations[0].State != tt.state {
				t.Fatalf("state %s", ac.Operations[0].State)
			}
			if len(g.posted) != 1 {
				t.Fatalf("POST count %d", len(g.posted))
			}
			signed, err := decodeOperation(key, ac.Operations[0])
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), signed.Signature) || strings.Contains(out.String(), testKey) || strings.Contains(out.String(), testPass) {
				t.Fatal("secret leaked in JSON")
			}
			disk, _ := os.ReadFile(filepath.Join(dir, "accounts", "alice.json"))
			if strings.Contains(string(disk), signed.Signature) || strings.Contains(string(disk), testKey) {
				t.Fatal("authorization or key stored in plaintext")
			}
		})
	}
}
func TestRestartAndIdenticalReplay(t *testing.T) {
	g := newGateway(t)
	g.postCode = 504
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != 4 {
		t.Fatalf("exit %d %s", code, out)
	}
	ac, _ := a.store.account("alice")
	id := ac.Operations[0].ID
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != 4 {
		t.Fatalf("new send not blocked: %d %s", code, out)
	}
	if len(g.posted) != 1 {
		t.Fatal("fresh nonce send escaped unresolved guard")
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "operation", "retry", id}); code != 4 {
		t.Fatalf("unknown replay not blocked: %d %s", code, out)
	}
	if len(g.posted) != 1 {
		t.Fatal("unknown status treated as safe")
	}
	g.status = client.OperationStatus{Status: "unknown", PendingUntil: time.Now().Unix()}
	g.postCode = 200
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "operation", "retry", id}); code != 0 {
		t.Fatalf("replay exit %d %s", code, out)
	}
	if len(g.posted) != 2 || !bytes.Equal(g.posted[0], g.posted[1]) {
		t.Fatal("retry changed submitted bytes")
	}
	ac, _ = a.store.account("alice")
	if ac.Operations[0].State != "sealed" {
		t.Fatal("seal not persisted")
	}
}
func TestTimeoutStaysUnknown(t *testing.T) {
	g := newGateway(t)
	g.delay = 100 * time.Millisecond
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs("--timeout", "20ms")); code != 4 {
		t.Fatalf("exit %d %s", code, out)
	}
	ac, _ := a.store.account("alice")
	if ac.Operations[0].State != "unresolved" {
		t.Fatal("timeout did not preserve uncertainty")
	}
}
func TestTrustAndIdentityChanges(t *testing.T) {
	for _, mutation := range []string{"decimals", "identity", "during-sign"} {
		t.Run(mutation, func(t *testing.T) {
			g := newGateway(t)
			dir, _ := setup(t, g)
			switch mutation {
			case "decimals":
				g.info.Assets[0].Decimals = 9
			case "identity":
				g.info.NetworkID = "YTES"
			case "during-sign":
				g.infoHook = func(n int) {
					if n == 4 {
						g.info.Assets[0].Symbol = "NEW"
					}
				}
			}
			a, out, diag := newTestApp(dir, &scriptedPrompt{approve: true})
			if code := a.execute(context.Background(), sendArgs()); code != 6 {
				t.Fatalf("exit %d %s %s", code, out, diag)
			}
			if len(g.posted) != 0 {
				t.Fatal("untrusted transfer submitted")
			}
		})
	}
}
func TestReadOutputAndPagination(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"balance"}); code != 0 {
		t.Fatal(code, out)
	}
	if !strings.Contains(out.String(), "1.25") || strings.Contains(out.String(), "L1") {
		t.Fatal("bad balance presentation", out)
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"--json", "transfer", "history", "--cursor", "page-one"}); code != 0 {
		t.Fatal(code, out)
	}
	assertOneJSON(t, out.String())
	if g.cursor != "page-one" || !strings.Contains(out.String(), "recipient credit not verified") || !strings.Contains(out.String(), "retentionDays") {
		t.Fatal("missing history semantics")
	}
	g.historyCode = 503
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"--json", "transfer", "history"}); code != 6 {
		t.Fatal("503 treated as no transfers", code, out)
	}
}
func TestInputAndCancellationNeverSubmit(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	for _, args := range [][]string{
		{"transfer", "send", "yellow://ynet/pool/eth", "--asset", "YELLOW=1", "--max-fee", "0"},
		{"transfer", "send", recipient, "--asset", "YELLOW=1.0000000000000000001", "--max-fee", "0"},
		{"transfer", "send", recipient, "--asset", "YELLOW=1", "--asset", "YELLOW=2", "--max-fee", "0"},
		{"transfer", "send", recipient, "--asset", "YELLOW=0", "--max-fee", "0"},
		{"transfer", "send", recipient, "--asset", "NOPE=1", "--max-fee", "0"},
		{"transfer", "send", recipient, "--asset", "YELLOW=1"},
	} {
		a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
		if code := a.execute(context.Background(), append([]string{"--json"}, args...)); code != 2 {
			t.Fatalf("invalid input exit %d: %s", code, out)
		}
	}
	a, _, _ := newTestApp(dir, &scriptedPrompt{approve: false})
	if code := a.execute(context.Background(), sendArgs()); code != 130 {
		t.Fatal("cancel exit", code)
	}
	if len(g.posted) != 0 {
		t.Fatal("invalid or cancelled transfer submitted")
	}
}
func TestUserRecipientsAndExactDecimals(t *testing.T) {
	for _, raw := range []string{recipient, "yellow://ynet/user/" + recipient} {
		if _, err := userAddress(raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"0x" + strings.Repeat("0", 40), "yellow://ynet/pool/eth", "yellow://ynet/treasury/fees", "yellow://ynet/user/" + recipient + "/tag/abc", "yellow://YNET/user/" + recipient} {
		if _, err := userAddress(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"-1", "+1", "1e9", "1.", ".1", "1.001", strings.Repeat("9", 79)} {
		if _, err := client.AmountInBaseUnits(raw, 2); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if got := displayUnits("100000000000000001", 18); got != "0.100000000000000001" {
		t.Fatal(got)
	}
}
func TestJournalCommitFailurePreventsPOST(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	g.infoHook = func(n int) {
		if n == 4 {
			if err := os.Rename(filepath.Join(dir, "accounts"), filepath.Join(dir, "saved")); err != nil {
				t.Error(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "accounts"), []byte("blocked"), 0600); err != nil {
				t.Error(err)
			}
		}
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != 2 {
		t.Fatal(code, out)
	}
	if len(g.posted) != 0 {
		t.Fatal("POST despite failed durable journal")
	}
}
func TestOutcomeCommitFailureRetainsSubmissionMarker(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.postHook = func() {
		if err := os.Rename(filepath.Join(dir, "accounts"), filepath.Join(dir, "saved")); err != nil {
			t.Error(err)
		}
		_ = os.WriteFile(filepath.Join(dir, "accounts"), []byte("blocked"), 0600)
		cancel()
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(ctx, sendArgs()); code != 4 {
		t.Fatal(code, out)
	}
	_ = os.Remove(filepath.Join(dir, "accounts"))
	_ = os.Rename(filepath.Join(dir, "saved"), filepath.Join(dir, "accounts"))
	ac, err := a.store.account("alice")
	if err != nil {
		t.Fatal(err)
	}
	if ac.Operations[0].State != "unresolved" || ac.Operations[0].Submitted == nil {
		t.Fatal("crash lost submitted marker")
	}
	g.postHook = nil
	g.status = client.OperationStatus{Status: "sealed", Receipt: &client.Receipt{BlockHash: "block", SealedAt: 42}}
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"--json", "operation", "show", ac.Operations[0].ID}); code != 0 {
		t.Fatal(code, out)
	}
}
func TestUnsubmittedRestartRecovery(t *testing.T) {
	g := newGateway(t)
	dir, key := setup(t, g)
	a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	a.cfg, _ = a.store.config()
	p, _ := a.selectedNetwork()
	c, _ := a.gateway(p, true)
	ac, _ := a.store.account("alice")
	signer, _, close, err := a.unlock(ac)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	intent, err := c.PrepareTransfer(context.Background(), signer, recipient, []client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "0")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := c.SignTransfer(context.Background(), signer, intent)
	if err != nil {
		t.Fatal(err)
	}
	op := operation{ID: signed.OperationID, Account: "alice", Sender: ac.Address, Nonce: intent.Nonce, Profile: p, Created: time.Now().UTC(), State: "signed_unsubmitted"}
	raw, _ := json.Marshal(signed)
	op.Payload, _ = seal(key, raw, opAAD(op))
	ac.Operations = append(ac.Operations, op)
	if err := a.store.saveAccount("alice", ac); err != nil {
		t.Fatal(err)
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "operation", "retry", op.ID}); code != 0 {
		t.Fatal(code, out)
	}
	if len(g.posted) != 1 {
		t.Fatal("not replayed once")
	}
}
func TestHelpSuggestionsAndCompletions(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	for _, tt := range []struct {
		args     []string
		contains string
		code     int
	}{
		{[]string{"help"}, "Start here:", 0},
		{[]string{"balanc"}, "balance", 2},
		{[]string{"account", "creat"}, "create", 2},
		{[]string{"__complete", "account", "use", ""}, "alice", 0},
		{[]string{"__complete", "network", "use", ""}, "dev", 0},
		{[]string{"__complete", "transfer", "send", recipient, "--asset", ""}, "YELLOW=", 0},
		{[]string{"completion", "zsh"}, "#compdef yellow", 0},
	} {
		a, out, diag := newTestApp(dir, &scriptedPrompt{})
		if code := a.execute(context.Background(), tt.args); code != tt.code {
			t.Fatal(tt.args, code, out, diag)
		}
		if !strings.Contains(out.String()+diag.String(), tt.contains) {
			t.Fatal(tt.args, out, diag)
		}
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"balanc", "--json"}); code != 2 {
		t.Fatal(code, out)
	}
	assertOneJSON(t, out.String())
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	_ = a.execute(context.Background(), []string{"help"})
	for _, word := range []string{"deposit", "withdraw", "custody", "swap", "pool", "validator"} {
		if strings.Contains(out.String(), word) {
			t.Fatal("out of scope help", word)
		}
	}
}
func TestEncryptionPermissionsAndBackupRestore(t *testing.T) {
	g := newGateway(t)
	g.postCode = 504
	dir, key := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	_ = a.execute(context.Background(), sendArgs())
	ac, _ := a.store.account("alice")
	if _, _, err := decryptSecret([]byte("wrong"), ac); err == nil {
		t.Fatal("wrong passphrase accepted")
	}
	for _, path := range []string{dir, filepath.Join(dir, "accounts"), filepath.Join(dir, "accounts", "alice.json")} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
			t.Fatal("permissions too broad", path, st.Mode())
		}
	}
	changed := ac.Operations[0]
	changed.Profile.Gateway = "https://evil.example"
	if _, err := decodeOperation(key, changed); err == nil {
		t.Fatal("journal identity tampering was not authenticated")
	}
	backup := filepath.Join(t.TempDir(), "backup.json")
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "account", "backup", "alice", "--out", backup}); code != 0 {
		t.Fatal(code, out)
	}
	newDir := t.TempDir()
	prompt := &scriptedPrompt{approve: true, lines: []string{"backup", backup}, secrets: []string{testPass, "new backup passphrase", "new backup passphrase"}}
	a, out, _ = newTestApp(newDir, prompt)
	if code := a.execute(context.Background(), []string{"--json", "account", "import", "restored"}); code != 0 {
		t.Fatal(code, out)
	}
	restored, err := a.store.account("restored")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Operations) != 1 || restored.Operations[0].State != "unresolved" {
		t.Fatal("restore lost unresolved journal")
	}
	sec, newKey, err := decryptSecret([]byte("new backup passphrase"), restored)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(newKey)
	if sec.Value != testKey {
		t.Fatal("restore changed identity")
	}
	original, err := decodeOperation(key, ac.Operations[0])
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := decodeOperation(newKey, restored.Operations[0])
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := json.Marshal(original)
	b2, _ := json.Marshal(replayed)
	if !bytes.Equal(b1, b2) {
		t.Fatal("backup changed authorization")
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "account", "backup", "alice", "--out", backup}); code != 2 {
		t.Fatal("backup overwritten", code, out)
	}
}
func TestAccountCreateImportAndNonTTY(t *testing.T) {
	dir := t.TempDir()
	p := &scriptedPrompt{approve: true}
	a, out, _ := newTestApp(dir, p)
	if code := a.execute(context.Background(), []string{"--json", "account", "create", "alice"}); code != 0 {
		t.Fatal(code, out)
	}
	assertOneJSON(t, out.String())
	if len(strings.Fields(p.words)) != 24 || strings.Contains(out.String(), p.words) {
		t.Fatal("recovery words leaked")
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true, lines: []string{"mnemonic"}, secrets: []string{p.words}})
	if code := a.execute(context.Background(), []string{"--json", "account", "import", "duplicate"}); code != 2 {
		t.Fatal("duplicate signing identity allowed", code, out)
	}
	f, err := os.CreateTemp(t.TempDir(), "pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out.Reset()
	diag := new(bytes.Buffer)
	if code := Execute(context.Background(), []string{"--json", "account", "create", "bob"}, f, out, diag, t.TempDir(), "test"); code != 2 {
		t.Fatal("non-TTY creation accepted", code, out)
	}
}
func TestNetworkReviewAndTrustUpdate(t *testing.T) {
	g := newGateway(t)
	dir := t.TempDir()
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "network", "add", "dev", "--gateway", g.server.URL, "--network-id", "YDEV"}); code != 0 {
		t.Fatal(code, out)
	}
	g.info.Assets[0].Decimals = 9
	a, out, diag := newTestApp(dir, &scriptedPrompt{approve: false})
	if code := a.execute(context.Background(), []string{"--json", "network", "trust-update", "dev"}); code != 130 {
		t.Fatal(code, out)
	}
	if !strings.Contains(diag.String(), "decimals 18 → 9") {
		t.Fatal("diff omitted", diag)
	}
	cfg, _ := a.store.config()
	if cfg.Networks["dev"].Identity.Assets[1].Decimals != 18 {
		t.Fatal("cancelled trust was saved")
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "network", "trust-update", "dev"}); code != 0 {
		t.Fatal(code, out)
	}
	a, out, _ = newTestApp(t.TempDir(), &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "network", "add", "bad", "--gateway", "http://remote.example", "--network-id", "YDEV"}); code != 6 {
		t.Fatal("remote HTTP accepted", code, out)
	}
}
func TestProcessLockAndCrashRelease(t *testing.T) {
	if dir := os.Getenv("YELLOW_TEST_LOCK_DIR"); dir != "" {
		unlock, err := (store{dir}).lock(context.Background())
		if err != nil {
			os.Exit(10)
		}
		defer unlock()
		fmt.Println("locked")
		time.Sleep(time.Minute)
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessLockAndCrashRelease$")
	cmd.Env = append(os.Environ(), "YELLOW_TEST_LOCK_DIR="+dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	reader := bufio.NewReader(stdout)
	if line, err := reader.ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatal("lock helper", line, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if unlock, err := (store{dir}).lock(ctx); err == nil {
		unlock()
		t.Fatal("concurrent process acquired reservation lock")
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	unlock, err := (store{dir}).lock(ctx2)
	if err != nil {
		t.Fatal("crashed process left stale lock", err)
	}
	unlock()
}

func TestSealedNonceCannotBeReservedAgain(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != 0 {
		t.Fatal(code, out)
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != ExitNetwork {
		t.Fatal("stale live nonce was signed again", code, out)
	}
	if len(g.posted) != 1 {
		t.Fatal("duplicate nonce reached POST")
	}
}
func TestWaitIsBoundedAndNeverReposts(t *testing.T) {
	g := newGateway(t)
	g.postCode = 202
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	start := time.Now()
	if code := a.execute(context.Background(), sendArgs("--wait", "100ms")); code != ExitPending {
		t.Fatal(code, out)
	}
	if time.Since(start) > 3*time.Second || len(g.posted) != 1 {
		t.Fatal("unbounded wait or automatic replay")
	}
}
func TestSubmissionMarkerFailurePreventsPOST(t *testing.T) {
	g := newGateway(t)
	dir, key := setup(t, g)
	a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	a.timeout = time.Second
	a.cfg, _ = a.store.config()
	p, _ := a.selectedNetwork()
	c, _ := a.gateway(p, true)
	ac, _ := a.store.account("alice")
	op := operation{ID: "0x" + strings.Repeat("ab", 32), Account: "alice", Sender: ac.Address, Nonce: 7, Profile: p, Created: time.Now().UTC(), State: "signed_unsubmitted"}
	op.Payload, _ = seal(key, []byte("test"), opAAD(op))
	ac.Operations = append(ac.Operations, op)
	if err := a.store.saveAccount("alice", ac); err != nil {
		t.Fatal(err)
	}
	_ = os.Rename(filepath.Join(dir, "accounts"), filepath.Join(dir, "saved"))
	_ = os.WriteFile(filepath.Join(dir, "accounts"), []byte("blocked"), 0600)
	if err := a.submit(context.Background(), c, "alice", ac, 0, &client.SignedTransfer{OperationID: op.ID}, 0); err == nil {
		t.Fatal("expected failed marker commit")
	}
	if len(g.posted) != 0 {
		t.Fatal("POST after submission marker failure")
	}
	var persisted accountFile
	if err := readJSON(filepath.Join(dir, "saved", "alice.json"), &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Operations[0].State != "signed_unsubmitted" {
		t.Fatal("failed marker changed original journal")
	}
}
func TestConcurrentCommandsReserveOnlyOnce(t *testing.T) {
	g := newGateway(t)
	g.postCode = 202
	dir, _ := setup(t, g)
	start := make(chan struct{})
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			results <- a.execute(context.Background(), sendArgs())
		}()
	}
	close(start)
	first, second := <-results, <-results
	if !((first == ExitPending && second == ExitUnknown) || (second == ExitPending && first == ExitUnknown)) {
		t.Fatal("concurrent result", first, second)
	}
	if len(g.posted) != 1 {
		t.Fatal("concurrent commands signed conflicting transfers")
	}
}

func TestSubmissionPreflightFailureIsUnsubmitted(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	g.infoHook = func(n int) {
		if n == 5 {
			g.info.Assets[0].Decimals = 9
		}
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != ExitNetwork {
		t.Fatal(code, out)
	}
	if len(g.posted) != 0 {
		t.Fatal("untrusted preflight reached POST")
	}
	ac, err := a.store.account("alice")
	if err != nil {
		t.Fatal(err)
	}
	if ac.Operations[0].State != "signed_unsubmitted" || ac.Operations[0].Submitted != nil {
		t.Fatal("known unsubmitted authorization was mislabeled unresolved")
	}
	if ac.Operations[0].Reason == "" || !strings.Contains(out.String(), "catalogue changed") || !strings.Contains(out.String(), "operation retry") {
		t.Fatal("preflight failure lost its cause or recovery command", out)
	}
	g.infoHook = nil
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "network", "trust-update", "dev"}); code != 0 {
		t.Fatal(code, out)
	}
	a, out, diag := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "operation", "retry", ac.Operations[0].ID}); code != 0 {
		t.Fatal("reviewed catalogue update failed to unblock original gateway recovery", code, out)
	}
	if !strings.Contains(diag.String(), "decimals 18 → 9") || !strings.Contains(diag.String(), "1250000000 ·") {
		t.Fatal("replay omitted changed asset precision or current amount")
	}
}

// A refused replay must stay unresolved: the original submission may have
// executed, and recording "rejected" would lift the unresolved-send guard.
func TestRefusedReplayStaysUnresolved(t *testing.T) {
	g := newGateway(t)
	dir, key := setup(t, g)
	a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	a.timeout = time.Second
	a.cfg, _ = a.store.config()
	p, _ := a.selectedNetwork()
	c, _ := a.gateway(p, true)
	ac, err := a.store.account("alice")
	if err != nil {
		t.Fatal(err)
	}
	signer, _, close, err := a.unlock(ac)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	intent, err := c.PrepareTransfer(context.Background(), signer, recipient,
		[]client.TransferAmount{{Asset: "YELLOW", Amount: "1.25"}}, "0")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := c.SignTransfer(context.Background(), signer, intent)
	if err != nil {
		t.Fatal(err)
	}
	submitted := time.Now().Add(-time.Minute).UTC()
	op := operation{ID: signed.OperationID, Account: "alice", Sender: ac.Address, Nonce: intent.Nonce, Profile: p,
		Created: submitted, Updated: submitted, State: "unresolved", Submitted: &submitted}
	raw, err := json.Marshal(signed)
	if err != nil {
		t.Fatal(err)
	}
	if op.Payload, err = seal(key, raw, opAAD(op)); err != nil {
		t.Fatal(err)
	}
	ac.Operations = append(ac.Operations, op)
	if err := a.store.saveAccount("alice", ac); err != nil {
		t.Fatal(err)
	}
	// The original gateway reports an ended admission window with no seal, so
	// the replay is permitted; the gateway then refuses it definitively.
	g.status = client.OperationStatus{Status: "unknown", PendingUntil: time.Now().Add(-30 * time.Second).Unix()}
	g.postCode = 400
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), []string{"--json", "operation", "retry", signed.OperationID}); code != ExitUnknown {
		t.Fatalf("refused replay exit = %d, want %d (%s)", code, ExitUnknown, out)
	}
	if len(g.posted) != 1 {
		t.Fatalf("replay POSTs = %d, want 1", len(g.posted))
	}
	persisted, err := a.store.account("alice")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Operations[0].State != "unresolved" {
		t.Fatalf("refused replay recorded %q, want unresolved", persisted.Operations[0].State)
	}
}

// --dry-run signs nothing, so an unrelated unresolved operation must not block it.
func TestDryRunIgnoresUnresolvedGuard(t *testing.T) {
	g := newGateway(t)
	g.postCode = 504
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs("--idempotency-key", "invoice-42")); code != ExitUnknown {
		t.Fatalf("first send exit = %d (%s)", code, out)
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs("--dry-run", "--idempotency-key", "invoice-42")); code != 0 {
		t.Fatalf("dry run exit = %d (%s)", code, out)
	}
	if !strings.Contains(out.String(), `"dryRun":true`) {
		t.Fatalf("dry run output = %s", out)
	}
	if len(g.posted) != 1 {
		t.Fatalf("dry run POSTs = %d, want 1 (the first send only)", len(g.posted))
	}
}

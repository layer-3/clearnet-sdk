package yellow

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestAccountExportKey(t *testing.T) {
	for _, tc := range []struct {
		name string
		sec  secret
		key  string
	}{
		{"imported key", secret{"key", testKey}, testKey},
		{"recovery words", secret{"mnemonic", "test test test test test test test test test test test junk"}, "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			a.cfg = config{Version: formatVersion, Networks: map[string]profile{}}
			if err := a.store.init(); err != nil {
				t.Fatal(err)
			}
			if err := a.saveNewAccount("alice", tc.sec, nil); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "accounts", "alice.json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"account", "export-key", "alice"}, {"account", "export-key"}, {"--account", "alice", "account", "export-key"}} {
				p := &scriptedPrompt{approve: true}
				a, out, diag := newTestApp(dir, p)
				if code := a.execute(context.Background(), args); code != 0 {
					t.Fatal(code, diag)
				}
				if p.exportedKey != tc.key || p.exportedName != "alice" {
					t.Fatal("wrong exported identity")
				}
				if !bytes.Equal(p.exportedBytes, make([]byte, 64)) {
					t.Fatal("export buffer was not cleared after reveal")
				}
				key, err := crypto.HexToECDSA(p.exportedKey)
				if err != nil || crypto.PubkeyToAddress(key.PublicKey).Hex() != p.exportedAddress {
					t.Fatal("export does not recover the saved address")
				}
				closeKey(key)
				if out.Len() != 0 || diag.Len() != 0 {
					t.Fatal("export wrote to stdout or stderr")
				}
			}
			for _, tc := range []struct {
				name string
				args []string
				p    *scriptedPrompt
				code int
			}{
				{"cancelled", []string{"account", "export-key", "alice"}, &scriptedPrompt{}, 130},
				{"wrong passphrase", []string{"account", "export-key", "alice"}, &scriptedPrompt{approve: true, secrets: []string{"wrong"}}, 2},
				{"json", []string{"--json", "account", "export-key", "alice"}, &scriptedPrompt{approve: true, secrets: []string{testPass}}, 2},
			} {
				t.Run(tc.name, func(t *testing.T) {
					a, out, diag := newTestApp(dir, tc.p)
					if code := a.execute(context.Background(), tc.args); code != tc.code {
						t.Fatal(code, out, diag)
					}
					if tc.p.exportedKey != "" || strings.Contains(out.String()+diag.String(), testKey) {
						t.Fatal("failed export revealed a key")
					}
					if tc.name == "json" {
						assertOneJSON(t, out.String())
						if len(tc.p.secrets) != 1 {
							t.Fatal("JSON export prompted for unlock")
						}
					}
				})
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("export modified encrypted account or journal")
			}
			out, diag := new(bytes.Buffer), new(bytes.Buffer)
			if code := Execute(context.Background(), []string{"account", "export-key", "alice"}, nil, out, diag, dir, "test"); code != 2 {
				t.Fatal("noninteractive export accepted", code)
			}
		})
	}
}

func TestAccountPrivateKeyImportRoundTrip(t *testing.T) {
	for _, prefix := range []string{"", "0x"} {
		t.Run("prefix="+prefix, func(t *testing.T) {
			p := &scriptedPrompt{approve: true, lines: []string{"key"}, secrets: []string{prefix + testKey, testPass, testPass}}
			a, out, diag := newTestApp(t.TempDir(), p)
			if code := a.execute(context.Background(), []string{"--json", "account", "import", "metamask"}); code != 0 {
				t.Fatal(code, out, diag)
			}
			assertOneJSON(t, out.String())
			if strings.Contains(out.String()+diag.String(), testKey) {
				t.Fatal("import echoed the private key")
			}
			ac, err := a.store.account("metamask")
			if err != nil || ac.Address != "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf" {
				t.Fatal("imported address does not match key")
			}
			raw, err := os.ReadFile(filepath.Join(a.store.dir, "accounts", "metamask.json"))
			if err != nil || bytes.Contains(raw, []byte(testKey)) || bytes.Contains(raw, []byte(testPass)) {
				t.Fatal("import stored plaintext secrets")
			}
			a, _, diag = newTestApp(a.store.dir, p)
			if code := a.execute(context.Background(), []string{"account", "export-key", "metamask"}); code != 0 || p.exportedKey != testKey {
				t.Fatal("import/export round trip failed", code, diag)
			}
		})
	}
}

func TestAccountExportRejectsMismatchedIdentity(t *testing.T) {
	dir := t.TempDir()
	encrypted, key, err := encryptSecret([]byte(testPass), recipient, secret{"key", testKey})
	if err != nil {
		t.Fatal(err)
	}
	clear(key)
	p := &scriptedPrompt{approve: true}
	a, out, diag := newTestApp(dir, p)
	if err := a.store.init(); err != nil {
		t.Fatal(err)
	}
	if err := a.store.saveAccount("alice", &accountFile{Version: 1, Address: recipient, Secret: encrypted}); err != nil {
		t.Fatal(err)
	}
	if code := a.execute(context.Background(), []string{"account", "export-key", "alice"}); code != 2 || p.exportedKey != "" {
		t.Fatal("mismatched identity exported", code, out, diag)
	}
}

package clearbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func TestSQLiteRestartDoesNotReseedAndRecoversPendingDelivery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clearbox.db")
	cfg := Config{DatabasePath: path, SealDelay: time.Second, DeliveryDelay: time.Second}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Created {
		t.Fatal("first initialization was not marked new")
	}
	address, key := s.SeedAddress, s.SeedPrivateKey
	server := httptest.NewServer(s)
	c, _ := client.New(server.URL, NetworkID)
	if getBalance(t, c, address, "YELLOW") != "1000000000000000000000000" || getBalance(t, c, address, "ETH") != "0" || getBalance(t, c, address, "USDC") != "0" {
		t.Fatal("initial allocation must fund only YELLOW")
	}
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, key, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1")
	result, err := c.SubmitTransfer(context.Background(), signed)
	if err != nil || result.Status != "pending" {
		t.Fatalf("admission: %+v %v", result, err)
	}
	before := getBalance(t, c, address, "YELLOW")
	server.Close()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.Created || reopened.SeedAddress != address || reopened.SeedPrivateKey != key {
		t.Fatal("restart regenerated the account or reseeded")
	}
	// The durable absolute event times survive a process that was down past
	// both deadlines; the next read completes the original operation once.
	reopened.now = func() time.Time { return time.Now().Add(time.Minute) }
	server = httptest.NewServer(reopened)
	defer server.Close()
	c, _ = client.New(server.URL, NetworkID)
	if got := getBalance(t, c, address, "YELLOW"); got != before {
		t.Fatalf("restart topped up sender: %s -> %s", before, got)
	}
	if got := getBalance(t, c, bob.Hex(), "YELLOW"); got != "25000000000000000000" {
		t.Fatal("pending delivery lost", got)
	}
	op, err := c.Operation(context.Background(), signed.OperationID)
	if err != nil || op.Status != "sealed" {
		t.Fatalf("durable operation: %+v %v", op, err)
	}
	replay, err := c.SubmitTransfer(context.Background(), signed)
	if err != nil || *replay.Receipt != *op.Receipt {
		t.Fatalf("durable replay: %+v %v", replay, err)
	}
	balance, _ := c.Balance(context.Background(), address)
	history, _ := c.Transfers(context.Background(), bob.Hex(), client.TransferPageOptions{})
	if balance.Nonce != 1 || len(history.Items) != 1 {
		t.Fatal("nonce/history not durable or replay credited twice")
	}
	if stat, err := os.Stat(path); err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("local key database must be owner-only", err)
	}
}

func TestSQLiteDatabaseHasOneOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clearbox.db")
	first, err := New(Config{DatabasePath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := New(Config{DatabasePath: path})
	if err == nil {
		second.Close()
		t.Fatal("two servers opened the same ledger")
	}
}

func TestFailedSQLiteCommitDoesNotAcknowledgeOrMutateFunds(t *testing.T) {
	s, server, c := fixture(t, Config{})
	bob, _ := sign.EthAddress(testSigner(t, bobKey))
	signed := prepared(t, c, s.SeedPrivateKey, bob.Hex(), []client.TransferAmount{{Asset: "YELLOW", Amount: "25"}}, "1")
	before := getBalance(t, c, s.SeedAddress, "YELLOW")
	if _, err := s.db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	if result, err := c.SubmitTransfer(context.Background(), signed); err == nil || result != nil {
		t.Fatal("failed commit was acknowledged", result)
	}
	health, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	health.Body.Close()
	if health.StatusCode != 503 {
		t.Fatal("storage failure was not reported by readiness")
	}
	if _, err := s.db.Exec("PRAGMA query_only=OFF"); err != nil {
		t.Fatal(err)
	}
	if got := getBalance(t, c, s.SeedAddress, "YELLOW"); got != before {
		t.Fatal("failed commit left uncommitted balance", got, server.URL)
	}
	balance, _ := c.Balance(context.Background(), s.SeedAddress)
	op, _ := c.Operation(context.Background(), signed.OperationID)
	if balance.Nonce != 0 || op.Status != "unknown" {
		t.Fatal("failed commit left a nonce/operation")
	}
	if _, err := c.SubmitTransfer(context.Background(), signed); err != nil {
		t.Fatal("storage did not recover", err)
	}
	health, err = http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	health.Body.Close()
	if health.StatusCode != 200 {
		t.Fatal("readiness did not recover after a successful commit")
	}
}

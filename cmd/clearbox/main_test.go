package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/layer-3/clearnet-sdk/internal/clearbox"
)

func TestBusyListenDoesNotCreateOrSeedDatabase(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	path := filepath.Join(t.TempDir(), "clearbox.db")
	if err := run(listener.Addr().String(), clearbox.Config{DatabasePath: path}); err == nil {
		t.Fatal("busy listener accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed startup initialized a seed without printing its key", err)
	}
}

func TestPublicServingRequiresPersistentDatabase(t *testing.T) {
	if err := run("127.0.0.1:0", clearbox.Config{PublicURL: "https://sandbox.example", DatabasePath: ":memory:"}); err == nil {
		t.Fatal("public in-memory ledger accepted")
	}
}

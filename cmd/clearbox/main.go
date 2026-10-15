// clearbox runs the public SDK's development account and transfer simulator.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/layer-3/clearnet-sdk/internal/clearbox"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "Loopback HTTP listen address")
	db := flag.String("db", ".clearbox/clearbox.db", "Local SQLite database (created and seeded only once)")
	seal := flag.Duration("seal-delay", 750*time.Millisecond, "Admission-to-seal delay; also paces new public writes (0 to 1m)")
	delivery := flag.Duration("delivery-delay", 500*time.Millisecond, "Sender-seal-to-recipient-credit delay (0 to 1m)")
	publicURL := flag.String("public-url", "", "External HTTPS origin; enables public serving and browser CORS")
	proxies := flag.String("trusted-proxies", "", "Comma-separated proxy CIDRs permitted to supply X-Forwarded-For")
	requestRate := flag.Float64("rate-limit", 10, "Public requests per second per client IP (burst is twice this)")
	maxOperations := flag.Int("max-operations", 2000, "Maximum retained operations; existing operations remain queryable at capacity")
	flag.Parse()
	cfg := clearbox.Config{DatabasePath: *db, SealDelay: *seal, DeliveryDelay: *delivery, PublicURL: *publicURL, RequestsPerSecond: *requestRate, MaxOperations: *maxOperations}
	if *proxies != "" {
		cfg.TrustedProxies = strings.Split(*proxies, ",")
	}
	if err := run(*listen, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "clearbox:", err)
		os.Exit(1)
	}
}

func run(address string, cfg clearbox.Config) error {
	if cfg.PublicURL != "" && (cfg.DatabasePath == "" || cfg.DatabasePath == ":memory:") {
		return fmt.Errorf("public serving requires an on-disk SQLite database")
	}
	var listener net.Listener
	var err error
	if cfg.PublicURL == "" {
		listener, err = clearbox.LoopbackListen(address)
	} else {
		listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		return err
	}
	defer listener.Close()
	sim, err := clearbox.New(cfg)
	if err != nil {
		return err
	}
	defer sim.Close()
	server := &http.Server{Handler: sim, MaxHeaderBytes: 16 << 10, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Printf("Clearbox listening at http://%s\n", listener.Addr())
	fmt.Printf("Local YDEV simulation. SQLite database: %s\nSeed account: %s\n", cfg.DatabasePath, sim.SeedAddress)
	if cfg.PublicURL != "" {
		fmt.Printf("Public developer sandbox: %s (one replica; persistent SQLite volume required)\n", cfg.PublicURL)
	}
	if sim.Created {
		fmt.Println("New database: seeded 1,000,000 YELLOW. No other tokens are funded.")
		fmt.Println("WARNING: this key is printed for DEVELOPMENT ONLY. Never reuse it for real funds, real networks or production wallets.")
		fmt.Printf("Private key: %s\n", sim.SeedPrivateKey)
	} else {
		fmt.Println("Reusing existing ledger and seed account; no funds added.")
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

// Command wait blocks until every devnet node answers RPC, then exits 0 — so
// `make devnet` returns only once the infra is ready to drive. Each endpoint is
// polled until it responds or the deadline elapses.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type probe struct {
	name string
	url  string
	user string
	pass string
	body string
}

func main() {
	networks := flag.String("networks", "", "comma-separated devnet networks to wait for: anvil,bitcoind,rippled,solana")
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "devnet: unexpected positional arguments: %s\n", strings.Join(flag.Args(), " "))
		os.Exit(2)
	}

	probes := []probe{
		{name: "anvil", url: envOr("EVM_RPC_URL", "http://127.0.0.1:8545"),
			body: `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`},
		{name: "bitcoind", url: envOr("BTC_RPC_URL", "http://127.0.0.1:18443"),
			user: envOr("BTC_RPC_USER", "sdk"), pass: envOr("BTC_RPC_PASS", "sdk"),
			body: `{"jsonrpc":"1.0","id":1,"method":"getblockchaininfo","params":[]}`},
		{name: "rippled", url: envOr("XRPL_RPC_URL", "http://127.0.0.1:5005"),
			body: `{"method":"server_info","params":[{}]}`},
		{name: "solana", url: envOr("SOL_RPC_URL", "http://127.0.0.1:8899"),
			body: `{"jsonrpc":"2.0","id":1,"method":"getHealth"}`},
	}

	selected, err := selectProbes(probes, *networks, flagWasSet("networks"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "devnet: %v\n", err)
		os.Exit(2)
	}

	deadline := time.Now().Add(90 * time.Second)
	client := &http.Client{Timeout: 3 * time.Second}
	for _, p := range selected {
		if err := waitOne(client, p, deadline); err != nil {
			fmt.Fprintf(os.Stderr, "devnet: %s not ready: %v\n", p.name, err)
			os.Exit(1)
		}
		fmt.Printf("devnet: %s ready\n", p.name)
	}
}

func selectProbes(probes []probe, networks string, specified bool) ([]probe, error) {
	if !specified {
		return probes, nil
	}

	byName := make(map[string]probe, len(probes))
	names := make([]string, 0, len(probes))
	for _, p := range probes {
		byName[p.name] = p
		names = append(names, p.name)
	}

	parts := strings.Split(networks, ",")
	selected := make([]probe, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("--networks must contain non-empty names from: %s", strings.Join(names, ","))
		}
		p, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("unsupported network %q; supported values: %s", name, strings.Join(names, ","))
		}
		selected = append(selected, p)
	}

	return selected, nil
}

func flagWasSet(name string) bool {
	wasSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

func waitOne(client *http.Client, p probe, deadline time.Time) error {
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, p.url, bytes.NewReader([]byte(p.body)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if p.user != "" {
			req.SetBasicAuth(p.user, p.pass)
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timed out: %v", last)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

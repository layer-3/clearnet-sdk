// Command abi_refresher regenerates the EVM contract bindings (the
// `pkg/blockchain/evm/*_abi.go` files) from the vendored ABI + bytecode files
// under `pkg/blockchain/evm/artifacts/`, using go-ethereum's abigen library directly
// — no bash, no jq, no external abigen binary, no forge build.
//
// The vendored `<Type>.abi` files and, for deployable contracts, `<Type>.bin`
// files are committed: they are the contract surface this package binds, so a
// contract change shows up as a reviewable diff here. Interface-only artifacts
// such as `IConfig.abi` intentionally have no bytecode. Regeneration is fully
// self-contained:
//
//	go generate ./pkg/blockchain/evm/...   # or: go run ./pkg/blockchain/evm/abi_refresher
//
// Refreshing the vendored files (only when a contract's ABI/bytecode actually
// changes) is done from a repo that owns the Solidity source, e.g.:
//
//	jq -r '.abi'             clearnet/contracts/evm/out/Custody.sol/Custody.json > artifacts/Custody.abi
//	jq -r '.bytecode.object' clearnet/contracts/evm/out/Custody.sol/Custody.json > artifacts/Custody.bin
//
// Artifacts owned by this repo are the exception — they are refreshed by a
// local `forge build` in contracts/, because their Solidity source lives here:
// the registry interfaces and the per-issuer config governance surface.
// This package binds the registry through those interfaces only — never through
// clearnet's private Registry.sol implementation — see artifacts/README.md.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
)

// pkgName is the package the generated bindings belong to.
const pkgName = "evm"

// artifactsSubdir is the vendored ABI + bytecode directory, relative to the
// evm package directory.
const artifactsSubdir = "artifacts"

// group is one abigen.Bind invocation. Contracts in the same group SHARE
// generated Go structs: abigen emits each distinct Solidity struct once per
// call, keyed by name + canonical type. Contracts that traffic in the same
// struct must therefore be bound together — ClearnetRegistry and
// ClearnetRegistryProtocol both return NodeRecord, and splitting them emits two
// `type NodeRecord struct` declarations into package evm.
type group struct {
	names []string // <name>.abi (+ optional <name>.bin) basenames; also the abigen --type
	out   string   // generated file written into the evm package
}

var groups = []group{
	{names: []string{"ClearnetRegistry", "ClearnetRegistryProtocol"}, out: "registry_abi.go"},
	{names: []string{"Slasher"}, out: "adjudicator_abi.go"},
	{names: []string{"MockERC20"}, out: "mockerc20_abi.go"},
	{names: []string{"Custody"}, out: "custody_abi.go"},
	{names: []string{"NodeID"}, out: "nodeid_abi.go"},
	{names: []string{"Faucet"}, out: "faucet_abi.go"},
	{names: []string{"YellowToken"}, out: "yellowtoken_abi.go"},
	{names: []string{"Config"}, out: "config_abi.go"},
	{names: []string{"ConfigRegistry"}, out: "config_registry_abi.go"},
	{names: []string{"IConfig"}, out: "iconfig_abi.go"},
}

func main() {
	evmDir := packageDir()
	artifactsDir := filepath.Join(evmDir, artifactsSubdir)

	for _, g := range groups {
		if err := generate(evmDir, artifactsDir, g); err != nil {
			fmt.Fprintf(os.Stderr, "abi_refresher: %s: %v\n", strings.Join(g.names, "+"), err)
			os.Exit(1)
		}
		fmt.Printf("abi_refresher: wrote %s\n", g.out)
	}
}

func generate(evmDir, artifactsDir string, g group) error {
	abis := make([]string, len(g.names))
	bins := make([]string, len(g.names))
	for i, name := range g.names {
		abiJSON, err := os.ReadFile(filepath.Join(artifactsDir, name+".abi"))
		if err != nil {
			return fmt.Errorf("read abi: %w", err)
		}
		abis[i] = string(abiJSON)

		// An interface has no deploy bytecode, so e.g. ClearnetRegistry.bin
		// will not exist. Treat that as "no bytecode" rather than an error —
		// abigen then emits no Deploy… helper for that contract.
		binHex, err := os.ReadFile(filepath.Join(artifactsDir, name+".bin"))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("read bin: %w", err)
			}
			bins[i] = ""
			continue
		}
		// abigen wants the deploy bytecode as a bare hex string with neither
		// the 0x prefix nor a trailing newline.
		bins[i] = strings.TrimPrefix(strings.TrimSpace(string(binHex)), "0x")
	}
	code, err := abigen.Bind(
		g.names,
		abis,
		bins,
		nil, // fsigs — only used by the combined-json path
		pkgName,
		nil, // libs
		nil, // aliases
	)
	if err != nil {
		return fmt.Errorf("abigen bind: %w", err)
	}

	if err := os.WriteFile(filepath.Join(evmDir, g.out), []byte(code), 0o644); err != nil {
		return fmt.Errorf("write binding: %w", err)
	}
	return nil
}

// packageDir returns the absolute path of the evm package directory (the
// parent of this abi_refresher command), resolved from this source file so the
// working directory is irrelevant.
func packageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("abi_refresher: runtime.Caller failed")
	}
	// file = <evm>/abi_refresher/main.go
	return filepath.Dir(filepath.Dir(file))
}

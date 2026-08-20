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
// For each contract abigen.Bind emits the Caller/Transactor/Filterer +
// Deploy tuple into `<out>` under `package evm`. Bytecode is kept for deployable
// contracts so custody (and devnet deploy paths) can deploy via the generated
// `Deploy*` helpers; interface-only bindings have empty deploy bytecode and are
// used only for calls/filters. Paths are resolved relative to this source file,
// so the working directory does not matter.
package main

import (
	"fmt"
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

// contract maps one vendored artifact to its generated binding. name is the
// `<name>.abi` / `<name>.bin` basename and the abigen --type; out is the
// generated file written into the evm package.
type contract struct {
	name          string
	out           string
	interfaceOnly bool
}

var contracts = []contract{
	{"Slasher", "adjudicator_abi.go", false},
	{"Registry", "registry_abi.go", false},
	{"MockERC20", "mockerc20_abi.go", false},
	{"Custody", "custody_abi.go", false},
	{"NodeID", "nodeid_abi.go", false},
	{"Faucet", "faucet_abi.go", false},
	{"YellowToken", "yellowtoken_abi.go", false},
	{"Config", "config_abi.go", false},
	{"ConfigGovernor", "config_governor_abi.go", false},
	{"ConfigRegistry", "config_registry_abi.go", false},
	{"IConfig", "iconfig_abi.go", true},
}

func main() {
	evmDir := packageDir()
	artifactsDir := filepath.Join(evmDir, artifactsSubdir)

	for _, c := range contracts {
		if err := generate(evmDir, artifactsDir, c); err != nil {
			fmt.Fprintf(os.Stderr, "abi_refresher: %s: %v\n", c.name, err)
			os.Exit(1)
		}
		fmt.Printf("abi_refresher: wrote %s\n", c.out)
	}
}

func generate(evmDir, artifactsDir string, c contract) error {
	abiJSON, err := os.ReadFile(filepath.Join(artifactsDir, c.name+".abi"))
	if err != nil {
		return fmt.Errorf("read abi: %w", err)
	}
	var binHex []byte
	if !c.interfaceOnly {
		binHex, err = os.ReadFile(filepath.Join(artifactsDir, c.name+".bin"))
		if err != nil {
			return fmt.Errorf("read bin: %w", err)
		}
	}
	// abigen wants the deploy bytecode as a bare hex string with neither the
	// 0x prefix nor a trailing newline.
	bytecode := strings.TrimPrefix(strings.TrimSpace(string(binHex)), "0x")

	code, err := abigen.Bind(
		[]string{c.name},
		[]string{string(abiJSON)},
		[]string{bytecode},
		nil, // fsigs — only used by the combined-json path
		pkgName,
		nil, // libs
		nil, // aliases
	)
	if err != nil {
		return fmt.Errorf("abigen bind: %w", err)
	}

	if err := os.WriteFile(filepath.Join(evmDir, c.out), []byte(code), 0o644); err != nil {
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

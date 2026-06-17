// Command idl_refresher regenerates the Solana program bindings (the
// pkg/blockchain/sol/custody package) from the vendored Anchor IDL using
// anchor-go's generator library directly — the Solana analog of the EVM
// abi_refresher (which drives go-ethereum's abigen). No external binary, no AI
// in the loop.
//
// The vendored artifacts/custody.json (the Anchor IDL — Solana's ABI analog)
// is committed; a contract change shows up as a reviewable IDL diff. Regenerate
// with:
//
//	go generate ./pkg/blockchain/sol/...   # or: go run ./pkg/blockchain/sol/idl_refresher
//
// Refreshing the IDL itself (only when the program changes) is `anchor build`
// in the repo that owns the Rust source — see artifacts/README.md.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gagliardetto/anchor-go/generator"
	"github.com/gagliardetto/anchor-go/idl"
	"github.com/gagliardetto/solana-go"
)

// programID is the custody program's fixed on-chain id (declare_id!).
const programID = "98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg"

// generated bindings package name + its module import path.
const (
	pkgName = "custody"
	modPath = "github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

func main() {
	solDir := packageDir()
	idlPath := filepath.Join(solDir, "artifacts", "custody.json")
	outDir := filepath.Join(solDir, "custody")

	parsed, err := idl.ParseFromFilepath(idlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "idl_refresher: parse %s: %v\n", idlPath, err)
		os.Exit(1)
	}
	pid := solana.MustPublicKeyFromBase58(programID)

	out, err := generator.NewGenerator(parsed, &generator.GeneratorOptions{
		OutputDir:   outDir,
		Package:     pkgName,
		ModPath:     modPath,
		ProgramId:   &pid,
		ProgramName: pkgName,
		SkipGoMod:   true, // bindings live inside the SDK module
	}).Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "idl_refresher: generate: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "idl_refresher: mkdir %s: %v\n", outDir, err)
		os.Exit(1)
	}
	for _, file := range out.Files {
		// Skip the generated placeholder test — it pulls heavy test deps
		// (gomega/goleak) into the SDK for no benefit.
		if file.Name == "tests_test.go" {
			continue
		}
		path := filepath.Join(outDir, file.Name)
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "idl_refresher: create %s: %v\n", path, err)
			os.Exit(1)
		}
		if err := file.File.Render(f); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "idl_refresher: render %s: %v\n", path, err)
			os.Exit(1)
		}
		f.Close()
		fmt.Printf("idl_refresher: wrote custody/%s\n", file.Name)
	}
}

// packageDir returns the pkg/blockchain/sol directory, resolved from this
// source file so the working directory is irrelevant.
func packageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("idl_refresher: runtime.Caller failed")
	}
	// file = <sol>/idl_refresher/main.go
	return filepath.Dir(filepath.Dir(file))
}

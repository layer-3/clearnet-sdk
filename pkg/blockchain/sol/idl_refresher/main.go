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
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gagliardetto/anchor-go/generator"
	"github.com/gagliardetto/anchor-go/idl"
	"github.com/gagliardetto/solana-go"
)

// defaultProgramID is the generated binding's default program id
// (declare_id!). Runtime SDK paths pass their configured program id into each
// instruction builder; see rewriteInstructionProgramIDs below.
const defaultProgramID = "98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg"

// generated bindings package name + its module import path.
const (
	pkgName = "custody"
	modPath = "github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

func main() {
	programID := flag.String("program-id", defaultProgramID, "Solana custody program id for generated default metadata")
	flag.Parse()

	solDir := packageDir()
	idlPath := filepath.Join(solDir, "artifacts", "custody.json")
	outDir := filepath.Join(solDir, "custody")

	parsed, err := idl.ParseFromFilepath(idlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "idl_refresher: parse %s: %v\n", idlPath, err)
		os.Exit(1)
	}
	pid := solana.MustPublicKeyFromBase58(*programID)

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
	instructionsPath := filepath.Join(outDir, "instructions.go")
	if err := rewriteInstructionProgramIDs(instructionsPath); err != nil {
		fmt.Fprintf(os.Stderr, "idl_refresher: rewrite %s: %v\n", instructionsPath, err)
		os.Exit(1)
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

// rewriteInstructionProgramIDs patches anchor-go's generated instruction
// builders so runtime callers can dispatch to a configured program id. The IDL
// has a required `program` account for every custody instruction, and the SDK's
// depositor/finalizers already pass their configured program id into that
// account. anchor-go still emits `NewInstruction(ProgramID, ...)`, which binds
// the generated default id into every transaction. We rewrite only functions
// that accept a `programAccount` parameter, replacing the first argument to
// solanago.NewInstruction with that parameter.
func rewriteInstructionProgramIDs(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return err
	}

	rewrites := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type == nil || fn.Type.Params == nil || fn.Body == nil {
			continue
		}
		if !hasParam(fn, "programAccount") {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			if !isSelector(call.Fun, "solanago", "NewInstruction") {
				return true
			}
			if ident, ok := call.Args[0].(*ast.Ident); !ok || ident.Name != "ProgramID" {
				return true
			}
			call.Args[0] = ast.NewIdent("programAccount")
			rewrites++
			return true
		})
	}
	if rewrites == 0 {
		return fmt.Errorf("no solanago.NewInstruction(ProgramID, ...) calls rewritten")
	}

	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func hasParam(fn *ast.FuncDecl, name string) bool {
	for _, field := range fn.Type.Params.List {
		for _, n := range field.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

func isSelector(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != name {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg
}

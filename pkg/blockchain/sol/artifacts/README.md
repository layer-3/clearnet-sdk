# Solana program artifacts

Vendored artifacts for the custody Anchor program:

- **`custody.json`** — the Anchor **IDL** (Solana's ABI analog). Source of truth
  for the generated `../custody` bindings: `idl_refresher` reads this and emits
  the Go client via anchor-go's generator library (the Solana parallel of the
  EVM `abi_refresher` driving abigen). A program change shows up here as a
  reviewable IDL diff.
- **`custody.so`** — the compiled BPF program (the bytecode analog). Used by the
  integration-test devnet, which preloads it into `solana-test-validator` at the
  fixed program id — no on-chain deploy or Anchor toolchain needed at test time.

Program id (fixed, `declare_id!`): `98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg`.

## Regenerate the bindings (common case)

```sh
make generate          # go generate ./...  → idl_refresher (+ evm abi_refresher, cbor-gen)
```

or directly:

```sh
go generate ./pkg/blockchain/sol/...
```

Rewrites `../custody/*.go` from `custody.json`. Commit the result.

## Refresh the artifacts (only when the program changes)

Both files come from the same source-generating `anchor build` in the repo that
owns the Rust source. That source lives in **custody** at
`chains/sol/contract`, which pins Anchor 1.1.2, Agave/Solana CLI 3.1.10,
platform-tools v1.52, and both Rust compilers in an immutable container build.
Docker is needed only to refresh the artifacts, never to run SDK tests:

```sh
cd ../custody
make solana-artifacts
```

That target updates the custody-owned artifact copies, copies both files here,
and regenerates `../custody/*.go`. Review all three outputs together.

Use that Linux amd64 container even on macOS. Native builds with the same tool
versions can embed different platform-tools library paths and produce a
different `.so`; they are not the canonical artifacts checked by custody CI.

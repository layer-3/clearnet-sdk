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

Both files come from `anchor build` in the repo that owns the Rust source
(`clearnet/contracts/solana`, Anchor 0.31). Requires the Solana + Anchor
toolchain (`solana` / `cargo-build-sbf` + `anchor` 0.31 via avm) — needed only
to refresh, never to run the tests:

```sh
cd ../clearnet/contracts/solana
anchor build
cp target/idl/custody.json    <sdk>/pkg/blockchain/sol/artifacts/custody.json
cp target/deploy/custody.so   <sdk>/pkg/blockchain/sol/artifacts/custody.so
# then, in the SDK:
make generate
```

Review the `custody.json` + generated `../custody/*.go` diffs together.

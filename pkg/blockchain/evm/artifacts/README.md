# EVM contract artifacts

Vendored **ABI + deploy bytecode** for the EVM contracts this package binds —
`<Contract>.abi` (interface JSON) and, for deployable contracts,
`<Contract>.bin` (deploy bytecode hex). Interface-only bindings such as
`IConfig.abi` intentionally do not have a `.bin` file. These artifacts are the
source of truth for the generated `../*_abi.go` bindings: the `abi_refresher`
command reads them and emits the bindings, so binding regeneration needs
**no Solidity source, no forge, no jq** in this repo.

The `.abi` is the contract's wire interface — a change here is a reviewable
diff. The `.bin` is kept where deployment helpers are needed (clearnet's
devnet/tests and integration tests deploy contracts such as `Custody` and
`ConfigRegistry`); interface-only bindings are call/filter surfaces.

## Regenerate the bindings (common case)

After editing nothing but bumping the SDK, or after refreshing the files below:

```sh
make generate          # go generate ./...  → runs abi_refresher (+ cbor-gen)
```

or directly:

```sh
go generate ./pkg/blockchain/evm/...
```

This rewrites `../*_abi.go` from the `.abi`/`.bin` here. Commit the result.

## Refresh the .abi / .bin (only when a contract changes)

The files are produced by `forge build` in the repo that owns each contract's
Solidity source. The source trees are split three ways:

- **This repo** owns the two registry interfaces and the **config governance
  contracts**. They compile here, so refreshing `.abi` and `.bin` artifacts
  is a local `forge build`, with no other repo in the loop:
  - `contracts/src/interfaces/IClearnetRegistry.sol` (issuer read subset) and
    `IClearnetRegistryProtocol.sol` (clearing-side superset).
  - `contracts/src/ConfigRegistry.sol`, `contracts/src/Config.sol`, and their
    interfaces + `libraries/ConfigRegistryDigests.sol` — the per-issuer config
    governance surface every issuer implementation must satisfy.
- **custody** (`chains/evm/contract`) owns `Custody` (`src/Custody.sol`).
- **clearnet** (`contracts/evm`) owns the remaining clearing contracts —
  `YellowToken`, `MockERC20`, `NodeID`, `Faucet`, `Slasher`.

### Locally-built artifacts (no other repo)

```sh
cd contracts && forge build
jq '.abi' out/IClearnetRegistry.sol/IClearnetRegistry.json                 > ../pkg/blockchain/evm/artifacts/ClearnetRegistry.abi
jq '.abi' out/IClearnetRegistryProtocol.sol/IClearnetRegistryProtocol.json > ../pkg/blockchain/evm/artifacts/ClearnetRegistryProtocol.abi
cd .. && make generate
```

Refresh **both**, always. They are bound in one `abigen.Bind` call so that they
share a single Go `NodeRecord`, and abigen dedupes that struct on its canonical
type string — which carries no field names. A half-refresh therefore does not
fail the build; it silently takes the field names of whichever contract abigen
bound first. `noderecord_identity_test.go` and the forge CI job both exist to
catch that.

The config governance surface is refreshed the same way, from the same build:

```sh
cd contracts && forge build
jq -r '.abi'             out/ConfigRegistry.sol/ConfigRegistry.json > ../pkg/blockchain/evm/artifacts/ConfigRegistry.abi
jq -r '.bytecode.object' out/ConfigRegistry.sol/ConfigRegistry.json > ../pkg/blockchain/evm/artifacts/ConfigRegistry.bin
jq -r '.abi'             out/IConfig.sol/IConfig.json               > ../pkg/blockchain/evm/artifacts/IConfig.abi
jq -r '.abi'             out/Config.sol/Config.json                 > ../pkg/blockchain/evm/artifacts/Config.abi
jq -r '.bytecode.object' out/Config.sol/Config.json                 > ../pkg/blockchain/evm/artifacts/Config.bin
cd .. && make generate
```

### The other contracts (built in their owning repo)

Build each in its own tree, then extract abi + bytecode into this directory.
Foundry nests output by source-file name, so the contract→json mapping matters
(note YellowToken lives in `Token.sol`):

```sh
# Clearing contracts from clearnet; refresh Custody from
# ../custody/chains/evm/contract/out (same jq extraction, different OUT tree).
OUT=../clearnet/contracts/evm/out          # a `forge build` output tree
DEST=pkg/blockchain/evm/artifacts          # this directory, from repo root

while read name json; do
  jq -r '.abi'             "$OUT/$json" > "$DEST/$name.abi"
  jq -r '.bytecode.object' "$OUT/$json" > "$DEST/$name.bin"
done <<'EOF'
Slasher      Slasher.sol/Slasher.json
MockERC20    MockERC20.sol/MockERC20.json
Custody      Custody.sol/Custody.json
NodeID       NodeID.sol/NodeID.json
Faucet       Faucet.sol/Faucet.json
YellowToken  Token.sol/YellowToken.json
EOF

make generate   # regenerate bindings from the refreshed files
```

Review the resulting `.abi`/`.bin` and `*_abi.go` diffs together — an ABI
change without a corresponding intentional code change is a red flag.

## Validate release artifacts

Run `make check-evm-artifacts CUSTODY_SOURCE=/path/to/custody` against the checkout
recorded in `custody-source-revision`. When artifacts change, validate the SDK
artifact commit in custody CI, then update `custody-source-revision` and
`validated-sdk-revision` together to record that source and artifact pair.

Before publishing, run `check-release-artifacts.yml` manually; release tags rerun
it. It requires a successful custody parity run at the recorded source revision
and unchanged artifacts from the validated SDK revision. Configure
`CUSTODY_ACTIONS_READ_TOKEN` with Actions-read access to custody; the public SDK
workflow reads result metadata only. If the result is older than the latest 100
runs, rerun custody validation.

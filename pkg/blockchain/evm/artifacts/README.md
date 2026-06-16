# EVM contract artifacts

Vendored **ABI + deploy bytecode** for the EVM contracts this package binds —
`<Contract>.abi` (interface JSON) and `<Contract>.bin` (deploy bytecode hex).
They are the source of truth for the generated `../*_abi.go` bindings: the
`abi_refresher` command reads these files and emits the bindings, so binding
regeneration needs **no Solidity source, no forge, no jq** in this repo.

The `.abi` is the contract's wire interface — a change here is a reviewable
diff. The `.bin` is kept so `Deploy*` helpers work (clearnet's devnet/tests and
the integration tests deploy `Custody`); this package itself only calls/reads.

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

The files are produced by `forge build` in a repo that owns the Solidity
source (clearnet for Registry/YellowToken/etc.; custody for Custody). Build
there, then extract abi + bytecode into this directory. Foundry nests output
by source-file name, so the contract→json mapping matters (note YellowToken
lives in `Token.sol`):

```sh
OUT=../clearnet/contracts/evm/out          # a `forge build` output tree
DEST=pkg/blockchain/evm/artifacts          # this directory, from repo root

while read name json; do
  jq -r '.abi'             "$OUT/$json" > "$DEST/$name.abi"
  jq -r '.bytecode.object' "$OUT/$json" > "$DEST/$name.bin"
done <<'EOF'
Slasher      Slasher.sol/Slasher.json
Registry     Registry.sol/Registry.json
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

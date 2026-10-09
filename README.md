# clearnet-sdk

SDKs and shared protocol libraries for Clearnet integrations.

This repository currently contains:

- a Go module, `github.com/layer-3/clearnet-sdk`, with core protocol types,
  signing helpers, p2p helpers, and blockchain adapters;
- a TypeScript package under `sdk/ts`, published/imported as
  `@yellow-org/clearnet-sdk`;
- Docker-backed local devnet tooling for Go and TypeScript integration tests.

The Go SDK is the broader backend-facing SDK. The TypeScript SDK currently
focuses on browser and application deposit flows for EVM, Solana, XRPL, and Bitcoin.

## Where This Fits

The wider system is organized into two layers. The **clearing layer** decides
what settlement should happen, and orders and finalizes it. The **issuers
layer** custodies real assets and issues and discharges obligations against
them.

This repository holds the shared protocol surface both layers must satisfy:
the wire types, digests, and verifiers, plus the governance contracts every
issuer must conform to. There is currently one Yellow-specific implementation
of each layer — Clearnet for clearing, Custody for issuance — and the protocol
surface in `pkg/core` is deliberately implementation-agnostic.

The rest of the repository is not. `pkg/blockchain/evm`, `pkg/blockchain/sol`,
`pkg/blockchain/xrpl`, and `pkg/blockchain/btc` are Custody's own
implementation-specific code, hosted here alongside the shared surface — for
example `pkg/blockchain/evm/custody_abi.go` and `adjudicator_abi.go` are
generated bindings for Custody's own on-chain contracts, and
`pkg/blockchain/btc`'s tagged-script and finalizer types encode Custody's own
vault design.

The rule going forward, so this does not get worse than it stands today: new
implementation-specific code goes in **its own package**, **labelled as such
in its doc comment**, and **importing nothing from the shared protocol
surface**. The four packages above are marked in the table below as the
existing instances of it, and that labelling is what makes a later extraction
mechanical rather than a fresh audit.

> **Cross-repo note.** Extracting the implementation-specific packages out of
> this repository is the subject of a future Custody-SDK extraction decision.

## Repository Layout

| Path | Purpose |
|---|---|
| `pkg/core` | Shared Clearnet data types, operations, transaction references, deposit destinations, and adapter interfaces. |
| `pkg/blockchain/evm` | **Custody-implementation-specific.** Go EVM adapters for vault deposits, withdrawals, signer rotation, registry/faucet/token/fraud interactions, and generated contract bindings. |
| `pkg/blockchain/sol` | **Custody-implementation-specific.** Go Solana custody adapter code, program bindings, deposits, withdrawals, and signer rotation. |
| `pkg/blockchain/xrpl` | **Custody-implementation-specific.** Go XRPL deposits, withdrawals, signer rotation, ticket handling, and payment wire helpers. |
| `pkg/blockchain/btc` | **Custody-implementation-specific.** Go Bitcoin vault deposit, withdrawal, rotation, consolidation, and RPC helpers. |
| `pkg/decimal` | Decimal amount type used by Go chain adapters. |
| `pkg/bls`, `pkg/eip712`, `pkg/sign` | Signature and digest helpers. |
| `pkg/p2p`, `pkg/receipt`, `pkg/log` | Supporting networking, receipt, and logging packages. |
| `sdk/ts` | TypeScript SDK package, tests, and browser demos. See `sdk/ts/README.md`. |
| `devnet` | Docker Compose local blockchain devnet and readiness probe. See `devnet/README.md`. |
| `contracts` | Shared Solidity governance contracts, interfaces, and Foundry tests. |
| `internal` | Private Go helpers and integration suites. The EIP-712 deployment test has an isolated test module; see `internal/integration/eip712deployment/README.md`. |
| `scripts` | Artifact validation and release checks. |
| `testdata` | Shared conformance fixtures, including vectors used by both Go and TypeScript. |
| `docs` | Release guidance. See `docs/releasing.md`. |

## Go SDK

The Go module is rooted at this repository:

```sh
go get github.com/layer-3/clearnet-sdk
```

Common entry points:

- `pkg/core`: chain-neutral interfaces such as `VaultDepositor`,
  `VaultWithdrawalFinalizer`, `SignerRotationFinalizer`, `TxRef`, and
  `DepositDestination`.
- `pkg/blockchain/evm`: EVM custody vault flows and generated bindings.
- `pkg/blockchain/sol`: Solana custody vault flows.
- `pkg/blockchain/xrpl`: XRPL custody vault flows.
- `pkg/blockchain/btc`: Bitcoin custody vault flows.
- `pkg/finality`: finalized withdrawal verification. Pass the signing cluster
  size from `evm.ReadRegistryFinalityPolicy` to
  `finality.NewFinalizedWithdrawalVerifier` before verifying withdrawals.

Registry policy is required for every signing cluster, including K=1.
`evm.ReadRegistryFinalityPolicy(ctx, reader, registry, confirmations)` requires
`CONFIG()` to return a nonzero Registry-owned Config with finality epoch 1 and
a known K=1..256 checksum. Reads are pinned to one confirmed block hash and
rechecked for an anchor reorg; missing or invalid policy fails closed.
Use a non-upgradeable registry with a frozen Config address and policy row:
the reader verifies the advertised state, not future immutability.
`finality.NewFinalizedWithdrawalVerifier(checker, policy.SigningClusterSize)`
snapshots that trusted K. An unconfigured verifier rejects withdrawals; there
is no implicit quorum. Never select the expected K from an incoming withdrawal.

Run the Go checks:

```sh
make build
make lint
make test
```

Generated Go files are committed. Regenerate them after changing generation
inputs:

```sh
make generate
```

## TypeScript SDK

The TypeScript package lives in `sdk/ts` and is ESM-first.

```sh
cd sdk/ts
npm ci
npm run typecheck
npm test
npm run build
```

Install from an application:

```sh
npm install @yellow-org/clearnet-sdk
```

The package currently exposes vault depositors for:

- EVM native ETH and ERC-20 deposits;
- Solana native SOL and SPL token deposits;
- XRPL native XRP and issued-currency deposits;
- Bitcoin native BTC deposits.

Read the package guide and API examples in `sdk/ts/README.md`.

## Browser Demos

The TypeScript package includes local demo apps for manual wallet testing:

```sh
npm --prefix sdk/ts run demo:evm
npm --prefix sdk/ts run demo:sol
npm --prefix sdk/ts run demo:xrpl
npm --prefix sdk/ts run demo:btc
```

The demos expect a local or configured chain endpoint, funded wallet accounts,
and the chain-specific wallet/browser extension needed by the demo. They are
developer aids, not production app templates.

## Devnet And Integration Tests

The local devnet runs the chain nodes used by the integration suites:

```sh
make devnet
npm --prefix sdk/ts ci
make integration
make devnet-down
```

Focused targets are available when iterating on one chain:

```sh
make devnet-evm
npm --prefix sdk/ts run test:integration:evm

make devnet-sol
npm --prefix sdk/ts run test:integration:sol

make devnet-xrpl
npm --prefix sdk/ts run test:integration:xrpl

make devnet-btc
npm --prefix sdk/ts run test:integration:btc
```

`make integration` runs the Go blockchain and cross-component integrations,
the isolated EIP-712 deployment test, and the TypeScript EVM, Solana, XRPL, and
Bitcoin integration tests. See `devnet/README.md` for ports,
provisioning behavior, and environment overrides.

## Development Notes

- Use `make test` for the Go race-enabled test suite.
- Use `npm --prefix sdk/ts test` to discover all TypeScript unit and example
  tests. Integration tests run separately through the `test:integration:*` scripts.
- Use `npm --prefix sdk/ts audit --omit=dev --audit-level=moderate` when
  checking runtime dependency advisories for the TypeScript package.
- Keep generated files and vendored chain artifacts in sync with their source
  inputs.
- Release maintainers: see the [release guide](docs/releasing.md).
- Keep public SDK documentation broad: this repository supports Clearnet
  integration surfaces, not only custody-specific flows.

## License

MIT. See `LICENSE`.

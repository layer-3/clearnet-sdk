# clearnet-sdk

SDKs and shared protocol libraries for Clearnet integrations.

This repository currently contains:

- a Go module, `github.com/layer-3/clearnet-sdk`, with core protocol types,
  signing helpers, p2p helpers, and blockchain adapters;
- a TypeScript package under `sdk/ts`, published/imported as
  `@yellow-org/clearnet-sdk`;
- Docker-backed local devnet tooling for Go and TypeScript integration tests.

The Go SDK is the broader backend-facing SDK. The TypeScript SDK currently
focuses on browser and application deposit flows for EVM, Solana, and XRPL.

## Repository Layout

| Path | Purpose |
|---|---|
| `pkg/core` | Shared Clearnet data types, operations, transaction references, deposit destinations, and adapter interfaces. |
| `pkg/blockchain/evm` | Go EVM adapters for vault deposits, withdrawals, signer rotation, registry/faucet/token/fraud interactions, and generated contract bindings. |
| `pkg/blockchain/sol` | Go Solana custody adapter code, program bindings, deposits, withdrawals, and signer rotation. |
| `pkg/blockchain/xrpl` | Go XRPL deposits, withdrawals, signer rotation, ticket handling, and payment wire helpers. |
| `pkg/blockchain/btc` | Go Bitcoin vault deposit, withdrawal, rotation, consolidation, and RPC helpers. |
| `pkg/decimal` | Decimal amount type used by Go chain adapters. |
| `pkg/bls`, `pkg/eip712`, `pkg/sign` | Signature and digest helpers. |
| `pkg/p2p`, `pkg/receipt`, `pkg/log` | Supporting networking, receipt, and logging packages. |
| `sdk/ts` | TypeScript SDK package, tests, and browser demos. See `sdk/ts/README.md`. |
| `devnet` | Docker Compose local blockchain devnet and readiness probe. See `devnet/README.md`. |

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
- XRPL native XRP and issued-currency deposits.

Read the package guide and API examples in `sdk/ts/README.md`.

## Browser Demos

The TypeScript package includes local demo apps for manual wallet testing:

```sh
npm --prefix sdk/ts run demo:evm
npm --prefix sdk/ts run demo:sol
npm --prefix sdk/ts run demo:xrpl
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
```

`make integration` runs the Go blockchain integrations and the TypeScript EVM,
Solana, and XRPL integration tests. See `devnet/README.md` for ports,
provisioning behavior, and environment overrides.

## Development Notes

- Use `make test` for the Go race-enabled test suite.
- Use `npm --prefix sdk/ts test` for TypeScript unit tests.
- Use `npm --prefix sdk/ts audit --omit=dev --audit-level=moderate` when
  checking runtime dependency advisories for the TypeScript package.
- Keep generated files and vendored chain artifacts in sync with their source
  inputs.
- Keep public SDK documentation broad: this repository supports Clearnet
  integration surfaces, not only custody-specific flows.

## License

MIT. See `LICENSE`.

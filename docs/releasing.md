# Releasing the SDK

This guide is for release maintainers. SDK consumers should start with the
[main SDK README](../README.md) or the [TypeScript guide](../sdk/ts/README.md).

## GitHub / Go release

Before tagging, confirm normal SDK CI passes on the release commit: Go and
TypeScript tests, SDK-owned contract compilation, generated EVM bindings,
devnet integration, and shipped artifact checks. Run
`check-release-artifacts.yml` manually on that commit if needed. Tags beginning
with `v` run the artifact checks again. Point the release draft and tag at the
verified commit.

SDK releases do not require custody CI results, custody dependency pin updates,
or access to the private custody repository. Custody owns source-to-artifact
parity and compatibility checks for its contract implementations and SDK pins.
SDK checks validate artifact integrity and recorded source-revision fields;
they do not prove compilation from private custody sources.

## Artifact maintenance

`check-release-artifacts.yml` runs on pull requests, master pushes, release tags,
and manual dispatch. It validates ABI/bytecode formats, EVM artifact checksums,
recorded source revisions, and the Solana program checksum using this checkout
only. Run its checks locally with:

```sh
python3 scripts/test-check-release-artifacts.py
python3 scripts/check-release-artifacts.py .
```

When EVM artifacts change intentionally, regenerate their checksum manifest
from `pkg/blockchain/evm/artifacts` with
`shasum -a 256 *.abi *.bin > artifacts.sha256`, and review it alongside the
artifact and binding changes. Keep the Solana source revision and program
checksum in sync when refreshing that program. See the
[EVM artifact guide](../pkg/blockchain/evm/artifacts/README.md) and
[Solana artifact guide](../pkg/blockchain/sol/artifacts/README.md).

Developers with custody sources can run the optional
`make check-evm-artifacts CUSTODY_SOURCE=/path/to/custody` check against the
recorded source revision. This is not an SDK release prerequisite.

## TypeScript npm release

Pushing a GitHub tag does not publish the TypeScript package to npm. From
`sdk/ts`, verify the intended version and run:

```sh
npm ci
npm run typecheck
npm test
npm audit --audit-level=moderate
npm run check:package
npm run build --workspaces
npm publish --dry-run --access public
```

The package check builds and packs the SDK, installs it into a fresh app,
audits that app without overrides, and verifies dependency resolution, public
API types, Solana preparation/signing/serialization/simulation, and RPC calls.
CI runs these checks and builds all four demos.

The package bundles its Solana v1 client with patched JSON-RPC dependencies
because npm consumers do not inherit this SDK's overrides. Prepack builds the
SDK and prepares the bundled dependency manifest. Preserve this preparation
when publishing the package, and keep the public `solana` namespace compatible
with the transactions returned by the SDK.

A maintainer with npm publishing access can then publish the verified package.

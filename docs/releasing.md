# Releasing the SDK

This guide is for release maintainers. SDK consumers should start with the
[main SDK README](../README.md), the [CLI README](../cmd/yellow/README.md),
or the [TypeScript guide](../sdk/ts/README.md).

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

## CI triggers

Pull requests targeting `master` and pushes to `master` run the full SDK checks.
New commits to a pull request cancel its older runs. `v*` tags run the
standalone artifact workflow.

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
with the transactions returned by the SDK. npm 12.2.0 rejects packing this
existing override-plus-bundle arrangement (`EBUNDLEOVERRIDE`); the release
workflow uses Node 22 with npm 10. Do not bypass the newer guard or publish
with npm 12 until the bundled dependency graph has been migrated and the packed
consumer check passes under it.

A maintainer with npm publishing access can then publish the verified package.

## Yellow CLI release

Yellow is included in SDK releases starting with `v0.9.0`. Publish CLI binaries
from the verified SDK release tag and keep released tags immutable.

The `Yellow CLI` workflow runs public HTTP-fixture, encryption, wallet and recovery
tests, checks that `cmd/yellow` imports no private Clearnet, custody, blockchain
adapter or node p2p packages, and cross-compiles Linux/macOS/Windows amd64 and arm64
archives with SHA-256 checksums. PR and branch checks retain these as the
`yellow-cli` Actions artifact. Publishing the SDK release builds the exact tag and
attaches archives and `SHA256SUMS` to that release only after the Go, TypeScript,
web wallet, Forge, artifact and devnet integration workflows all pass on the
release tag.
Release publication must follow normal SDK validation; this workflow does not
create or move tags.

Before publishing, review `yellow help`, `yellow transfer send --help`,
`yellow operation retry --help`, `yellow shell --help`, and the [CLI README](../cmd/yellow/README.md). Confirm help has
only accounts, networks, balances, transfers, recent history, operation recovery
and CLI tooling. Build the CLI from the public module without private repository
access. Run the CLI checks from the repository root:

```sh
go test -race ./cmd/yellow ./internal/yellow ./pkg/wallet ./pkg/client
python3 scripts/test-yellow-terminal.py
./scripts/build-yellow.sh v0.9.0 /tmp/yellow-release
```

Test interactive prompts and platform binaries on their target operating systems,
including live suggestions, resizing, history, scrollback, secure input handoff and
Ctrl+C/Ctrl+D restoration. Confirm direct JSON commands remain suitable for scripts.
Verify private-key export requires unlock and confirmation, and keeps the revealed
key out of redirected stdout, stderr and JSON. Check private-key import preserves
the signing address and stores the key encrypted.
Each release archive includes the CLI README, license and the complete
`skills/yellow-network` agent skill. Review the skill's commands, JSON fields,
terminal handoffs and recovery rules against the release CLI when those contracts
change. Verify both archive formats retain `SKILL.md`, `agents/openai.yaml` and
the linked references. See [skill installation](../cmd/yellow/README.md#scripts-and-agents).

Previously submitted operation replay requires the original gateway's optional
`pendingUntil` status evidence. Gateways without it can still handle account,
balance, transfer, history and status commands, and sealed receipt reconciliation;
the CLI safely refuses replay of submitted uncertain authorizations.

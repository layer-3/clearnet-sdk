#!/usr/bin/env bash
# The caller checks out the owning custody source; no network or credentials here.
set -euo pipefail
sdk_root=$(cd "$(dirname "$0")/.." && pwd)
custody_root=$(cd "${1:?usage: check-evm-artifacts.sh PATH_TO_CUSTODY}" && pwd)
expected=$(cat "$sdk_root/pkg/blockchain/evm/artifacts/custody-source-revision")
[[ "$expected" =~ ^[0-9a-f]{40}$ ]] || { echo 'invalid custody source revision'; exit 1; }
actual=$(git -C "$custody_root" rev-parse HEAD)
[ "$actual" = "$expected" ] || { echo "custody source revision mismatch: expected $expected, got $actual"; exit 1; }
# Bytecode/ABI parity is custody's canonical check: it recompiles both
# owners' Solidity under custody's Foundry profile and diffs the result
# against the SDK's vendored artifacts.
"$custody_root/scripts/check-sdk-artifact-parity.sh" "$sdk_root"

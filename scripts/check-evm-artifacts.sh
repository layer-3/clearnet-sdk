#!/usr/bin/env bash
# The caller checks out the owning custody source; no network or credentials here.
set -euo pipefail
sdk_root=$(cd "$(dirname "$0")/.." && pwd)
custody_root=$(cd "${1:?usage: check-evm-artifacts.sh PATH_TO_CUSTODY}" && pwd)
expected=$(cat "$sdk_root/pkg/blockchain/evm/artifacts/custody-source-revision")
[[ "$expected" =~ ^[0-9a-f]{40}$ ]] || { echo 'invalid custody source revision'; exit 1; }
actual=$(git -C "$custody_root" rev-parse HEAD)
[ "$actual" = "$expected" ] || { echo "custody source revision mismatch: expected $expected, got $actual"; exit 1; }
validated=$(cat "$sdk_root/pkg/blockchain/evm/artifacts/validated-sdk-revision")
[ "$validated" = "$(cat "$custody_root/.github/eip712-sdk-revision")" ] || { echo 'validated SDK revision disagrees with custody CI'; exit 1; }
"$custody_root/scripts/check-evm-eip712-local.sh" "$sdk_root"

#!/usr/bin/env bash
# Test the actual gateway handlers against the CLI, TS SDK and web price math.
set -euo pipefail
sdk_root="$(cd "$(dirname "$0")/.." && pwd)"
gateway_root="${CLEARNET_SOURCE:-$sdk_root/../clearnet}"
if [[ ! -f "$gateway_root/gateway/public_sdk_contract_test.go" ]]; then
  echo 'Set CLEARNET_SOURCE to the companion Clearnet checkout.' >&2
  exit 1
fi
gateway_root="$(cd "$gateway_root" && pwd)"
contract_work="$(mktemp -d)"
trap 'rm -rf "$contract_work"' EXIT
cp "$gateway_root/go.mod" "$contract_work/go.mod"
cp "$gateway_root/go.sum" "$contract_work/go.sum"
GOWORK=off go mod edit -modfile "$contract_work/go.mod" -replace "github.com/layer-3/clearnet-sdk=$sdk_root"
npm --prefix "$sdk_root/sdk/ts" run build
cd "$gateway_root"
GOWORK=off CLEARNET_SDK_CONTRACT_DIR="$sdk_root" go test -mod=mod -modfile "$contract_work/go.mod" -race ./gateway -run '^TestPublicSDKContract_' -count=1 -v

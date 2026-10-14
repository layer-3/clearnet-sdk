#!/usr/bin/env bash
# Build public CLI archives without any consumer repository.
set -euo pipefail
cd "$(dirname "$0")/.."
yellow_version="${1:-dev}"
yellow_output="${2:-dist/yellow}"
[[ "$yellow_version" =~ ^[a-zA-Z0-9._+-]+$ ]] || { echo "Invalid release version" >&2; exit 1; }
mkdir -p "$yellow_output"
yellow_output="$(cd "$yellow_output" && pwd)"

if go list -deps ./cmd/yellow | grep -E '^github.com/layer-3/(clearnet/|custody/|clearnet-sdk/pkg/(blockchain|p2p)/)'; then
  echo 'yellow must only use the public user gateway API' >&2
  exit 1
fi

yellow_work="$(mktemp -d)"
trap 'rm -rf "$yellow_work"' EXIT
for yellow_platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  yellow_os="${yellow_platform%/*}"
  yellow_arch="${yellow_platform#*/}"
  yellow_name="yellow_${yellow_version}_${yellow_os}_${yellow_arch}"
  mkdir -p "$yellow_work/$yellow_name"
  yellow_binary=yellow
  if [ "$yellow_os" = windows ]; then yellow_binary=yellow.exe; fi
  CGO_ENABLED=0 GOOS="$yellow_os" GOARCH="$yellow_arch" go build -trimpath \
    -ldflags "-s -w -X main.version=$yellow_version" \
    -o "$yellow_work/$yellow_name/$yellow_binary" ./cmd/yellow
  cp LICENSE cmd/yellow/README.md "$yellow_work/$yellow_name/"
  if [ "$yellow_os" = windows ]; then
    (cd "$yellow_work/$yellow_name" && zip -q "$yellow_output/$yellow_name.zip" "$yellow_binary" LICENSE README.md)
  else
    tar -czf "$yellow_output/$yellow_name.tar.gz" -C "$yellow_work/$yellow_name" "$yellow_binary" LICENSE README.md
  fi
  echo "Built $yellow_name"
done
(cd "$yellow_output" && shasum -a 256 yellow_*.tar.gz yellow_*.zip > SHA256SUMS)

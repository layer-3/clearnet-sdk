.PHONY: build lint test generate devnet devnet-down integration

build:
	go build ./...

lint:
	go vet ./...

test:
	go test -race ./...

# Regenerate all generated code:
#   - pkg/blockchain/evm/*_abi.go from the vendored ABI/bytecode in
#     pkg/blockchain/evm/artifacts (see that dir's README to refresh them)
#   - pkg/core/cbor_gen.go from pkg/core/gen
# Requires go-ethereum's abigen logic (vendored via the module — no external
# binary needed); commit the result.
generate:
	go generate ./...

# Bring the local devnet up and block until every node answers RPC.
devnet:
	docker compose -f devnet/docker-compose.yml up -d
	go run ./devnet/wait

devnet-down:
	docker compose -f devnet/docker-compose.yml down -v

# Build-tagged blockchain flow tests (deposit + withdrawal per chain). Every
# test self-provisions against the devnet — no setup, no env. See devnet/README.md.
integration:
	go test -tags integration ./pkg/blockchain/... -v

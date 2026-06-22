.PHONY: build lint test generate devnet devnet-evm devnet-down ts-deps integration

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

devnet-evm:
	docker compose -f devnet/docker-compose.yml up -d anvil
	go run ./devnet/wait --networks anvil

devnet-down:
	docker compose -f devnet/docker-compose.yml down -v

ts-deps:
	npm --prefix sdk/ts ci

# Blockchain flow tests against the devnet. Go tests cover deposit + withdrawal
# per chain; the TS suite covers EVM deposits. See devnet/README.md.
integration: ts-deps
	go test -tags integration ./pkg/blockchain/... -v
	npm --prefix sdk/ts run test:integration:evm

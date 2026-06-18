.PHONY: build lint test generate devnet devnet-evm devnet-down integration

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
	@for i in $$(seq 1 60); do \
		if curl -fsS -H 'content-type: application/json' \
			--data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
			http://127.0.0.1:8545 >/dev/null; then \
			echo "anvil ready"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "anvil did not become ready" >&2; \
	exit 1

devnet-down:
	docker compose -f devnet/docker-compose.yml down -v

# Build-tagged blockchain flow tests (deposit + withdrawal per chain). Every
# test self-provisions against the devnet — no setup, no env. See devnet/README.md.
integration:
	go test -tags integration ./pkg/blockchain/... -v

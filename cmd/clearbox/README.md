# Clearbox: Clearnet development gateway

Clearbox is a lightweight Go simulator for building and testing Clearnet
integrations. Use it with the Yellow CLI, TypeScript client or web app to create
accounts, transfer test YELLOW and inspect balance changes without external
services or network credentials. State persists in a local SQLite database.

```sh
# From the clearnet-sdk repository root:
go run ./cmd/clearbox
# Equivalent: make clearbox
```

Default gateway: `http://127.0.0.1:8080`. Network: `YDEV`. Wallet/EIP-712 chain ID:
`1497646422` (`0x59444556`). Wallet RPC: `http://127.0.0.1:8080/rpc`.

## First launch and funding

On the **first launch with a new database**, clearbox generates an Ethereum
account, seeds it with **1,000,000 YELLOW only**, and prints its address and
private key in the startup log. There is no predefined account or key. ETH and
USDC remain in the asset catalogue for interface compatibility but start at zero.
New user accounts also start at zero; fund them by transferring YELLOW from the
newly generated seed account.

**The printed key is for DEVELOPMENT ONLY. Never reuse it for real funds,
real networks or production wallets.** Import it only into a dedicated development
wallet/CLI account. The source and initialization log carry the same warning.
The key is stored in the local development database and is never exposed by HTTP.
Do not commit the database or its initialization log.

The default database is `.clearbox/clearbox.db` (git-ignored). Its directory and
file are created owner-only. **Restarts reuse the same account and ledger;
no funds are added again.** Balances, nonces, operations, history, block hashes
and pending deliveries survive restart. The private key is printed only during
initialization; import it then into your local wallet. A separate database starts
a separate development network state:

```sh
go run ./cmd/clearbox -db /tmp/my-clearbox.db -listen 127.0.0.1:8081
```

Only one process may own a database. A corrupt or incompatible checkpoint causes
startup to fail; it is never silently replaced or reseeded.

## Yellow CLI

Build the CLI and review a local network:

```sh
go build -o /tmp/yellow ./cmd/yellow
/tmp/yellow network add local --gateway http://127.0.0.1:8080 --network-id YDEV
/tmp/yellow account import clearbox-seed
```

At import choose `key`, enter the generated key from the first startup log at the
hidden prompt, and choose an encryption passphrase. The CLI retains its normal
network trust, encryption and transfer approvals. Explicitly select the local
network/account for development commands:

```sh
/tmp/yellow --network local --account clearbox-seed balance
/tmp/yellow account create clearbox-user
/tmp/yellow account address clearbox-user

# Replace <recipient-address> with that new account's address:
/tmp/yellow --network local --account clearbox-seed transfer send <recipient-address> \
  --asset YELLOW=25 --max-fee 1 --wait 5s

/tmp/yellow --network local --account clearbox-user balance

# Replace <seed-address> with the generated address printed at startup:
/tmp/yellow --network local --account clearbox-user transfer send <seed-address> \
  --asset YELLOW=2 --max-fee 0.1 --wait 5s
/tmp/yellow --network local --account clearbox-user transfer history
```

Users must retain at least 10 YELLOW after principal and fees. A sender seal and
recipient credit are separate steps; live balance determines spendability.

## Web app

Leave clearbox running, then in another terminal:

```sh
cd web
pnpm install
pnpm dev
```

Open `http://localhost:3000`, import the generated seed key into a dedicated
browser-wallet account, and connect. The default **Local** network points at
clearbox through `/api/gw/local`, including wallet RPC. Approve the local Yellow
chain switch when signing. The imported seed identity has the same balances and
nonce as in the CLI. Transfers made in either client appear in the other's
balance/history reads.

For a different port configure `NEXT_PUBLIC_YELLOW_NETWORKS` and the CLI profile
accordingly. The normal TypeScript `GatewayClient` also works directly with
`baseUrl: "http://127.0.0.1:8080", networkId: "YDEV"`.

## Configuration

Defaults work for local development. Use launch flags to adjust timing, state
and request limits:

| Flag | Default | Purpose |
|---|---|---|
| `-listen` | `127.0.0.1:8080` | HTTP listen address; loopback-only unless `-public-url` is set. |
| `-db` | `.clearbox/clearbox.db` | SQLite state file, initialized and seeded only once. |
| `-seal-delay` | `750ms` | Delay from transfer acceptance to sender seal (`0` to `1m`). |
| `-delivery-delay` | `500ms` | Delay from sender seal to recipient credit (`0` to `1m`). |
| `-max-operations` | `2000` | Maximum retained operations. |
| `-public-url` | Unset | Optional external HTTPS origin for wallet RPC metadata and cross-origin browser access. |
| `-rate-limit` | `10` | Requests per second per client IP when `-public-url` is set; burst is twice this value. |
| `-trusted-proxies` | Unset | Proxy CIDRs allowed to supply client IPs through `X-Forwarded-For`. |

For instant local tests:

```sh
go run ./cmd/clearbox -seal-delay 0 -delivery-delay 0
```

Increase `-seal-delay` to test pending operations and polling. When `-public-url`
is set, it also spaces new transfer admissions by that interval; with zero delay,
new transfers are limited to 2/sec with a burst of 4. Reads and identical replays
are not delayed. This mode permits browser requests from any origin without
cookies and limits total requests to 50/sec with a burst of 100. Only trust proxy
CIDRs you control; forwarding headers from other callers are ignored.

Clearbox allows 16 in-flight requests, request bodies up to 64 KiB and RPC batches
of up to 100 calls. Rate/busy responses return `429` with `Retry-After` before
accepting a write. At the operation limit, new transfers return `429`; existing
balances, history, status and identical replays remain available. Increase
`-max-operations` and restart to keep working with the same ledger. State is never
automatically reset or reseeded.

`GET /healthz` reports readiness and operation usage/capacity. It is exempt from
request limits and returns `503` after a storage failure. Every response includes
`X-Clearbox-Simulated: true`.

## Interface coverage

The complete gateway surface currently used by CLI/web account and transfer
flows is implemented with the public gateway's wire schemas and error envelope:

| Endpoint | Behaviour |
|---|---|
| `GET /v1/gateway/info` | YDEV identity, router, assets/decimals, capabilities and wallet RPC URL |
| `GET /v1/accounts/{address}/balances` | Base-unit balances, nonce and account-code projections |
| `GET /v1/accounts/{address}/transfers` | Stable history, limit 1–100, account-bound opaque cursors, 30-day retention |
| `GET /v1/pools` | Static ETH/YELLOW and USDC/YELLOW valuation fixtures |
| `POST /v1/transfers` | Real EIP-712 recovery, asset/uint256 validation, fee cap, nonce ownership and balance checks |
| `GET /v1/operations/{id}` | Pending/sealed status and stable receipt, or unknown |
| `POST /rpc` | Wallet chain discovery and balance/nonce/block reads; batches, errors and notifications |
| `GET /healthz` | Readiness and explicit simulation marker |

Clearbox reuses `pkg/eip712`, `pkg/client`, `pkg/core` and `pkg/decimal`, including
canonical transfer/block encodings, hashes and operation IDs. SQLite keeps
development state across restarts.

Admission reserves funds and a nonce atomically. Duplicate submissions return
the same operation; a different intent at that signer/nonce conflicts. Every
mutation is committed to SQLite before releasing a response. Pending work
resumes from stored event times after restart. The small simulator uses one
atomic SQLite checkpoint for its state, not production's database schema.

Fees are 0.10% of each leg's YELLOW-equivalent value, rounded down, with a
0.0001 YELLOW minimum per leg. Zero `maxFee` authorizes zero fees. Users must
hold and retain at least 10 YELLOW. Rejections do not consume funds or a nonce.

Default timing is deterministic: `202 pending`, seal after 750 ms, recipient
credit 500 ms later. Reads materialize due events. Replay after sealing returns
the same `200` receipt.

History matches `out`/`sender_sealed`, `in`/`not_verified`, and `self` semantics;
incoming history does not assert recipient spendability.

## Simulation boundary

This is account/transfer API compatibility, not a full distributed network.
Consensus, BLS attestations, P2P, registry policy, security ceilings, distributed
recovery, AMM execution, treasury/pool transfers, custody deposits and withdrawals
are not simulated. Pool prices are fixed fixtures. Blocks use real SDK encodings
but no validator attestation; `stateRoot` is a local snapshot hash, not a
production sparse Merkle root or finality proof.

Wallet RPC supports `eth_chainId`, `net_version`, `web3_clientVersion`,
`net_listening`, `net_peerCount`, `eth_syncing`, `eth_accounts` (empty),
`eth_blockNumber`, `eth_gasPrice`, `eth_maxPriorityFeePerGas`, `eth_getBalance`,
`eth_getTransactionCount`, `eth_getCode`, `eth_getBlockByNumber` and
`eth_getBlockByHash`.
Account-state reads support `latest` and `pending`. As in Clearnet, block numbers
are **account-scoped**: `eth_blockNumber` takes `[account]`, and
`eth_getBlockByNumber` takes `[account, tag, fullTransactions]`. Accounts may be
addresses or canonical user URIs. Sequences start at 1; an account with no sealed
blocks returns count 0 and a `null` block. Block responses carry the actual SDK
entries, and `eth_getBlockByHash` resolves the same sealed block.

TCP clients, WebSocket subscriptions, virtual-router `eth_call`, transaction/log
queries and Registry/L1 forwarding are outside this simulator's interface.
Unsupported REST routes return 404; unsupported RPC methods return `-32601`.
It does not fabricate EVM execution or historical account state.

## Checks

```sh
make test-clearbox
cd web && pnpm test && pnpm typecheck && pnpm lint

# With clearbox and the web app running, use ONLY the generated local test key:
cd web
CLEARBOX_PRIVATE_KEY=<key-from-initialization> BASE_URL=http://localhost:3000 pnpm shots
```

Go tests cover signatures, concurrent spending, fees/floors, cursors, error/RPC
handling, SQLite restart recovery and commit failures, plus the real CLI account,
trust and operation-journal workflow. TypeScript tests launch an isolated Go
clearbox and sign transfers with viem. Browser smoke tests send YELLOW through
the web relay and check both participants' live balances on the Go server.

Building clearbox requires Go and a C compiler for the SQLite driver (as in real
Clearnet). Tests use isolated databases and accounts.

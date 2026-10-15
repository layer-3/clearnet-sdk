# Yellow web app

The Yellow Network web wallet: connect a wallet, see balances (in YELLOW and per
token), send multi-asset transfers, and follow transfer history. It is the web
counterpart of the `yellow` CLI and talks to the network only through the
TypeScript SDK's gateway client (`@yellow-org/clearnet-sdk/gateway`, linked from
`../sdk/ts`).

Stack: Next.js (App Router), Tailwind CSS 4, wagmi + viem, Reown AppKit,
TanStack Query.

## Run locally

```sh
# In another terminal, from the repository root:
go run ./cmd/clearbox

# Then start the web app:
cd web
pnpm install          # Node 22+, pnpm 10 (or: npx pnpm@10 install)
pnpm dev              # builds ../sdk/ts first, then http://localhost:3000
```

With no configuration the app uses **Local**, the standalone Go clearbox at
`http://127.0.0.1:8080`, through the site's `/api/gw/local` relay. The CLI and web
app share its balances, nonces and history. New accounts start empty; import the
generated key printed on clearbox's first initialization into a dedicated wallet
account to use the seeded YELLOW, or receive funds from it. See [clearbox setup,
test-key precautions and interface coverage](../cmd/clearbox/README.md).
Clearbox persists its ledger and generated account in SQLite across restarts.

## Configuration

Networks are fixed when the site is built. Users never enter gateway URLs.

| Variable | Purpose |
|---|---|
| `NEXT_PUBLIC_YELLOW_NETWORKS` | JSON array of networks. See below. Defaults to local clearbox. |
| `NEXT_PUBLIC_YELLOW_DEFAULT_NETWORK` | Slug selected on first visit. Defaults to the first entry. |
| `NEXT_PUBLIC_REOWN_PROJECT_ID` | Enables Reown AppKit (WalletConnect, mobile wallets, 500+ wallets). Without it only injected browser wallets are offered. |
| `NEXT_PUBLIC_SITE_URL` | Public origin, used for wallet metadata. |

Each network entry:

```json
{
  "slug": "testnet",
  "name": "Testnet",
  "gateway": "https://gateway.testnet.yellow.org",
  "networkId": "YTES",
  "proxy": true,
  "fingerprint": "<sha256 from `yellow network show`>",
  "rpcUrl": "https://gateway.testnet.yellow.org/rpc",
  "explorer": "https://explorer.example"
}
```

- `gateway` is the gateway base URL. Local development uses `http://127.0.0.1:8080`;
  set this explicitly when connecting to another network.
- `proxy: true` routes browser requests through this site's `/api/gw/<slug>`
  relay, for gateways that don't send CORS headers. The relay only forwards
  gateway routes and never follows redirects.
- `fingerprint` pins the network identity exactly like the CLI's trust check.
  If the gateway's identity changes, signing stops until you review and update the
  pinned fingerprint, then rebuild the app.
- `rpcUrl` is the JSON-RPC URL wallets get when adding the Yellow chain
  (defaults to the site's relay when proxied, otherwise `<gateway>/rpc`). MetaMask only signs typed data for the active
  chain, so the app adds/switches to the Yellow chain before asking to sign.

See `.env.example`. Invalid configuration fails the production build.

## How sending stays safe

The web app follows the CLI's operation journal rules (exported by the SDK):

1. The signed transfer is saved in IndexedDB before anything is sent, and
   marked "possibly submitted" before the POST.
2. It is posted exactly once. A timeout is "outcome unknown", never a reason
   to sign again at a new nonce.
3. New sends from an account wait until its earlier transfer is settled.
   A cross-tab lock stops two tabs from signing at once.
4. Open transfers are reconciled in the background. "Send saved transfer"
   replays the identical signature only when the gateway confirms its window
   closed without a seal.

## Scripts

```sh
pnpm test        # unit tests
pnpm lint        # ESLint (Next + React Compiler rules)
pnpm typecheck
pnpm build
# End-to-end smoke test + screenshots (injects the generated local test wallet; start clearbox first):
pnpm build && pnpm start -p 3200 &
CLEARBOX_PRIVATE_KEY=<key-from-clearbox-initialization> BASE_URL=http://localhost:3200 pnpm shots
```

# XRPL Deposit Demo

This browser demo exercises `XrplVaultDepositor` against the local standalone
XRPL devnet. It supports two signer paths:

- a local XRPL wallet generated in the browser for the fastest local smoke test
- GemWallet signing through a custom WSS endpoint that points at the same local
  chain

The local devnet uses `network_id: 31337`. Do not change it to `21337` or
`21338`: those IDs are used by Xahau mainnet and testnet, and wallets may treat
the local chain as a Xahau network if those IDs are reused.

## Start The Demo

From the repository root:

```sh
make devnet-xrpl
```

From `sdk/ts`:

```sh
npm run demo:xrpl
```

Open `http://127.0.0.1:5173/`.

The default fields are for the local devnet:

| Field | Default | Purpose |
|---|---|---|
| WebSocket URL | `ws://127.0.0.1:6006` | XRPL WebSocket URL used by the SDK. |
| Admin HTTP URL | `/xrpl-admin` | Vite proxy to `http://127.0.0.1:5005` for `ledger_accept`. |
| Vault Address | `rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh` | Standalone genesis account used as the demo vault. |
| Fund Drops | `1000000000` | Amount sent from the standalone genesis wallet to the selected signer. |

## Local Signer Flow

Use this path first when checking that the devnet, SDK, and demo are working.
It does not require a browser wallet.

1. Leave `WebSocket URL` as `ws://127.0.0.1:6006`.
2. Click `Use Local Signer`.
3. Click `Fund Wallet`.
4. Click `Submit Deposit`.
5. Click `Verify Last Tx`.

Expected result:

```text
status: confirmed
```

If the `Local Wallet Seed` field is blank, the demo generates a fresh local
wallet and writes the seed back into the field. Reuse that seed if you need to
repeat the same local-wallet test after a page reload.

## GemWallet Flow

GemWallet custom networks require a `wss://` endpoint. The local rippled
container exposes raw WebSocket at `ws://127.0.0.1:6006`, so the wallet cannot
use that URL directly. Put a trusted WSS tunnel or TLS reverse proxy in front of
the local WebSocket port.

One local option is ngrok:

```sh
ngrok http 6006
```

If ngrok prints `https://example.ngrok-free.app`, use
`wss://example.ngrok-free.app` as the wallet and demo WebSocket URL.

Then:

1. In GemWallet, add a custom network for the WSS endpoint.
2. Select that custom network in GemWallet.
3. In the demo, set `WebSocket URL` to the same WSS endpoint.
4. Keep `Admin HTTP URL` as `/xrpl-admin`.
5. Click `Connect GemWallet` and approve the address request.
6. Click `Fund Wallet`.
7. Click `Submit Deposit` and approve the signing request in GemWallet.
8. Click `Verify Last Tx`.

Expected result:

```text
status: confirmed
```

The demo checks GemWallet's selected WebSocket endpoint before signing. If the
wallet is still on Xahau testnet or another network, the demo reports both
network IDs and stops before opening the signing request.

## What Went Wrong During Setup

There were three separate issues:

1. The first local chain ID collided with Xahau. `21337` and `21338` are public
   Xahau IDs, so the local standalone chain now uses `31337`.
2. GemWallet would not connect to `ws://127.0.0.1:6006` as a custom network.
   The wallet-facing endpoint must be `wss://`, so the demo needs a WSS tunnel
   or TLS proxy for the GemWallet path.
3. GemWallet 3.8.2 failed to render its signing review for prepared
   transactions that already included the custom `NetworkID: 31337` field.

The demo handles the third issue by verifying that GemWallet and the demo are
pointed at endpoints with the same `network_id`, then omitting `NetworkID` only
from the transaction object passed into `signTransaction()`. GemWallet autofills
`NetworkID` from its selected custom endpoint before signing. The submitted
transaction is still expected to include `NetworkID: 31337`; verify this through
`tx` lookup if the wallet flow is changed.

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---|---|---|
| `GemWallet is on XAHAU Testnet ... The demo RPC is ... 31337` | GemWallet is not on the local custom network. | Select the custom WSS network in GemWallet and set the demo `WebSocket URL` to the same WSS URL. |
| GemWallet custom network form rejects the URL | The endpoint is `ws://`, not `wss://`. | Put ngrok, Caddy, or another trusted TLS proxy in front of `127.0.0.1:6006`. |
| `GemWallet address approval timed out` | The extension did not return an address approval result. | Reopen GemWallet, unlock it if needed, reload the demo, and retry `Connect GemWallet`. |
| `status: pending` immediately after submit | The transaction is accepted but not in a validated ledger yet. | The demo calls `ledger_accept`; click `Verify Last Tx` again if needed. |
| `actNotFound` or account lookup failure | The signer account is not funded on the local standalone ledger. | Click `Fund Wallet`, then submit again. |
| GemWallet signing window shows an error before approval | The wallet path may be receiving a prepared transaction shape GemWallet cannot render. | Confirm the demo is using the current code path that removes `NetworkID` only for GemWallet signing, then retry after reloading the page. |

## Local-Only Notes

The `Fund Wallet` button and `ledger_accept` call are for the repository's local
standalone devnet. They are not public XRPL or Xahau testnet flows. On a public
network, fund the wallet through that network's faucet or normal account
funding process and remove the standalone admin assumptions.

The page reuses the active `XrplVaultDepositor` for `Verify Last Tx` and closes
that WebSocket connection when the signer is replaced or the page is unloaded.
Changing the signer, RPC URL, or vault address means submitting again before
verifying.

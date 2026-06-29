# Bitcoin Deposit Demo

Browser smoke demo for native BTC deposits against the local Bitcoin Core
regtest node.

## Run

From the repository root:

```sh
make devnet-btc
npm --prefix sdk/ts ci
npm --prefix sdk/ts run build
npm --prefix sdk/ts run demo:btc
```

Open the Vite URL, usually `http://127.0.0.1:5173/`.

The demo talks to Bitcoin Core through the Vite proxy at `/btc-rpc`. Browser
code does not include the Bitcoin Core username or password; the proxy injects
Basic Auth before forwarding to `http://127.0.0.1:18443`.

## Local signer flow

1. Click **Generate Keys** to create a local depositor key and three vault
   public keys.
2. Click **Fund Local** to create/load the `sdk` wallet, mine spendable regtest
   funds, import the generated addresses, and fund the depositor P2WPKH address.
3. Click **Submit Local** to spend from the depositor address into the
   per-account P2WSH deposit address.
4. Click **Verify Last Tx** before mining to observe `pending`.
5. Click **Mine Block**, then **Verify Last Tx** again to observe `confirmed`.

## Xverse PSBT flow

The Xverse path exercises browser-wallet signing without putting wallet-specific
code in the SDK. The demo asks Xverse to add/switch to the local regtest network,
prepares an unsigned PSBT, asks Xverse to sign the selected inputs, finalizes the
signed PSBT locally, and broadcasts it through the same `/btc-rpc` proxy.

Xverse uses the configured custom-network URL as an Electrs-style API while
rendering signing previews. The Vite demo server therefore exposes a small
Electrs-compatible facade under `/btc-rpc` for reads such as
`GET /address/:address/utxo`, backed by Bitcoin Core JSON-RPC. Browser SDK calls
still use JSON-RPC `POST /btc-rpc`, and the proxy keeps Bitcoin Core credentials
server-side.

1. Install Xverse and unlock a Bitcoin account.
2. Confirm **Xverse Electrs URL** points at the running Vite proxy. The default
   is `http://127.0.0.1:5174/btc-rpc`; update the port if Vite uses another one.
3. Click **Add/Switch Xverse Network** and approve the Xverse prompts.
4. Click **Connect Xverse**. The demo requests the payment address on `Regtest`
   and accepts `p2wpkh` or nested-SegWit `p2sh` payment addresses.
5. Click **Fund Xverse** to import and fund the returned address from the local
   `sdk` wallet.
6. Click **Submit Xverse** to prepare the PSBT, approve the Xverse signing
   request, finalize, and broadcast.
7. Use **Verify Last Tx**, **Mine Block**, and **Verify Last Tx** again to check
   `pending` then `confirmed`.

If Xverse cannot add or reach the local RPC URL, use a reachable HTTP(S) tunnel
for the Vite proxy or use the local signer flow. The local signer path remains
the deterministic regtest fallback.

## Defaults

| Field | Default |
|---|---|
| RPC URL | `/btc-rpc` |
| Wallet Name | `sdk` |
| Network | `regtest` |
| Xverse Network Name | `clearnet-regtest` |
| Xverse Electrs URL | `http://127.0.0.1:5174/btc-rpc` |
| Xverse Network | `Regtest` |
| Fund Sats | `100000000` |
| Fallback Fee Sat/VB | `5` |
| Amount Sats | `20000000` |

Override the backend target with `BTC_RPC_URL`, `BTC_RPC_USER`, or
`BTC_RPC_PASS` when starting Vite.

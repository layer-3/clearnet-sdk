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

Open the Vite URL printed by Vite, usually `http://127.0.0.1:5173/`.

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
2. Confirm **Xverse Electrs URL** points at the running Vite proxy. The page
   derives the default from its own origin, such as
   `http://127.0.0.1:5173/btc-rpc`.
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

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| **Add/Switch Xverse Network** opens a prompt, then the demo logs `Access denied`. | The network request was rejected in Xverse, or a previous prompt was still open. | Approve the Xverse prompt. If a stale popup is open, close it and click **Add/Switch Xverse Network** again. The demo only calls `wallet_addNetwork` with `switch: true`; it does not require a separate network-switch request. |
| **Connect Xverse** disables briefly but does not fill the address fields. | Xverse did not return a payment account, the wallet is locked, or the request was rejected. | Unlock Xverse, make sure the custom regtest network is selected, then click **Connect Xverse** again and approve the connection prompt. The demo requests a payment account with `getAccounts` and falls back to `getAddresses` only when that method is unavailable. |
| The log says `Xverse getAccounts failed: Access denied`. | Xverse rejected the account-access request. | Re-run **Connect Xverse** and approve the prompt. If no prompt appears, open Xverse directly, unlock it, confirm the active account, then retry from the demo tab. |
| **Fund Xverse** says to connect first. | The demo has not stored an Xverse payment address yet. | Complete **Add/Switch Xverse Network** and **Connect Xverse** before funding. |
| **Fund Xverse** rejects the returned address. | The active Xverse account is not returning a regtest-compatible payment address. | Use an Xverse Bitcoin payment address on the custom regtest network. The demo accepts `bcrt1...` Native SegWit (`p2wpkh`) and `2...` nested-SegWit (`p2sh`) addresses. |
| Xverse shows `Transaction Error` with `404` for `/address/<address>/utxo` while signing. | The Xverse custom-network URL points at plain Bitcoin Core JSON-RPC or the wrong Vite port. Xverse needs Electrs-style read endpoints while rendering the PSBT review. | Set **Xverse Electrs URL** to the running Vite server's `/btc-rpc` path, for example `http://127.0.0.1:5173/btc-rpc`. Restart the demo after Vite config changes. |
| **Submit Xverse** logs `PSBT has no inputs for Xverse to sign`. | The funded UTXO is not visible from the wallet/funding address selected in the demo. | Click **Fund Xverse** again, confirm it reports at least one UTXO, then retry **Submit Xverse**. |
| **Submit Xverse** signs but broadcast or verify fails. | The transaction may still be in the mempool, the local regtest node has not mined a block, or the demo is pointed at a different Bitcoin Core wallet/node. | Click **Verify Last Tx** before mining to check for `pending`, then click **Mine Block** and verify again. Confirm **RPC URL** and **Xverse Electrs URL** point at the same Vite proxy/node. |
| Xverse cannot reach `127.0.0.1` from the extension environment. | Browser extension networking or custom-network policy is blocking the local URL. | Use a reachable HTTP(S) tunnel to the Vite proxy and put that tunnel URL in **Xverse Electrs URL**, or use the local signer flow. |
| Leather shows local networks as disabled or returns a non-regtest address. | Leather's local network entries are not usable for this demo's regtest funding and PSBT path. | Use Xverse for the browser-wallet path, or use **Submit Local** for deterministic local-signer validation. |

## Defaults

| Field | Default |
|---|---|
| RPC URL | `/btc-rpc` |
| Wallet Name | `sdk` |
| Network | `regtest` |
| Xverse Network Name | `clearnet-regtest` |
| Xverse Electrs URL | Derived from the current page origin, for example `http://127.0.0.1:5173/btc-rpc` |
| Xverse Network | `Regtest` |
| Fund Sats | `100000000` |
| Fallback Fee Sat/VB | `5` |
| Amount BTC | `0.2` |

Override the backend target with `BTC_RPC_URL`, `BTC_RPC_USER`, or
`BTC_RPC_PASS` when starting Vite.

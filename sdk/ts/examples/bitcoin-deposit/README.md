# Bitcoin Deposit Demo

Browser smoke demo for native BTC deposits. It supports the full local custody
devnet as well as the SDK's standalone Bitcoin Core regtest.

## Custody devnet quickstart

Start custody first:

```sh
cd /path/to/custody
make start
```

Then start the demo from the clearnet SDK repository:

```sh
cd /path/to/clearnet-sdk
npm --prefix sdk/ts ci
npm --prefix sdk/ts run build
npm --prefix sdk/ts run demo:btc
```

Open the Vite URL printed in the terminal, normally
`http://127.0.0.1:5173/`.

The demo reads the active 5-of-7 signer configuration from the sibling
`custody/testenv/btc.env`, derives the shared deposit address, and refuses to
broadcast if it differs from custody's configured `BTC_DEPOSIT_ADDRESS`. If the
repositories are not siblings, specify the file explicitly:

```sh
CUSTODY_BTC_ENV=/absolute/path/to/custody/testenv/btc.env \
  npm --prefix sdk/ts run demo:btc
```

### Submit and confirm

1. Confirm the page says it loaded the custody `5-of-7` configuration.
2. Click **Fund Local**.
3. Enter the clearnet account, optional 32-byte reference, and BTC amount.
4. Click **Submit Local** and copy the transaction ID.
5. Click **Mine Block twice**. The first block confirms the transaction; the
   second advances custody's watcher beyond its local confirmation boundary.
6. Click **Verify Last Tx** to check Bitcoin confirmation.

### Check the MintReceipt manually

The browser deliberately does not access custody's clearing stand-in. Replace
`TXID` with the submitted transaction ID:

```sh
curl -sS 'http://localhost:50052/events?type=mint&tx_id=TXID:0' | jq
```

A completed deposit returns a non-empty `events` array containing the credited
account, amount, asset URI, and at least five signatures. If it is empty, mine
another block and retry after a few seconds.

## Standalone SDK regtest

To exercise only the SDK depositor without custody:

```sh
make devnet-btc
npm --prefix sdk/ts ci
npm --prefix sdk/ts run build
npm --prefix sdk/ts run demo:btc
```

The demo talks to Bitcoin Core through the Vite proxy at `/btc-rpc`. Browser
code does not include the Bitcoin Core username or password; the proxy injects
Basic Auth before forwarding to `http://127.0.0.1:18443`.

## Local signer flow

1. Click **Generate Depositor Key** to create a local depositor key. In
   standalone mode the page also starts with a generated 2-of-3 demo vault.
2. Click **Fund Local** to create/load the `sdk` wallet, mine spendable regtest
   funds, import the generated addresses, and fund the depositor P2WPKH address.
3. Click **Submit Local** to pay the shared P2WSH deposit address with the
   account attribution carried in the `YNET` `OP_RETURN` marker.
4. Click **Verify Last Tx** before mining to observe `pending`.
5. Click **Mine Block**, then **Verify Last Tx** again to observe `confirmed`.

## Xverse PSBT flow

The Xverse path exercises browser-wallet signing without putting wallet-specific
code in the SDK. The demo asks Xverse to add/switch to the local regtest network,
prepares an unsigned PSBT, asks Xverse to sign the selected inputs, finalizes the
signed PSBT locally, and broadcasts it through the same `/btc-rpc` proxy.

Xverse uses the configured custom-network URL as an Electrs-style API while
rendering signing previews. The Vite server exposes a small Electrs-compatible
facade under `/btc-rpc`, backed by Bitcoin Core JSON-RPC.

1. Install Xverse and unlock a Bitcoin account.
2. Confirm **Xverse Electrs URL** points at the running Vite proxy, such as
   `http://127.0.0.1:5173/btc-rpc`.
3. Click **Add/Switch Xverse Network** and approve the prompt.
4. Click **Connect Xverse** and approve account access.
5. Click **Fund Xverse**.
6. Click **Submit Xverse** and approve signing.
7. Use **Verify Last Tx** and **Mine Block** to check confirmation. In custody
   mode, mine a second block before querying the MintReceipt.

If Xverse cannot reach localhost, use a reachable HTTP(S) tunnel for the Vite
proxy or use the deterministic local-signer flow.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| The page says custody configuration was not found. | `custody/testenv/btc.env` does not exist at the expected sibling path. | Run `make start` in custody, restart Vite, or set `CUSTODY_BTC_ENV`. |
| Deposit is refused because the derived address differs from custody. | The displayed signer keys or threshold were edited or are stale. | Restart the demo so it reloads the active custody configuration. |
| **Add/Switch Xverse Network** reports `Access denied`. | The request was rejected or a previous prompt remains open. | Unlock Xverse, close stale prompts, retry, and approve the request. |
| Xverse reports `404` for `/address/<address>/utxo`. | Its custom-network URL points at plain Bitcoin Core or the wrong Vite port. | Point it at the demo's `/btc-rpc` URL. |
| **Submit Xverse** reports no inputs to sign. | The selected wallet address has no visible confirmed UTXO. | Click **Fund Xverse**, then retry. |
| Xverse cannot reach `127.0.0.1`. | The extension blocks the local endpoint. | Use an HTTP(S) tunnel or the local-signer flow. |
| The MintReceipt query returns an empty array. | Custody has not scanned past the transaction block yet. | Mine a second block, wait a few seconds, and retry the query. |

## Defaults and overrides

| Field | Default |
|---|---|
| RPC URL | `/btc-rpc` |
| Wallet Name | `sdk` |
| Network | `regtest` |
| Xverse Network Name | `clearnet-regtest` |
| Xverse Electrs URL | Current page origin plus `/btc-rpc` |
| Fund Sats | `100000000` |
| Fallback Fee Sat/VB | `5` |
| Amount BTC | `0.2` |

Override the backend with `BTC_RPC_URL`, `BTC_RPC_USER`, `BTC_RPC_PASS`, or
`BTC_RPC_WALLET`. This setup is for local development only. A public frontend
should submit through a user wallet and observe the resulting account credit
through clearnet, not custody's internal receipt transport.

# EVM Deposit Demo

Browser smoke demo for `EvmVaultDepositor` against an EIP-1193 wallet and an
EVM RPC endpoint.

## Run

From the repository root:

```sh
make devnet-evm
npm --prefix sdk/ts ci
npm --prefix sdk/ts run build
npm --prefix sdk/ts run demo:evm
```

Open the Vite URL, usually `http://127.0.0.1:5173/`.

## Requirements

- A browser wallet that exposes `window.ethereum`.
- The wallet account must be funded on the configured network.
- A deployed `Custody` contract address. The demo does not deploy contracts.
  The TypeScript integration test self-deploys fresh contracts for automated
  coverage, but the browser page expects you to paste the target custody
  address into `Custody Address`.
- The wallet chain and `RPC URL` field must point at the same chain ID.

This page is therefore a wallet/RPC smoke test for an already deployed custody
contract, not a one-click local deployment harness. For a self-contained local
run that starts Anvil, deploys `Custody`, funds accounts, submits deposits, and
verifies receipts, use `npm --prefix sdk/ts run test:integration:evm`.

## Fields

| Field | Default | Purpose |
|---|---|---|
| RPC URL | `http://127.0.0.1:8545` | EVM JSON-RPC endpoint used for chain checks, receipt waits, and verification. |
| Chain ID | `31337` | Local Anvil chain ID. |
| Custody Address | blank | Deployed `Custody` contract that receives deposits. |
| Credited Account | blank | Clearnet account credited by the deposit; must be an EVM address-shaped value. |
| Asset | blank | Empty string native ETH sentinel. Use an ERC-20 address for token deposits. |
| Reference | blank | Optional `bytes32` deposit reference. |
| Amount | `0.01` | Decimal token amount. |
| Decimals | `18` | Native ETH decimal override. ERC-20 decimals are read from the token contract. |

## Demo Flow

1. Start Anvil with `make devnet-evm`.
2. Deploy or identify the `Custody` contract you want to test. The repository
   integration test deploys from `pkg/blockchain/evm/artifacts/Custody.bin`;
   the browser demo intentionally leaves deployment to your chosen EVM tooling.
3. Add or switch your browser wallet to chain ID `31337` at
   `http://127.0.0.1:8545`.
4. Fund the wallet account on that chain.
5. Paste the custody address and credited account.
6. Click **Connect Wallet**.
7. Click **Submit Deposit** and approve the wallet transaction.
8. Click **Verify Last Tx**.

For native ETH, the SDK sends `Custody.deposit(...)` with `msg.value`. For an
ERC-20 asset, the SDK sends an exact `approve(...)` first and then the custody
deposit transaction.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `No EIP-1193 wallet found` | No compatible browser wallet is installed or enabled for the page. | Install or enable a wallet that injects `window.ethereum`, then reload the demo. |
| `RPC URL reports chain ... but Chain ID is ...` | The page RPC and chain ID fields do not describe the same network. | Update both fields to the same chain, then reconnect. |
| Wallet balance and configured RPC balance differ | The wallet is connected to a different RPC than the demo page. | Switch the wallet to the RPC URL shown in the page, then reconnect. |
| You do not have a `Custody Address` | The browser demo does not deploy contracts. | Deploy `Custody` with your preferred EVM tooling, or use `npm --prefix sdk/ts run test:integration:evm` for the self-contained local path. |
| `CHAIN_MISMATCH` | Wallet or public RPC chain ID does not match the configured `Chain ID`. | Use **Connect Wallet** again after switching the wallet network. |
| `INVALID_ADDRESS` for `Custody Address`, `Credited Account`, or `Asset` | A required address field is blank or malformed, or the asset is the zero address. | Paste a valid `0x` address. Leave `Asset` blank for native ETH. |
| ERC-20 deposit fails during approval | The wallet rejected the approval, the token address is wrong, or the wallet lacks token balance. | Confirm the ERC-20 contract address and wallet balance, then submit again. |
| `TX_REVERTED` | The custody call or approval mined with a reverted receipt. | Check the custody address, asset, allowance/balance, and chain. |
| `Verify Last Tx` reports `pending` | The transaction is known but not confirmed enough for `minConfirmations = 1`. | Wait for another block on the configured chain, then verify again. |

## Notes

This page is a browser-wallet smoke test. For deterministic local coverage,
use `npm --prefix sdk/ts run test:integration:evm`; that test provisions fresh
contracts and accounts against the Anvil devnet.

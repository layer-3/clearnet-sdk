# Solana Deposit Demo

Browser smoke demo for `SolanaVaultDepositor` using a Wallet Standard Solana
wallet.

## Run

From the repository root:

```sh
make devnet-sol
npm --prefix sdk/ts ci
npm --prefix sdk/ts run build
npm --prefix sdk/ts run demo:sol
```

Open the Vite URL, usually `http://127.0.0.1:5173/`.

## Requirements

- A Solana browser wallet that implements Wallet Standard `standard:connect`
  and `solana:signTransaction`.
- The wallet must expose an account for the selected wallet chain, such as
  `solana:localnet`.
- The wallet account must be funded on the configured RPC.
- Solana CLI if you want to use the documented local `solana airdrop` command;
  otherwise fund the wallet through another localnet-compatible method.
- The custody program must be deployed at the configured `Program ID`. The
  repository devnet starts `solana-test-validator` with the custody program
  preloaded at the default program ID.
- SPL token deposits require the depositor associated token account and vault
  associated token account to exist before submission. The browser demo does
  not mint tokens or create token accounts.

## Fields

| Field | Default | Purpose |
|---|---|---|
| RPC URL | `http://127.0.0.1:8899` | Solana RPC endpoint used for balances, blockhashes, send, and verification. |
| Wallet Chain | `solana:localnet` | Wallet Standard chain requested from the browser wallet. |
| Commitment | `confirmed` | Commitment used for send and confirmation checks. |
| Program ID | `98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg` | Custody program ID expected by the SDK. |
| Credited Account | `00000000000000000000000000000000000000a1` | 20-byte Clearnet account credited by the deposit. |
| Asset | `SOL` | Native SOL sentinel. Use an SPL mint address for token deposits. |
| Reference | blank | Optional `bytes32` deposit reference. |
| Amount | `100000000` | Base-unit amount. For native SOL this is lamports. |

## Demo Flow

1. Start the local validator with `make devnet-sol`.
2. Open the demo and confirm `RPC URL`, `Wallet Chain`, and `Program ID`.
3. Fund the wallet account on the configured RPC. For localnet, copy the
   wallet public key after connecting and run:

   ```sh
   solana airdrop 2 <wallet-public-key> --url http://127.0.0.1:8899
   ```

4. Click **Connect Wallet** and approve the wallet prompt.
5. Keep `Asset` as `SOL` for a native deposit, or paste an SPL mint address.
6. Click **Submit Deposit** and approve the signing request.
7. Click **Verify Last Tx**.

The demo builds a single custody instruction, asks the wallet to sign the
transaction, sends the signed bytes through the configured RPC, and returns the
Go-compatible `TxRef` shape.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `No Wallet Standard Solana wallet found for solana:localnet` | The installed wallet does not expose the selected chain or required features. | Select a chain your wallet supports, or use a Wallet Standard wallet that supports localnet signing. |
| `wallet did not return an account for ...` | The wallet connected, but no account supports the selected chain and signing feature. | Switch the wallet network/account and reconnect. |
| `network settings changed after wallet connection` | RPC URL, chain, or commitment was changed after connecting. | Click **Connect Wallet** again so the signer state matches the form. |
| Wallet balance is `0` | The account has not been funded on the configured RPC. | Airdrop or transfer funds to the connected account on that same RPC. |
| SPL deposit fails with an account error | The SPL depositor ATA or vault ATA does not exist, or the depositor ATA has no token balance. | Create and fund both token accounts before submitting, or use native `SOL` for the browser smoke test. |
| `INVALID_ADDRESS` for credited account | The credited account is not a 20-byte hex account. | Use a 40-character hex value with or without `0x`, or a supported account URI. |
| `INVALID_REFERENCE` | Reference is not a 32-byte hex value. | Leave it blank or provide `0x` plus 64 hex characters. |
| `TX_REVERTED` | The submitted transaction reported an instruction error. | Check the program ID, account funding, asset/mint, and program initialization on the target validator. |
| `Verify Last Tx` reports `pending` | The transaction has not reached the requested commitment. | Wait and verify again, or use the same commitment level selected during submission. |

## Notes

This page is a browser-wallet smoke test. For deterministic local coverage,
use `npm --prefix sdk/ts run test:integration:sol`; that test funds local
signers and exercises native SOL plus SPL deposit paths against the devnet.

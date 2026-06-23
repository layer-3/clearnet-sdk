# Clearnet TypeScript SDK

TypeScript SDK for Clearnet integration. This package currently exposes EVM and
Solana vault depositors. EVM supports native ETH and ERC-20 deposits. Solana
supports native SOL and SPL token deposits. Deposits credit a `destination` made
of an account and an optional ADR-015 opaque reference.

The package is ESM-first. EVM callers use `viem` clients and primitives. Solana
callers provide an SDK-owned signer adapter around their wallet or local keypair.

## Install

```sh
npm install @yellow-org/clearnet-sdk viem @solana/web3.js
```

For local development in this repository:

```sh
cd sdk/ts
npm ci
```

## EVM Quick Start

Native ETH deposits use `EVM_NATIVE_ASSET`, which is the EVM zero address. Amounts
must be `bigint` values in base units.

```ts
import {
  ClearnetSdkError,
  EVM_NATIVE_ASSET,
  EvmVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import {
  createPublicClient,
  createWalletClient,
  custom,
  http,
  parseEther,
} from "viem";
import type { Address, EIP1193Provider } from "viem";

declare const window: Window & { ethereum?: EIP1193Provider };

const chainId = 31_337;
const rpcUrl = "http://127.0.0.1:8545";
// Replace with the Custody contract deployed on the configured chain.
const custodyAddress = "0x0000000000000000000000000000000000001000" as Address;

if (window.ethereum === undefined) {
  throw new Error("No EIP-1193 wallet found");
}

const accounts = (await window.ethereum.request({
  method: "eth_requestAccounts",
})) as Address[];
const walletAccount = accounts[0];
if (walletAccount === undefined) {
  throw new Error("Wallet did not return an account");
}

const publicClient = createPublicClient({
  transport: http(rpcUrl),
});
const walletClient = createWalletClient({
  account: walletAccount,
  transport: custom(window.ethereum),
});

const depositor = new EvmVaultDepositor({
  publicClient,
  walletClient,
  walletAccount,
  custodyAddress,
  chainId,
});

try {
  const ref = await depositor.submitDeposit(
    {
      destination: { account: walletAccount },
      asset: EVM_NATIVE_ASSET,
      amount: parseEther("0.01"),
    },
    {
      onSubmitted(submittedRef) {
        console.log("deposit submitted", submittedRef.hash);
      },
    },
  );

  console.log("deposit mined", ref.hash);
  console.log("status", await depositor.verifyDeposit(ref, 1));
} catch (error) {
  if (error instanceof ClearnetSdkError) {
    console.error(error.code, error.txRef?.hash);
  }
  throw error;
}
```

The SDK checks the configured public RPC chain and wallet chain before signing
or submitting a deposit. If either chain does not match `chainId`, it throws
`CHAIN_MISMATCH`.

## ERC-20 Deposits

For ERC-20 deposits, pass the token contract address as `asset`.

```ts
import { parseUnits } from "viem";

const ref = await depositor.submitDeposit({
  destination: { account: walletAccount },
  asset: "0x0000000000000000000000000000000000003000",
  amount: parseUnits("25", 18),
});
```

The SDK submits an exact-amount `approve(custodyAddress, amount)` transaction
before submitting the custody `deposit(...)` transaction. A successful
`submitDeposit` call returns the deposit transaction hash, not the approval hash.
If an ERC-20 approval fails before the deposit is submitted, `error.txRef` may
refer to the approval transaction.

## Solana Deposits

Solana deposits use `SolanaVaultDepositor`. The SDK builds the custody
instruction and delegates signing/broadcast to a caller-provided `SolanaSigner`.
The signer boundary is small so browser-wallet, Wallet Standard, and local
keypair adapters can live outside the core SDK.

```ts
import {
  SOLANA_NATIVE_ASSET,
  SolanaVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import { Connection, Keypair, sendAndConfirmTransaction } from "@solana/web3.js";
import type { SolanaSigner } from "@yellow-org/clearnet-sdk";

const rpcUrl = "http://127.0.0.1:8899";
const keypair = Keypair.generate();
const connection = new Connection(rpcUrl, "confirmed");

const signer: SolanaSigner = {
  publicKey: keypair.publicKey.toBase58(),
  async signAndSend(transaction) {
    return sendAndConfirmTransaction(connection, transaction, [keypair], {
      commitment: "confirmed",
      preflightCommitment: "confirmed",
    });
  },
};

const depositor = new SolanaVaultDepositor({
  rpcUrl,
  signer,
  commitment: "confirmed",
});

const ref = await depositor.submitDeposit({
  destination: {
    account: "00000000000000000000000000000000000000a1",
    ref: "0x3333333333333333333333333333333333333333333333333333333333333333",
  },
  asset: SOLANA_NATIVE_ASSET,
  amount: 100_000_000n,
});

console.log(ref.raw); // Solana base58 signature
console.log(ref.hash); // 0x + sha256(signature bytes)
console.log(await depositor.verifyDeposit(ref, 0));
```

Native asset aliases are `SOL`, `sol`, `native`, and an empty string. For SPL
deposits, pass the mint public key as `asset` and the amount in token base units.
The SDK does not mint tokens or create token accounts. SPL callers must ensure
the depositor ATA and vault ATA exist before submitting the deposit.

## Deposit References

Pass `destination.ref` to attach a 32-byte opaque sub-account reference to the
deposit. Omit it when there is no sub-account reference.

```ts
const ref = await depositor.submitDeposit({
  destination: {
    account: walletAccount,
    ref: "0x3333333333333333333333333333333333333333333333333333333333333333",
  },
  asset: EVM_NATIVE_ASSET,
  amount: parseEther("0.01"),
});
```

For EVM, the reference is passed to `Custody.deposit(...)` as `bytes32`. For
Solana, it is encoded into `deposit_sol` or `deposit_spl` as `[u8; 32]`. The SDK
does not interpret it. Omitted references are sent as 32 zero bytes.

## Verify A Deposit

```ts
const status = await depositor.verifyDeposit(ref, 1);
```

`verifyDeposit` returns:

| Status | Meaning |
|---|---|
| `confirmed` | The transaction has a successful receipt and at least `minConfirmations` confirmations. |
| `pending` | The transaction is known but is not confirmed enough yet. |
| `absent` | The transaction is unknown or has a reverted receipt. |

`minConfirmations` accepts a non-negative safe integer `number` or a non-negative
`bigint`. EVM treats it as an inclusive receipt confirmation count. Solana maps
it onto the commitment ladder: `0` accepts `confirmed`; `>= 1` requires
`finalized`.

## API Reference

### `EvmVaultDepositor`

```ts
new EvmVaultDepositor(config: EvmDepositorConfig)
```

Config fields:

| Field | Type | Notes |
|---|---|---|
| `publicClient` | `viem` `PublicClient` | Used for chain checks, receipt waits, and verification. |
| `walletClient` | `viem` `WalletClient` | Used to submit approval and deposit transactions. |
| `walletAccount` | `Account \| Address` | Must match `walletClient.account` when the wallet client has one. |
| `custodyAddress` | `Address` | The deployed custody contract address. |
| `chainId` | `number` | Positive safe integer; must match the public RPC and wallet chain. |
| `receiptTimeoutMs` | `number` | Optional default timeout for receipt waits. |

### `submitDeposit(input, options?)`

Input fields:

| Field | Type | Notes |
|---|---|---|
| `destination.account` | `Address` | Clearnet account credited by the custody deposit. |
| `destination.ref` | `Hash \| undefined` | Optional 32-byte opaque reference. Omitted values are sent as `bytes32(0)`. |
| `asset` | `Address` | Use `EVM_NATIVE_ASSET` for native ETH, or an ERC-20 token address. |
| `amount` | `bigint` | Positive base-unit amount that fits in `uint256`. |

Options:

| Field | Notes |
|---|---|
| `signal` | Aborts the receipt wait. |
| `receiptTimeoutMs` | Overrides the receipt wait timeout for this call. |
| `onSubmitted` | Called with the deposit `TxRef` after the deposit transaction is submitted. |

Returns:

```ts
type TxRef = {
  hash: `0x${string}`;
  raw: string;
};
```

For EVM, `hash` and `raw` are both the transaction hash.

### `SolanaVaultDepositor`

```ts
new SolanaVaultDepositor(config: SolanaDepositorConfig)
```

Config fields:

| Field | Type | Notes |
|---|---|---|
| `rpcUrl` | `string` | Used for signature-status verification and commitment waits. |
| `signer` | `SolanaSigner` | Provides `publicKey` and `signAndSend(transaction)`. |
| `programId` | `string` | Optional. Must be the default custody program ID in this version. |
| `commitment` | `"processed" \| "confirmed" \| "finalized"` | Optional; defaults to `finalized`. |
| `receiptTimeoutMs` | `number` | Optional default timeout for commitment waits. |

Solana input fields:

| Field | Type | Notes |
|---|---|---|
| `destination.account` | `string` | 20-byte Clearnet account as hex, optional `0x`, or URI-like value whose final path segment is that hex. |
| `destination.ref` | `` `0x${string}` \| undefined `` | Optional 32-byte opaque reference. |
| `asset` | `string` | Native alias (`SOL`, `sol`, `native`, or empty string) or SPL mint public key. |
| `amount` | `bigint` | Positive base-unit amount that fits in `uint64`. |

For Solana, `TxRef.raw` is the base58 signature and `TxRef.hash` is `0x` plus
the SHA-256 digest of the signature bytes.

### `verifyDeposit(ref, minConfirmations)`

Returns `Promise<"absent" | "pending" | "confirmed">`.

## Local Development

Run package checks from `sdk/ts`:

```sh
npm run typecheck
npm test
npm run build
npm --workspace @yellow-org/evm-deposit-demo run build
npm --workspace @yellow-org/solana-deposit-demo run build
```

Run the EVM integration test against local Anvil:

```sh
# From the repository root:
make devnet-evm

# From sdk/ts:
npm run test:integration:evm

# From the repository root:
make devnet-down
```

The integration test deploys fresh `Custody` and `MockERC20` contracts on each
run.

Run the Solana integration test against the local validator:

```sh
# From the repository root:
make devnet-sol

# From sdk/ts:
npm run test:integration:sol

# From the repository root:
make devnet-down
```

The Solana devnet preloads the custody program at
`98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg`. The integration test creates
and funds local signers, creates SPL token accounts needed for the test, submits
native SOL and SPL deposits, and verifies each returned transaction reference.

To run the repository integration suite, including the TS EVM and Solana
integration tests:

```sh
# From the repository root:
make devnet
make integration
make devnet-down
```

## Demo

Start the browser demo from `sdk/ts`:

```sh
npm run demo:evm
npm run demo:sol
```

The EVM demo expects:

- an EIP-1193 wallet, such as MetaMask
- an RPC URL and chain ID that match the wallet's selected network
- a deployed `Custody` contract address
- a funded wallet account on that network

`make devnet-evm` starts Anvil on `http://127.0.0.1:8545` with chain ID `31337`,
but it does not predeploy `Custody` for the browser demo.

The Solana demo discovers wallets through Wallet Standard, uses
`solana:signTransaction`, and broadcasts the signed transaction through the
configured RPC URL. The selected wallet chain must be one the wallet advertises,
such as `solana:localnet` for a local validator. The local devnet preloads the
custody program, but the wallet must be funded and SPL token accounts must
already exist for SPL deposits.

## Troubleshooting

Errors thrown by the SDK use `ClearnetSdkError` with a stable `code`.

| Code | Common cause |
|---|---|
| `INVALID_ADDRESS` | EVM address, Solana public key, Solana mint, program ID, or Clearnet account is invalid. |
| `INVALID_AMOUNT` | `amount` is not a positive `bigint` or exceeds the chain limit (`uint256` for EVM, `uint64` for Solana). |
| `INVALID_CONFIRMATIONS` | `minConfirmations` is negative, fractional, or an unsafe number. |
| `INVALID_REFERENCE` | `destination.ref` is not a 32-byte hex value. |
| `INVALID_TX_REF` | `ref.hash` is not bytes32, or Solana `ref.raw` is not a 64-byte signature. |
| `MISSING_WALLET_ACCOUNT` | The EVM wallet account is missing/mismatched, or the Solana signer is missing. |
| `CHAIN_MISMATCH` | EVM only: the public RPC or wallet chain does not match `chainId`. |
| `TX_REVERTED` | A submitted approval or deposit transaction reverted. |
| `RECEIPT_TIMEOUT` | Waiting for a receipt timed out or was aborted. |
| `RPC_ERROR` | The public RPC or wallet provider returned an unexpected error. |

When a transaction may already have been submitted, `ClearnetSdkError` can include
`txRef`. Use that hash to let a user inspect or retry verification of the
submitted transaction.

# Clearnet TypeScript SDK

TypeScript SDK for Clearnet integration. This package currently exposes
the EVM vault depositor, with support for native ETH deposits and ERC-20
deposits. Deposits credit a `destination` made of an account and an optional
ADR-015 opaque reference.

The package is ESM-first and uses `viem` for EVM clients and primitives.

## Install

```sh
npm install @yellow-org/clearnet-sdk viem
```

For local development in this repository:

```sh
cd sdk/ts
npm ci
```

## Quick Start

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

For EVM, the reference is passed to `Custody.deposit(...)` as `bytes32`. The SDK
does not interpret it.

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
`bigint`.

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

### `verifyDeposit(ref, minConfirmations)`

Returns `Promise<"absent" | "pending" | "confirmed">`.

## Local Development

Run package checks from `sdk/ts`:

```sh
npm run typecheck
npm test
npm run build
npm --workspace @yellow-org/evm-deposit-demo run build
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

To run the repository integration suite, including this TS EVM integration test:

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
```

The demo expects:

- an EIP-1193 wallet, such as MetaMask
- an RPC URL and chain ID that match the wallet's selected network
- a deployed `Custody` contract address
- a funded wallet account on that network

`make devnet-evm` starts Anvil on `http://127.0.0.1:8545` with chain ID `31337`,
but it does not predeploy `Custody` for the browser demo.

## Troubleshooting

Errors thrown by the SDK use `ClearnetSdkError` with a stable `code`.

| Code | Common cause |
|---|---|
| `INVALID_ADDRESS` | `account`, `asset`, `custodyAddress`, or `walletAccount` is not a valid EVM address. |
| `INVALID_AMOUNT` | `amount` is not a positive `bigint` or does not fit in `uint256`. |
| `INVALID_CONFIRMATIONS` | `minConfirmations` is negative, fractional, or an unsafe number. |
| `INVALID_REFERENCE` | `destination.ref` is not a 32-byte hex value. |
| `INVALID_TX_REF` | `ref.hash` is missing or is not a 32-byte EVM transaction hash. |
| `MISSING_WALLET_ACCOUNT` | The wallet account is missing or does not match `walletClient.account`. |
| `CHAIN_MISMATCH` | The public RPC or wallet chain does not match `chainId`. |
| `TX_REVERTED` | A submitted approval or deposit transaction reverted. |
| `RECEIPT_TIMEOUT` | Waiting for a receipt timed out or was aborted. |
| `RPC_ERROR` | The public RPC or wallet provider returned an unexpected error. |

When a transaction may already have been submitted, `ClearnetSdkError` can include
`txRef`. Use that hash to let a user inspect or retry verification of the
submitted transaction.

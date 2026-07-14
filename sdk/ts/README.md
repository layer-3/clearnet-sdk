# Clearnet TypeScript SDK

TypeScript SDK for Clearnet integration. This package currently exposes EVM,
Solana, XRPL, and Bitcoin vault depositors. EVM supports native ETH and ERC-20
deposits. Solana supports native SOL and SPL token deposits. XRPL supports
native XRP and issued-currency deposits. Bitcoin supports native BTC deposits.
Deposits credit a `destination` account. EVM, Solana, and XRPL can also carry an
optional ADR-015 opaque reference; Bitcoin deposits reject non-zero references.

The package is ESM-first. EVM callers use `viem` clients and primitives. Solana
and XRPL callers provide SDK-owned signer adapters around their wallet or local
keypair.

## Install

```sh
npm install @yellow-org/clearnet-sdk viem @solana/web3.js xrpl @scure/btc-signer
```

For local development in this repository:

```sh
cd sdk/ts
npm ci
```

## Bitcoin Quick Start

Bitcoin deposits use `BitcoinVaultDepositor`. Native BTC uses
`BITCOIN_NATIVE_ASSET`, which is an empty string, and deposit amounts are
positive decimal BTC strings. The depositor spends from a P2WPKH address derived
from the signer public key and pays the per-account P2WSH deposit address
derived from the configured vault keys. The SDK signs digest bytes through a
caller-provided `BitcoinSigner` so local keys, HSMs, or wallet adapters can live
outside the core depositor.

```ts
import {
  BITCOIN_NATIVE_ASSET,
  BitcoinCoreRpcClient,
  BitcoinVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import {
  pubECDSA,
  randomPrivateKeyBytes,
  signECDSA,
} from "@scure/btc-signer/utils.js";
import type { BitcoinSigner } from "@yellow-org/clearnet-sdk";

const privateKey = randomPrivateKeyBytes();
const signer: BitcoinSigner = {
  algorithm: "secp256k1",
  getPublicKeyCompressed() {
    return pubECDSA(privateKey, true);
  },
  signDigest32(digest) {
    return signECDSA(digest, privateKey);
  },
};

const rpc = new BitcoinCoreRpcClient({
  url: "http://127.0.0.1:18443",
  username: "sdk",
  password: "sdk",
  wallet: "sdk",
});

const vaultPubkeys = [
  pubECDSA(randomPrivateKeyBytes(), true),
  pubECDSA(randomPrivateKeyBytes(), true),
  pubECDSA(randomPrivateKeyBytes(), true),
];

const depositor = new BitcoinVaultDepositor({
  network: "regtest",
  rpc,
  signer,
  vaultPubkeys,
  threshold: 2,
  minFundingConfirmations: 1,
  fallbackFeeRateSatPerVByte: 5n,
});

console.log(await depositor.depositorAddress()); // fund this P2WPKH address
console.log(depositor.depositAddress("yellow://ynet/user/btc-a1"));

const ref = await depositor.submitDeposit({
  destination: { account: "yellow://ynet/user/btc-a1" },
  asset: BITCOIN_NATIVE_ASSET,
  amount: "0.0002",
});

console.log(ref); // display txid
console.log(await depositor.verifyDeposit(ref, 1));
```

`BitcoinCoreRpcClient` sends Basic Auth only when both `username` and `password`
are supplied. Browser demos should use a same-origin proxy, omit credentials in
the client config, and inject Bitcoin Core credentials server-side.

## EVM Quick Start

Native ETH deposits use `EVM_NATIVE_ASSET`, which is an empty string. Amounts
are positive decimal strings. Native ETH uses 18 decimals by default, or the
`nativeDecimals` override in depositor config; ERC-20 decimals are read from the
token contract and cached.

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
      amount: "0.01",
    },
    {
      onSubmitted(submittedRef) {
        console.log("deposit submitted", submittedRef);
      },
    },
  );

  console.log("deposit mined", ref);
  console.log("status", await depositor.verifyDeposit(ref, 1));
} catch (error) {
  if (error instanceof ClearnetSdkError) {
    console.error(error.code, error.txID);
  }
  throw error;
}
```

The SDK checks the configured public RPC chain and wallet chain before signing
or submitting a deposit. If either chain does not match `chainId`, it throws
`CHAIN_MISMATCH`.

## ERC-20 Deposits

For ERC-20 deposits, pass the token contract address as `asset` and the token
amount as a decimal string.

```ts
const ref = await depositor.submitDeposit({
  destination: { account: walletAccount },
  asset: "0x0000000000000000000000000000000000003000",
  amount: "25",
});
```

The SDK reads and caches the token's `decimals()`, converts the decimal amount
to base units, submits an exact-amount `approve(custodyAddress, amount)`
transaction, then submits the custody `deposit(...)` transaction. A successful
`submitDeposit` call returns the deposit transaction hash, not the approval
hash. If an ERC-20 approval fails before the deposit is submitted, `error.txID`
may refer to the approval transaction.

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
import {
  Connection,
  Keypair,
  LAMPORTS_PER_SOL,
  sendAndConfirmTransaction,
} from "@solana/web3.js";
import type { SolanaSigner } from "@yellow-org/clearnet-sdk";

const rpcUrl = "http://127.0.0.1:8899";
const keypair = Keypair.generate();
const connection = new Connection(rpcUrl, "confirmed");

const airdrop = await connection.requestAirdrop(
  keypair.publicKey,
  LAMPORTS_PER_SOL,
);
await connection.confirmTransaction(airdrop, "confirmed");

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
  amount: "0.1",
});

console.log(ref); // Solana base58 signature
console.log(ref); // 0x + sha256(signature bytes)
console.log(await depositor.verifyDeposit(ref, 0));
```

Native SOL uses `SOLANA_NATIVE_ASSET`, which is an empty string. For SPL
deposits, pass the mint public key as `asset` and the token amount as a decimal
string. The SDK reads and caches SPL mint decimals. It does not mint tokens or
create token accounts. SPL callers must ensure the depositor ATA and vault ATA
exist before submitting the deposit.

## XRPL Deposits

XRPL deposits use `XrplVaultDepositor`. Amounts are positive decimal strings.
Native XRP uses an empty asset string. Issued-currency assets use `CUR.rIssuer`
and require decimals in depositor config.

```ts
import {
  XRPL_NATIVE_ASSET,
  XrplVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import { Wallet, hashes, type SubmittableTransaction } from "xrpl";
import type {
  XrplPreparedPayment,
  XrplSigner,
} from "@yellow-org/clearnet-sdk";

const wallet = Wallet.generate();
const signer: XrplSigner = {
  classicAddress: wallet.classicAddress,
  async sign(payment: XrplPreparedPayment) {
    const signed = wallet.sign(payment as SubmittableTransaction);
    return { txBlob: signed.tx_blob, hash: hashes.hashSignedTx(signed.tx_blob) };
  },
};

const depositor = new XrplVaultDepositor({
  rpcUrl: "ws://127.0.0.1:6006",
  vaultAddress: "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
  signer,
});

try {
  const ref = await depositor.submitDeposit({
    destination: {
      account: "00000000000000000000000000000000000000a1",
      ref: "0x3333333333333333333333333333333333333333333333333333333333333333",
    },
    asset: XRPL_NATIVE_ASSET,
    amount: "1",
  });

  console.log(ref); // uppercase XRPL transaction hash
  console.log(ref); // same bytes as 0x-prefixed hex
  console.log(await depositor.verifyDeposit(ref, 0));
} finally {
  await depositor.disconnect();
}
```

For issued currencies, pass the asset key, decimal amount, and configured
decimals:

```ts
const usdAsset = "USD.rN7n7otQDd6FczFgLdSqtcsAUxDkw6fzRH";
const depositor = new XrplVaultDepositor({
  rpcUrl: "ws://127.0.0.1:6006",
  vaultAddress: "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
  signer,
  issuedAssetDecimals: { [usdAsset]: 2 },
});
const ref = await depositor.submitDeposit({
  destination: { account: "00000000000000000000000000000000000000a1" },
  asset: usdAsset,
  amount: "25.50",
});
```

Trustlines and balances must already exist before an issued-currency deposit.
The SDK builds one XRPL `Payment`, adds one `ynet-account` memo carrying the
Clearnet account/reference, asks the caller-provided signer to sign, submits the
signed blob, and returns after rippled accepts the submit result as `tesSUCCESS`
or `terQUEUED`. Use `verifyDeposit` to observe validated-ledger finality; a
just-submitted XRPL payment can return `pending` until it appears in a validated
ledger.

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
  amount: "0.01",
});
```

For EVM, the reference is passed to `Custody.deposit(...)` as `bytes32`. For
Solana, it is encoded into `deposit_sol` or `deposit_spl` as `[u8; 32]`. For
XRPL, it is appended after the 20-byte Clearnet account in the `ynet-account`
payment memo. The SDK does not interpret it. Omitted references are sent as 32
zero bytes. Bitcoin deposits do not attach a reference and reject non-zero
`destination.ref` values.

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
`finalized`. XRPL validates the shape for cross-chain parity but treats XRPL
finality as binary: a validated transaction is `confirmed`. Bitcoin returns
`confirmed` for any known transaction when `minConfirmations` is `0`; otherwise
it returns `pending` for mempool or shallow transactions and `confirmed` when
the transaction has at least `minConfirmations` confirmations.

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
| `nativeDecimals` | `number` | Optional native ETH decimals override; defaults to `18`. |

### `submitDeposit(input, options?)`

Input fields:

| Field | Type | Notes |
|---|---|---|
| `destination.account` | `Address` | Clearnet account credited by the custody deposit. |
| `destination.ref` | `Hash \| undefined` | Optional 32-byte opaque reference. Omitted values are sent as `bytes32(0)`. |
| `asset` | `Address \| ""` | Use `EVM_NATIVE_ASSET` for native ETH, or an ERC-20 token address. |
| `amount` | `string` | Positive decimal amount. Native uses `nativeDecimals`; ERC-20 tokens use on-chain `decimals()`. |

Options:

| Field | Notes |
|---|---|
| `signal` | Aborts the receipt wait. |
| `receiptTimeoutMs` | Overrides the receipt wait timeout for this call. |
| `onSubmitted` | Called with the deposit `txID` string. |

Returns:

```ts
type TxID = string;
```

For EVM deposits, `txID` is `txHash/logIndex`, identifying the exact
`Deposited` log. Verification also accepts a raw transaction hash for
transaction-level status checks.

### `BitcoinVaultDepositor`

```ts
new BitcoinVaultDepositor(config: BitcoinDepositorConfig)
```

Config fields:

| Field | Type | Notes |
|---|---|---|
| `network` | `"mainnet" \| "testnet" \| "signet" \| "regtest"` | Selects address parameters. |
| `rpc` | `BitcoinRpc` | Used for UTXO lookup, fee estimates, broadcast, and verification. |
| `signer` | `BitcoinSigner` | Provides a compressed secp256k1 public key and DER signature over 32-byte digests. |
| `vaultPubkeys` | `(Uint8Array \| string)[]` | Compressed secp256k1 vault public keys. The SDK sorts them before building the P2WSH script. |
| `threshold` | `number` | Multisig threshold, from 1 to the number of vault keys. |
| `minFundingConfirmations` | `bigint \| number` | Optional UTXO confirmation floor; defaults to `1`. |
| `feeTargetBlocks` | `bigint \| number` | Optional `estimatesmartfee` target; defaults to `6`. |
| `fallbackFeeRateSatPerVByte` | `bigint \| number` | Optional fallback fee rate; defaults to `5`. |

Bitcoin input fields:

| Field | Type | Notes |
|---|---|---|
| `destination.account` | `string` | Opaque Clearnet account. The SDK hashes it to derive the per-account P2WSH address. |
| `destination.ref` | `undefined \| 0x00...00` | Non-zero references are rejected. |
| `asset` | `string` | Use `BITCOIN_NATIVE_ASSET`, the empty string. Other asset values are rejected. |
| `amount` | `string` | Positive decimal BTC amount that fits in signed 64-bit satoshis. |

For Bitcoin, `txID` is the display txid. `submitDeposit` returns after Bitcoin
Core accepts the raw transaction; use `verifyDeposit` to observe mempool,
shallow, and confirmed states.

For PSBT wallet signing, `prepareDepositPsbt` returns an `unsignedTxID` for the
unsigned transaction shape. Use the `txID` returned by
`submitSignedDepositPsbt` for verification because wallet finalization can
change the final txid for nested-SegWit inputs.

`BitcoinCoreRpcClient` is a convenience adapter for Bitcoin Core JSON-RPC:

```ts
new BitcoinCoreRpcClient({
  url: "http://127.0.0.1:18443",
  username: "sdk",
  password: "sdk",
  wallet: "sdk",
});
```

Omit `username` and `password` together when using a same-origin proxy that adds
Basic Auth outside the browser.

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
| `asset` | `string` | Use `SOLANA_NATIVE_ASSET`, the empty string, or an SPL mint public key. |
| `amount` | `string` | Positive decimal amount. Native SOL uses 9 decimals; SPL tokens use mint decimals. |

For Solana, `txID` is the base58 signature.

### `XrplVaultDepositor`

```ts
new XrplVaultDepositor(config: XrplDepositorConfig)
```

Config fields:

| Field | Type | Notes |
|---|---|---|
| `rpcUrl` | `string` | XRPL WebSocket URL used for autofill, submit, and verification. |
| `vaultAddress` | `string` | XRPL classic address that receives deposits. |
| `signer` | `XrplSigner` | Provides `classicAddress` and `sign(payment)`. |
| `maxFeeDrops` | `bigint \| number` | Optional positive fee ceiling checked after autofill and before signing. |
| `issuedAssetDecimals` | `Record<string, number>` | Required per issued-currency asset used for deposits. Native XRP uses 6 decimals. |

XRPL input fields:

| Field | Type | Notes |
|---|---|---|
| `destination.account` | `string` | 20-byte Clearnet account as hex, with optional `0x`. |
| `destination.ref` | `` `0x${string}` \| undefined `` | Optional 32-byte opaque reference. |
| `asset` | `string` | Empty string for native, or issued-currency `CUR.rIssuer`. |
| `amount` | `string` | Positive decimal amount; native XRP uses 6 decimals, issued currencies use configured decimals. |

For XRPL, `txID` is the uppercase 64-hex transaction hash.

`XrplVaultDepositor` owns an XRPL WebSocket client. Call
`await depositor.disconnect()` when the depositor is no longer needed, such as
when replacing the signer or shutting down a long-lived process.

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
npm --workspace @yellow-org/xrpl-deposit-demo run build
npm --workspace @yellow-org/bitcoin-deposit-demo run build
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

Run the Bitcoin integration test against local Bitcoin Core regtest:

```sh
# From the repository root:
make devnet-btc

# From sdk/ts:
npm run test:integration:btc

# From the repository root:
make devnet-down
```

The Bitcoin integration test creates a fresh signer set, mines regtest funds,
funds the generated depositor P2WPKH address, submits a native BTC deposit, and
verifies the returned transaction reference before and after mining.

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

Run the XRPL integration test against local rippled:

```sh
# From the repository root:
make devnet-xrpl

# From sdk/ts:
npm run test:integration:xrpl

# From the repository root:
make devnet-down
```

The XRPL integration test uses rippled WebSocket `ws://127.0.0.1:6006` for SDK
calls and admin JSON-RPC `http://127.0.0.1:5005` for `ledger_accept`. Override
with `XRPL_WS_URL` and `XRPL_ADMIN_RPC_URL` if needed. It creates fresh accounts,
funds them from the standalone genesis wallet, submits native XRP and issued
currency deposits, and verifies each returned transaction reference.

To run the repository integration suite, including the TS EVM, Solana, XRPL, and
Bitcoin integration tests:

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
npm run demo:xrpl
npm run demo:btc
```

The EVM demo expects:

- an EIP-1193 wallet, such as MetaMask
- an RPC URL and chain ID that match the wallet's selected network
- a deployed `Custody` contract address
- a funded wallet account on that network

`make devnet-evm` starts Anvil on `http://127.0.0.1:8545` with chain ID `31337`,
but it does not predeploy `Custody` for the browser demo. See
[examples/evm-deposit/README.md](examples/evm-deposit/README.md) for the
browser flow, required contract setup, and troubleshooting notes.

The Solana demo discovers wallets through Wallet Standard, uses
`solana:signTransaction`, and broadcasts the signed transaction through the
configured RPC URL. The selected wallet chain must be one the wallet advertises,
such as `solana:localnet` for a local validator. The local devnet preloads the
custody program, but the wallet must be funded and SPL token accounts must
already exist for SPL deposits. See
[examples/solana-deposit/README.md](examples/solana-deposit/README.md) for
local funding notes and troubleshooting.

The XRPL demo supports a local signer for standalone-devnet smoke tests and
GemWallet for browser-wallet signing. The GemWallet path requires a custom
`wss://` endpoint that points at the same local chain as the demo because wallet
network selection and SDK submission must agree. See
[examples/xrpl-deposit/README.md](examples/xrpl-deposit/README.md) for the full
local signer flow, GemWallet custom-network setup, funding steps, and
troubleshooting notes.

The Bitcoin demo uses local generated keys and Bitcoin Core regtest through a
Vite proxy at `/btc-rpc`. Start `make devnet-btc`, then `npm run demo:btc`.
The browser never receives the Bitcoin Core username or password; the Vite proxy
adds Basic Auth before forwarding to `http://127.0.0.1:18443`. The demo includes
both a deterministic local-signer flow and an Xverse PSBT signing flow; see
[examples/bitcoin-deposit/README.md](examples/bitcoin-deposit/README.md) for the
Xverse Electrs facade, local funding steps, and troubleshooting notes.

## Troubleshooting

Errors thrown by the SDK use `ClearnetSdkError` with a stable `code`.

| Code | Common cause |
|---|---|
| `INVALID_INPUT` | Submit options are missing or have the wrong shape, an asset is unsupported, or Bitcoin transaction/PSBT finalization input is invalid. |
| `INVALID_ADDRESS` | EVM address, Solana public key, Solana mint, program ID, XRPL classic address, XRPL issued-currency key, or Clearnet account is invalid. |
| `INVALID_AMOUNT` | `amount` is not positive, has the wrong type/precision, or exceeds the chain limit (`uint256` for EVM, `uint64` for Solana/XRPL native drops, signed 64-bit satoshis for Bitcoin). |
| `INVALID_CONFIRMATIONS` | `minConfirmations` is negative, fractional, or an unsafe number. |
| `INVALID_REFERENCE` | `destination.ref` is not a 32-byte hex value, or Bitcoin received a non-zero reference. |
| `INVALID_TX_ID` | `txID` is not valid for the chain: EVM transaction hash or `txHash/logIndex`, Solana 64-byte signature, XRPL 64-hex hash, or Bitcoin 64-hex txid. |
| `MISSING_WALLET_ACCOUNT` | The EVM wallet account is missing/mismatched, or the Solana/XRPL signer is missing. |
| `CHAIN_MISMATCH` | The configured chain or network does not match the RPC or wallet network, such as an EVM chain ID mismatch or unsupported Bitcoin network. |
| `INSUFFICIENT_FUNDS` | Bitcoin only: confirmed depositor UTXOs cannot cover the deposit amount plus fee. |
| `TX_REVERTED` | A submitted approval/deposit transaction reverted, or XRPL rejected the payment engine result. |
| `RECEIPT_TIMEOUT` | Waiting for a receipt timed out or was aborted. |
| `RPC_ERROR` | The public RPC or wallet provider returned an unexpected error. |

When a transaction may already have been submitted, `ClearnetSdkError` can include
`txID`. Use that hash to let a user inspect or retry verification of the
submitted transaction.

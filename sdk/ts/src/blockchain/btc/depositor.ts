import {
  OutScript,
  SigHash,
  Transaction,
} from "@scure/btc-signer";

import { bytesToHex, concatBytes, hexToBytes } from "../../core/bytes.js";
import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  SubmitDepositOptions,
  VaultDepositor,
} from "../../core/types.js";
import { normalizeMinConfirmations as normalizeSharedMinConfirmations } from "../../core/validation.js";
import { decimalToBaseUnits } from "../amounts.js";
import {
  depositAddress,
  depositPayment,
  fundingAddress,
  fundingPayment,
  requireCompressedPublicKey,
} from "./address.js";
import { BITCOIN_DUST_THRESHOLD_SATS } from "./constants.js";
import { BitcoinRpcError } from "./types.js";
import type {
  BitcoinDepositorConfig,
  BitcoinPreparedDepositPsbt,
  BitcoinPsbtSignerInfo,
  BitcoinSigner,
  BitcoinSubmitDepositInput,
  BitcoinUnspent,
  BitcoinWalletAddressType,
  NormalizedBitcoinConfig,
} from "./types.js";
import { txIDFromTxid, requireBitcoinTxID } from "./txid.js";
import { compareUtxoForInputOrder, selectDepositUtxos } from "./utxo.js";
import {
  normalizeConfig,
  requireConfiguredSigner,
  requireBitcoinAmount,
  requireBitcoinAsset,
  requireDepositDestination,
  requireReference,
  requireSubmitDepositOptions,
  signerPublicKey,
} from "./validation.js";

export class BitcoinVaultDepositor
  implements VaultDepositor<BitcoinSubmitDepositInput>
{
  private readonly config: NormalizedBitcoinConfig;

  constructor(config: BitcoinDepositorConfig) {
    this.config = normalizeConfig(config);
  }

  async submitDeposit(
    input: BitcoinSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<string> {
    const submitOptions = requireSubmitDepositOptions(options);
    const fields = this.requireDepositFields(input);
    const signer = requireConfiguredSigner(this.config.signer);
    const publicKey = await signerPublicKey(signer);
    const prepared = await this.prepareUnsignedDepositTx({
      account: fields.account,
      amount: fields.amount,
      publicKey,
    });
    const tx = await this.signDepositTx(prepared, signer, publicKey);
    return this.broadcastTransaction(tx, submitOptions);
  }

  async prepareDepositPsbt(
    input: BitcoinSubmitDepositInput,
    wallet: BitcoinPsbtSignerInfo,
  ): Promise<BitcoinPreparedDepositPsbt> {
    const fields = this.requireDepositFields(input);
    const publicKey = requireWalletPublicKey(wallet);
    const addressType = requireWalletAddressType(wallet.addressType);
    const funding = fundingPayment(this.config.network, publicKey, addressType);
    if (wallet.address !== undefined && wallet.address !== funding.address) {
      throw new ClearnetSdkError(
        "INVALID_ADDRESS",
        "wallet address does not match wallet public key",
      );
    }
    const prepared = await this.prepareUnsignedDepositTx({
      account: fields.account,
      amount: fields.amount,
      publicKey,
      addressType,
    });
    return {
      psbtHex: bytesToHex(prepared.tx.toPSBT()),
      inputIndexesToSign: prepared.orderedUtxos.map((_, index) => index),
      unsignedTxID: txIDFromTxid(prepared.tx.id),
      fundingAddress: prepared.fundingAddress,
      depositAddress: prepared.depositAddress,
      feeSats: prepared.feeSats,
      selectedUtxos: prepared.orderedUtxos,
    };
  }

  async submitSignedDepositPsbt(
    psbtHex: string,
    options: SubmitDepositOptions = {},
  ): Promise<string> {
    const submitOptions = requireSubmitDepositOptions(options);
    const tx = finalizableTransactionFromPsbt(psbtHex);
    return this.broadcastTransaction(tx, submitOptions);
  }

  async verifyDeposit(
    txID: string,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const normalized = requireBitcoinTxID(txID);
    const minConf = normalizeMinConfirmations(minConfirmations);
    let raw;
    try {
      raw = await this.config.rpc.getRawTransaction(normalized);
    } catch (error) {
      if (error instanceof BitcoinRpcError && error.code === -5) {
        return "absent";
      }
      throw new ClearnetSdkError("RPC_ERROR", "btc: getrawtransaction", {
        cause: error,
      });
    }
    if (raw === null) {
      return "absent";
    }
    return minConf === 0 || (raw.confirmations > 0 && raw.confirmations >= minConf)
      ? "confirmed"
      : "pending";
  }

  async depositorAddress(): Promise<string> {
    return fundingAddress(
      this.config.network,
      await signerPublicKey(requireConfiguredSigner(this.config.signer)),
    );
  }

  depositAddress(account: string): string {
    return depositAddress(
      this.config.network,
      account,
      this.config.threshold,
      this.config.vaultPubkeys,
    );
  }

  txIDFromTxid(txid: string): string {
    return txIDFromTxid(txid);
  }

  private requireDepositFields(input: BitcoinSubmitDepositInput): {
    account: string;
    amount: bigint;
  } {
    const fields =
      input && typeof input === "object"
        ? (input as Partial<BitcoinSubmitDepositInput>)
        : {};
    const destination = requireDepositDestination(fields.destination);
    requireBitcoinAsset(fields.asset);
    requireReference(destination.ref);
    return {
      account: destination.account,
      amount: requireBitcoinAmount(decimalToBaseUnits(fields.amount, 8)),
    };
  }

  private async prepareUnsignedDepositTx(input: {
    account: string;
    amount: bigint;
    publicKey: Uint8Array;
    addressType?: BitcoinWalletAddressType;
  }): Promise<PreparedUnsignedDepositTx> {
    const funding = fundingPayment(
      this.config.network,
      input.publicKey,
      input.addressType,
    );
    const fundingAddressValue = funding.address;
    const utxos = await this.config.rpc.listUnspent(
      this.config.minFundingConfirmations,
      [fundingAddressValue],
    );
    const feeRate = await this.config.rpc.estimateSmartFeeSatPerVByte(
      this.config.feeTargetBlocks,
      this.config.fallbackFeeRateSatPerVByte,
    );
    const fundingScriptHex = bytesToHex(funding.script);
    const eligible = utxos.filter(
      (utxo) => utxo.scriptPubKey.toLowerCase() === fundingScriptHex,
    );
    const selected = selectDepositUtxos(
      eligible,
      input.amount,
      feeRate,
      input.addressType,
    );
    const tx = new Transaction({ version: 1 });
    const deposit = depositPayment(
      this.config.network,
      input.account,
      this.config.threshold,
      this.config.vaultPubkeys,
    );
    tx.addOutput({ script: deposit.script, amount: input.amount });
    const total = selected.utxos.reduce((sum, utxo) => sum + utxo.amountSats, 0n);
    const change = total - input.amount - selected.feeSats;
    if (change >= (this.config.dustThresholdSats ?? BITCOIN_DUST_THRESHOLD_SATS)) {
      tx.addOutput({ script: funding.script, amount: change });
    }

    const orderedUtxos = [...selected.utxos].sort(compareUtxoForInputOrder);
    for (const utxo of orderedUtxos) {
      const txInput = {
        txid: hexToBytes(utxo.txid, "txid"),
        index: utxo.vout,
        sequence: 0xffffffff,
        witnessUtxo: { amount: utxo.amountSats, script: funding.script },
      };
      tx.addInput(
        funding.redeemScript === undefined
          ? txInput
          : { ...txInput, redeemScript: funding.redeemScript },
      );
    }

    return {
      tx,
      orderedUtxos,
      fundingScript: funding.script,
      fundingHash: funding.pubkeyHash,
      fundingAddress: fundingAddressValue,
      depositAddress: deposit.address,
      feeSats: selected.feeSats,
    };
  }

  private async signDepositTx(
    prepared: PreparedUnsignedDepositTx,
    signer: BitcoinSigner,
    publicKey: Uint8Array,
  ): Promise<Transaction> {
    const scriptCode = OutScript.encode({
      type: "pkh",
      hash: prepared.fundingHash,
    });
    for (let index = 0; index < prepared.orderedUtxos.length; index += 1) {
      const utxo = prepared.orderedUtxos[index];
      if (utxo === undefined) {
        throw new ClearnetSdkError("RPC_ERROR", "btc: missing selected input");
      }
      const digest = prepared.tx.preimageWitnessV0(
        index,
        scriptCode,
        SigHash.ALL,
        utxo.amountSats,
      );
      const der = await signer.signDigest32(digest);
      requireDerSignature(der);
      prepared.tx.updateInput(
        index,
        {
          partialSig: [[
            publicKey,
            concatBytes(der, new Uint8Array([SigHash.ALL])),
          ]],
        },
        true,
      );
    }
    try {
      prepared.tx.finalize();
    } catch (error) {
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        "btc: transaction finalization failed",
        { cause: error },
      );
    }
    return prepared.tx;
  }

  private async broadcastTransaction(
    tx: Transaction,
    submitOptions: SubmitDepositOptions,
  ): Promise<string> {
    const txID = txIDFromTxid(tx.id);
    try {
      await this.config.rpc.sendRawTransaction(tx.hex);
      submitOptions.onSubmitted?.(txID);
      return txID;
    } catch (error) {
      if (isAlreadyKnown(error)) {
        submitOptions.onSubmitted?.(txID);
        return txID;
      }
      if (isMissingOrSpent(error)) {
        let raw;
        try {
          raw = await this.config.rpc.getRawTransaction(txID);
        } catch {
          raw = null;
        }
        if (raw?.txid.toLowerCase() === txID) {
          submitOptions.onSubmitted?.(txID);
          return txID;
        }
      }
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "btc: sendrawtransaction", {
        txID,
        cause: error,
      });
    }
  }
}

interface PreparedUnsignedDepositTx {
  tx: Transaction;
  orderedUtxos: readonly BitcoinUnspent[];
  fundingScript: Uint8Array;
  fundingHash: Uint8Array;
  fundingAddress: string;
  depositAddress: string;
  feeSats: bigint;
}

function isAlreadyKnown(error: unknown): boolean {
  return (
    (error instanceof BitcoinRpcError && error.code === -27) ||
    errorMessage(error).includes("already in block chain") ||
    errorMessage(error).includes("txn-already-known")
  );
}

function isMissingOrSpent(error: unknown): boolean {
  return (
    (error instanceof BitcoinRpcError && error.code === -25) ||
    errorMessage(error).includes("missingorspent") ||
    errorMessage(error).includes("missing inputs")
  );
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message.toLowerCase() : "";
}

function normalizeMinConfirmations(value: bigint | number): number {
  const normalized = normalizeSharedMinConfirmations(value);
  if (normalized > BigInt(Number.MAX_SAFE_INTEGER)) {
    throw new ClearnetSdkError(
      "INVALID_CONFIRMATIONS",
      "minConfirmations must be a non-negative safe integer",
    );
  }
  return Number(normalized);
}

function requireDerSignature(signature: Uint8Array): void {
  if (signature.length < 8 || signature[0] !== 0x30) {
    throw new ClearnetSdkError(
      "RPC_ERROR",
      "btc: signer returned invalid DER signature",
    );
  }
}

function requireWalletPublicKey(wallet: BitcoinPsbtSignerInfo): Uint8Array {
  if (!wallet || typeof wallet !== "object") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "wallet signer info is required",
    );
  }
  const publicKey = (wallet as Partial<BitcoinPsbtSignerInfo>).publicKey;
  if (!(publicKey instanceof Uint8Array) && typeof publicKey !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "wallet.publicKey must be a compressed secp256k1 public key",
    );
  }
  return requireCompressedPublicKey(publicKey, "wallet.publicKey");
}

function requireWalletAddressType(
  addressType: BitcoinPsbtSignerInfo["addressType"],
): BitcoinWalletAddressType {
  if (addressType === undefined) {
    return "p2wpkh";
  }
  if (addressType !== "p2wpkh" && addressType !== "p2sh") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "wallet.addressType must be p2wpkh or p2sh",
    );
  }
  return addressType;
}

function finalizableTransactionFromPsbt(psbtHex: string): Transaction {
  if (typeof psbtHex !== "string" || psbtHex.trim() === "") {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "signed PSBT must be an even-length hex string",
    );
  }
  try {
    const tx = Transaction.fromPSBT(hexToBytes(psbtHex.trim(), "psbt"));
    tx.finalize();
    return tx;
  } catch (error) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "signed PSBT must be finalizable",
      { cause: error },
    );
  }
}

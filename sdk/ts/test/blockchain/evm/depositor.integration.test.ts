import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { beforeAll, describe, expect, it } from "vitest";
import {
  createPublicClient,
  createWalletClient,
  defineChain,
  getAddress,
  http,
  parseAbiItem,
  parseEther,
  parseEventLogs,
  zeroAddress,
} from "viem";
import type { Address, Hash, Hex, Log } from "viem";
import { privateKeyToAccount } from "viem/accounts";

import { EvmVaultDepositor } from "../../../src/blockchain/evm/depositor.js";
import { custodyAbi, erc20Abi } from "../../../src/blockchain/evm/abi.js";
import { composeNonce, depositId } from "../../../src/blockchain/evm/depositId.js";
import type { EvmSubmitDepositInput } from "../../../src/core/types.js";

const RPC_URL = process.env.EVM_RPC_URL ?? "http://127.0.0.1:8545";
const CHAIN_ID = 31_337;
const DEPLOYER_PRIVATE_KEY =
  "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
const ACCOUNT_PRIVATE_KEYS = [
  DEPLOYER_PRIVATE_KEY,
  "0x0000000000000000000000000000000000000000000000000000000000000001",
  "0x0000000000000000000000000000000000000000000000000000000000000002",
] as const;
const DEPOSIT_REFERENCE =
  "0x3333333333333333333333333333333333333333333333333333333333333333" as Hash;
const anvil = defineChain({
  id: CHAIN_ID,
  name: "Anvil",
  nativeCurrency: { decimals: 18, name: "Ether", symbol: "ETH" },
  rpcUrls: { default: { http: [RPC_URL] } },
});
const deployer = privateKeyToAccount(DEPLOYER_PRIVATE_KEY);
const publicClient = createPublicClient({
  chain: anvil,
  transport: http(RPC_URL),
});
const walletClient = createWalletClient({
  account: deployer,
  chain: anvil,
  transport: http(RPC_URL),
});

type AnvilPublicClient = typeof publicClient;
type AnvilWalletClient = typeof walletClient;

describe("EvmVaultDepositor Anvil integration", () => {
  beforeAll(async () => {
    const chainId = await publicClient.getChainId();
    expect(chainId).toBe(CHAIN_ID);
  });

  it("deposits native ETH and verifies the deposit tx", async () => {
    const custodyAddress = await deployCustody(publicClient, walletClient);
    const depositor = new EvmVaultDepositor({
      publicClient,
      walletClient,
      walletAccount: deployer,
      custodyAddress,
      chainId: CHAIN_ID,
    });
    const amount = parseEther("0.01");
    const beforeBalance = await publicClient.getBalance({
      address: custodyAddress,
    });

    const result = await depositor.submitDeposit({
      destination: { account: deployer.address, ref: DEPOSIT_REFERENCE },
      asset: "",
      amount: "0.01",
    });
    const txHash = result.txHash as Hash;
    const afterBalance = await publicClient.getBalance({
      address: custodyAddress,
    });
    const receipt = await publicClient.getTransactionReceipt({
      hash: txHash,
    });

    expect(afterBalance - beforeBalance).toBe(amount);
    expect(
      hasDepositedLog(
        receipt.logs,
        custodyAddress,
        deployer.address,
        DEPOSIT_REFERENCE,
        zeroAddress,
        amount,
      ),
    ).toBe(true);
    await expect(
      depositor.chainDepositStatus(result.txHash, result.depositId, 1),
    ).resolves.toBe("confirmed");
  });

  it("approves an exact ERC-20 amount, deposits, and verifies the deposit tx", async () => {
    const custodyAddress = await deployCustody(publicClient, walletClient);
    const tokenAddress = await deployMockErc20(publicClient, walletClient);
    const depositor = new EvmVaultDepositor({
      publicClient,
      walletClient,
      walletAccount: deployer,
      custodyAddress,
      chainId: CHAIN_ID,
    });
    const amount = parseEther("25");

    await mine(
      publicClient,
      await walletClient.writeContract({
        address: tokenAddress,
        abi: erc20Abi,
        functionName: "mint",
        args: [deployer.address, amount],
      }),
    );
    const beforeBalance = await publicClient.readContract({
      address: tokenAddress,
      abi: erc20Abi,
      functionName: "balanceOf",
      args: [custodyAddress],
    });
    const startBlock = await publicClient.getBlockNumber();

    const result = await depositor.submitDeposit({
      destination: { account: deployer.address, ref: DEPOSIT_REFERENCE },
      asset: tokenAddress,
      amount: "25",
    });
    const allowance = await publicClient.readContract({
      address: tokenAddress,
      abi: erc20Abi,
      functionName: "allowance",
      args: [deployer.address, custodyAddress],
    });
    const txHash = result.txHash as Hash;
    const afterBalance = await publicClient.readContract({
      address: tokenAddress,
      abi: erc20Abi,
      functionName: "balanceOf",
      args: [custodyAddress],
    });
    const receipt = await publicClient.getTransactionReceipt({
      hash: txHash,
    });
    const approvalLogs = await publicClient.getLogs({
      address: tokenAddress,
      event: parseAbiItem(
        "event Approval(address indexed owner, address indexed spender, uint256 value)",
      ),
      args: { owner: deployer.address, spender: custodyAddress },
      fromBlock: startBlock,
      toBlock: receipt.blockNumber,
    });
    const approvalLog = approvalLogs.find(
      (log) => log.args.value === amount && log.transactionHash !== txHash,
    );
    if (approvalLog === undefined) {
      throw new Error("expected exact approval log before deposit");
    }
    const approvalReceipt = await publicClient.getTransactionReceipt({
      hash: approvalLog.transactionHash,
    });

    expect(allowance).toBe(0n);
    expect(approvalReceipt.status).toBe("success");
    expect(approvalReceipt.blockNumber < receipt.blockNumber).toBe(true);
    expect(afterBalance - beforeBalance).toBe(amount);
    expect(
      hasDepositedLog(
        receipt.logs,
        custodyAddress,
        deployer.address,
        DEPOSIT_REFERENCE,
        tokenAddress,
        amount,
      ),
    ).toBe(true);
    await expect(
      depositor.chainDepositStatus(result.txHash, result.depositId, 1),
    ).resolves.toBe("confirmed");
  });

  it("uses consecutive nonces and the requested key, and reports pending, confirmed and absent", async () => {
    const custodyAddress = await deployCustody(publicClient, walletClient);
    const depositor = new EvmVaultDepositor({
      publicClient,
      walletClient,
      walletAccount: deployer,
      custodyAddress,
      chainId: CHAIN_ID,
    });
    const input: EvmSubmitDepositInput = {
      destination: { account: deployer.address },
      asset: "",
      amount: "0.001",
    };
    const idFor = (nonce: bigint) =>
      depositId(BigInt(CHAIN_ID), custodyAddress, deployer.address, nonce);

    const first = await depositor.submitDeposit(input);
    const second = await depositor.submitDeposit(input);
    expect(first.depositId).toBe(idFor(0n));
    expect(second.depositId).toBe(idFor(1n));

    const keyed = await depositor.submitDeposit({ ...input, nonceKey: 7n });
    expect(keyed.depositId).toBe(idFor(composeNonce(7n, 0n)));

    const mined = await publicClient.getTransactionReceipt({
      hash: keyed.txHash as Hash,
    });
    const head = await publicClient.getBlockNumber({ cacheTime: 0 });
    const depth = head - mined.blockNumber + 1n;
    await expect(
      depositor.chainDepositStatus(keyed.txHash, keyed.depositId, depth + 2n),
    ).resolves.toBe("pending");
    await anvilRpc("evm_mine");
    await anvilRpc("evm_mine");
    await expect(
      depositor.chainDepositStatus(keyed.txHash, keyed.depositId, depth + 2n),
    ).resolves.toBe("confirmed");

    await expect(
      depositor.chainDepositStatus(first.txHash, second.depositId, 1),
    ).resolves.toBe("absent");
  });

  it("reports STALE_NONCE when another deposit consumed the nonce it read", async () => {
    const custodyAddress = await deployCustody(publicClient, walletClient);
    const depositor = new EvmVaultDepositor({
      publicClient,
      walletClient,
      walletAccount: deployer,
      custodyAddress,
      chainId: CHAIN_ID,
    });
    const amount = parseEther("0.001");

    await anvilRpc("evm_setAutomine", [false]);
    try {
      // Sits in the pool with nonce 0; getNonce at `latest` still reads 0.
      await walletClient.writeContract({
        address: custodyAddress,
        abi: custodyAbi,
        functionName: "deposit",
        args: [deployer.address, zeroAddress, amount, DEPOSIT_REFERENCE, 0n],
        value: amount,
        gas: 200_000n,
      });
      const stale = depositor
        .submitDeposit({
          destination: { account: deployer.address },
          asset: "",
          amount: "0.001",
        })
        .then(
          () => undefined,
          (error: unknown) => error,
        );
      const miner = setInterval(() => {
        void anvilRpc("evm_mine");
      }, 500);
      try {
        await expect(stale).resolves.toMatchObject({
          code: "STALE_NONCE",
          step: "deposit",
          nonceKey: 0n,
          nonce: 0n,
        });
      } finally {
        clearInterval(miner);
      }
    } finally {
      await anvilRpc("evm_setAutomine", [true]);
    }
  }, 60_000);

  it("sends no second approve when an earlier approve already covers the amount", async () => {
    const custodyAddress = await deployCustody(publicClient, walletClient);
    const tokenAddress = await deployMockErc20(publicClient, walletClient);
    const depositor = new EvmVaultDepositor({
      publicClient,
      walletClient,
      walletAccount: deployer,
      custodyAddress,
      chainId: CHAIN_ID,
    });
    const amount = parseEther("5");
    await mine(
      publicClient,
      await walletClient.writeContract({
        address: tokenAddress,
        abi: erc20Abi,
        functionName: "mint",
        args: [deployer.address, amount],
      }),
    );
    await mine(
      publicClient,
      await walletClient.writeContract({
        address: tokenAddress,
        abi: erc20Abi,
        functionName: "approve",
        args: [custodyAddress, amount],
      }),
    );
    const before = await publicClient.getTransactionCount({
      address: deployer.address,
      blockTag: "pending",
    });

    await depositor.submitDeposit({
      destination: { account: deployer.address },
      asset: tokenAddress,
      amount: "5",
    });

    const after = await publicClient.getTransactionCount({
      address: deployer.address,
      blockTag: "pending",
    });
    expect(after - before).toBe(1);
  });
});

async function anvilRpc(method: string, params: unknown[] = []): Promise<unknown> {
  const response = await fetch(RPC_URL, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
  });
  const body = (await response.json()) as { result?: unknown; error?: unknown };
  if (body.error !== undefined) {
    throw new Error(`${method}: ${JSON.stringify(body.error)}`);
  }
  return body.result;
}

async function deployCustody(
  publicClient: AnvilPublicClient,
  walletClient: AnvilWalletClient,
): Promise<Address> {
  const signers = ACCOUNT_PRIVATE_KEYS.map((key) =>
    getAddress(privateKeyToAccount(key).address),
  ).sort((left, right) => left.toLowerCase().localeCompare(right.toLowerCase()));
  const hash = await walletClient.deployContract({
    abi: custodyAbi,
    bytecode: artifactBytecode("Custody.bin"),
    args: [signers, 2n],
  });
  const receipt = await mine(publicClient, hash);
  if (receipt.contractAddress === null || receipt.contractAddress === undefined) {
    throw new Error("Custody deployment did not return a contract address");
  }
  return receipt.contractAddress;
}

async function deployMockErc20(
  publicClient: AnvilPublicClient,
  walletClient: AnvilWalletClient,
): Promise<Address> {
  const hash = await walletClient.deployContract({
    abi: erc20Abi,
    bytecode: artifactBytecode("MockERC20.bin"),
    args: ["Mock Token", "MOCK"],
  });
  const receipt = await mine(publicClient, hash);
  if (receipt.contractAddress === null || receipt.contractAddress === undefined) {
    throw new Error("MockERC20 deployment did not return a contract address");
  }
  return receipt.contractAddress;
}

async function mine(publicClient: AnvilPublicClient, hash: Hex) {
  return publicClient.waitForTransactionReceipt({ hash });
}

function artifactBytecode(fileName: "Custody.bin" | "MockERC20.bin"): Hex {
  const contents = readFileSync(
    resolve("../../pkg/blockchain/evm/artifacts", fileName),
    "utf8",
  ).trim();
  return contents.startsWith("0x") ? (contents as Hex) : `0x${contents}`;
}

function hasDepositedLog(
  logs: readonly Log[],
  custodyAddress: Address,
  account: Address,
  reference: Hash,
  asset: Address,
  amount: bigint,
): boolean {
  return parseEventLogs({
    abi: custodyAbi,
    eventName: "Deposited",
    logs: [...logs],
  }).some(
    (log) =>
      log.address.toLowerCase() === custodyAddress.toLowerCase() &&
      log.args.account.toLowerCase() === account.toLowerCase() &&
      log.args.depositReference.toLowerCase() === reference.toLowerCase() &&
      log.args.asset.toLowerCase() === asset.toLowerCase() &&
      log.args.amount === amount,
  );
}

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
  zeroAddress,
} from "viem";
import type { Address, Hex, Log } from "viem";
import { privateKeyToAccount } from "viem/accounts";

import { EvmVaultDepositor } from "../../../src/blockchain/evm/depositor.js";
import { custodyAbi, erc20Abi } from "../../../src/blockchain/evm/abi.js";

const RPC_URL = process.env.EVM_RPC_URL ?? "http://127.0.0.1:8545";
const CHAIN_ID = 31_337;
const DEPLOYER_PRIVATE_KEY =
  "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
const ACCOUNT_PRIVATE_KEYS = [
  DEPLOYER_PRIVATE_KEY,
  "0x0000000000000000000000000000000000000000000000000000000000000001",
  "0x0000000000000000000000000000000000000000000000000000000000000002",
] as const;
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

    const ref = await depositor.submitDeposit({
      account: deployer.address,
      asset: zeroAddress,
      amount,
    });
    const afterBalance = await publicClient.getBalance({
      address: custodyAddress,
    });
    const receipt = await publicClient.getTransactionReceipt({
      hash: ref.hash,
    });

    expect(afterBalance - beforeBalance).toBe(amount);
    expect(hasDepositedLog(receipt.logs, custodyAddress)).toBe(true);
    await expect(depositor.verifyDeposit(ref, 1)).resolves.toBe("confirmed");
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

    const ref = await depositor.submitDeposit({
      account: deployer.address,
      asset: tokenAddress,
      amount,
    });
    const allowance = await publicClient.readContract({
      address: tokenAddress,
      abi: erc20Abi,
      functionName: "allowance",
      args: [deployer.address, custodyAddress],
    });
    const afterBalance = await publicClient.readContract({
      address: tokenAddress,
      abi: erc20Abi,
      functionName: "balanceOf",
      args: [custodyAddress],
    });
    const receipt = await publicClient.getTransactionReceipt({
      hash: ref.hash,
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
      (log) => log.args.value === amount && log.transactionHash !== ref.hash,
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
    expect(hasDepositedLog(receipt.logs, custodyAddress)).toBe(true);
    await expect(depositor.verifyDeposit(ref, 1)).resolves.toBe("confirmed");
  });
});

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

function hasDepositedLog(logs: readonly Log[], custodyAddress: Address): boolean {
  return logs.some(
    (log) => log.address.toLowerCase() === custodyAddress.toLowerCase(),
  );
}

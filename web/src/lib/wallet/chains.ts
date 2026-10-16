import { defineChain } from "viem";
import { walletRpcUrl, type YellowNetwork } from "@/config/networks";

/**
 * The Yellow network as an EVM chain for wallets. Its chain ID is the EIP-712
 * domain chain ID, so wallets that enforce "domain chainId == active chain"
 * (MetaMask) can sign Clearnet transfers once this chain is active.
 */
export function yellowChain(n: YellowNetwork, origin: string) {
  return defineChain({
    id: n.chainId,
    name: `Yellow ${n.name}`,
    nativeCurrency: { name: "YELLOW", symbol: "YELLOW", decimals: 18 },
    rpcUrls: { default: { http: [walletRpcUrl(n, origin)] } },
    ...(n.explorer && { blockExplorers: { default: { name: "Yellow Explorer", url: n.explorer } } }),
    testnet: n.networkId !== "YNET",
  });
}

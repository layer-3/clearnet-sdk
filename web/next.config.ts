import type { NextConfig } from "next";
import { fileURLToPath } from "node:url";

// The app consumes the TypeScript SDK from ../sdk/ts (pnpm link:), so the
// bundler root is the repository root.
const repoRoot = fileURLToPath(new URL("..", import.meta.url));

const nextConfig: NextConfig = {
  reactStrictMode: true,
  agentRules: false,
  poweredByHeader: false,
  turbopack: {
    root: repoRoot,
    // One viem instance for wagmi, AppKit and the SDK.
    resolveAlias: { viem: "./node_modules/viem" },
  },
  outputFileTracingRoot: repoRoot,
  // WalletConnect pulls optional Node-only loggers; keep them out of the bundle.
  serverExternalPackages: ["pino", "pino-pretty", "thread-stream"],
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "X-Frame-Options", value: "DENY" },
        ],
      },
    ];
  },
};

export default nextConfig;

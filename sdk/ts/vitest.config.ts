import { configDefaults, defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => ({
  test: {
    include: mode === "integration"
      ? ["test/**/*.integration.test.ts"]
      : ["test/**/*.test.ts"],
    exclude: [
      ...configDefaults.exclude,
      ...(mode === "integration" ? [] : ["**/*.integration.test.ts"]),
    ],
    environment: "node",
    globals: true,
    restoreMocks: true,
    testTimeout: 120_000,
  },
}));

import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    restoreMocks: true,
    testTimeout: 120_000,
  },
});

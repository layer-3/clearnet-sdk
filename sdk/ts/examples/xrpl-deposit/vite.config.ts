import { defineConfig } from "vite";

export default defineConfig({
  server: {
    proxy: {
      "/xrpl-admin": {
        target: "http://127.0.0.1:5005",
        changeOrigin: true,
        rewrite: () => "/",
      },
    },
  },
});

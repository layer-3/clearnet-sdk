import assert from "node:assert/strict";
import { readFileSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const sdk = require("../package.json");
const path = require.resolve("@solana/web3.js/package.json");
const solana = JSON.parse(readFileSync(path, "utf8"));
const jayson = createRequire(path)("jayson/package.json");

// npm ignores a library's overrides in consuming apps. Bundle the patched
// v1 client, and make its manifest describe the dependency actually shipped.
assert.equal(jayson.version, sdk.overrides.jayson, "Run npm ci before packing");
assert.equal(jayson.version, "5.0.0");
solana.dependencies.jayson = jayson.version;
writeFileSync(path, `${JSON.stringify(solana, null, 2)}\n`);

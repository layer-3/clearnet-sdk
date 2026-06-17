# Devnet & blockchain integration tests

Full **deposit** and **withdrawal** flows per chain, exercised end-to-end
against real nodes. The tests are build-tagged `integration` and live next to
each adapter (`pkg/blockchain/<chain>/vault_integration_test.go`). Each
withdrawal test runs the whole *k-of-n* quorum in-process — it holds N local
`sign.KeySigner`s and drives `Pack → Validate → Sign → Merge → Submit →
VerifyExecution` itself, so no p2p mesh is needed.

## Run

```sh
make devnet        # anvil + bitcoind + rippled + solana-test-validator; blocks until all answer RPC
make integration   # go test -tags integration ./pkg/blockchain/...
make devnet-down
```

`make devnet` returns only once every node answers (the `devnet/wait` probe).
`make integration` then needs **no setup and no env** — every test
self-provisions against the devnet and is **idempotent**: each run uses fresh
keys / accounts / a freshly-deployed contract, so re-running is a clean run.
Only each node's funder persists (anvil account 0, the bitcoind coinbase
wallet, the XRPL genesis master).

## What each test provisions

- **EVM** — deploys a fresh `Custody` vault over N freshly-generated signer
  keys (funded from anvil account 0), deposits native ETH, then runs the quorum
  withdrawal.
- **BTC** — creates a legacy wallet, mines to maturity, generates a fresh vault
  + depositor, watch-imports their addresses, funds the depositor, deposits to
  the per-account P2WSH address, then runs the quorum withdrawal (mining to
  confirm between steps).
- **XRPL** — funds a fresh vault + depositor from the genesis master,
  `SignerListSet`s the vault over fresh signer keys, `TicketCreate`s a ticket,
  then deposits and runs the quorum withdrawal. Standalone rippled does not
  auto-close ledgers, so the test calls `ledger_accept` after each submit.
- **Solana** — the validator preloads the custody program **upgradeable** at its
  fixed id (`--upgradeable-program`), upgrade authority = the vendored
  `devnet/sol-upgrade-authority.json`. The test airdrop-funds the authority +
  depositor, `Initialize`s the Config once (idempotent; gated on the upgrade
  authority), deposits native SOL, then runs the quorum withdrawal. The Config
  PDA is a singleton, so the signer set is **fixed** across runs and only the
  withdrawalID is fresh — re-runs stay clean without a validator restart. The
  validator image is multi-arch (no `platform:` pin — the Agave validator needs
  AVX, which isn't emulable on Apple silicon).

## Optional overrides

Defaults target the devnet; override the endpoints if pointing elsewhere:

| env | default |
|---|---|
| `EVM_RPC_URL` / `EVM_DEPLOYER_KEY` | `http://127.0.0.1:8545` / anvil account 0 |
| `BTC_RPC_URL` / `BTC_RPC_USER` / `BTC_RPC_PASS` | `http://127.0.0.1:18443` / `sdk` / `sdk` |
| `XRPL_RPC_URL` | `http://127.0.0.1:5005` |

## Notes

- The tests double as the **executable spec** for the adapter interfaces: the
  withdrawal flow shows exactly how a custody node calls the SDK
  (`Pack`/`Validate`/`Sign`/`Merge`/`Submit`/`VerifyExecution`); the only piece
  left to the caller is collecting the quorum's signatures over its mesh.
- The rippled image is `linux/amd64` (pinned via `platform:`); on Apple
  silicon it runs under emulation — works, just slower to start.

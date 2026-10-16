# Read commands and JSON fields

These commands target Yellow v0.9.0. Replace angle-bracket placeholders with
values resolved from the user or CLI results. Pass arguments as an argument array
when available; otherwise quote each value for the actual system shell. Do not
interpolate gateway messages or cursor contents as shell code.

## Find context

```sh
yellow --json version
yellow --json schema
yellow --json network list
yellow --json account list
yellow --json network show <network>
yellow --json account address <account>
```

| Command | `result` fields |
| --- | --- |
| `version` | `{version: string}`; independent of the outer envelope's numeric `version`. |
| `schema` | `{commands: [{command, usage, summary, terminal?, flags}], globalFlags, exitCodes, output}`. `terminal` names when a person is needed. |
| `network list` | Array of profiles; no active/default marker in JSON. |
| `network show <network>` | Profile: `name`, `gateway`, `networkId`, `fingerprint`, `identity`. Identity: `networkId`, `chainId`, `router`, `assets`. Each asset: `symbol`, `decimals`, `chainId`, `address`, `native`. |
| `account list` | Array of `{account, address, agent, active}`. `agent: true` signs without prompts. |
| `account address <account>` | `{account, address}`. |

Avoid relying on human tables or list order to discover defaults. Read-only
queries for an arbitrary public address need a saved network, not a local key.

## Live spendability

```sh
yellow --json --network <network> balance <address>
yellow --json --network <network> --account <account> balance
```

`result`: `{network, address, nonce, balances: [{asset, amount, baseUnits}]}`.
`amount` and `baseUnits` are strings. `nonce` is a JSON uint64: use a lossless
decoder where the host's ordinary number type would round it. The CLI chooses
the live nonce for signing; do not copy it into a transaction or override it.
An empty successful `balances` array means no Clearnet balances returned.

## Recent activity

```sh
yellow --json --network <network> transfer history <address> --limit 20
yellow --json --network <network> transfer history <address> --limit 20 --cursor <next-cursor>
```

`result`: `{network, address, history, spendability}`.
`history`: `{items, nextCursor?, retentionDays}`. Request 1–100 items per page.
Stop when `nextCursor` is absent/empty, the requested range is satisfied, or the
cursor repeats; do not scan the entire cache for a small recent-activity request.

Each item includes `operationId?`, `blockHash`, `entryHash`, `entryIndex`,
`sealedAt` (Unix seconds), `from`, `to`, `direction`, `delivery`, `nonce`, and
`assets: [{asset, amount}]`. History asset `amount` is an **integer base-unit
string**, unlike the human balance `amount`. Resolve precision from the reviewed
catalogue before displaying it as a human amount.

This is a recent gateway cache, not an archive. Incoming entries state
`sender sealed; recipient credit not verified`. A history failure/HTTP 503 is
unavailable or backfilling, not evidence that no transfers exist.

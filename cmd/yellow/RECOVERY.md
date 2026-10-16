# Operator-assisted transfer recovery

This procedure is for a transfer saved as `pending` or `unresolved`. It preserves
the user's encrypted journal and reuses the original signed authorization.
It requires the companion gateway changes shipped for SDK v0.9.0: retained
operations expose `pendingUntil`, and gateway startup accepts `-backfill-from`.
Upgrade older gateways before using the replay path below.

## 1. Preserve and identify the operation

The user runs:

```sh
yellow operation list
yellow operation show <operation-id>
```

Give the operator the operation ID, original gateway URL, network ID, sender,
nonce and approximate submission time. Do not share keys, passphrases, unlock
files or the encrypted authorization. Keep a normal account backup and the
current journal. Do not delete a record, restore an older journal, change account
aliases or sign a replacement payment at another nonce.

## 2. Repair the original gateway

The operator preserves the existing SQLite database and its operation records,
and checks P2P connectivity, proposer availability, Registry/L1 access and verified
block ingestion. Restore those services before asking the user to retry.

If the network sealed the transfer but this gateway missed its block, restart
the gateway with its **existing deployment arguments and database**, adding:

```sh
-backfill-from <unix-seconds-before-the-original-submission>
```

This requests peer history from the selected time instead of resuming only from
the newest indexed block. Inspect startup logs for connected peers, successful
backfill and verification failures. Backfill uses the normal Registry/BLS checks;
never enable verification bypasses. Peers must still retain the required history.
The bounded backfill is not an archive restore: if its response limit or peer
retention excludes the block, restore a consistent gateway backup containing the
operation and verified block, or escalate to network support. Do not modify SQL
rows to fabricate a rejection or a verified receipt.

## 3. Reconcile or replay

The user runs `yellow operation show <operation-id>` again.

- `sealed`: the CLI saves the matching receipt. Continue to step 4.
- `rejected` with a definitive reason: the CLI saves that result. Continue to
  step 4; never fabricate this state from a missing block or advanced nonce.
- `pending`: wait and query again; do not replay yet.
- `unknown`: the original gateway must provide a positive `pendingUntil` that
  has ended and is no earlier than the saved submission time. Once the operator
  has repaired routing/indexing, run:

  ```sh
  yellow operation retry <operation-id>
  yellow operation show <operation-id>
  ```

The CLI checks the evidence itself, unlocks the account and reviews the original
nonce, recipient, amounts, fee cap and signature. Personal accounts require
confirmation; explicitly designated agent accounts retain their normal unattended
behavior. An ended gateway wait window is **not signature expiry or proof of
non-execution**. Only the identical authorization is replayed; the network's nonce
and operation identity prevent a second execution.

If the catalogue changed, review `yellow network trust-update <original-profile>`
first and carefully review the current meaning of the retained base units during
retry. This changes only the local submission guard after approval; the journal
retains its original catalogue and encrypted authorization. Missing assets or a
different signing domain cannot be bypassed.

If a query, trust check or replay fails, keep the operation and inspect it again.
Missing `pendingUntil` on an `unknown` result remains blocked. Repair or upgrade
the original gateway; neither elapsed wall-clock time alone nor a refused replay
establishes a final outcome. Repeated uncertain outcomes require operator/network
investigation, not a journal override.

## 4. Confirm recovery

`yellow operation show <operation-id>` must report `sealed` or `rejected` and the
original operation must remain visible in `yellow operation list`. For a sealed
operation, the live account nonce must advance past the retained nonce before the
CLI permits a new send. A changed nonce alone never resolves an unknown operation.
Prepare any new payment only after these checks pass.

## Automated verification

From the SDK checkout, with the companion gateway source available:

```sh
CLEARNET_SOURCE=/path/to/clearnet bash scripts/test-gateway-contract.sh
```

The local contract suite runs the actual gateway HTTP handlers with deterministic
proposer/index fixtures, the public Go SDK, the real CLI command handlers and the
TypeScript wallet calculations. It verifies uncertain submission, blocked new
sends and early replay, operator repair, ended-window evidence, identical replay,
receipt reconciliation, preserved encrypted journal and a subsequent send. Gateway
tests separately cover delayed verified indexing and index restart. This is local
integration validation; it does not certify a deployed network or browser wallet.

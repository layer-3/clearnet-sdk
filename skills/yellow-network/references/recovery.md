# Reconcile a transfer

Use this workflow for a pending, unresolved, interrupted, timed-out or saved
unsubmitted transfer. Reconcile before preparing any replacement payment.

```sh
yellow --json operation list
yellow --json operation show <operation-id>
```

`operation list` returns **all** saved operations across accounts and networks,
without contacting a gateway. It exits 0 regardless of listed states. Filter its
array locally using `account`/`sender` and `networkId`, not just profile name.
If the send produced no ID, compare pre-send and current journal entries. Do not
guess an ID when several entries could match. If the journal is unavailable or
missing after a possible submission, stop and reconcile with the original journal
and operator; an empty replacement journal is not proof of failure.

`operation show` queries the saved operation's **original gateway** and updates
the local journal. Global `--network`/`--account` do not redirect this lookup.
Use the full lowercase `0x` operation ID exactly as returned; it is not an L1
transaction hash. Query failure leaves the last saved state intact.

## Operation result

Fields: `operationId`, `account`, `network` (saved profile name), `networkId`,
`gateway`, `profileFingerprint`, `sender`, `nonce`, `state`, `createdAt`,
`updatedAt`, optional `submittedAt`, `receipt`, `idempotencyKey`, `reason`, `message`,
`nonceConsumedBy` (sealed activity for a different outgoing transfer at this nonce).
`reason` retains a submission error or the gateway's rejection explanation.
Pre-submit check failures exit 6 while retaining the prior journal state; a first
attempt stays unsubmitted. Use `operation retry` after correcting the cause. Cancellation after submission
retains pending/unresolved exit 3/4 and does not establish that nothing happened.
Journal timestamps are RFC3339 strings. Receipt fields include `operationId`,
`blockHash`, `entryIndex`, `sealedAt` (Unix seconds), `stateRoot`, `entryHash?`.
Signed payloads and the original payment's asset/recipient details are not
exposed by this view; retain the user's public payment intent alongside the ID
when available. `nonceConsumedBy` contains the other transfer's public history
entry, including its sender, recipient, nonce, assets and seal location.

| `state` | Meaning | Next step |
| --- | --- | --- |
| `sealed` | Matching receipt returned | Report receipt and completion. No replay. |
| `rejected` | Original gateway established definitive rejection | Report rejection. No replay; a new/changed payment is a separate decision. |
| `pending` | Accepted, execution not confirmed | Check the same ID later. Never send again. |
| `unresolved` | May have executed | Preserve the ID and journal; query the original gateway. |
| `signed_unsubmitted` | Authorization saved without a committed submission marker | Review a possible identical replay through the CLI. |

## Exit codes

| Exit | Interpretation | Response |
| --- | --- | --- |
| 0 | Successful read/setup, or sealed operation | A payment is complete only with `state: sealed` and its matching receipt. |
| 2 | Invalid input or local storage problem | Correct the specific issue; a busy lock may mean another command awaits approval. A reused idempotency key for a different transfer is also exit 2. |
| 3 | Pending | Save the operation ID; check status, never send a replacement. |
| 4 | Unresolved, saved unsubmitted, or blocked | Reconcile the journal before another send. |
| 5 | Definitively rejected | Report rejection; a changed payment needs user intent, not automatic fee escalation. |
| 6 | Gateway, trust, capability or history failure | Report unavailable information; preserve existing operation state. |
| 130 | Cancelled | If a send/replay had started, inspect the journal before assuming nothing happened. |

`operation list` exits 0 even when listed states are unresolved. Inspect the
operation state as well as the exit code. Missing/malformed output, a transport
failure or an unexpected exit after launching a send/replay also requires journal
reconciliation. Do not mask exit codes in a pipeline. The only safe rerun of a
send is the identical command with its original `--idempotency-key`.

## Waiting for an outcome

For a status check, query once and report the state. When the user asks to wait,
use their requested duration as the deadline; for an unspecified brief wait,
default to 30 seconds. Poll the same operation with increasing delays, capped
so another check fits within the remaining time. There is no fixed check count.

Measure elapsed time with a monotonic clock. Bound both each subprocess's total
execution timeout and its CLI `--timeout` by the remaining budget, including the
Info preflight and status request. Do not sleep or start another query after the
deadline. Stop early on sealed, rejected, cancellation or a query/trust failure.
A timeout leaves the last observed state intact; it does not authorize replay.

At the deadline, report the last observed state and the exact status command.
An unresolved outcome stays unresolved. Use scheduling only when the user asks
for a later/background check and the host supports it; a bounded wait in this
conversation does not require a scheduled task. If the host cannot sustain the
requested wait, explain that limit rather than silently shortening the wait or
claiming to keep monitoring. Never run an unbounded loop.

## Replay

Replay only while completing a payment the user asked for, or when they ask for
recovery; a status request does not authorize it. For an agent account, run it
yourself. For a personal account, give the person the command to run in their
own terminal.

```sh
yellow --json operation retry <operation-id>
```

Replay unlocks and submits the **identical saved authorization**: same nonce,
recipient, assets, fee, signature and operation ID. It does not create a fresh
payment. Sealed/rejected operations cannot replay. Pending operations cannot
replay. `operation retry` has no `--wait`; check status afterward.

For an unsubmitted record, the CLI requires original-gateway `unknown` status.
For a previously submitted record, it requires original-gateway `unknown` plus
the optional `pendingUntil` admission-window evidence: positive Unix seconds,
already ended and no earlier than the last local submission time. **Unknown
alone is insufficient.** The public operation view does not expose this raw
evidence; let `operation retry` enforce the checks rather than fabricating them
or calling private endpoints. Missing evidence means stop and report blocked.
An ended window permits identical replay; it does not revoke the signature or prove non-execution.

Trust changes require human review via `network trust-update <original-profile>`.
An explicitly reviewed catalogue change on the same gateway/signing domain can
support replay; changed domain, missing asset, unsupported status or unavailable
gateway cannot be bypassed. The CLI shows any catalogue changes and the retained
base units during replay review.

Never clear the journal, reimport a key under another alias, restore an older
backup, change gateway, or re-sign at a fresh nonce to unblock an uncertain
payment. A changed balance or advanced nonce does not establish this payment's
outcome. After a refusal, preserve the state and explain the specific blocker.

The CLI and web wallet have separate local journals. Resolve uncertain operations
in the original tool before sending from the same account through another wallet,
browser or machine. If the live nonce advanced past a rejected authorization,
the CLI automatically checks the original gateway's operation status and recent
sealed history. A matching seal for the original payment updates its state and
stops a new send for review. A different outgoing transfer at that exact sender
nonce is retained as `nonceConsumedBy`, clearing this rejected operation's block
without deleting history or requiring acknowledgement. `operation show` can do
the same reconciliation; an operation with `nonceConsumedBy` remains rejected
and exits 5, not sealed or paid.

The lookup covers up to 1,000 recent activity entries. Missing/expired evidence,
index outages or contradictory reports keep sends blocked. Report the blocker
and ask the original gateway operator to restore verified history. Do not work
around it by signing again or editing the journal; an advanced nonce and an
empty history are not proof of non-execution.

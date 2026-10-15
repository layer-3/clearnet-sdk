# Yellow CLI

Your Clearnet account, in your terminal. Yellow manages encrypted local accounts,
live balances, transfers, recent activity and safe transfer recovery. It uses the
public gateway API.

## Install

Install Yellow v0.9.0 with Go:

```sh
go install github.com/layer-3/clearnet-sdk/cmd/yellow@v0.9.0
```

Go 1.25.6 or newer is required. Add Go's `bin` directory to your shell's PATH,
then run `yellow help`.

You can also download binaries for Linux, macOS and Windows on amd64 and arm64
from the [v0.9.0 release](https://github.com/layer-3/clearnet-sdk/releases/tag/v0.9.0).
Use the included `SHA256SUMS` to verify your download.

## First run

For development without a live network, start the SDK's [local clearbox
gateway](../clearbox/README.md). It shares balances and history with the web app
and generates a local test account on first initialization. Use that key only for local tests.

Use the gateway URL and four-byte network ID supplied by your network operator.
The following URL is an example, not a running gateway:

```sh
yellow network add testnet --gateway https://gateway.example.org --network-id YTES
yellow account create personal
yellow account backup personal --out personal.backup.json
yellow balance
```

Yellow shows the gateway's EIP-712 network identity, router, asset precision and
fingerprint for approval. The first network and account become the defaults.
Network IDs are case sensitive. Remote gateways require HTTPS. Explicitly adding
an HTTP loopback gateway (for example `http://127.0.0.1:8080`) creates a local
development profile; remote HTTP is refused.

Account creation displays 24 numbered recovery words once, directly on your
terminal. Write them down in order, then confirm you saved them. Set an encryption
passphrase of at least 12 characters. An encrypted backup includes the key and
saved operation journal; keep it separate from its passphrase and refresh it
after sending transfers. Recovery words restore the account identity but cannot
restore a lost operation journal.

Imported mnemonics use `m/44'/60'/0'/0/0`, Ethereum index zero, with no BIP-39
extension passphrase. `account import` asks whether you have recovery words, a
raw secp256k1 key, or an encrypted Yellow backup. Secret input is hidden. Importing
a backup asks for its existing passphrase and a new local passphrase, preserving
all saved authorizations and unresolved states. Duplicate local signing addresses
are refused so two aliases cannot bypass nonce reservations.

## Import an account by private key

To bring an existing Ethereum account into Yellow:

```sh
yellow account import personal
```

Choose `key` when asked for the import type, paste the 64-character hexadecimal
private key into the hidden prompt, then set a Yellow encryption passphrase.
Keys with or without a leading `0x` are accepted. The account address stays the
same. Yellow encrypts the key locally; your other wallet's password is not needed.
Run `yellow account address personal` to verify the address and create an
encrypted backup with `yellow account backup personal --out personal.backup.json`.

The same steps work inside `yellow shell` using `account import personal`.
Enter the key only at the hidden prompt, never as part of the command.

## Import your account into MetaMask

Reveal the private key for a saved account:

```sh
yellow account export-key personal
```

Omit the name to use your active account, or run `account export-key personal`
inside `yellow shell`. Enter your Yellow encryption passphrase at the hidden
prompt, check the displayed account and address, then answer `yes` to reveal the
64-character Ethereum private key. For accounts created from recovery words,
Yellow exports the key at `m/44'/60'/0'/0/0`, matching your saved address.

In MetaMask, open the account selector, choose **Add wallet → Import an account**,
and import the key as an Ethereum account. Verify that its address matches
`yellow account address personal`. See
[MetaMask's private-key import instructions](https://support.metamask.io/start/use-an-existing-wallet/#import-using-a-private-key).
Importing the key gives MetaMask the same account identity; use Yellow to view
your Clearnet balances and send Clearnet transfers.

Anyone with this key can control your account. The key remains in your terminal
scrollback after export; keep your terminal private and clear your scrollback
after importing it. Yellow displays it directly in the terminal, never in
redirected stdout, stderr, JSON or a file, and does not copy it to the clipboard.
`--json` and noninteractive input are refused for this command.

Keep your encrypted Yellow backup: private-key import does not restore the
operation journal, and a private key imported into MetaMask is separate from
MetaMask's original recovery phrase. Yellow's encrypted backup is its own format
and cannot be imported through MetaMask's JSON-file option.

## Interactive shell

Open Yellow's interactive prompt:

```sh
yellow shell
```

The prompt shows your active network and account. Commands, flags, saved accounts,
addresses, assets and operation IDs appear as you type, with short descriptions.
You can begin with `help`, `network`, `account` or `balance`. No shell-completion
setup is needed inside this prompt.

```text
yellow> account list
yellow> network use testnet
yellow> balance
yellow> transfer send <recipient-address> --asset YELLOW=1.25 --max-fee 0.01
yellow> operation list
yellow> exit
```

Enter commands with or without the `yellow` prefix. All commands use the same
validation, encrypted storage, journal and transfer approval as direct CLI calls.
Keys and passphrases are entered in separate hidden prompts. Selecting a suggestion
only fills in text; it never runs a command or approves a transfer.

| Key | Action |
| --- | --- |
| Tab | Complete the current argument |
| ↓ | Select a suggestion |
| ↑ / ↓ with a selection | Navigate suggestions |
| Enter with a selection | Accept the suggestion |
| Esc | Leave suggestion selection |
| Enter without a selection | Run the typed command |
| ↑ without a selection | Recall a command from this session |
| ↓ while recalling history | Move forward, then restore your draft |
| Ctrl+C with typed input | Clear the current line |
| Ctrl+C with an empty line | Cancel and close the shell |
| Ctrl+D with an empty line, or `exit` | Close the shell |

History holds up to 100 recognized commands for the current session and is never
written to disk. Hidden secret input never enters history. Suggestions use a
snapshot of public local metadata, refreshed after each command; completion never
calls the gateway or unlocks an account. Backup output paths also support local
file completion. An initial `--network` or `--account` selects the session context;
`network use` and `account use` update that context and your saved defaults.

Quote arguments that contain spaces, for example:

```text
account backup personal --out 'my account backup.json'
```

The prompt parses arguments without executing a system shell. Variables, pipes,
redirection and command substitution are treated as literal input. Pasted text
never runs automatically; pasted line breaks become spaces. Results remain in
normal terminal scrollback, and the prompt adapts when you resize the terminal.
`--no-color` and `NO_COLOR` disable colors. Interrupting the process cancels
any running command and closes the shell; saved transfer authorizations remain in
the journal.

## Completion in your terminal shell

Completion discovers commands, options, saved networks, account names, operation
IDs and reviewed asset symbols. Mistyped commands suggest nearby spellings.
Completions use saved metadata and never unlock an account or make gateway calls.
See `yellow completion --help` for installation details.

```sh
# Bash: enable for this shell
source <(yellow completion bash)

# Zsh: enable for this shell
autoload -Uz compinit && compinit
source <(yellow completion zsh)

# Fish: persist for future shells
mkdir -p ~/.config/fish/completions
yellow completion fish > ~/.config/fish/completions/yellow.fish
```

PowerShell: run `yellow completion powershell > yellow.ps1`, then dot-source that
file from your PowerShell profile. Persist Bash/Zsh completion by adding the
corresponding source line to your shell's startup file.

## Send a transfer

```sh
yellow transfer send 0x00000000000000000000000000000000000000bb \
  --asset YELLOW=1.25 --max-fee 0.01

# More than one asset; optional bounded wait for a seal
yellow transfer send <recipient-address> \
  --asset ETH=0.1 --asset YELLOW=5 --max-fee 0 --wait 20s
```

Choose a full, nonzero Ethereum address or canonical
`yellow://ynet/user/<lowercase-address>` URI. Only user recipients are supported.
Enter each asset once and spell its symbol as shown by `network show`. Amounts are
converted exactly using reviewed asset precision. Extra fractional digits,
scientific notation, negative values, duplicates, zero transfer amounts and
uint256 overflow are refused. `--max-fee` is required and denominated in YELLOW:
`0` authorizes **zero fees**.

Yellow asks you to unlock, then shows the network, gateway, full sender and
recipient, sorted assets, base units, live nonce and fee cap for approval. It signs
once, encrypts and durably saves the exact authorization, then sends one POST.
There are no automatic submission retries or redirects. `--wait` polls status; it
never resubmits. Human results go to stdout, while prompts and diagnostics go to
stderr. For personal accounts, secret input and approval require an interactive
terminal, including with `--json`. Agent accounts sign without prompts; see
[Scripts and agents](#scripts-and-agents).

## Understand the result

| State | Meaning | Next action |
| --- | --- | --- |
| `sealed` | Gateway returned a verified seal receipt. | Inspect the displayed block hash and entry index. |
| `pending` | Gateway accepted the authorization; execution is unconfirmed. | `yellow operation show <id>` |
| `unresolved` | Timeout, unknown response or crash; it may have executed. | Query the original gateway. Do not send again at a fresh nonce. |
| `signed_unsubmitted` | Signed bytes are saved; no submission marker was committed. | Review and unlock with `yellow operation retry <id>`. |
| `rejected` | A definitive gateway rejection was established. | Review the rejected operation before preparing a new transfer. |

All unresolved authorizations remain in the journal and block new sends by that
account on the same network, even when another gateway profile is selected.
An advanced nonce, changed balance, or another gateway's `unknown` cannot resolve
an operation. Yellow never silently deletes the journal or re-signs an unresolved
transfer at a new nonce.

```sh
yellow operation list
yellow operation show <operation-id>
yellow operation retry <operation-id>
```

Status and replay always use the original gateway URL. A previously submitted
operation can only replay when the **original gateway** reports `unknown` and
provides `pendingUntil`, the original admission window's end in Unix seconds,
already in the past and no earlier than the recorded submission time. Yellow
refuses replay when the gateway does not provide
this information, when the query fails, or while the operation is pending. A sealed or rejected operation is
never replayed. Replay requires unlock and explicit confirmation and sends exactly
the retained nonce, recipient, assets, fee and signature.

Gateway status is trusted through HTTPS and your reviewed signing-domain profile;
Yellow does not independently verify validator quorum proofs. A catalogue change
requires review with `yellow network trust-update <name>`. That explicit review
also allows recovery against the same original gateway and signing domain; the
operation keeps its original fingerprint and authorization. Replay review shows
any catalogue differences again and uses the current reviewed precision to display
the retained base units. An asset removed from the catalogue prevents replay.

## Read balances and recent history

```sh
yellow balance
yellow balance <address>
yellow transfer history --limit 20
yellow transfer history --limit 20 --cursor <next-cursor>
```

Balance is live Clearnet spendability. History is a cursor-paginated recent gateway
cache, **not an archive**; output includes retention and the next-page cursor.
Incoming history says “sender sealed; recipient credit not verified.” Use live
balance to determine whether credit is spendable. A history HTTP 503 means the
index is unavailable or backfilling, never “no transfers.”

## Commands and options

```text
yellow network add <name> --gateway <url> --network-id <id> [--fingerprint <fp>]
yellow network list
yellow network show [name]
yellow network use <name>
yellow network trust-update <name> [--fingerprint <fp>]
yellow account create <name> [--agent]
yellow account import <name>
yellow account list
yellow account address [name]
yellow account use <name>
yellow account backup <name> --out <path>
yellow account export-key [name]
yellow balance [address]
yellow transfer send <recipient> --asset SYMBOL=AMOUNT --max-fee <YELLOW-amount>
                     [--asset ...] [--from <name>] [--wait <duration>]
                     [--dry-run] [--idempotency-key <key>]
yellow transfer history [address] [--limit <n>] [--cursor <cursor>]
yellow operation list
yellow operation show <operation-id>
yellow operation retry <operation-id>
yellow completion bash|zsh|fish|powershell
yellow shell
yellow schema
yellow version
yellow help
```

Global options: `--network`, `--account`, `--json`, `--timeout` (per gateway request,
default `15s`), `--no-color`. `NO_COLOR` also disables color. Colors are disabled
when output is redirected. Names may contain letters, digits, dashes and
underscores, must start with a letter or digit, and are at most 48 characters.
No mnemonic, private key, unlock
passphrase or nonce option exists, in arguments or environment variables.

Use `--network` and `--account` for one invocation; `network use` and `account use`
change your defaults. `--from` is an account selector for sending and must agree
with `--account` when both are provided.

## JSON and exit codes

Successful command results have this envelope:

```json
{"version":1,"result":{"operationId":"0x…","state":"pending","network":"testnet"}}
```

Failures use `{"version":1,"error":"…","exitCode":2}`. Every executed data command
returns one JSON object; help and generated completion scripts remain text.
Keys, recovery words, passphrases and signed payloads are never emitted as JSON. Prompts
remain on stderr. A personal account's approval cannot be bypassed by `--json` or
piped input. `yellow --json schema` describes every command and option.

| Exit | Meaning |
| --- | --- |
| 0 | Successful read, setup, backup, or sealed transfer |
| 2 | Input or local storage error |
| 3 | Pending transfer |
| 4 | Unknown outcome, unresolved authorization or blocked replay |
| 5 | Definitive rejection |
| 6 | Network, trust, capability or index failure |
| 130 | Cancellation |

## Scripts and agents

AI agents and scripts can use Yellow on their own. Give them an **agent
account**, a separate account that signs without prompts:

```sh
yellow --json account create assistant --agent
```

It has no recovery words and no passphrase prompt. Yellow encrypts its key with a
generated passphrase saved next to it (`accounts/assistant.unlock`, mode 0600), so
anyone, or any program, that can read your Yellow folder can spend its balance.
**Its balance is the limit of what it can spend.** Fund it from your personal
account with only what the agent may use, and move the rest back when you are done:

```sh
yellow transfer send <agent-address> --asset USDC=100 --max-fee 1
```

An agent account is never selected as the default when it is created; agents are
used by passing `--account` (or explicitly with `account use`). Your personal
accounts keep their passphrase and approval prompts,
and no agent can unlock them. `account backup` of an agent account asks you for a
new passphrase and writes an ordinary encrypted backup, which restores as a
personal account. `account export-key` works as for any account.

Everything an agent needs runs without a terminal:

```sh
yellow --json schema                       # every command, option and exit code
yellow --json network add testnet --gateway <url> --network-id YTES --fingerprint <fp>
yellow --json account create assistant --agent
yellow --json --account assistant balance
yellow --json --account assistant transfer send <recipient> \
  --asset USDC=50 --max-fee 1 --dry-run   # exact preview; nothing signed
yellow --json --account assistant transfer send <recipient> \
  --asset USDC=50 --max-fee 1 --idempotency-key invoice-42 --wait 20s
yellow --json operation show <operation-id>
yellow --json operation retry <operation-id>
```

- `--fingerprint` pins the gateway fingerprint your operator gave you, instead of
  the interactive review. A different fingerprint saves nothing. Without it, a
  noninteractive `network add` fails and reports the gateway's fingerprint, so you
  can confirm it with the operator.
- `--dry-run` returns the exact transfer with the live nonce, without unlocking,
  signing, saving or sending. It works for personal accounts too.
- `--idempotency-key` makes a repeated command safe. If the account already saved
  a transfer with that key on this network, Yellow returns that operation and
  signs nothing. Reusing a key for a different transfer is refused. Use one key
  per intended payment, for example after a tool timeout.
- The unresolved-operation guard applies to agent accounts as well: an unknown
  outcome blocks new sends until it is reconciled.

The [Yellow Network skill](https://github.com/layer-3/clearnet-sdk/tree/v0.9.0/skills/yellow-network) teaches agents
which commands to use, how to interpret JSON and operation states, and how to
recover uncertain transfers without paying twice. It uses direct commands and
loads task-specific references only when needed.

Each CLI release archive includes `skills/yellow-network`. From the extracted
archive or the SDK repository root, install the skill for Codex on macOS/Linux:

```sh
mkdir -p "$HOME/.agents/skills"
cp -R skills/yellow-network "$HOME/.agents/skills/"
```

On Windows, copy the entire `skills/yellow-network` folder into
`%USERPROFILE%\.agents\skills`. If a previous copy exists, replace that skill
folder with the new copy. Codex detects skills automatically; restart it if the
skill does not appear. See the [official skill documentation](https://learn.chatgpt.com/docs/build-skills#where-codex-loads-local-skills).
For other agents supporting `SKILL.md`, install the same folder in their skill
directory, preserving `references` and `agents`.
The skill uses the installed `yellow` executable on PATH.

Try “Use $yellow-network to show my Clearnet balances” or “Use $yellow-network
to send 5 USDC to 0x… from my agent account.” With an agent account, the agent
sends, waits and reconciles on its own.

Direct commands do not open the interactive shell. Data results and errors use the
versioned JSON envelopes described above; diagnostics stay on stderr. Check both
the exit code and operation state. Exit 3 means pending and exit 4 means an unknown
outcome; neither is a sealed transfer. Preserve the operation ID and inspect it
before attempting another transfer.

`yellow shell --json` and `yellow account export-key --json` are refused. For
personal accounts, creation, import, backup, sending and replay require a terminal
for secrets and explicit approval, including when their results use JSON. No key
or passphrase is accepted through command arguments or environment variables.

## Local data and recovery

Files live in your OS user configuration directory under `yellow`: typically
`~/.config/yellow` on Linux, `~/Library/Application Support/yellow` on macOS, or
`%AppData%\yellow` on Windows. Configuration contains public profile metadata;
account files contain encrypted secrets and encrypted signed authorizations,
plus nonsecret operation metadata for inspection without unlock.

Encryption format v1 uses scrypt (N=131072, r=8, p=1), a random 32-byte salt,
AES-256-GCM and random 12-byte nonces. Authorization encryption authenticates its
operation ID, sender, nonce, original profile and creation time. Unix directories
use mode 0700 and files 0600. Windows stores files inside your user profile;
Windows permissions follow that directory's ACLs. Atomic writes sync data and
replace the destination, with directory sync on Unix and write-through replacement
on Windows. Backups commit a fully written file exclusively, refusing overwrite.
OS file locks serialize local commands, preventing conflicting nonce reservations,
and release on process exit or crash. Unlock holds keys only during the command;
Go cannot guarantee erasure of every temporary copy of a secret in memory.

If the gateway is unavailable, the last durable state remains intact. Keep both
local account files and current encrypted backups. A restored, older backup cannot
reconstruct authorizations signed after that backup: reconcile those transfers
with your original local journal and gateway before sending from another machine.

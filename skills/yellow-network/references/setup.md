# Setup, accounts and gateway trust

This skill targets Yellow v0.9.0. If the executable is unavailable, use the user's
installation method; with Go 1.25.6+:

```sh
go install github.com/layer-3/clearnet-sdk/cmd/yellow@v0.9.0
```

`yellow --json schema` lists every command, option and exit code, and which
commands need a person at a terminal.

## Add a network

Obtain the HTTPS gateway URL, the case-sensitive four-byte network ID and the
gateway **fingerprint** from the user or network operator. Then:

```sh
yellow --json network add <name> --gateway <operator-https-url> --network-id <id> --fingerprint <fingerprint>
```

A mismatch saves nothing (exit 6). Without `--fingerprint` the command fails
(exit 2) and reports the gateway's fingerprint. Show it to the user and use it
only after they confirm it matches what their operator published; never pin a
fingerprint you only read from the CLI or gateway yourself.

A catalogue change blocks sends until reviewed. Show the change to the user;
run `network trust-update <name> --fingerprint <new-fingerprint>` only with a
fingerprint they confirmed. Never switch gateways to make an error pass.

## Agent accounts

```sh
yellow --json account create <name> --agent
```

Returns `{account, address, agent: true}`. No recovery words or prompts. It
never becomes the default account; always pass `--account <name>`. Tell the
user the address, that its balance is the limit of what you can spend, and that
they fund it themselves. Create one only when the user wants you to make
payments.

## Personal accounts and secrets: the person's terminal only

Give these commands to the person; never run or drive them:

```sh
yellow account create <name>
yellow account import <name>
yellow account backup <name> --out <new-backup-path>
yellow account export-key [name]
```

Create **or** import, not both. Import accepts recovery words
(`m/44'/60'/0'/0/0`, no BIP-39 extension passphrase), a raw secp256k1 private
key (64 hex digits, optional `0x`) or an encrypted Yellow backup. Duplicate
signing addresses are refused. New passphrases need 12+ characters. A recovery
phrase or key restores identity, not the operation journal; keep current
encrypted backups. Backing up an agent account asks the person for a new
passphrase and restores as a personal account.

Export prints the key on the terminal only and refuses JSON; never capture it.

Names: 1–48 letters, digits, dashes or underscores, starting with a letter or
digit. HTTP gateways are allowed only for explicitly added loopback
development profiles. No secrets are accepted through flags or environment.

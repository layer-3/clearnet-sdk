package yellow

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
	"github.com/layer-3/clearnet-sdk/pkg/wallet"
	"github.com/spf13/cobra"
)

func secretKey(sec secret) (*ecdsa.PrivateKey, error) {
	switch sec.Kind {
	case "mnemonic":
		return wallet.DeriveKey(sec.Value)
	case "key":
		k, err := crypto.HexToECDSA(sec.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid secp256k1 private key")
		}
		return k, nil
	default:
		return nil, fmt.Errorf("unsupported account secret")
	}
}
func closeKey(k *ecdsa.PrivateKey) {
	if k != nil {
		k.D.SetInt64(0)
	}
}
func (a *app) newPassphrase() ([]byte, error) {
	pass, err := a.prompt.secret("New encryption passphrase (at least 12 characters)")
	if err != nil {
		return nil, err
	}
	if utf8.RuneCount(pass) < 12 {
		clear(pass)
		return nil, fail(ExitInput, "choose a passphrase of at least 12 characters")
	}
	again, err := a.prompt.secret("Repeat passphrase")
	defer clear(again)
	if err != nil {
		clear(pass)
		return nil, err
	}
	if string(pass) != string(again) {
		clear(pass)
		return nil, fail(ExitInput, "passphrases do not match")
	}
	return pass, nil
}
func (a *app) unlockKey(ac *accountFile) (*ecdsa.PrivateKey, []byte, func(), error) {
	pass, err := a.prompt.secret("Unlock account")
	if err != nil {
		return nil, nil, nil, err
	}
	defer clear(pass)
	sec, key, err := decryptSecret(pass, ac)
	if err != nil {
		return nil, nil, nil, err
	}
	private, err := secretKey(sec)
	sec.Value = ""
	if err != nil {
		clear(key)
		return nil, nil, nil, err
	}
	if crypto.PubkeyToAddress(private.PublicKey).Hex() != ac.Address {
		clear(key)
		closeKey(private)
		return nil, nil, nil, fmt.Errorf("encrypted key does not match the account address")
	}
	return private, key, func() { closeKey(private); clear(key) }, nil
}
func (a *app) unlock(ac *accountFile) (sign.Signer, []byte, func(), error) {
	private, key, close, err := a.unlockKey(ac)
	if err != nil {
		return nil, nil, nil, err
	}
	signer := sign.NewKeySignerFromECDSA(private)
	return signer, key, func() { _ = signer.Close(); close() }, nil
}
func (a *app) ensureNewAccount(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	path, _ := a.store.accountPath(name)
	if _, err := os.Lstat(path); err == nil {
		return fail(ExitInput, "account %q already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

type restoredOperation struct {
	op     operation
	signed *client.SignedTransfer
}

func (a *app) saveNewAccount(name string, sec secret, restored []restoredOperation) error {
	private, err := secretKey(sec)
	if err != nil {
		return err
	}
	defer closeKey(private)
	address := crypto.PubkeyToAddress(private.PublicKey).Hex()
	names, err := a.store.accounts()
	if err != nil {
		return err
	}
	for _, n := range names {
		ac, err := a.store.account(n)
		if err != nil {
			return err
		}
		if strings.EqualFold(ac.Address, address) {
			return fail(ExitInput, "this address is already saved as %s; use yellow account use %s", n, n)
		}
	}
	pass, err := a.newPassphrase()
	if err != nil {
		return err
	}
	defer clear(pass)
	encrypted, key, err := encryptSecret(pass, address, sec)
	if err != nil {
		return err
	}
	defer clear(key)
	ac := &accountFile{Version: 1, Address: address, Secret: encrypted, Operations: []operation{}}
	for _, saved := range restored {
		op := saved.op
		op.Account = name
		if op.Sender != address {
			return fmt.Errorf("backup authorization does not belong to this account")
		}
		raw, err := json.Marshal(saved.signed)
		if err != nil {
			return err
		}
		payload, err := seal(key, raw, opAAD(op))
		clear(raw)
		if err != nil {
			return err
		}
		op.Payload = payload
		ac.Operations = append(ac.Operations, op)
	}
	if err := a.store.saveAccount(name, ac); err != nil {
		return err
	}
	if a.cfg.Account == "" {
		a.cfg.Account = name
		if err := a.store.saveConfig(a.cfg); err != nil {
			return err
		}
	}
	return a.emit(map[string]string{"account": name, "address": address}, func() {
		a.title("Account ready · " + name)
		fmt.Fprintln(a.out, address)
		fmt.Fprintf(a.out, "Next: yellow account backup %s --out %s.backup.json\n", name, name)
	})
}
func (a *app) accountCommands() *cobra.Command {
	parent := group("account", "Create, import and back up encrypted local accounts")
	create := a.cmd("create <name>", "Create an account with 24 recovery words", cobra.ExactArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		if err := a.ensureNewAccount(args[0]); err != nil {
			return err
		}
		words, err := wallet.NewMnemonic()
		if err != nil {
			return err
		}
		if err := a.prompt.recovery(words); err != nil {
			return err
		}
		return a.saveNewAccount(args[0], secret{"mnemonic", words}, nil)
	})
	imp := a.cmd("import <name>", "Import recovery words, a private key or an encrypted backup interactively", cobra.ExactArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		if err := a.ensureNewAccount(args[0]); err != nil {
			return err
		}
		kind, err := a.prompt.line("Import type: mnemonic, key or backup")
		if err != nil {
			return err
		}
		var sec secret
		var restored []restoredOperation
		switch kind {
		case "mnemonic", "key":
			value, err := a.prompt.secret("Enter " + kind + " (hidden)")
			if err != nil {
				return err
			}
			defer clear(value)
			sec = secret{kind, strings.TrimSpace(string(value))}
			if kind == "key" {
				sec.Value = strings.TrimPrefix(sec.Value, "0x")
			}
		case "backup":
			path, err := a.prompt.line("Encrypted backup path")
			if err != nil {
				return err
			}
			var backup accountFile
			if err := readJSON(path, &backup); err != nil {
				return fmt.Errorf("cannot read encrypted backup")
			}
			if backup.Version != 1 {
				return fmt.Errorf("unsupported backup format")
			}
			pass, err := a.prompt.secret("Backup passphrase")
			if err != nil {
				return err
			}
			defer clear(pass)
			var key []byte
			sec, key, err = decryptSecret(pass, &backup)
			if err != nil {
				return err
			}
			defer clear(key)
			for _, op := range backup.Operations {
				signed, err := decodeOperation(key, op)
				if err != nil {
					return err
				}
				restored = append(restored, restoredOperation{op, signed})
			}
		default:
			return fail(ExitInput, "choose mnemonic, key or backup")
		}
		return a.saveNewAccount(args[0], sec, restored)
	})
	list := a.cmd("list", "View account names and public addresses", cobra.NoArgs, func(context.Context, *cobra.Command, []string) error {
		names, err := a.store.accounts()
		if err != nil {
			return err
		}
		results := []map[string]string{}
		rows := [][]string{}
		for _, name := range names {
			ac, err := a.store.account(name)
			if err != nil {
				return err
			}
			results = append(results, map[string]string{"account": name, "address": ac.Address})
			mark := ""
			if name == a.cfg.Account {
				mark = "active"
			}
			rows = append(rows, []string{name, mark, ac.Address})
		}
		return a.emit(results, func() {
			a.title("Your accounts")
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No accounts yet. Run yellow account create personal.")
			} else {
				a.table([]string{"NAME", "", "ADDRESS"}, rows)
			}
		})
	})
	address := a.cmd("address [name]", "Show the full public address", cobra.MaximumNArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		explicit := ""
		if len(args) > 0 {
			explicit = args[0]
		}
		name, ac, err := a.selectedAccount(explicit)
		if err != nil {
			return err
		}
		return a.emit(map[string]string{"account": name, "address": ac.Address}, func() { fmt.Fprintln(a.out, ac.Address) })
	})
	use := a.cmd("use <name>", "Select the default account", cobra.ExactArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		name, ac, err := a.selectedAccount(args[0])
		if err != nil {
			return err
		}
		a.cfg.Account = name
		if err := a.store.saveConfig(a.cfg); err != nil {
			return err
		}
		return a.emit(map[string]string{"account": name, "address": ac.Address}, func() { fmt.Fprintln(a.out, "Active account:", name) })
	})
	var outPath string
	backup := a.cmd("backup <name>", "Export an encrypted backup (never plaintext)", cobra.ExactArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		_, ac, err := a.selectedAccount(args[0])
		if err != nil {
			return err
		}
		signer, _, close, err := a.unlock(ac)
		if err != nil {
			return err
		}
		defer close()
		_ = signer
		// Backups retain encrypted operation authorizations for crash recovery.
		if _, err := os.Lstat(outPath); err == nil {
			return fail(ExitInput, "backup path already exists; choose a new --out path")
		} else if !os.IsNotExist(err) {
			return err
		}
		raw, err := json.MarshalIndent(ac, "", "  ")
		if err != nil {
			return err
		}
		defer clear(raw)
		if err := atomicWriteNew(outPath, raw); err != nil {
			return err
		}
		return a.emit(map[string]string{"account": args[0], "backup": outPath}, func() {
			a.title("Encrypted backup saved")
			fmt.Fprintln(a.out, clean(outPath))
			fmt.Fprintln(a.out, "Keep the backup and its passphrase in separate, safe places.")
		})
	})
	backup.Flags().StringVar(&outPath, "out", "", "New file path for the encrypted backup")
	_ = backup.MarkFlagRequired("out")
	_ = backup.MarkFlagFilename("out")
	export := a.cmd("export-key [name]", "Reveal your private key in the terminal to import into another wallet", cobra.MaximumNArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		if a.json {
			return fail(ExitInput, "private-key export requires interactive text mode; remove --json")
		}
		explicit := ""
		if len(args) > 0 {
			explicit = args[0]
		}
		name, ac, err := a.selectedAccount(explicit)
		if err != nil {
			return err
		}
		private, _, close, err := a.unlockKey(ac)
		if err != nil {
			return err
		}
		defer close()
		raw := crypto.FromECDSA(private)
		defer clear(raw)
		encoded := make([]byte, hex.EncodedLen(len(raw)))
		defer clear(encoded)
		hex.Encode(encoded, raw)
		return a.prompt.privateKey(name, ac.Address, encoded)
	})
	for _, c := range []*cobra.Command{address, use, backup, export} {
		c.ValidArgsFunction = oneArgument(a.completeAccounts)
	}
	parent.AddCommand(create, imp, list, address, use, backup, export)
	return parent
}

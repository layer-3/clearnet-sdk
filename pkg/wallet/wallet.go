// Package wallet derives user signing keys using public BIP-39/BIP-32 standards.
// It does not store keys or read secrets from the environment.
package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

// DerivationPath is the BIP-44 path DeriveKey follows: the first Ethereum
// account, m/44'/60'/0'/0/0.
const DerivationPath = "m/44'/60'/0'/0/0"

// derivationIndexes is DerivationPath as BIP-32 child indexes; DeriveKey uses
// it so the documented path and the derived key cannot drift apart.
var derivationIndexes = [...]uint32{44 | hdkeychain.HardenedKeyStart, 60 | hdkeychain.HardenedKeyStart, hdkeychain.HardenedKeyStart, 0, 0}

// NewMnemonic returns 24 recovery words generated from OS randomness.
func NewMnemonic() (string, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	defer clear(entropy)
	return bip39.NewMnemonic(entropy)
}

// DeriveKey derives Ethereum index zero with no BIP-39 extension passphrase.
func DeriveKey(mnemonic string) (*ecdsa.PrivateKey, error) {
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid BIP-39 recovery words or checksum")
	}
	seed := bip39.NewSeed(mnemonic, "")
	defer clear(seed)
	master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	defer master.Zero()
	k := master
	for _, index := range derivationIndexes {
		child, err := k.Derive(index)
		if k != master {
			k.Zero()
		}
		if err != nil {
			return nil, err
		}
		k = child
	}
	defer k.Zero()
	key, err := k.ECPrivKey()
	if err != nil {
		return nil, err
	}
	defer key.Zero()
	raw := key.Serialize()
	defer clear(raw)
	return crypto.ToECDSA(raw)
}

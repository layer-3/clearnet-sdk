//go:build integration

package eip712deployment_test

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	evm "github.com/layer-3/clearnet-sdk/pkg/blockchain/evm"
)

// Deploy the committed bytecode and execute all six operations signed by the Go
// helpers. This catches stale bytecode/bindings even when Solidity tests pass.
func TestEIP712DeploymentAllOperations(t *testing.T) {
	ctx := context.Background()
	chain := big.NewInt(1337)
	var privateKeys []*ecdsa.PrivateKey
	for i := int64(1); i <= 3; i++ {
		k, err := crypto.ToECDSA(common.LeftPadBytes(big.NewInt(i).Bytes(), 32))
		if err != nil {
			t.Fatal(err)
		}
		privateKeys = append(privateKeys, k)
	}
	sort.Slice(privateKeys, func(i, j int) bool {
		return crypto.PubkeyToAddress(privateKeys[i].PublicKey).Cmp(crypto.PubkeyToAddress(privateKeys[j].PublicKey)) < 0
	})
	var keys []common.Address
	for _, k := range privateKeys {
		keys = append(keys, crypto.PubkeyToAddress(k.PublicKey))
	}
	backend := simulated.NewBackend(types.GenesisAlloc{keys[0]: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)}})
	defer backend.Close()
	client := backend.Client()
	auth, err := bind.NewKeyedTransactorWithChainID(privateKeys[0], chain)
	if err != nil {
		t.Fatal(err)
	}
	mined := func(tx *types.Transaction, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		backend.Commit()
		r, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != types.ReceiptStatusSuccessful {
			t.Fatal("reverted", tx.Hash())
		}
	}
	sign := func(d common.Hash) [][]byte {
		out := make([][]byte, 2)
		for i := range out {
			sig, err := crypto.Sign(d[:], privateKeys[i])
			if err != nil {
				t.Fatal(err)
			}
			sig[64] += 27
			out[i] = sig
		}
		return out
	}
	vaultAddr, tx, vault, err := evm.DeployCustody(auth, client, keys, big.NewInt(2))
	mined(tx, err)
	registryAddr, tx, registry, err := evm.DeployConfigRegistry(auth, client)
	mined(tx, err)
	vd, err := vault.Eip712Domain(nil)
	if err != nil {
		t.Fatal(err)
	}
	if vd.Name != evm.CustodyDomainName || vd.Version != evm.EIP712Version || vd.VerifyingContract != vaultAddr || vd.ChainId.Cmp(chain) != 0 {
		t.Fatal("vault domain mismatch")
	}
	rd, err := registry.Eip712Domain(nil)
	if err != nil {
		t.Fatal(err)
	}
	if rd.Name != evm.ConfigRegistryDomainName || rd.Version != evm.EIP712Version || rd.VerifyingContract != registryAddr || rd.ChainId.Cmp(chain) != 0 {
		t.Fatal("registry domain mismatch")
	}
	auth.Value = big.NewInt(1000)
	mined(vault.Deposit(auth, keys[0], common.Address{}, big.NewInt(1000), [32]byte{}))
	auth.Value = nil
	recipient := common.HexToAddress("0x1234")
	wid := [32]byte{1}
	deadline := big.NewInt(4000000000)
	mined(vault.Execute(auth, recipient, common.Address{}, big.NewInt(100), wid, deadline, sign(evm.ComputeWithdrawalDigest(1337, vaultAddr, recipient, common.Address{}, big.NewInt(100), wid, deadline))))
	balance, err := client.BalanceAt(ctx, recipient, nil)
	if err != nil || balance.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("withdrawal balance %v: %v", balance, err)
	}
	mined(vault.UpdateSigners(auth, keys, big.NewInt(2), sign(evm.ComputeRotationDigest(1337, vaultAddr, keys, big.NewInt(2), big.NewInt(0)))))
	nonce, err := vault.SignerNonce(nil)
	if err != nil || nonce.Uint64() != 1 {
		t.Fatal("vault rotation nonce", nonce, err)
	}
	issuer, err := registry.ComputeIssuerId(nil, keys, big.NewInt(2))
	if err != nil {
		t.Fatal(err)
	}
	mined(registry.RegisterIssuer(auth, keys, big.NewInt(2), sign(evm.ComputeConfigRegistryRegistrationDigest(1337, registryAddr, keys, big.NewInt(2)))))
	key := [32]byte{2}
	checksum := [32]byte{3}
	data := []byte("typed config")
	mined(registry.SetConfig(auth, issuer, key, checksum, big.NewInt(0), sign(evm.ComputeConfigRegistrySetConfigDigest(1337, registryAddr, issuer, key, checksum, big.NewInt(0)))))
	mined(registry.SetConfigWithData(auth, issuer, key, data, big.NewInt(1), sign(evm.ComputeConfigRegistrySetConfigWithDataDigest(1337, registryAddr, issuer, key, data, big.NewInt(1)))))
	mined(registry.UpdateIssuerSettings(auth, issuer, keys, big.NewInt(2), big.NewInt(2), sign(evm.ComputeConfigRegistryUpdateIssuerSettingsDigest(1337, registryAddr, issuer, keys, big.NewInt(2), big.NewInt(2)))))
	nonce, err = registry.Nonce(nil, issuer)
	if err != nil || nonce.Uint64() != 3 {
		t.Fatal("registry nonce", nonce, err)
	}
	config, err := evm.NewConfig(issuer, client)
	if err != nil {
		t.Fatal(err)
	}
	got, err := config.LatestConfigChecksum(nil, key)
	if err != nil || got != crypto.Keccak256Hash(data) {
		t.Fatal("config checksum", got, err)
	}
}

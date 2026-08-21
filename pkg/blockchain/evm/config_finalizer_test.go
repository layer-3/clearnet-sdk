package evm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestParseBytes32_RoundTrip(t *testing.T) {
	var b [32]byte
	b[0] = 0xDE
	b[31] = 0xAD
	s := hexBytes32(b)
	got, err := parseBytes32(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got != b {
		t.Fatalf("round trip mismatch: %x != %x", got, b)
	}
	// Accepts upper-case 0X and bare hex too.
	if _, err := parseBytes32("0X" + "00"); err == nil {
		t.Fatal("expected length error for short hex")
	}
	if _, err := parseBytes32("zz"); err == nil {
		t.Fatal("expected error for non-hex")
	}
}

func TestConfigRegistryCommitPacked_JSONStable(t *testing.T) {
	p := configRegistryCommitPacked{
		Op:            configRegistryOpSetConfig,
		IssuerID:      "0x0000000000000000000000000000000000000001",
		Key:           "0x01",
		Checksum:      "0x02",
		ExpectedNonce: "7",
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var got configRegistryCommitPacked
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Fatalf("round trip mismatch: %+v != %+v", got, p)
	}
}

func TestConfigRegistryCommitFinalizer_DigestFromPacked(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuerID := common.HexToAddress("0x000000000000000000000000000000000000b0b0")
	f := &ConfigRegistryCommitFinalizer{registryAddr: registry, issuerID: issuerID, chainID: 31337}
	var key, checksum [32]byte
	key[31] = 1
	checksum[31] = 2

	packed, err := json.Marshal(configRegistryCommitPacked{
		Op:            configRegistryOpSetConfig,
		IssuerID:      issuerID.Hex(),
		Key:           hexBytes32(key),
		Checksum:      hexBytes32(checksum),
		ExpectedNonce: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.digestFromPacked(packed)
	if err != nil {
		t.Fatalf("digestFromPacked: %v", err)
	}
	want := ComputeConfigRegistrySetConfigDigest(31337, registry, issuerID, key, checksum, big.NewInt(7))
	if got != want {
		t.Fatalf("digest mismatch\nwant %x\ngot  %x", want, got)
	}
}

func TestConfigRegistryCommitFinalizer_DigestFromPackedWithData(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuerID := common.HexToAddress("0x000000000000000000000000000000000000b0b0")
	f := &ConfigRegistryCommitFinalizer{registryAddr: registry, issuerID: issuerID, chainID: 31337}
	var key [32]byte
	key[31] = 1
	data := []byte("signers")

	packed, err := json.Marshal(configRegistryCommitPacked{
		Op:            configRegistryOpSetConfigWithData,
		IssuerID:      issuerID.Hex(),
		Key:           hexBytes32(key),
		Data:          "0x" + common.Bytes2Hex(data),
		ExpectedNonce: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.digestFromPacked(packed)
	if err != nil {
		t.Fatalf("digestFromPacked: %v", err)
	}
	want := ComputeConfigRegistrySetConfigWithDataDigest(31337, registry, issuerID, key, data, big.NewInt(7))
	if got != want {
		t.Fatalf("digest mismatch\nwant %x\ngot  %x", want, got)
	}
}

func TestConfigRegistryIssuerRegistrationFinalizer_DigestFromPacked(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	f := &ConfigRegistryIssuerRegistrationFinalizer{registryAddr: registry, chainID: 31337}
	keys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}

	packed, err := json.Marshal(configRegistryIssuerRegistrationPacked{
		Op:         configRegistryOpRegisterIssuer,
		IssuerKeys: addrsToHex(keys),
		Threshold:  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.digestFromPacked(packed)
	if err != nil {
		t.Fatalf("digestFromPacked: %v", err)
	}
	want := ComputeConfigRegistryRegistrationDigest(31337, registry, keys, big.NewInt(2))
	if got != want {
		t.Fatalf("digest mismatch\nwant %x\ngot  %x", want, got)
	}
}

func TestConfigRegistryIssuerSettingsUpdateFinalizer_DigestFromPacked(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuerID := common.HexToAddress("0x000000000000000000000000000000000000b0b0")
	f := &ConfigRegistryIssuerSettingsUpdateFinalizer{registryAddr: registry, issuerID: issuerID, chainID: 31337}
	keys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000004"),
		common.HexToAddress("0x0000000000000000000000000000000000000005"),
		common.HexToAddress("0x0000000000000000000000000000000000000006"),
	}

	packed, err := json.Marshal(configRegistryIssuerSettingsUpdatePacked{
		Op:            configRegistryOpUpdateIssuerSettings,
		IssuerID:      issuerID.Hex(),
		NewIssuerKeys: addrsToHex(keys),
		NewThreshold:  2,
		ExpectedNonce: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.digestFromPacked(packed)
	if err != nil {
		t.Fatalf("digestFromPacked: %v", err)
	}
	want := ComputeConfigRegistryUpdateIssuerSettingsDigest(31337, registry, issuerID, keys, big.NewInt(2), big.NewInt(7))
	if got != want {
		t.Fatalf("digest mismatch\nwant %x\ngot  %x", want, got)
	}
}

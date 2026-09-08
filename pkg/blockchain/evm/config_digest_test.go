package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestConfigRegistryDigests_InputsDifferentiate(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuerID := common.HexToAddress("0x000000000000000000000000000000000000b0b0")
	keys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000010"),
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
		common.HexToAddress("0x0000000000000000000000000000000000000030"),
	}
	otherKeys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000010"),
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
		common.HexToAddress("0x0000000000000000000000000000000000000040"),
	}
	threshold := big.NewInt(2)
	nonce := big.NewInt(0)
	var key, checksum [32]byte
	key[31] = 0x10
	checksum[31] = 0x20

	reg0 := ComputeConfigRegistryRegistrationDigest(1, registry, keys, threshold)
	regVariants := map[string][32]byte{
		"chainID":   ComputeConfigRegistryRegistrationDigest(2, registry, keys, threshold),
		"registry":  ComputeConfigRegistryRegistrationDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), keys, threshold),
		"keys":      ComputeConfigRegistryRegistrationDigest(1, registry, otherKeys, threshold),
		"threshold": ComputeConfigRegistryRegistrationDigest(1, registry, keys, big.NewInt(3)),
	}
	for name, d := range regVariants {
		if d == reg0 {
			t.Errorf("registry registration digest unchanged when %s changed", name)
		}
	}

	set0 := ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, checksum, nonce)
	setVariants := map[string][32]byte{
		"chainID":  ComputeConfigRegistrySetConfigDigest(2, registry, issuerID, key, checksum, nonce),
		"registry": ComputeConfigRegistrySetConfigDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, key, checksum, nonce),
		"issuerID": ComputeConfigRegistrySetConfigDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), key, checksum, nonce),
		"key":      ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, [32]byte{0x11}, checksum, nonce),
		"checksum": ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, [32]byte{0x21}, nonce),
		"nonce":    ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, checksum, big.NewInt(1)),
	}
	for name, d := range setVariants {
		if d == set0 {
			t.Errorf("registry setConfig digest unchanged when %s changed", name)
		}
	}

	data0 := ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("data"), nonce)
	dataVariants := map[string][32]byte{
		"chainID":  ComputeConfigRegistrySetConfigWithDataDigest(2, registry, issuerID, key, []byte("data"), nonce),
		"registry": ComputeConfigRegistrySetConfigWithDataDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, key, []byte("data"), nonce),
		"issuerID": ComputeConfigRegistrySetConfigWithDataDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), key, []byte("data"), nonce),
		"key":      ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, [32]byte{0x11}, []byte("data"), nonce),
		"data":     ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("other"), nonce),
		"nonce":    ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("data"), big.NewInt(1)),
	}
	for name, d := range dataVariants {
		if d == data0 {
			t.Errorf("registry setConfigWithData digest unchanged when %s changed", name)
		}
	}

	update0 := ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, threshold, nonce)
	updateVariants := map[string][32]byte{
		"chainID":      ComputeConfigRegistryUpdateIssuerSettingsDigest(2, registry, issuerID, otherKeys, threshold, nonce),
		"registry":     ComputeConfigRegistryUpdateIssuerSettingsDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, otherKeys, threshold, nonce),
		"issuerID":     ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), otherKeys, threshold, nonce),
		"newKeys":      ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, keys, threshold, nonce),
		"newThreshold": ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, big.NewInt(3), nonce),
		"nonce":        ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, threshold, big.NewInt(1)),
	}
	for name, d := range updateVariants {
		if d == update0 {
			t.Errorf("registry updateIssuerSettings digest unchanged when %s changed", name)
		}
	}

	if set0 == data0 {
		t.Errorf("registry setConfig and setConfigWithData digests collide")
	}
	if reg0 == update0 {
		t.Errorf("registry registerIssuer and updateIssuerSettings digests collide")
	}
}

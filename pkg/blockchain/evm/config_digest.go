package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ComputeConfigRegistryRegistrationDigest returns the EIP-712 RegisterIssuer authorization.
// Invalid uint256 arguments panic with the offending field name.
func ComputeConfigRegistryRegistrationDigest(chainID uint64, registry common.Address, issuerKeys []common.Address, threshold *big.Int) common.Hash {
	keys := addressArrayHash(issuerKeys)
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(registerIssuerType, keys[:], uint256Word("threshold", threshold)))
}

// ComputeConfigRegistrySetConfigDigest returns the EIP-712 SetConfig authorization.
// Invalid uint256 arguments panic with the offending field name.
func ComputeConfigRegistrySetConfigDigest(chainID uint64, registry, issuerID common.Address, key, checksum [32]byte, expectedNonce *big.Int) common.Hash {
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(setConfigType, addressWord(issuerID), key[:], checksum[:], uint256Word("expectedNonce", expectedNonce)))
}

// ComputeConfigRegistrySetConfigWithDataDigest returns the EIP-712 SetConfigWithData authorization.
// Invalid uint256 arguments panic with the offending field name.
func ComputeConfigRegistrySetConfigWithDataDigest(chainID uint64, registry, issuerID common.Address, key [32]byte, data []byte, expectedNonce *big.Int) common.Hash {
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(setConfigWithDataType, addressWord(issuerID), key[:], crypto.Keccak256(data), uint256Word("expectedNonce", expectedNonce)))
}

// ComputeConfigRegistryUpdateIssuerSettingsDigest returns the EIP-712 UpdateIssuerSettings authorization.
// Invalid uint256 arguments panic with the offending field name.
func ComputeConfigRegistryUpdateIssuerSettingsDigest(chainID uint64, registry, issuerID common.Address, newIssuerKeys []common.Address, newThreshold, expectedNonce *big.Int) common.Hash {
	keys := addressArrayHash(newIssuerKeys)
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(updateIssuerSettingsType, addressWord(issuerID), keys[:], uint256Word("newThreshold", newThreshold), uint256Word("expectedNonce", expectedNonce)))
}

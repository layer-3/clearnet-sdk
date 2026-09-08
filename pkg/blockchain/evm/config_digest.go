package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

// ComputeConfigRegistryRegistrationDigest returns the EIP-712 RegisterIssuer authorization.
func ComputeConfigRegistryRegistrationDigest(chainID uint64, registry common.Address, issuerKeys []common.Address, threshold *big.Int) [32]byte {
	keys := addressArrayHash(issuerKeys)
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(registerIssuerType, keys[:], uint256Word(threshold)))
}

// ComputeConfigRegistrySetConfigDigest returns the EIP-712 SetConfig authorization.
func ComputeConfigRegistrySetConfigDigest(chainID uint64, registry, issuerID common.Address, key, checksum [32]byte, expectedNonce *big.Int) [32]byte {
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(setConfigType, addressWord(issuerID), key[:], checksum[:], uint256Word(expectedNonce)))
}

// ComputeConfigRegistrySetConfigWithDataDigest returns the EIP-712 SetConfigWithData authorization.
func ComputeConfigRegistrySetConfigWithDataDigest(chainID uint64, registry, issuerID common.Address, key [32]byte, data []byte, expectedNonce *big.Int) [32]byte {
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(setConfigWithDataType, addressWord(issuerID), key[:], crypto.Keccak256(data), uint256Word(expectedNonce)))
}

// ComputeConfigRegistryUpdateIssuerSettingsDigest returns the EIP-712 UpdateIssuerSettings authorization.
func ComputeConfigRegistryUpdateIssuerSettingsDigest(chainID uint64, registry, issuerID common.Address, newIssuerKeys []common.Address, newThreshold, expectedNonce *big.Int) [32]byte {
	keys := addressArrayHash(newIssuerKeys)
	return typedDigest(ConfigRegistryDomainName, chainID, registry, typedStructHash(updateIssuerSettingsType, addressWord(issuerID), keys[:], uint256Word(newThreshold), uint256Word(expectedNonce)))
}

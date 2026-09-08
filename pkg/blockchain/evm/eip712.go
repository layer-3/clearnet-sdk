package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

// EIP-712 signing domains. Version describes the signing protocol, not the SDK release.
const CustodyDomainName = "YellowCustody"
const ConfigRegistryDomainName = "YellowConfigRegistry"
const EIP712Version = "1"
const eip712DomainType = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
const executeType = "Execute(address to,address asset,uint256 amount,bytes32 withdrawalId,uint256 deadline)"
const updateSignersType = "UpdateSigners(address[] newSigners,uint256 newThreshold,uint256 signerNonce)"
const registerIssuerType = "RegisterIssuer(address[] issuerKeys,uint256 threshold)"
const setConfigType = "SetConfig(address issuerId,bytes32 key,bytes32 checksum,uint256 expectedNonce)"
const setConfigWithDataType = "SetConfigWithData(address issuerId,bytes32 key,bytes data,uint256 expectedNonce)"
const updateIssuerSettingsType = "UpdateIssuerSettings(address issuerId,address[] newIssuerKeys,uint256 newThreshold,uint256 expectedNonce)"

func uint256Word(n *big.Int) []byte {
	if n == nil || n.Sign() < 0 || n.BitLen() > 256 {
		panic("evm: invalid EIP-712 uint256")
	}
	return common.LeftPadBytes(n.Bytes(), 32)
}
func addressWord(a common.Address) []byte { return common.LeftPadBytes(a.Bytes(), 32) }
func addressArrayHash(addresses []common.Address) common.Hash {
	words := make([]byte, 0, 32*len(addresses))
	for _, a := range addresses {
		words = append(words, addressWord(a)...)
	}
	return crypto.Keccak256Hash(words)
}
func typedStructHash(schema string, words ...[]byte) common.Hash {
	parts := append([][]byte{crypto.Keccak256([]byte(schema))}, words...)
	return crypto.Keccak256Hash(parts...)
}
func domainSeparator(name string, chainID uint64, contract common.Address) common.Hash {
	return typedStructHash(eip712DomainType, crypto.Keccak256([]byte(name)), crypto.Keccak256([]byte(EIP712Version)), uint256Word(new(big.Int).SetUint64(chainID)), addressWord(contract))
}
func typedDigest(name string, chainID uint64, contract common.Address, structure common.Hash) common.Hash {
	domain := domainSeparator(name, chainID, contract)
	return crypto.Keccak256Hash([]byte{0x19, 0x01}, domain[:], structure[:])
}

// ComputeWithdrawalDigest returns the EIP-712 Execute authorization. All integers
// are Solidity uint256 values; invalid (negative, nil or overflowing) values panic.
func ComputeWithdrawalDigest(chainID uint64, vault, to, asset common.Address, amount *big.Int, withdrawalID [32]byte, deadline *big.Int) common.Hash {
	return typedDigest(CustodyDomainName, chainID, vault, typedStructHash(executeType, addressWord(to), addressWord(asset), uint256Word(amount), withdrawalID[:], uint256Word(deadline)))
}

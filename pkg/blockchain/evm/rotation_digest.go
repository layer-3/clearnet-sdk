package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ComputeRotationDigest returns the EIP-712 UpdateSigners authorization. Signer
// order is preserved; rotationNonce is the current on-chain rotation nonce.
func ComputeRotationDigest(chainID uint64, vault common.Address, newSigners []common.Address, newThreshold, rotationNonce *big.Int) common.Hash {
	keys := addressArrayHash(newSigners)
	return typedDigest(CustodyDomainName, chainID, vault, typedStructHash(updateSignersType, keys[:], uint256Word("newThreshold", newThreshold), uint256Word("rotationNonce", rotationNonce)))
}

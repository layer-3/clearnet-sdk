package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// ComputeRotationDigest returns the EIP-712 UpdateSigners authorization. Signer
// order is preserved; signerNonce is the current on-chain rotation nonce.
func ComputeRotationDigest(chainID uint64, vault common.Address, newSigners []common.Address, newThreshold, signerNonce *big.Int) [32]byte {
	keys := addressArrayHash(newSigners)
	return typedDigest(CustodyDomainName, chainID, vault, typedStructHash(updateSignersType, keys[:], uint256Word(newThreshold), uint256Word(signerNonce)))
}

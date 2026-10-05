package bls

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// RegistryProofOfPossessionDigest matches Registry's abi.encode exactly. G2 coordinates
// are in EVM (imaginary-first) order. A proof is bound to chain, Registry,
// operator and BOTH representations of the key; it is not the legacy node PoP.
func RegistryProofOfPossessionDigest(chainID *big.Int, registry, operator common.Address, g1 [2]*big.Int, g2 [4]*big.Int) ([32]byte, error) {
	if chainID == nil || chainID.Sign() < 0 || chainID.BitLen() > 256 {
		return [32]byte{}, fmt.Errorf("invalid Registry chain ID")
	}
	for _, c := range append(g1[:], g2[:]...) {
		if c == nil || c.Sign() < 0 || c.BitLen() > 256 {
			return [32]byte{}, fmt.Errorf("invalid Registry key coordinate")
		}
	}
	types := []string{"bytes32", "uint256", "address", "address", "uint256[2]", "uint256[4]"}
	args := make(abi.Arguments, len(types))
	for i, name := range types {
		t, err := abi.NewType(name, "", nil)
		if err != nil {
			return [32]byte{}, err
		}
		args[i] = abi.Argument{Type: t}
	}
	domain := crypto.Keccak256Hash([]byte("CLEARNET_REGISTRY_BLS_POP_V1"))
	encoded, err := args.Pack(domain, chainID, registry, operator, g1, g2)
	if err != nil {
		return [32]byte{}, err
	}
	return crypto.Keccak256Hash(encoded), nil
}

// RegistryPossessionDigest is the original name used by coordinated callers.
// Deprecated: use RegistryProofOfPossessionDigest.
func RegistryPossessionDigest(chainID *big.Int, registry, operator common.Address, g1 [2]*big.Int, g2 [4]*big.Int) ([32]byte, error) {
	return RegistryProofOfPossessionDigest(chainID, registry, operator, g1, g2)
}

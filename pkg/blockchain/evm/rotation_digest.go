package evm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
)

// rotationInnerArgs / rotationOuterArgs mirror Custody.sol's updateSigners
// digest layout. The inner hash commits to the dynamic newSigners array and the
// threshold; the outer commits to the chain id, vault, the "updateSigners"
// domain string, that inner hash, and the signer nonce.
var (
	rotationInnerArgs = abi.Arguments{{Type: abiutil.AddressArr}, {Type: abiutil.Uint256}}
	rotationOuterArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Uint256},
	}
)

// ComputeRotationDigest mirrors Custody.sol's updateSigners digest:
//
//	keccak256(abi.encode(
//	    block.chainid, address(this), "updateSigners",
//	    keccak256(abi.encode(newSigners, newThreshold)),
//	    signerNonce))
//
// newSigners must be in the same (ascending) order the contract stores them;
// signerNonce is the on-chain Custody.signerNonce() — the rotation replay token.
func ComputeRotationDigest(chainID uint64, vault common.Address, newSigners []common.Address, newThreshold, signerNonce *big.Int) [32]byte {
	innerEncoded, err := rotationInnerArgs.Pack(newSigners, newThreshold)
	if err != nil {
		// rotationInnerArgs is fixed; Pack only fails on a type mismatch.
		panic(fmt.Errorf("evm: ComputeRotationDigest inner pack: %w", err))
	}
	innerHash := crypto.Keccak256Hash(innerEncoded)

	outerEncoded, err := rotationOuterArgs.Pack(
		new(big.Int).SetUint64(chainID),
		vault,
		"updateSigners",
		innerHash,
		new(big.Int).Set(signerNonce),
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeRotationDigest outer pack: %w", err))
	}
	return crypto.Keccak256Hash(outerEncoded)
}

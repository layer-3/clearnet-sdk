package evm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
)

// configCommitArgs / operatorRotateInnerArgs / operatorRotateOuterArgs mirror
// ConfigGovernor.sol's digest layouts (ADR-017). They share Custody.sol's
// domain-separation discipline: chain id + governor address + an operation tag,
// so a signature can never be replayed across chains or sibling deployments, or
// between the two operations.
var (
	// keccak256(abi.encode(chainid, governor, "setConfig", key, checksum, expectedEpoch))
	configCommitArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Uint64},
	}
	// inner: keccak256(abi.encode(newOperators, newThreshold))
	operatorRotateInnerArgs = abi.Arguments{{Type: abiutil.AddressArr}, {Type: abiutil.Uint256}}
	// outer: keccak256(abi.encode(chainid, governor, "updateOperators", innerHash, operatorNonce))
	operatorRotateOuterArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Uint256},
	}
)

// ComputeConfigCommitDigest mirrors ConfigGovernor.setConfig's digest:
//
//	keccak256(abi.encode(
//	    block.chainid, address(this), "setConfig", key, checksum, expectedEpoch))
//
// expectedEpoch is the registry's current configEpoch(key) — the only replay
// token for a config commit, since Config.sol has no nonce. It is bound into the
// signed digest, never into the hashed payload, so content-addressing and free
// rollback (re-committing a checksum already held) keep working.
func ComputeConfigCommitDigest(chainID uint64, governor common.Address, key [32]byte, checksum [32]byte, expectedEpoch uint64) [32]byte {
	encoded, err := configCommitArgs.Pack(
		new(big.Int).SetUint64(chainID),
		governor,
		"setConfig",
		key,
		checksum,
		expectedEpoch,
	)
	if err != nil {
		// configCommitArgs is fixed; Pack only fails on a type mismatch.
		panic(fmt.Errorf("evm: ComputeConfigCommitDigest pack: %w", err))
	}
	return crypto.Keccak256Hash(encoded)
}

// ComputeOperatorRotationDigest mirrors ConfigGovernor.updateOperators's digest:
//
//	keccak256(abi.encode(
//	    block.chainid, address(this), "updateOperators",
//	    keccak256(abi.encode(newOperators, newThreshold)),
//	    operatorNonce))
//
// newOperators must be ascending (the order the contract stores and verifies);
// operatorNonce is the on-chain ConfigGovernor.operatorNonce() — consumed on a
// successful rotation, so the signature set authorises exactly one rotation.
func ComputeOperatorRotationDigest(chainID uint64, governor common.Address, newOperators []common.Address, newThreshold, operatorNonce *big.Int) [32]byte {
	innerEncoded, err := operatorRotateInnerArgs.Pack(newOperators, newThreshold)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeOperatorRotationDigest inner pack: %w", err))
	}
	innerHash := crypto.Keccak256Hash(innerEncoded)

	outerEncoded, err := operatorRotateOuterArgs.Pack(
		new(big.Int).SetUint64(chainID),
		governor,
		"updateOperators",
		innerHash,
		new(big.Int).Set(operatorNonce),
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeOperatorRotationDigest outer pack: %w", err))
	}
	return crypto.Keccak256Hash(outerEncoded)
}

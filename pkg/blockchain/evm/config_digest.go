package evm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
)

// configRegistryIssuerSettingsArgs / configRegistryRegistrationArgs /
// configRegistrySetConfigArgs / configRegistrySetConfigWithDataArgs /
// configRegistryUpdateIssuerSettingsArgs mirror ConfigRegistryDigests.sol's
// digest layouts. They share Custody.sol's domain-separation discipline:
// chain id + registry address + an operation tag, so a signature can never
// be replayed across chains, sibling deployments, or between operations.
var (
	// ConfigRegistryDigests.registrationDigest:
	// inner: keccak256(abi.encode(issuerKeys, threshold))
	configRegistryIssuerSettingsArgs = abi.Arguments{{Type: abiutil.AddressArr}, {Type: abiutil.Uint256}}
	// outer: keccak256(abi.encode(chainid, registry, "registerIssuer", innerHash))
	configRegistryRegistrationArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Bytes32},
	}
	// keccak256(abi.encode(chainid, registry, "setConfig", issuerId, key, checksum, expectedNonce))
	configRegistrySetConfigArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Address},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Uint256},
	}
	// keccak256(abi.encode(chainid, registry, "setConfigWithData", issuerId, key, data, expectedNonce))
	configRegistrySetConfigWithDataArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Address},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Bytes},
		{Type: abiutil.Uint256},
	}
	// outer: keccak256(abi.encode(chainid, registry, "updateIssuerSettings", issuerId, innerHash, expectedNonce))
	configRegistryUpdateIssuerSettingsArgs = abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Address},
		{Type: abiutil.String},
		{Type: abiutil.Address},
		{Type: abiutil.Bytes32},
		{Type: abiutil.Uint256},
	}
)

// ComputeConfigRegistryRegistrationDigest mirrors
// ConfigRegistryDigests.registrationDigest:
//
//	keccak256(abi.encode(
//	    block.chainid,
//	    registry,
//	    "registerIssuer",
//	    keccak256(abi.encode(issuerKeys, threshold))))
//
// issuerKeys must be in the same ascending order the contract validates and
// threshold is a Solidity uint256.
func ComputeConfigRegistryRegistrationDigest(chainID uint64, registry common.Address, issuerKeys []common.Address, threshold *big.Int) [32]byte {
	innerEncoded, err := configRegistryIssuerSettingsArgs.Pack(issuerKeys, threshold)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistryRegistrationDigest inner pack: %w", err))
	}
	innerHash := crypto.Keccak256Hash(innerEncoded)

	outerEncoded, err := configRegistryRegistrationArgs.Pack(
		new(big.Int).SetUint64(chainID),
		registry,
		"registerIssuer",
		innerHash,
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistryRegistrationDigest outer pack: %w", err))
	}
	return crypto.Keccak256Hash(outerEncoded)
}

// ComputeConfigRegistrySetConfigDigest mirrors
// ConfigRegistryDigests.setConfigDigest:
//
//	keccak256(abi.encode(
//	    block.chainid, registry, "setConfig",
//	    issuerId, key, checksum, expectedNonce))
//
// expectedNonce is ConfigRegistry's per-issuer nonce, not Config.configEpoch(key).
func ComputeConfigRegistrySetConfigDigest(chainID uint64, registry common.Address, issuerID common.Address, key [32]byte, checksum [32]byte, expectedNonce *big.Int) [32]byte {
	encoded, err := configRegistrySetConfigArgs.Pack(
		new(big.Int).SetUint64(chainID),
		registry,
		"setConfig",
		issuerID,
		key,
		checksum,
		expectedNonce,
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistrySetConfigDigest pack: %w", err))
	}
	return crypto.Keccak256Hash(encoded)
}

// ComputeConfigRegistrySetConfigWithDataDigest mirrors
// ConfigRegistryDigests.setConfigWithDataDigest:
//
//	keccak256(abi.encode(
//	    block.chainid, registry, "setConfigWithData",
//	    issuerId, key, data, expectedNonce))
//
// The digest binds the raw data bytes. ConfigRegistry separately commits
// keccak256(data) to the issuer Config.
func ComputeConfigRegistrySetConfigWithDataDigest(chainID uint64, registry common.Address, issuerID common.Address, key [32]byte, data []byte, expectedNonce *big.Int) [32]byte {
	encoded, err := configRegistrySetConfigWithDataArgs.Pack(
		new(big.Int).SetUint64(chainID),
		registry,
		"setConfigWithData",
		issuerID,
		key,
		data,
		expectedNonce,
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistrySetConfigWithDataDigest pack: %w", err))
	}
	return crypto.Keccak256Hash(encoded)
}

// ComputeConfigRegistryUpdateIssuerSettingsDigest mirrors
// ConfigRegistryDigests.updateIssuerSettingsDigest:
//
//	keccak256(abi.encode(
//	    block.chainid,
//	    registry,
//	    "updateIssuerSettings",
//	    issuerId,
//	    keccak256(abi.encode(newIssuerKeys, newThreshold)),
//	    expectedNonce))
//
// Signatures over this digest are verified against the issuer's current key set;
// newIssuerKeys/newThreshold are the replacement settings being authorised.
func ComputeConfigRegistryUpdateIssuerSettingsDigest(chainID uint64, registry common.Address, issuerID common.Address, newIssuerKeys []common.Address, newThreshold *big.Int, expectedNonce *big.Int) [32]byte {
	innerEncoded, err := configRegistryIssuerSettingsArgs.Pack(newIssuerKeys, newThreshold)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistryUpdateIssuerSettingsDigest inner pack: %w", err))
	}
	innerHash := crypto.Keccak256Hash(innerEncoded)

	outerEncoded, err := configRegistryUpdateIssuerSettingsArgs.Pack(
		new(big.Int).SetUint64(chainID),
		registry,
		"updateIssuerSettings",
		issuerID,
		innerHash,
		expectedNonce,
	)
	if err != nil {
		panic(fmt.Errorf("evm: ComputeConfigRegistryUpdateIssuerSettingsDigest outer pack: %w", err))
	}
	return crypto.Keccak256Hash(outerEncoded)
}

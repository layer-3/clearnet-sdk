// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @notice Required extension for Registry-owned finalized-withdrawal policy.
/// @dev CONFIG() must return a nonzero Config whose owner() is this Registry,
/// not a custody issuer. For key keccak256("CLEARNET_FINALITY_POLICY_V1"),
/// configEpoch(key) must be exactly 1 and latestConfigChecksum(key) must equal
/// keccak256(abi.encode(uint64(K))), where K is the signing cluster size in 1..256.
/// The Registry must freeze the CONFIG address and this policy row, including
/// against its owner; live policy changes and upgradeable trust roots are unsupported.
/// Consumers pin all reads to one confirmed block hash and recheck its canonicality.
/// These reads validate the current policy, not future immutability.
interface IClearnetRegistryConfig {
    function CONFIG() external view returns (address);
}

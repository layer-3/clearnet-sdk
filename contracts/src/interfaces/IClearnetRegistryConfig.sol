// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @notice Required extension for Registry-owned finalized-withdrawal policy.
/// The Config is owned by this registry, not by a custody issuer.
interface IClearnetRegistryConfig {
    function CONFIG() external view returns (address);
}

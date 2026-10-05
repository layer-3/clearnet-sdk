// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @notice Optional extension: legacy registries need not implement this.
/// The Config is owned by this registry, not by a custody issuer.
interface IClearnetRegistryConfig {
    function CONFIG() external view returns (address);
}

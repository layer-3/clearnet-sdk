// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @title ConfigRegistryDigests
/// @notice Pure digest-construction formulas for every signed message
///         `ConfigRegistry` verifies (`registerIssuer`, `setConfig`,
///         `setConfigWithData`, `updateIssuerSettings`).
library ConfigRegistryDigests {
    /// @notice Digest for `registerIssuer`: self-referential over the
    ///         claimed key set, proving control of it.
    function registrationDigest(address registry, address[] memory issuerKeys, uint256 threshold)
        internal
        view
        returns (bytes32)
    {
        return keccak256(
            abi.encode(block.chainid, registry, "registerIssuer", keccak256(abi.encode(issuerKeys, threshold)))
        );
    }

    /// @notice Digest for `setConfig`. Binds `key` (which key is being
    ///         written) and `expectedNonce`.
    function setConfigDigest(address registry, address issuerId, bytes32 key, bytes32 checksum, uint256 expectedNonce)
        internal
        view
        returns (bytes32)
    {
        return keccak256(abi.encode(block.chainid, registry, "setConfig", issuerId, key, checksum, expectedNonce));
    }

    /// @notice Digest for `setConfigWithData`. Binds the raw `data` bytes
    ///         rather than a pre-computed checksum.
    function setConfigWithDataDigest(
        address registry,
        address issuerId,
        bytes32 key,
        bytes memory data,
        uint256 expectedNonce
    ) internal view returns (bytes32) {
        return keccak256(abi.encode(block.chainid, registry, "setConfigWithData", issuerId, key, data, expectedNonce));
    }

    /// @notice Digest for `updateIssuerSettings`. Binds the pre-hashed proposed
    ///         key set/threshold and the current per-issuer `nonce`, verified
    ///         against the issuer's *current* (outgoing) key set.
    function updateIssuerSettingsDigest(
        address registry,
        address issuerId,
        address[] memory newIssuerKeys,
        uint256 newThreshold,
        uint256 expectedNonce
    ) internal view returns (bytes32) {
        return keccak256(
            abi.encode(
                block.chainid,
                registry,
                "updateIssuerSettings",
                issuerId,
                keccak256(abi.encode(newIssuerKeys, newThreshold)),
                expectedNonce
            )
        );
    }
}

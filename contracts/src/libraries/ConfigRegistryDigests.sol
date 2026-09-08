// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @notice EIP-712 struct hashes; ConfigRegistry applies its own domain.
library ConfigRegistryDigests {
    bytes32 internal constant REGISTER_ISSUER_TYPEHASH =
        keccak256("RegisterIssuer(address[] issuerKeys,uint256 threshold)");
    bytes32 internal constant SET_CONFIG_TYPEHASH =
        keccak256("SetConfig(address issuerId,bytes32 key,bytes32 checksum,uint256 expectedNonce)");
    bytes32 internal constant SET_CONFIG_WITH_DATA_TYPEHASH =
        keccak256("SetConfigWithData(address issuerId,bytes32 key,bytes data,uint256 expectedNonce)");
    bytes32 internal constant UPDATE_ISSUER_SETTINGS_TYPEHASH = keccak256(
        "UpdateIssuerSettings(address issuerId,address[] newIssuerKeys,uint256 newThreshold,uint256 expectedNonce)"
    );

    function registrationStructHash(address[] memory issuerKeys, uint256 threshold) internal pure returns (bytes32) {
        return keccak256(abi.encode(REGISTER_ISSUER_TYPEHASH, keccak256(abi.encodePacked(issuerKeys)), threshold));
    }

    function setConfigStructHash(address issuerId, bytes32 key, bytes32 checksum, uint256 expectedNonce)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(SET_CONFIG_TYPEHASH, issuerId, key, checksum, expectedNonce));
    }

    function setConfigWithDataStructHash(address issuerId, bytes32 key, bytes memory data, uint256 expectedNonce)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(SET_CONFIG_WITH_DATA_TYPEHASH, issuerId, key, keccak256(data), expectedNonce));
    }

    function updateIssuerSettingsStructHash(
        address issuerId,
        address[] memory newIssuerKeys,
        uint256 newThreshold,
        uint256 expectedNonce
    ) internal pure returns (bytes32) {
        return keccak256(
            abi.encode(
                UPDATE_ISSUER_SETTINGS_TYPEHASH,
                issuerId,
                keccak256(abi.encodePacked(newIssuerKeys)),
                newThreshold,
                expectedNonce
            )
        );
    }
}

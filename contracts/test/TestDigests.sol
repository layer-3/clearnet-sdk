// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;
import {ConfigRegistryDigests} from "../src/libraries/ConfigRegistryDigests.sol";

// Test-only domain application. Production applies OpenZeppelin EIP712 in the contract.
library TestDigests {
    function wrap(address registry, bytes32 structure) internal view returns (bytes32) {
        bytes32 domain = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("YellowConfigRegistry"),
                keccak256("1"),
                block.chainid,
                registry
            )
        );
        return keccak256(abi.encodePacked(hex"1901", domain, structure));
    }

    function registrationDigest(address registry, address[] memory keys, uint256 threshold)
        internal
        view
        returns (bytes32)
    {
        return wrap(registry, ConfigRegistryDigests.registrationStructHash(keys, threshold));
    }

    function setConfigDigest(address registry, address issuer, bytes32 key, bytes32 checksum, uint256 nonce)
        internal
        view
        returns (bytes32)
    {
        return wrap(registry, ConfigRegistryDigests.setConfigStructHash(issuer, key, checksum, nonce));
    }

    function setConfigWithDataDigest(address registry, address issuer, bytes32 key, bytes memory data, uint256 nonce)
        internal
        view
        returns (bytes32)
    {
        return wrap(registry, ConfigRegistryDigests.setConfigWithDataStructHash(issuer, key, data, nonce));
    }

    function updateIssuerSettingsDigest(
        address registry,
        address issuer,
        address[] memory keys,
        uint256 threshold,
        uint256 nonce
    ) internal view returns (bytes32) {
        return wrap(registry, ConfigRegistryDigests.updateIssuerSettingsStructHash(issuer, keys, threshold, nonce));
    }
}

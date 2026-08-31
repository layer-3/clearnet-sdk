// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {Test} from "forge-std/Test.sol";

/// @title SignerSetBase
/// @notice Shared signer-set generation and signing helpers for `Config` / `ConfigRegistry` tests.
abstract contract SignerSetBase is Test {
    // -------------------------------------------------------------------------
    // Signer-set generation
    // -------------------------------------------------------------------------

    /// @dev Generate `count` signer key pairs, sorted ascending by address.
    function _generateSigners(uint256 count) internal pure returns (uint256[] memory keys, address[] memory addrs) {
        keys = new uint256[](count);
        addrs = new address[](count);
        for (uint256 i = 0; i < count; i++) {
            keys[i] = i + 1;
            addrs[i] = vm.addr(keys[i]);
        }
        _sortByAddress(keys, addrs);
    }

    /// @dev Generate a second, disjoint sorted signer set, keyed off `seed` so callers can
    ///      produce multiple independent non-issuer / rotation-target sets without collisions
    ///      with `_generateSigners` or with each other (distinct `seed`s never overlap).
    function _makeSignerSet(uint256 seed, uint256 count)
        internal
        pure
        returns (uint256[] memory keys, address[] memory addrs)
    {
        keys = new uint256[](count);
        addrs = new address[](count);
        for (uint256 i = 0; i < count; i++) {
            keys[i] = ((seed + 1) * 1_000_000) + i + 1;
            addrs[i] = vm.addr(keys[i]);
        }
        _sortByAddress(keys, addrs);
    }

    /// @dev Insertion sort both arrays by address ascending.
    function _sortByAddress(uint256[] memory keys, address[] memory addrs) internal pure {
        for (uint256 i = 1; i < addrs.length; i++) {
            address addrKey = addrs[i];
            uint256 privKey = keys[i];
            uint256 j = i;
            while (j > 0 && addrs[j - 1] > addrKey) {
                addrs[j] = addrs[j - 1];
                keys[j] = keys[j - 1];
                j--;
            }
            addrs[j] = addrKey;
            keys[j] = privKey;
        }
    }

    // -------------------------------------------------------------------------
    // Signing
    // -------------------------------------------------------------------------

    /// @dev Sign `digest` with the first `count` of `keys`. `keys` must be sorted ascending by
    ///      address so the produced signatures satisfy the contracts' ascending-order checks.
    function _signDigestWithKeys(bytes32 digest, uint256[] memory keys, uint256 count)
        internal
        pure
        returns (bytes[] memory sigs)
    {
        sigs = new bytes[](count);
        for (uint256 i = 0; i < count; i++) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(keys[i], digest);
            sigs[i] = abi.encodePacked(r, s, v);
        }
    }

    /// @dev Swap the first two signatures, breaking ascending-recovered-address order without
    ///      duplicating the `SignaturesNotOrdered` construction logic at every call site.
    function _swapFirstTwo(bytes[] memory sigs) internal pure returns (bytes[] memory) {
        require(sigs.length >= 2, "need at least 2 signatures to swap");
        bytes memory tmp = sigs[0];
        sigs[0] = sigs[1];
        sigs[1] = tmp;
        return sigs;
    }
}

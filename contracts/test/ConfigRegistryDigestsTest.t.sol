// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {Test} from "forge-std/Test.sol";

import {TestDigests} from "./TestDigests.sol";

/// @title ConfigRegistryDigestsTest
/// @notice EIP-712 field/domain differentiation. Byte-exact encoding is covered
/// by EIP712Vectors.t.sol and the independent Go typed-data reference encoder.
contract ConfigRegistryDigestsTest is Test {
    address internal constant BASE_REGISTRY = address(0xA11CE);
    address internal constant BASE_ISSUER_ID = address(0xB0B);
    bytes32 internal constant BASE_KEY = keccak256("key");
    bytes32 internal constant BASE_CHECKSUM = keccak256("checksum");
    bytes internal constant BASE_DATA = "data";
    uint256 internal constant BASE_NONCE = 7;
    uint256 internal constant BASE_THRESHOLD = 3;

    address[] internal baseKeys;
    address[] internal baseNewKeys;

    function setUp() public {
        baseKeys = new address[](3);
        baseKeys[0] = address(0x1);
        baseKeys[1] = address(0x2);
        baseKeys[2] = address(0x3);

        baseNewKeys = new address[](3);
        baseNewKeys[0] = address(0x4);
        baseNewKeys[1] = address(0x5);
        baseNewKeys[2] = address(0x6);
    }

    // -------------------------------------------------------------------------
    // registrationDigest
    // -------------------------------------------------------------------------

    function test_registrationDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != TestDigests.registrationDigest(registry, baseKeys, BASE_THRESHOLD)
        );
    }

    function test_registrationDigest_changes_ifKeysDiffer(address[] memory keys) public view {
        vm.assume(keccak256(abi.encode(keys)) != keccak256(abi.encode(baseKeys)));
        assertTrue(
            TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != TestDigests.registrationDigest(BASE_REGISTRY, keys, BASE_THRESHOLD)
        );
    }

    function test_registrationDigest_changes_ifThresholdDiffers(uint256 threshold_) public view {
        vm.assume(threshold_ != BASE_THRESHOLD);
        assertTrue(
            TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, threshold_)
        );
    }

    function test_registrationDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD);
        vm.chainId(otherChainId);
        bytes32 after_ = TestDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD);
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // setConfigDigest
    // -------------------------------------------------------------------------

    function test_setConfigDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != TestDigests.setConfigDigest(registry, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != TestDigests.setConfigDigest(BASE_REGISTRY, issuerId, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifKeyDiffers(bytes32 key) public view {
        vm.assume(key != BASE_KEY);
        assertTrue(
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, key, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifChecksumDiffers(bytes32 checksum) public view {
        vm.assume(checksum != BASE_CHECKSUM);
        assertTrue(
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, checksum, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, expectedNonce)
        );
    }

    function test_setConfigDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        vm.chainId(otherChainId);
        bytes32 after_ = TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // setConfigWithDataDigest
    // -------------------------------------------------------------------------

    function test_setConfigWithDataDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
                != TestDigests.setConfigWithDataDigest(registry, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
        );
    }

    function test_setConfigWithDataDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
                != TestDigests.setConfigWithDataDigest(BASE_REGISTRY, issuerId, BASE_KEY, BASE_DATA, BASE_NONCE)
        );
    }

    function test_setConfigWithDataDigest_changes_ifKeyDiffers(bytes32 key) public view {
        vm.assume(key != BASE_KEY);
        assertTrue(
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
                != TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, key, BASE_DATA, BASE_NONCE)
        );
    }

    function test_setConfigWithDataDigest_changes_ifDataDiffers(bytes memory data) public view {
        vm.assume(keccak256(data) != keccak256(BASE_DATA));
        assertTrue(
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
                != TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, data, BASE_NONCE)
        );
    }

    function test_setConfigWithDataDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE)
                != TestDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, expectedNonce
                )
        );
    }

    function test_setConfigWithDataDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before =
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE);
        vm.chainId(otherChainId);
        bytes32 after_ =
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE);
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // updateIssuerSettingsDigest
    // -------------------------------------------------------------------------

    function test_updateIssuerSettingsDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
                != TestDigests.updateIssuerSettingsDigest(
                    registry, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
                != TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, issuerId, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNewKeysDiffer(address[] memory newIssuerKeys) public view {
        vm.assume(keccak256(abi.encode(newIssuerKeys)) != keccak256(abi.encode(baseNewKeys)));
        assertTrue(
            TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
                != TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, newIssuerKeys, BASE_THRESHOLD, BASE_NONCE
                )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNewThresholdDiffers(uint256 newThreshold) public view {
        vm.assume(newThreshold != BASE_THRESHOLD);
        assertTrue(
            TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
                != TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, newThreshold, BASE_NONCE
                )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
                )
                != TestDigests.updateIssuerSettingsDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, expectedNonce
                )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = TestDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        vm.chainId(otherChainId);
        bytes32 after_ = TestDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // Tag separation
    // -------------------------------------------------------------------------

    function test_tagSeparation_setConfigVsSetConfigWithData() public view {
        bytes32 setConfigDigest =
            TestDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        bytes32 setConfigWithDataDigest =
            TestDigests.setConfigWithDataDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE);
        assertTrue(setConfigDigest != setConfigWithDataDigest);
    }

    function test_tagSeparation_registrationVsUpdateIssuerSettings() public view {
        bytes32 registrationDigest = TestDigests.registrationDigest(BASE_REGISTRY, baseNewKeys, BASE_THRESHOLD);
        bytes32 updateDigest = TestDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        assertTrue(registrationDigest != updateDigest);
    }
}

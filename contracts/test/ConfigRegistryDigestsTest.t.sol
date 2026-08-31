// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {Test, console} from "forge-std/Test.sol";

import {ConfigRegistryDigests} from "../src/libraries/ConfigRegistryDigests.sol";

/// @title ConfigRegistryDigestsTest
/// @notice `ConfigRegistryDigests` is all-`internal`, so this contract calls it directly with no
///         harness. Every digest is checked against a hand-written `keccak256(abi.encode(...))`
///         expansion (pinning the exact tag string and, where applicable, the inner
///         `keccak256(abi.encode(keys, threshold))` pre-hash), then probed for separation across
///         each input, chain id, registry, and entrypoint tag.
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

    function test_registrationDigest_matchesHandExpansion(address registry, address[] memory keys, uint256 threshold_)
        public
        view
    {
        bytes32 expected =
            keccak256(abi.encode(block.chainid, registry, "registerIssuer", keccak256(abi.encode(keys, threshold_))));
        assertEq(ConfigRegistryDigests.registrationDigest(registry, keys, threshold_), expected);
    }

    /// @dev Non-fuzzed reference for off-chain (BE) implementers: pinned inputs so the logged
    ///      intermediate encodings and final digest are stable and reproducible across runs, run
    ///      with `-vvv` to see them. `test_registrationDigest_matchesHandExpansion` above is the
    ///      actual correctness test; fuzzed console output there would change every run.
    function test_registrationDigest_pinnedExample() public view {
        bytes memory encodedIssuerSettings = abi.encode(baseKeys, BASE_THRESHOLD);
        // 0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003
        console.log("encoded issuer settings:");
        console.logBytes(encodedIssuerSettings);

        bytes32 hashedIssuerSettings = keccak256(encodedIssuerSettings);
        // 0x2d214bb58c991848772f27b1a03f84fd68bc7b1a80392f338801ebf3697d8e83
        console.log("hashed issuer settings:");
        console.logBytes32(hashedIssuerSettings);

        bytes memory encoded = abi.encode(block.chainid, BASE_REGISTRY, "registerIssuer", hashedIssuerSettings);
        // 0x0000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000000000000000000000000000000000000000a11ce00000000000000000000000000000000000000000000000000000000000000802d214bb58c991848772f27b1a03f84fd68bc7b1a80392f338801ebf3697d8e83000000000000000000000000000000000000000000000000000000000000000e7265676973746572497373756572000000000000000000000000000000000000
        console.log("encoded registration message:");
        console.logBytes(encoded);

        bytes32 expected = keccak256(encoded);
        // 0x5b3d8e9ac53b97a28f70f5231cb17fa5034bd2e77fe1f5e256c66b790e068300
        console.log("hashed registration digest:");
        console.logBytes32(expected);

        assertEq(ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD), expected);
    }

    function test_registrationDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != ConfigRegistryDigests.registrationDigest(registry, baseKeys, BASE_THRESHOLD)
        );
    }

    function test_registrationDigest_changes_ifKeysDiffer(address[] memory keys) public view {
        vm.assume(keccak256(abi.encode(keys)) != keccak256(abi.encode(baseKeys)));
        assertTrue(
            ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, keys, BASE_THRESHOLD)
        );
    }

    function test_registrationDigest_changes_ifThresholdDiffers(uint256 threshold_) public view {
        vm.assume(threshold_ != BASE_THRESHOLD);
        assertTrue(
            ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD)
                != ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, threshold_)
        );
    }

    function test_registrationDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD);
        vm.chainId(otherChainId);
        bytes32 after_ = ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseKeys, BASE_THRESHOLD);
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // setConfigDigest
    // -------------------------------------------------------------------------

    function test_setConfigDigest_matchesHandExpansion(
        address registry,
        address issuerId,
        bytes32 key,
        bytes32 checksum,
        uint256 expectedNonce
    ) public view {
        bytes32 expected = keccak256(
            abi.encode(block.chainid, registry, "setConfig", issuerId, key, checksum, expectedNonce)
        );
        assertEq(ConfigRegistryDigests.setConfigDigest(registry, issuerId, key, checksum, expectedNonce), expected);
    }

    /// @dev Non-fuzzed reference for off-chain (BE) implementers: pinned inputs so the logged
    ///      encoding and final digest are stable and reproducible across runs, run with `-vvv`
    ///      to see them. `test_setConfigDigest_matchesHandExpansion` above is the actual
    ///      correctness test; fuzzed console output there would change every run.
    function test_setConfigDigest_pinnedExample() public view {
        bytes memory encoded =
            abi.encode(block.chainid, BASE_REGISTRY, "setConfig", BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        // 0x0000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000000000000000000000000000000000000000a11ce00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000b0b07855b46a623a8ecabac76ed697aa4e13631e3b6718c8a0d342860c13c30d2fcb902dd46ff1895e2e3efbe277c6b8857ffc82d78a1f8a424cb69a98d1d793a7800000000000000000000000000000000000000000000000000000000000000070000000000000000000000000000000000000000000000000000000000000009736574436f6e6669670000000000000000000000000000000000000000000000
        console.log("encoded setConfig message:");
        console.logBytes(encoded);

        bytes32 expected = keccak256(encoded);
        // 0x4421090980e2549f86f38e4ea769c68c0142f0a1b3cd446971ff12241743c3b0
        console.log("hashed setConfig digest:");
        console.logBytes32(expected);

        assertEq(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE),
            expected
        );
    }

    function test_setConfigDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != ConfigRegistryDigests.setConfigDigest(registry, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, issuerId, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifKeyDiffers(bytes32 key) public view {
        vm.assume(key != BASE_KEY);
        assertTrue(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, key, BASE_CHECKSUM, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifChecksumDiffers(bytes32 checksum) public view {
        vm.assume(checksum != BASE_CHECKSUM);
        assertTrue(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, checksum, BASE_NONCE)
        );
    }

    function test_setConfigDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE)
                != ConfigRegistryDigests.setConfigDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, expectedNonce
                )
        );
    }

    function test_setConfigDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before =
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        vm.chainId(otherChainId);
        bytes32 after_ =
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // setConfigWithDataDigest
    // -------------------------------------------------------------------------

    function test_setConfigWithDataDigest_matchesHandExpansion(
        address registry,
        address issuerId,
        bytes32 key,
        bytes memory data,
        uint256 expectedNonce
    ) public view {
        bytes32 expected = keccak256(
            abi.encode(block.chainid, registry, "setConfigWithData", issuerId, key, data, expectedNonce)
        );
        assertEq(ConfigRegistryDigests.setConfigWithDataDigest(registry, issuerId, key, data, expectedNonce), expected);
    }

    /// @dev Non-fuzzed reference for off-chain (BE) implementers: pinned inputs so the logged
    ///      encoding and final digest are stable and reproducible across runs, run with `-vvv`
    ///      to see them. `test_setConfigWithDataDigest_matchesHandExpansion` above is the actual
    ///      correctness test; fuzzed console output there would change every run.
    function test_setConfigWithDataDigest_pinnedExample() public view {
        bytes memory encoded = abi.encode(
            block.chainid, BASE_REGISTRY, "setConfigWithData", BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
        );
        // 0x0000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000000000000000000000000000000000000000a11ce00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000b0b07855b46a623a8ecabac76ed697aa4e13631e3b6718c8a0d342860c13c30d2fc000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000070000000000000000000000000000000000000000000000000000000000000011736574436f6e666967576974684461746100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046461746100000000000000000000000000000000000000000000000000000000
        console.log("encoded setConfigWithData message:");
        console.logBytes(encoded);

        bytes32 expected = keccak256(encoded);
        // 0x8be03704cad22b4cdd16d94e0f555ebfc1799059ae2984c5ae706323dde1c560
        console.log("hashed setConfigWithData digest:");
        console.logBytes32(expected);

        assertEq(
            ConfigRegistryDigests.setConfigWithDataDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
            ),
            expected
        );
    }

    function test_setConfigWithDataDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            ConfigRegistryDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                )
                != ConfigRegistryDigests.setConfigWithDataDigest(
                        registry, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                    )
        );
    }

    function test_setConfigWithDataDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            ConfigRegistryDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                )
                != ConfigRegistryDigests.setConfigWithDataDigest(
                        BASE_REGISTRY, issuerId, BASE_KEY, BASE_DATA, BASE_NONCE
                    )
        );
    }

    function test_setConfigWithDataDigest_changes_ifKeyDiffers(bytes32 key) public view {
        vm.assume(key != BASE_KEY);
        assertTrue(
            ConfigRegistryDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                )
                != ConfigRegistryDigests.setConfigWithDataDigest(
                        BASE_REGISTRY, BASE_ISSUER_ID, key, BASE_DATA, BASE_NONCE
                    )
        );
    }

    function test_setConfigWithDataDigest_changes_ifDataDiffers(bytes memory data) public view {
        vm.assume(keccak256(data) != keccak256(BASE_DATA));
        assertTrue(
            ConfigRegistryDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                )
                != ConfigRegistryDigests.setConfigWithDataDigest(
                        BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, data, BASE_NONCE
                    )
        );
    }

    function test_setConfigWithDataDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            ConfigRegistryDigests.setConfigWithDataDigest(
                    BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
                )
                != ConfigRegistryDigests.setConfigWithDataDigest(
                        BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, expectedNonce
                    )
        );
    }

    function test_setConfigWithDataDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = ConfigRegistryDigests.setConfigWithDataDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
        );
        vm.chainId(otherChainId);
        bytes32 after_ = ConfigRegistryDigests.setConfigWithDataDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
        );
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // updateIssuerSettingsDigest
    // -------------------------------------------------------------------------

    function test_updateIssuerSettingsDigest_matchesHandExpansion(
        address registry,
        address issuerId,
        address[] memory newIssuerKeys,
        uint256 newThreshold,
        uint256 expectedNonce
    ) public view {
        bytes32 expected = keccak256(
            abi.encode(
                block.chainid,
                registry,
                "updateIssuerSettings",
                issuerId,
                keccak256(abi.encode(newIssuerKeys, newThreshold)),
                expectedNonce
            )
        );
        assertEq(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                registry, issuerId, newIssuerKeys, newThreshold, expectedNonce
            ),
            expected
        );
    }

    /// @dev Non-fuzzed reference for off-chain (BE) implementers: pinned inputs so the logged
    ///      intermediate encodings and final digest are stable and reproducible across runs, run
    ///      with `-vvv` to see them. `test_updateIssuerSettingsDigest_matchesHandExpansion` above
    ///      is the actual correctness test; fuzzed console output there would change every run.
    function test_updateIssuerSettingsDigest_pinnedExample() public view {
        bytes memory encodedIssuerSettings = abi.encode(baseNewKeys, BASE_THRESHOLD);
        // 0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000006
        console.log("encoded issuer settings:");
        console.logBytes(encodedIssuerSettings);

        bytes32 hashedIssuerSettings = keccak256(encodedIssuerSettings);
        // 0x6e6a324991d52b4d10f97a47318de54c5c1aebe1a786e5f219f9a2c23b11ded5
        console.log("hashed issuer settings:");
        console.logBytes32(hashedIssuerSettings);

        bytes memory encoded = abi.encode(
            block.chainid, BASE_REGISTRY, "updateIssuerSettings", BASE_ISSUER_ID, hashedIssuerSettings, BASE_NONCE
        );
        // 0x0000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000000000000000000000000000000000000000a11ce00000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000b0b6e6a324991d52b4d10f97a47318de54c5c1aebe1a786e5f219f9a2c23b11ded50000000000000000000000000000000000000000000000000000000000000007000000000000000000000000000000000000000000000000000000000000001475706461746549737375657253657474696e6773000000000000000000000000
        console.log("encoded updateIssuerSettings message:");
        console.logBytes(encoded);

        bytes32 expected = keccak256(encoded);
        // 0x7ac8089411d6fdf6a9de709a227cde395af11729cb64aff0722dce7878941d42
        console.log("hashed updateIssuerSettings digest:");
        console.logBytes32(expected);

        assertEq(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            ),
            expected
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifRegistryDiffers(address registry) public view {
        vm.assume(registry != BASE_REGISTRY);
        assertTrue(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
            != ConfigRegistryDigests.updateIssuerSettingsDigest(
                registry, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifIssuerIdDiffers(address issuerId) public view {
        vm.assume(issuerId != BASE_ISSUER_ID);
        assertTrue(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
            != ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, issuerId, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNewKeysDiffer(address[] memory newIssuerKeys) public view {
        vm.assume(keccak256(abi.encode(newIssuerKeys)) != keccak256(abi.encode(baseNewKeys)));
        assertTrue(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
            != ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, newIssuerKeys, BASE_THRESHOLD, BASE_NONCE
            )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNewThresholdDiffers(uint256 newThreshold) public view {
        vm.assume(newThreshold != BASE_THRESHOLD);
        assertTrue(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
            != ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, newThreshold, BASE_NONCE
            )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifNonceDiffers(uint256 expectedNonce) public view {
        vm.assume(expectedNonce != BASE_NONCE);
        assertTrue(
            ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
            )
            != ConfigRegistryDigests.updateIssuerSettingsDigest(
                BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, expectedNonce
            )
        );
    }

    function test_updateIssuerSettingsDigest_changes_ifChainIdDiffers(uint64 otherChainId) public {
        vm.assume(otherChainId != block.chainid);
        bytes32 before = ConfigRegistryDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        vm.chainId(otherChainId);
        bytes32 after_ = ConfigRegistryDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        assertTrue(before != after_);
    }

    // -------------------------------------------------------------------------
    // Tag separation
    // -------------------------------------------------------------------------

    /// @dev `setConfig` and `setConfigWithData` digests are distinct even over comparable
    ///      inputs (checksum vs. the bytes that would hash to it), because the tag string and
    ///      encoded shape differ.
    function test_tagSeparation_setConfigVsSetConfigWithData() public view {
        bytes32 setConfigDigest =
            ConfigRegistryDigests.setConfigDigest(BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_CHECKSUM, BASE_NONCE);
        bytes32 setConfigWithDataDigest = ConfigRegistryDigests.setConfigWithDataDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, BASE_KEY, BASE_DATA, BASE_NONCE
        );
        assertTrue(setConfigDigest != setConfigWithDataDigest);
    }

    /// @dev `registrationDigest` and `updateIssuerSettingsDigest` are distinct for the same
    ///      `(keys, threshold)` pre-hash — `updateIssuerSettingsDigest` additionally binds
    ///      `issuerId`, but even holding that aside the tag string alone separates them.
    function test_tagSeparation_registrationVsUpdateIssuerSettings() public view {
        bytes32 registrationDigest =
            ConfigRegistryDigests.registrationDigest(BASE_REGISTRY, baseNewKeys, BASE_THRESHOLD);
        bytes32 updateDigest = ConfigRegistryDigests.updateIssuerSettingsDigest(
            BASE_REGISTRY, BASE_ISSUER_ID, baseNewKeys, BASE_THRESHOLD, BASE_NONCE
        );
        assertTrue(registrationDigest != updateDigest);
    }
}

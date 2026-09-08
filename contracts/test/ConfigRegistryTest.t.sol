// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {Test} from "forge-std/Test.sol";

import {Config} from "../src/Config.sol";
import {ConfigRegistry} from "../src/ConfigRegistry.sol";
import {IConfig} from "../src/interfaces/IConfig.sol";
import {IConfigRegistry} from "../src/interfaces/IConfigRegistry.sol";
import {TestDigests} from "./TestDigests.sol";

import {SignerSetBase} from "./SignerSetBase.sol";

/// @title ConfigRegistryTestBase
/// @notice Shared `setUp` and signing helpers for every `ConfigRegistryTest*` contract in this
///         file. Deploys a fresh `ConfigRegistry` and registers one issuer (7 sorted signer
///         keys, `THRESHOLD` of 5) so per-function test contracts start from a common,
///         already-registered issuer without duplicating the registration dance.
abstract contract ConfigRegistryTestBase is SignerSetBase {
    uint256 internal constant SIGNER_COUNT = 7;
    uint256 internal constant THRESHOLD = 5;

    ConfigRegistry internal registry;

    /// @dev The registered issuer's signer set (sorted ascending by address) and its `issuerId`.
    uint256[] internal issuerPrivKeys;
    address[] internal issuerSignerAddrs;
    address internal issuerId;

    function setUp() public virtual {
        registry = new ConfigRegistry();

        (issuerPrivKeys, issuerSignerAddrs) = _generateSigners(SIGNER_COUNT);
        issuerId = _registerIssuer(issuerSignerAddrs, THRESHOLD, issuerPrivKeys, THRESHOLD);
    }

    // -------------------------------------------------------------------------
    // Digest signing helpers (always via `ConfigRegistryDigests`, never a hand-rolled encoding)
    // -------------------------------------------------------------------------

    /// @dev Sign a `registerIssuer` message; does not submit it.
    function _signRegistration(
        address[] memory keys,
        uint256 threshold_,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal view returns (bytes[] memory sigs) {
        bytes32 digest = TestDigests.registrationDigest(address(registry), keys, threshold_);
        sigs = _signDigestWithKeys(digest, signingKeys, signerCount);
    }

    /// @dev Sign and submit `registerIssuer`.
    function _registerIssuer(
        address[] memory keys,
        uint256 threshold_,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal returns (address) {
        bytes[] memory sigs = _signRegistration(keys, threshold_, signingKeys, signerCount);
        return registry.registerIssuer(keys, threshold_, sigs);
    }

    /// @dev Sign a `setConfig` message; does not submit it.
    function _signSetConfig(
        address issuerId_,
        bytes32 key,
        bytes32 checksum,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal view returns (bytes[] memory sigs) {
        bytes32 digest = TestDigests.setConfigDigest(address(registry), issuerId_, key, checksum, expectedNonce);
        sigs = _signDigestWithKeys(digest, signingKeys, signerCount);
    }

    /// @dev Sign and submit `setConfig`.
    function _setConfig(
        address issuerId_,
        bytes32 key,
        bytes32 checksum,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal {
        bytes[] memory sigs = _signSetConfig(issuerId_, key, checksum, expectedNonce, signingKeys, signerCount);
        registry.setConfig(issuerId_, key, checksum, expectedNonce, sigs);
    }

    /// @dev Sign a `setConfigWithData` message; does not submit it.
    function _signSetConfigWithData(
        address issuerId_,
        bytes32 key,
        bytes memory data,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal view returns (bytes[] memory sigs) {
        bytes32 digest = TestDigests.setConfigWithDataDigest(address(registry), issuerId_, key, data, expectedNonce);
        sigs = _signDigestWithKeys(digest, signingKeys, signerCount);
    }

    /// @dev Sign and submit `setConfigWithData`.
    function _setConfigWithData(
        address issuerId_,
        bytes32 key,
        bytes memory data,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal {
        bytes[] memory sigs = _signSetConfigWithData(issuerId_, key, data, expectedNonce, signingKeys, signerCount);
        registry.setConfigWithData(issuerId_, key, data, expectedNonce, sigs);
    }

    /// @dev Sign an `updateIssuerSettings` message; does not submit it.
    function _signUpdateIssuerSettings(
        address issuerId_,
        address[] memory newKeys,
        uint256 newThreshold,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal view returns (bytes[] memory sigs) {
        bytes32 digest = TestDigests.updateIssuerSettingsDigest(
            address(registry), issuerId_, newKeys, newThreshold, expectedNonce
        );
        sigs = _signDigestWithKeys(digest, signingKeys, signerCount);
    }

    /// @dev Sign and submit `updateIssuerSettings`.
    function _updateIssuerSettings(
        address issuerId_,
        address[] memory newKeys,
        uint256 newThreshold,
        uint256 expectedNonce,
        uint256[] memory signingKeys,
        uint256 signerCount
    ) internal {
        bytes[] memory sigs = _signUpdateIssuerSettings(
            issuerId_, newKeys, newThreshold, expectedNonce, signingKeys, signerCount
        );
        registry.updateIssuerSettings(issuerId_, newKeys, newThreshold, expectedNonce, sigs);
    }
}

/// @title ConfigRegistryTest
/// @notice Deployment and view-function behaviour of `ConfigRegistry`, independent of any single entrypoint.
contract ConfigRegistryTest is ConfigRegistryTestBase {
    function test_isRegistered_false_forUnregisteredIssuer(address addr) public view {
        vm.assume(addr != issuerId);
        assertFalse(registry.isRegistered(addr));
    }

    function test_threshold_zero_forUnregisteredIssuer(address addr) public view {
        vm.assume(addr != issuerId);
        assertEq(registry.threshold(addr), 0);
    }

    function test_nonce_zero_forUnregisteredIssuer(address addr) public view {
        vm.assume(addr != issuerId);
        assertEq(registry.nonce(addr), 0);
    }

    function test_issuerKeys_empty_forUnregisteredIssuer(address addr) public view {
        vm.assume(addr != issuerId);
        assertEq(registry.issuerKeys(addr).length, 0);
    }

    function test_issuerSettings_matchesIndividualGetters() public view {
        (address[] memory keys, uint256 threshold_, uint256 nonce_) = registry.issuerSettings(issuerId);
        assertEq(keys, registry.issuerKeys(issuerId));
        assertEq(threshold_, registry.threshold(issuerId));
        assertEq(nonce_, registry.nonce(issuerId));
    }

    /// @dev Cross-checks `computeIssuerId` against the same `vm.computeCreate2Address` oracle the
    ///      `registerIssuer` CREATE2 tests use, rather than against itself.
    function test_computeIssuerId_matchesCreate2Address(address[] memory keys, uint256 threshold_) public view {
        address predicted = registry.computeIssuerId(keys, threshold_);
        address expected = vm.computeCreate2Address(
            keccak256(abi.encode(keys, threshold_)),
            keccak256(abi.encodePacked(type(Config).creationCode, abi.encode(address(registry)))),
            address(registry)
        );
        assertEq(predicted, expected);
    }
}

/// @title ConfigRegistryTest_registerIssuer
/// @notice `registerIssuer` behaviour: CREATE2 determinism/idempotency, validation ordering, and
///         the deliberate no-key-count-floor behaviour.
contract ConfigRegistryTest_registerIssuer is ConfigRegistryTestBase {
    function test_registerIssuer_returnsIssuerIdWithDeployedCode() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(1, SIGNER_COUNT);
        address newIssuerId = _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD);

        assertGt(newIssuerId.code.length, 0);
        assertEq(Config(newIssuerId).owner(), address(registry));
    }

    function test_registerIssuer_storesSettingsVerbatim() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(2, SIGNER_COUNT);
        address newIssuerId = _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD);

        assertTrue(registry.isRegistered(newIssuerId));
        assertEq(registry.issuerKeys(newIssuerId), addrs);
        assertEq(registry.threshold(newIssuerId), THRESHOLD);
        assertEq(registry.nonce(newIssuerId), 0);
    }

    function test_registerIssuer_emitsIssuerRegistered() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(3, SIGNER_COUNT);
        bytes[] memory sigs = _signRegistration(addrs, THRESHOLD, keys, THRESHOLD);

        vm.expectEmit();
        emit IConfigRegistry.IssuerRegistered(
            vm.computeCreate2Address(
                keccak256(abi.encode(addrs, THRESHOLD)),
                keccak256(abi.encodePacked(type(Config).creationCode, abi.encode(address(registry)))),
                address(registry)
            ),
            addrs,
            THRESHOLD
        );
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_create2AddressIsDeterministic() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(4, SIGNER_COUNT);

        address predicted = vm.computeCreate2Address(
            keccak256(abi.encode(addrs, THRESHOLD)),
            keccak256(abi.encodePacked(type(Config).creationCode, abi.encode(address(registry)))),
            address(registry)
        );

        address actual = _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD);
        assertEq(actual, predicted);
    }

    /// @dev Registering the exact same `(keys, threshold)` tuple twice predicts the same
    ///      `issuerId` and is rejected by the explicit pre-check before any deployment is attempted.
    function test_registerIssuer_revert_ifTupleAlreadyRegistered() public {
        vm.expectRevert(IConfigRegistry.IssuerAlreadyRegistered.selector);
        _registerIssuer(issuerSignerAddrs, THRESHOLD, issuerPrivKeys, THRESHOLD);
    }

    function test_registerIssuer_differentKeySet_yieldsDifferentIssuerId() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(5, SIGNER_COUNT);
        address newIssuerId = _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD);
        assertTrue(newIssuerId != issuerId);
    }

    function test_registerIssuer_differentThreshold_yieldsDifferentIssuerId() public {
        address newIssuerId = _registerIssuer(issuerSignerAddrs, THRESHOLD - 1, issuerPrivKeys, THRESHOLD - 1);
        assertTrue(newIssuerId != issuerId);
    }

    function test_computeIssuerId_matchesActualIssuerIdAfterRegistration() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(60, SIGNER_COUNT);
        address predicted = registry.computeIssuerId(addrs, THRESHOLD);

        address actual = _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD);
        assertEq(predicted, actual);
    }

    /// @dev `computeIssuerId` is a pure prediction: it does not require `isRegistered` to be true.
    function test_computeIssuerId_worksForNeverRegisteredTuple() public view {
        (, address[] memory addrs) = _makeSignerSet(61, SIGNER_COUNT);
        address predicted = registry.computeIssuerId(addrs, THRESHOLD);
        assertFalse(registry.isRegistered(predicted));
    }

    function test_computeIssuerId_differentKeySet_yieldsDifferentPredictedIssuerId() public view {
        (, address[] memory addrs) = _makeSignerSet(62, SIGNER_COUNT);
        assertTrue(registry.computeIssuerId(addrs, THRESHOLD) != registry.computeIssuerId(issuerSignerAddrs, THRESHOLD));
    }

    function test_computeIssuerId_differentThreshold_yieldsDifferentPredictedIssuerId() public view {
        assertTrue(
            registry.computeIssuerId(issuerSignerAddrs, THRESHOLD)
                != registry.computeIssuerId(issuerSignerAddrs, THRESHOLD - 1)
        );
    }

    function test_registerIssuer_differentRegistry_yieldsDifferentIssuerId() public {
        ConfigRegistry otherRegistry = new ConfigRegistry();
        bytes32 digest = TestDigests.registrationDigest(address(otherRegistry), issuerSignerAddrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, issuerPrivKeys, THRESHOLD);
        address otherIssuerId = otherRegistry.registerIssuer(issuerSignerAddrs, THRESHOLD, sigs);

        assertTrue(otherIssuerId != issuerId);
    }

    function test_registerIssuer_revert_ifThresholdIsZero() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(6, SIGNER_COUNT);
        bytes[] memory sigs = _signRegistration(addrs, 1, keys, 1);

        vm.expectRevert(IConfigRegistry.InvalidThreshold.selector);
        registry.registerIssuer(addrs, 0, sigs);
    }

    function test_registerIssuer_revert_ifKeyCountBelowThreshold() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(7, 3);
        bytes[] memory sigs = _signRegistration(addrs, 4, keys, 3);

        vm.expectRevert(IConfigRegistry.NotEnoughIssuerKeys.selector);
        registry.registerIssuer(addrs, 4, sigs);
    }

    function test_registerIssuer_revert_ifKeyCountBelowThreshold_emptyArray() public {
        address[] memory addrs = new address[](0);
        bytes[] memory sigs = new bytes[](0);

        vm.expectRevert(IConfigRegistry.NotEnoughIssuerKeys.selector);
        registry.registerIssuer(addrs, 1, sigs);
    }

    function test_registerIssuer_revert_ifAnyKeyIsZero() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(8, SIGNER_COUNT);
        addrs[0] = address(0);

        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, keys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.ZeroIssuerKey.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_revert_ifKeysUnsorted() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(9, SIGNER_COUNT);
        // Swap two entries to break ascending order.
        (addrs[0], addrs[1]) = (addrs[1], addrs[0]);
        (keys[0], keys[1]) = (keys[1], keys[0]);

        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, keys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.IssuerKeysNotSorted.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_revert_ifKeysContainDuplicate() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(10, SIGNER_COUNT);
        addrs[1] = addrs[0];
        keys[1] = keys[0];

        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, keys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.IssuerKeysNotSorted.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_revert_ifTooFewSignatures() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(11, SIGNER_COUNT);

        vm.expectRevert(IConfigRegistry.BelowThreshold.selector);
        _registerIssuer(addrs, THRESHOLD, keys, THRESHOLD - 1);
    }

    function test_registerIssuer_revert_ifSignaturesUnordered() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(12, SIGNER_COUNT);
        bytes[] memory sigs = _swapFirstTwo(_signRegistration(addrs, THRESHOLD, keys, THRESHOLD));

        vm.expectRevert(IConfigRegistry.SignaturesNotOrdered.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_revert_ifSameSignerTwice() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(13, SIGNER_COUNT);
        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, keys, THRESHOLD);
        sigs[1] = sigs[0]; // duplicate the first signer's signature into the second slot

        vm.expectRevert(IConfigRegistry.SignaturesNotOrdered.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    function test_registerIssuer_revert_ifSignerIsNotAnIssuerKey() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(14, SIGNER_COUNT);
        (uint256[] memory outsiderKeys,) = _makeSignerSet(15, 1);

        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        // Sign with threshold - 1 real keys plus one outsider key; the outsider is not in `addrs`.
        bytes[] memory realSigs = _signDigestWithKeys(digest, keys, THRESHOLD - 1);
        bytes[] memory outsiderSigs = _signDigestWithKeys(digest, outsiderKeys, 1);

        bytes[] memory sigs = new bytes[](THRESHOLD);
        for (uint256 i = 0; i < THRESHOLD - 1; i++) {
            sigs[i] = realSigs[i];
        }
        sigs[THRESHOLD - 1] = outsiderSigs[0];

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }

    /// @dev Deliberate: `_validateIssuerKeys` enforces `keys.length >= threshold_` only, no
    ///      absolute floor. A 1-key/threshold-1 issuer must register successfully; see the NOTE in
    ///      `ConfigRegistry._validateIssuerKeys`.
    function test_registerIssuer_singleKeyThresholdOne_succeeds() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(16, 1);
        address newIssuerId = _registerIssuer(addrs, 1, keys, 1);

        assertTrue(registry.isRegistered(newIssuerId));
        assertEq(registry.threshold(newIssuerId), 1);
    }

    function test_registerIssuer_excessSignatures_succeeds() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(17, SIGNER_COUNT);
        // Supply all 7 signatures when threshold is 5.
        address newIssuerId = _registerIssuer(addrs, THRESHOLD, keys, SIGNER_COUNT);

        assertTrue(registry.isRegistered(newIssuerId));
    }

    function test_registerIssuer_revert_ifChainIdDiffers() public {
        (uint256[] memory keys, address[] memory addrs) = _makeSignerSet(18, SIGNER_COUNT);

        vm.chainId(block.chainid + 1);
        bytes32 digest = TestDigests.registrationDigest(address(registry), addrs, THRESHOLD);
        bytes[] memory sigs = _signDigestWithKeys(digest, keys, THRESHOLD);
        vm.chainId(block.chainid - 1);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.registerIssuer(addrs, THRESHOLD, sigs);
    }
}

/// @title ConfigRegistryTest_setConfig
/// @notice `setConfig` behaviour: happy path, event shapes, replay/tamper/domain-separation, and
///         the shared validation-order reverts.
contract ConfigRegistryTest_setConfig is ConfigRegistryTestBase {
    function test_setConfig_forwardsToConfig(bytes32 key, bytes32 checksum) public {
        vm.assume(key != bytes32(0));

        _setConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD);

        assertEq(Config(issuerId).latestConfigChecksum(key), checksum);
        assertEq(Config(issuerId).configEpoch(key), 1);
        assertEq(registry.nonce(issuerId), 1);
    }

    function test_setConfig_emitsConfigCommittedAndInnerConfigSet(bytes32 key, bytes32 checksum) public {
        vm.assume(key != bytes32(0));
        bytes[] memory sigs = _signSetConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD);

        vm.expectEmit();
        emit IConfig.ConfigSet(key, address(registry), checksum, 1);
        vm.expectEmit();
        emit IConfigRegistry.ConfigCommitted(issuerId, key, checksum, 1);
        registry.setConfig(issuerId, key, checksum, 0, sigs);
    }

    function test_setConfig_sequentialCommits_nonceAdvancesAcrossKeys(bytes32 keyA, bytes32 keyB) public {
        vm.assume(keyA != bytes32(0) && keyB != bytes32(0));
        vm.assume(keyA != keyB);

        _setConfig(issuerId, keyA, keccak256("1"), 0, issuerPrivKeys, THRESHOLD);
        _setConfig(issuerId, keyB, keccak256("2"), 1, issuerPrivKeys, THRESHOLD);
        _setConfig(issuerId, keyA, keccak256("3"), 2, issuerPrivKeys, THRESHOLD);

        assertEq(registry.nonce(issuerId), 3);
        // Per-key epochs advance independently of the shared nonce.
        assertEq(Config(issuerId).configEpoch(keyA), 2);
        assertEq(Config(issuerId).configEpoch(keyB), 1);
    }

    function test_setConfig_revert_ifIssuerNotRegistered(address randomIssuer, bytes32 key, bytes32 checksum) public {
        vm.assume(!registry.isRegistered(randomIssuer));
        bytes[] memory sigs = new bytes[](0);

        vm.expectRevert(IConfigRegistry.IssuerNotRegistered.selector);
        registry.setConfig(randomIssuer, key, checksum, 0, sigs);
    }

    function test_setConfig_revert_ifNonceIsStale(uint256 supplied) public {
        vm.assume(supplied != 0);
        bytes[] memory sigs =
            _signSetConfig(issuerId, keccak256("k"), keccak256("v"), supplied, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 0, supplied));
        registry.setConfig(issuerId, keccak256("k"), keccak256("v"), supplied, sigs);
    }

    function test_setConfig_revert_ifNonceIsAhead(uint256 supplied) public {
        vm.assume(supplied != 1);
        // Advance the real nonce to 1 first.
        _setConfig(issuerId, keccak256("bump"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD);

        bytes[] memory sigs =
            _signSetConfig(issuerId, keccak256("k"), keccak256("v"), supplied, issuerPrivKeys, THRESHOLD);
        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, supplied));
        registry.setConfig(issuerId, keccak256("k"), keccak256("v"), supplied, sigs);
    }

    function test_setConfig_revert_ifTooFewSignatures() public {
        bytes[] memory sigs = _signSetConfig(issuerId, keccak256("k"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD - 1);

        vm.expectRevert(IConfigRegistry.BelowThreshold.selector);
        registry.setConfig(issuerId, keccak256("k"), keccak256("v"), 0, sigs);
    }

    function test_setConfig_revert_ifSignerIsNotAnIssuerKey() public {
        (uint256[] memory outsiderKeys,) = _makeSignerSet(20, THRESHOLD);
        bytes[] memory sigs = _signSetConfig(issuerId, keccak256("k"), keccak256("v"), 0, outsiderKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.setConfig(issuerId, keccak256("k"), keccak256("v"), 0, sigs);
    }

    function test_setConfig_revert_ifSignaturesUnordered() public {
        bytes[] memory sigs =
            _swapFirstTwo(_signSetConfig(issuerId, keccak256("k"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD));

        vm.expectRevert(IConfigRegistry.SignaturesNotOrdered.selector);
        registry.setConfig(issuerId, keccak256("k"), keccak256("v"), 0, sigs);
    }

    /// @dev The registry does not pre-check the key; a zero key is rejected by `Config` itself,
    ///      bubbling `EmptyConfigKey` up through `setConfig`.
    function test_setConfig_revert_ifKeyIsZero() public {
        _setConfig_revert_ifKeyIsZero_impl();
    }

    function _setConfig_revert_ifKeyIsZero_impl() internal {
        bytes[] memory sigs = _signSetConfig(issuerId, bytes32(0), keccak256("v"), 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfig.EmptyConfigKey.selector);
        registry.setConfig(issuerId, bytes32(0), keccak256("v"), 0, sigs);
    }

    function test_setConfig_revert_ifResubmittedAfterSuccess() public {
        bytes32 key = keccak256("k");
        bytes32 checksum = keccak256("v");
        bytes[] memory sigs = _signSetConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD);

        registry.setConfig(issuerId, key, checksum, 0, sigs);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.setConfig(issuerId, key, checksum, 0, sigs);
    }

    /// @dev A signature set bound to `(key, checksum)` fails when submitted with a different
    ///      `checksum`: the recovered addresses depend on the tampered digest, so the failure is
    ///      either `NotAnIssuerKey` or `SignaturesNotOrdered` depending on how the wrong-digest
    ///      recovery happens to order.
    function test_setConfig_revert_ifChecksumTampered() public {
        bytes32 key = keccak256("k");
        bytes[] memory sigs = _signSetConfig(issuerId, key, keccak256("v1"), 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfig(issuerId, key, keccak256("v2"), 0, sigs);
    }

    function test_setConfig_revert_ifKeyTampered() public {
        bytes32 checksum = keccak256("v");
        bytes[] memory sigs = _signSetConfig(issuerId, keccak256("k1"), checksum, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfig(issuerId, keccak256("k2"), checksum, 0, sigs);
    }

    function test_setConfig_revert_ifSignedForDifferentIssuer() public {
        (uint256[] memory keysB, address[] memory addrsB) = _makeSignerSet(21, SIGNER_COUNT);
        address issuerB = _registerIssuer(addrsB, THRESHOLD, keysB, THRESHOLD);

        bytes32 key = keccak256("k");
        bytes32 checksum = keccak256("v");
        // Signed for issuer B's digest, submitted against issuer A (this contract's `issuerId`).
        bytes32 digestForB = TestDigests.setConfigDigest(address(registry), issuerB, key, checksum, 0);
        bytes[] memory sigs = _signDigestWithKeys(digestForB, keysB, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.setConfig(issuerId, key, checksum, 0, sigs);
    }

    function test_setConfig_revert_ifSignedForSetConfigWithData() public {
        bytes32 key = keccak256("k");
        bytes32 checksum = keccak256("v");
        bytes32 wrongTagDigest =
            TestDigests.setConfigWithDataDigest(address(registry), issuerId, key, abi.encode(checksum), 0);
        bytes[] memory sigs = _signDigestWithKeys(wrongTagDigest, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfig(issuerId, key, checksum, 0, sigs);
    }
}

/// @title ConfigRegistryTest_setConfigWithData
/// @notice `setConfigWithData` behaviour, mirroring `ConfigRegistryTest_setConfig`'s coverage
///         plus the raw-`data` specifics.
contract ConfigRegistryTest_setConfigWithData is ConfigRegistryTestBase {
    function test_setConfigWithData_forwardsToConfig(bytes32 key, bytes memory data) public {
        vm.assume(key != bytes32(0));

        _setConfigWithData(issuerId, key, data, 0, issuerPrivKeys, THRESHOLD);

        assertEq(Config(issuerId).latestConfigChecksum(key), keccak256(data));
        assertEq(Config(issuerId).configEpoch(key), 1);
        assertEq(registry.nonce(issuerId), 1);
    }

    function test_setConfigWithData_acceptsEmptyData(bytes32 key) public {
        vm.assume(key != bytes32(0));

        _setConfigWithData(issuerId, key, "", 0, issuerPrivKeys, THRESHOLD);

        assertEq(Config(issuerId).latestConfigChecksum(key), keccak256(""));
    }

    function test_setConfigWithData_emitsConfigWithDataCommittedAndInnerConfigSetWithData(
        bytes32 key,
        bytes memory data
    ) public {
        vm.assume(key != bytes32(0));
        bytes[] memory sigs = _signSetConfigWithData(issuerId, key, data, 0, issuerPrivKeys, THRESHOLD);

        vm.expectEmit();
        emit IConfig.ConfigSetWithData(key, address(registry), keccak256(data), 1, data);
        vm.expectEmit();
        emit IConfigRegistry.ConfigWithDataCommitted(issuerId, key, keccak256(data), data, 1);
        registry.setConfigWithData(issuerId, key, data, 0, sigs);
    }

    function test_setConfigWithData_revert_ifIssuerNotRegistered(address randomIssuer, bytes32 key, bytes memory data)
        public
    {
        vm.assume(!registry.isRegistered(randomIssuer));
        bytes[] memory sigs = new bytes[](0);

        vm.expectRevert(IConfigRegistry.IssuerNotRegistered.selector);
        registry.setConfigWithData(randomIssuer, key, data, 0, sigs);
    }

    function test_setConfigWithData_revert_ifNonceIsStale(uint256 supplied, bytes memory data) public {
        vm.assume(supplied != 0);
        bytes[] memory sigs =
            _signSetConfigWithData(issuerId, keccak256("k"), data, supplied, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 0, supplied));
        registry.setConfigWithData(issuerId, keccak256("k"), data, supplied, sigs);
    }

    function test_setConfigWithData_revert_ifNonceIsAhead(uint256 supplied, bytes memory data) public {
        vm.assume(supplied != 1);
        _setConfigWithData(issuerId, keccak256("bump"), "x", 0, issuerPrivKeys, THRESHOLD);

        bytes[] memory sigs =
            _signSetConfigWithData(issuerId, keccak256("k"), data, supplied, issuerPrivKeys, THRESHOLD);
        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, supplied));
        registry.setConfigWithData(issuerId, keccak256("k"), data, supplied, sigs);
    }

    function test_setConfigWithData_revert_ifTooFewSignatures() public {
        bytes[] memory sigs = _signSetConfigWithData(issuerId, keccak256("k"), "v", 0, issuerPrivKeys, THRESHOLD - 1);

        vm.expectRevert(IConfigRegistry.BelowThreshold.selector);
        registry.setConfigWithData(issuerId, keccak256("k"), "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifSignerIsNotAnIssuerKey() public {
        (uint256[] memory outsiderKeys,) = _makeSignerSet(22, THRESHOLD);
        bytes[] memory sigs = _signSetConfigWithData(issuerId, keccak256("k"), "v", 0, outsiderKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.setConfigWithData(issuerId, keccak256("k"), "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifSignaturesUnordered() public {
        bytes[] memory sigs =
            _swapFirstTwo(_signSetConfigWithData(issuerId, keccak256("k"), "v", 0, issuerPrivKeys, THRESHOLD));

        vm.expectRevert(IConfigRegistry.SignaturesNotOrdered.selector);
        registry.setConfigWithData(issuerId, keccak256("k"), "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifKeyIsZero() public {
        bytes[] memory sigs = _signSetConfigWithData(issuerId, bytes32(0), "v", 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfig.EmptyConfigKey.selector);
        registry.setConfigWithData(issuerId, bytes32(0), "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifResubmittedAfterSuccess() public {
        bytes32 key = keccak256("k");
        bytes[] memory sigs = _signSetConfigWithData(issuerId, key, "v", 0, issuerPrivKeys, THRESHOLD);

        registry.setConfigWithData(issuerId, key, "v", 0, sigs);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.setConfigWithData(issuerId, key, "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifDataTampered() public {
        bytes32 key = keccak256("k");
        bytes[] memory sigs = _signSetConfigWithData(issuerId, key, "v1", 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfigWithData(issuerId, key, "v2", 0, sigs);
    }

    function test_setConfigWithData_revert_ifKeyTampered() public {
        bytes[] memory sigs = _signSetConfigWithData(issuerId, keccak256("k1"), "v", 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfigWithData(issuerId, keccak256("k2"), "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifSignedForDifferentIssuer() public {
        (uint256[] memory keysB, address[] memory addrsB) = _makeSignerSet(23, SIGNER_COUNT);
        address issuerB = _registerIssuer(addrsB, THRESHOLD, keysB, THRESHOLD);

        bytes32 key = keccak256("k");
        bytes32 digestForB = TestDigests.setConfigWithDataDigest(address(registry), issuerB, key, "v", 0);
        bytes[] memory sigs = _signDigestWithKeys(digestForB, keysB, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.setConfigWithData(issuerId, key, "v", 0, sigs);
    }

    function test_setConfigWithData_revert_ifSignedForSetConfig() public {
        bytes32 key = keccak256("k");
        bytes32 wrongTagDigest = TestDigests.setConfigDigest(address(registry), issuerId, key, keccak256("v"), 0);
        bytes[] memory sigs = _signDigestWithKeys(wrongTagDigest, issuerPrivKeys, THRESHOLD);

        vm.expectRevert();
        registry.setConfigWithData(issuerId, key, "v", 0, sigs);
    }
}

/// @title ConfigRegistryTest_updateIssuerSettings
/// @notice Issuer key/threshold rotation: wholesale replacement, validation on the *new* set,
///         verification against the *outgoing* set, and the A -> B -> A replay case.
contract ConfigRegistryTest_updateIssuerSettings is ConfigRegistryTestBase {
    /// @dev Issuer threshold is policy, not a registry-wide majority floor. The
    ///      current quorum may intentionally rotate to a centralized 1-of-1 policy.
    function test_updateIssuerSettings_currentQuorumCanChooseSingleKeyThresholdOne() public {
        (uint256[] memory newKeys, address[] memory newAddrs) = _makeSignerSet(29, 1);

        _updateIssuerSettings(issuerId, newAddrs, 1, 0, issuerPrivKeys, THRESHOLD);

        assertEq(registry.issuerKeys(issuerId), newAddrs);
        assertEq(registry.threshold(issuerId), 1);
        assertEq(registry.nonce(issuerId), 1);

        _setConfig(issuerId, keccak256("k"), keccak256("v"), 1, newKeys, 1);
        assertEq(Config(issuerId).latestConfigChecksum(keccak256("k")), keccak256("v"));
    }

    function test_updateIssuerSettings_replacesKeysAndThreshold() public {
        (, address[] memory newAddrs) = _makeSignerSet(30, SIGNER_COUNT);
        uint256 newThreshold = 4;

        _updateIssuerSettings(issuerId, newAddrs, newThreshold, 0, issuerPrivKeys, THRESHOLD);

        assertEq(registry.issuerKeys(issuerId), newAddrs);
        assertEq(registry.threshold(issuerId), newThreshold);
        assertEq(registry.nonce(issuerId), 1);

        for (uint256 i = 0; i < issuerSignerAddrs.length; i++) {
            assertFalse(_isIssuerKey(issuerSignerAddrs[i], newAddrs));
        }
    }

    function test_updateIssuerSettings_emitsIssuerSettingsUpdated() public {
        (, address[] memory newAddrs) = _makeSignerSet(31, SIGNER_COUNT);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        vm.expectEmit();
        emit IConfigRegistry.IssuerSettingsUpdated(issuerId, newAddrs, THRESHOLD, 1);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    function test_updateIssuerSettings_thresholdOnlyChange_consumesNonce() public {
        uint256 newThreshold = 3;
        _updateIssuerSettings(issuerId, issuerSignerAddrs, newThreshold, 0, issuerPrivKeys, THRESHOLD);

        assertEq(registry.issuerKeys(issuerId), issuerSignerAddrs);
        assertEq(registry.threshold(issuerId), newThreshold);
        assertEq(registry.nonce(issuerId), 1);
    }

    function test_updateIssuerSettings_newQuorumCanImmediatelySetConfig() public {
        (uint256[] memory newKeys, address[] memory newAddrs) = _makeSignerSet(32, SIGNER_COUNT);
        _updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        _setConfig(issuerId, keccak256("k"), keccak256("v"), 1, newKeys, THRESHOLD);
        assertEq(Config(issuerId).latestConfigChecksum(keccak256("k")), keccak256("v"));
    }

    function test_updateIssuerSettings_revert_ifIssuerNotRegistered(
        address randomIssuer,
        address[] memory newAddrs,
        uint256 newThreshold
    ) public {
        vm.assume(!registry.isRegistered(randomIssuer));
        bytes[] memory sigs = new bytes[](0);

        vm.expectRevert(IConfigRegistry.IssuerNotRegistered.selector);
        registry.updateIssuerSettings(randomIssuer, newAddrs, newThreshold, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifNonceIsStale(uint256 supplied) public {
        vm.assume(supplied != 0);
        (, address[] memory newAddrs) = _makeSignerSet(33, SIGNER_COUNT);
        bytes[] memory sigs =
            _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, supplied, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 0, supplied));
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, supplied, sigs);
    }

    function test_updateIssuerSettings_revert_ifNonceIsAhead(uint256 supplied) public {
        vm.assume(supplied != 1);
        _setConfig(issuerId, keccak256("bump"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD);

        (, address[] memory newAddrs) = _makeSignerSet(34, SIGNER_COUNT);
        bytes[] memory sigs =
            _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, supplied, issuerPrivKeys, THRESHOLD);
        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, supplied));
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, supplied, sigs);
    }

    function test_updateIssuerSettings_revert_ifNewThresholdIsZero() public {
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, issuerSignerAddrs, 0, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.InvalidThreshold.selector);
        registry.updateIssuerSettings(issuerId, issuerSignerAddrs, 0, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifNewKeyCountBelowThreshold() public {
        (, address[] memory shortAddrs) = _makeSignerSet(35, 3);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, shortAddrs, 4, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotEnoughIssuerKeys.selector);
        registry.updateIssuerSettings(issuerId, shortAddrs, 4, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifNewKeyIsZero() public {
        (, address[] memory newAddrs) = _makeSignerSet(36, SIGNER_COUNT);
        newAddrs[0] = address(0);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.ZeroIssuerKey.selector);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifNewKeysUnsorted() public {
        (, address[] memory newAddrs) = _makeSignerSet(37, SIGNER_COUNT);
        (newAddrs[0], newAddrs[1]) = (newAddrs[1], newAddrs[0]);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.IssuerKeysNotSorted.selector);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifTooFewSignatures() public {
        (, address[] memory newAddrs) = _makeSignerSet(38, SIGNER_COUNT);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD - 1);

        vm.expectRevert(IConfigRegistry.BelowThreshold.selector);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    /// @dev Verification is against the *outgoing* set: a member of the *new* set signing the
    ///      rotation is not (yet) an issuer key, so it is rejected.
    function test_updateIssuerSettings_revert_ifSignedByNewSetKey() public {
        (uint256[] memory newKeys, address[] memory newAddrs) = _makeSignerSet(39, SIGNER_COUNT);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, newKeys, THRESHOLD);

        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifSignaturesUnordered() public {
        (, address[] memory newAddrs) = _makeSignerSet(40, SIGNER_COUNT);
        bytes[] memory sigs =
            _swapFirstTwo(_signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD));

        vm.expectRevert(IConfigRegistry.SignaturesNotOrdered.selector);
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    function test_updateIssuerSettings_revert_ifResubmittedAfterSuccess() public {
        (, address[] memory newAddrs) = _makeSignerSet(41, SIGNER_COUNT);
        bytes[] memory sigs = _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, sigs);
    }

    /// @dev Rotate A -> B, then B -> A. The set is A again, so the captured A -> B signatures
    ///      recover to live keys once more — only the advanced nonce (now 2) stops the replay.
    function test_updateIssuerSettings_revert_replayAfterConfigRevisited() public {
        (uint256[] memory bKeys, address[] memory bAddrs) = _makeSignerSet(42, SIGNER_COUNT);

        bytes[] memory sigsAtoB = _signUpdateIssuerSettings(issuerId, bAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);
        registry.updateIssuerSettings(issuerId, bAddrs, THRESHOLD, 0, sigsAtoB);
        assertEq(registry.nonce(issuerId), 1);

        bytes[] memory sigsBtoA = _signUpdateIssuerSettings(issuerId, issuerSignerAddrs, THRESHOLD, 1, bKeys, THRESHOLD);
        registry.updateIssuerSettings(issuerId, issuerSignerAddrs, THRESHOLD, 1, sigsBtoA);
        assertEq(registry.nonce(issuerId), 2);

        // A is current again: under the old nonce, sigsAtoB would recover to live keys. The nonce
        // (now 2) is what makes the replay fail.
        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 2, 0));
        registry.updateIssuerSettings(issuerId, bAddrs, THRESHOLD, 0, sigsAtoB);
    }

    function _isIssuerKey(address candidate, address[] memory keys) internal pure returns (bool) {
        for (uint256 i = 0; i < keys.length; i++) {
            if (keys[i] == candidate) return true;
        }
        return false;
    }
}

/// @title ConfigRegistryTest_nonce
/// @notice The shared-nonce cross-entrypoint invalidation property: a single per-issuer `nonce`
///         is consumed by any successful call to `setConfig`, `setConfigWithData`, or
///         `updateIssuerSettings`, invalidating any other pending pre-signed request of any of
///         the three types at that nonce.
contract ConfigRegistryTest_nonce is ConfigRegistryTestBase {
    function test_nonce_landingRotation_invalidatesPendingSetConfig() public {
        bytes32 key = keccak256("k");
        bytes32 checksum = keccak256("v");
        bytes[] memory pendingSetConfigSigs = _signSetConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD);

        (uint256[] memory newKeys, address[] memory newAddrs) = _makeSignerSet(50, SIGNER_COUNT);
        _updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.setConfig(issuerId, key, checksum, 0, pendingSetConfigSigs);

        // Re-signing at the new nonce with the *old* (now-rotated-out) keys still fails.
        bytes[] memory oldKeysAtNewNonce = _signSetConfig(issuerId, key, checksum, 1, issuerPrivKeys, THRESHOLD);
        vm.expectRevert(IConfigRegistry.NotAnIssuerKey.selector);
        registry.setConfig(issuerId, key, checksum, 1, oldKeysAtNewNonce);

        // Only the new quorum can proceed.
        _setConfig(issuerId, key, checksum, 1, newKeys, THRESHOLD);
        assertEq(Config(issuerId).latestConfigChecksum(key), checksum);
    }

    function test_nonce_landingSetConfig_invalidatesPendingRotation() public {
        (, address[] memory newAddrs) = _makeSignerSet(51, SIGNER_COUNT);
        bytes[] memory pendingRotationSigs =
            _signUpdateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, issuerPrivKeys, THRESHOLD);

        _setConfig(issuerId, keccak256("k"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 0, pendingRotationSigs);

        // Re-signing at the new nonce with the same (unrotated) keys succeeds.
        _updateIssuerSettings(issuerId, newAddrs, THRESHOLD, 1, issuerPrivKeys, THRESHOLD);
        assertEq(registry.issuerKeys(issuerId), newAddrs);
    }

    /// @dev The nonce is not scoped per config key: pre-signing `setConfig` and
    ///      `setConfigWithData` for two *different* keys at the same nonce, landing either
    ///      invalidates the other.
    function test_nonce_isNotScopedPerConfigKey() public {
        bytes32 keyA = keccak256("keyA");
        bytes32 keyB = keccak256("keyB");

        bytes[] memory sigsA = _signSetConfig(issuerId, keyA, keccak256("va"), 0, issuerPrivKeys, THRESHOLD);
        bytes[] memory sigsB = _signSetConfigWithData(issuerId, keyB, "vb", 0, issuerPrivKeys, THRESHOLD);

        registry.setConfig(issuerId, keyA, keccak256("va"), 0, sigsA);

        vm.expectRevert(abi.encodeWithSelector(IConfigRegistry.UnexpectedNonce.selector, 1, 0));
        registry.setConfigWithData(issuerId, keyB, "vb", 0, sigsB);
    }

    function test_nonce_isPerIssuer() public {
        (uint256[] memory keysB, address[] memory addrsB) = _makeSignerSet(52, SIGNER_COUNT);
        address issuerB = _registerIssuer(addrsB, THRESHOLD, keysB, THRESHOLD);

        _setConfig(issuerId, keccak256("k"), keccak256("v"), 0, issuerPrivKeys, THRESHOLD);

        assertEq(registry.nonce(issuerId), 1);
        assertEq(registry.nonce(issuerB), 0);
    }

    /// @dev A failed call (too few signatures) does not consume the nonce: the original valid
    ///      call at the same nonce still succeeds afterwards.
    function test_nonce_notConsumedByFailedCall() public {
        bytes32 key = keccak256("k");
        bytes32 checksum = keccak256("v");

        bytes[] memory tooFewSigs = _signSetConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD - 1);
        vm.expectRevert(IConfigRegistry.BelowThreshold.selector);
        registry.setConfig(issuerId, key, checksum, 0, tooFewSigs);

        assertEq(registry.nonce(issuerId), 0);

        _setConfig(issuerId, key, checksum, 0, issuerPrivKeys, THRESHOLD);
        assertEq(registry.nonce(issuerId), 1);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {Test} from "forge-std/Test.sol";

import {Config} from "../src/Config.sol";
import {IConfig} from "../src/interfaces/IConfig.sol";

/// @title ConfigTest
/// @notice Tests for `Config` deployed directly with a plain EOA/`address(this)` owner (not
///         routed through `ConfigRegistry` — see `ConfigRegistryTest*` for that).
contract ConfigTest is Test {
    Config internal config;

    /// @dev The test contract itself is the owner, so calls made directly from test functions
    ///      (without `vm.prank`) are already authorized.
    address internal owner;

    function setUp() public {
        owner = address(this);
        config = new Config(owner);
    }

    // -------------------------------------------------------------------------
    // Constructor
    // -------------------------------------------------------------------------

    function test_constructor_setsOwner(address owner_) public {
        vm.assume(owner_ != address(0));
        Config c = new Config(owner_);
        assertEq(c.owner(), owner_);
    }

    function test_constructor_revert_ifOwnerIsZero() public {
        vm.expectRevert(IConfig.InvalidOwner.selector);
        new Config(address(0));
    }

    // -------------------------------------------------------------------------
    // Views — untouched key
    // -------------------------------------------------------------------------

    function test_configEpoch_zero_forUntouchedKey(bytes32 key) public view {
        assertEq(config.configEpoch(key), 0);
    }

    function test_latestConfigChecksum_zero_forUntouchedKey(bytes32 key) public view {
        assertEq(config.latestConfigChecksum(key), bytes32(0));
    }

    function test_configChecksums_empty_forUntouchedKey(bytes32 key) public view {
        assertEq(config.configChecksums(key).length, 0);
    }

    function test_configChecksumAtEpoch_revert_ifUntouchedKey(bytes32 key) public {
        // Epoch 0 is never a valid epoch, even for a written key.
        vm.expectRevert(IConfig.EpochOutOfRange.selector);
        config.configChecksumAtEpoch(key, 0);

        // The first real epoch is out of range too while the history is empty.
        vm.expectRevert(IConfig.EpochOutOfRange.selector);
        config.configChecksumAtEpoch(key, 1);
    }

    // -------------------------------------------------------------------------
    // setConfig
    // -------------------------------------------------------------------------

    function test_setConfig_appendsChecksum(bytes32 key, bytes32 checksum) public {
        vm.assume(key != bytes32(0));

        config.setConfig(key, checksum);

        assertEq(config.configEpoch(key), 1);
        assertEq(config.latestConfigChecksum(key), checksum);
        assertEq(config.configChecksumAtEpoch(key, 1), checksum);
    }

    function test_setConfig_emitsConfigSet(bytes32 key, bytes32 checksum) public {
        vm.assume(key != bytes32(0));

        vm.expectEmit();
        emit IConfig.ConfigSet(key, owner, checksum, 1);
        config.setConfig(key, checksum);
    }

    function test_setConfig_revert_ifKeyIsZero(bytes32 checksum) public {
        vm.expectRevert(IConfig.EmptyConfigKey.selector);
        config.setConfig(bytes32(0), checksum);
    }

    function test_setConfig_revert_ifCallerIsNotOwner(address caller, bytes32 key, bytes32 checksum) public {
        vm.assume(caller != owner);
        vm.prank(caller);
        vm.expectRevert(IConfig.NotConfigOwner.selector);
        config.setConfig(key, checksum);
    }

    // -------------------------------------------------------------------------
    // setConfigWithData
    // -------------------------------------------------------------------------

    function test_setConfigWithData_checksumIsHashOfData(bytes32 key, bytes memory data) public {
        vm.assume(key != bytes32(0));

        config.setConfigWithData(key, data);

        assertEq(config.latestConfigChecksum(key), keccak256(data));
    }

    function test_setConfigWithData_acceptsEmptyData(bytes32 key) public {
        vm.assume(key != bytes32(0));

        config.setConfigWithData(key, "");

        // Empty data is a valid payload, not an error case — checksum is keccak256("").
        assertEq(config.latestConfigChecksum(key), keccak256(""));
        assertEq(config.configEpoch(key), 1);
    }

    function test_setConfigWithData_emitsConfigSetWithData(bytes32 key, bytes memory data) public {
        vm.assume(key != bytes32(0));

        vm.expectEmit();
        emit IConfig.ConfigSetWithData(key, owner, keccak256(data), 1, data);
        config.setConfigWithData(key, data);
    }

    function test_setConfigWithData_revert_ifKeyIsZero(bytes memory data) public {
        vm.expectRevert(IConfig.EmptyConfigKey.selector);
        config.setConfigWithData(bytes32(0), data);
    }

    function test_setConfigWithData_revert_ifCallerIsNotOwner(address caller, bytes32 key, bytes memory data) public {
        vm.assume(caller != owner);
        vm.prank(caller);
        vm.expectRevert(IConfig.NotConfigOwner.selector);
        config.setConfigWithData(key, data);
    }

    // -------------------------------------------------------------------------
    // Append-only history
    // -------------------------------------------------------------------------

    function test_setConfig_history_appendsAcrossMultipleWrites(bytes32 key, bytes32 checksum1, bytes32 checksum2)
        public
    {
        vm.assume(key != bytes32(0));

        config.setConfig(key, checksum1);
        config.setConfig(key, checksum2);
        // A repeated checksum value must still append a new entry, not deduplicate.
        config.setConfig(key, checksum2);

        assertEq(config.configEpoch(key), 3);

        bytes32[] memory history = config.configChecksums(key);
        assertEq(history.length, 3);
        assertEq(history[0], checksum1);
        assertEq(history[1], checksum2);
        assertEq(history[2], checksum2);

        assertEq(config.configChecksumAtEpoch(key, 1), checksum1);
        assertEq(config.configChecksumAtEpoch(key, 2), checksum2);
        assertEq(config.configChecksumAtEpoch(key, 3), checksum2);

        // Nothing is overwritten: the earliest entry is still readable after later writes.
        assertEq(config.configChecksumAtEpoch(key, 1), checksum1);
        assertEq(config.latestConfigChecksum(key), checksum2);
    }

    function test_setConfig_configChecksumAtEpoch_matchesEmittedEpoch(bytes32 key, bytes32 checksum) public {
        vm.assume(key != bytes32(0));

        vm.expectEmit();
        emit IConfig.ConfigSet(key, owner, checksum, 1);
        config.setConfig(key, checksum);

        uint64 epoch = config.configEpoch(key);
        assertEq(epoch, 1);
        assertEq(config.configChecksumAtEpoch(key, epoch), checksum);
    }

    function test_setConfig_and_setConfigWithData_shareOneCounter(bytes32 key, bytes32 checksum, bytes memory data)
        public
    {
        vm.assume(key != bytes32(0));

        config.setConfig(key, checksum);
        assertEq(config.configEpoch(key), 1);

        config.setConfigWithData(key, data);
        assertEq(config.configEpoch(key), 2);

        assertEq(config.configChecksumAtEpoch(key, 1), checksum);
        assertEq(config.configChecksumAtEpoch(key, 2), keccak256(data));
    }

    function test_setConfig_keyIsolation_doesNotAffectOtherKeys(bytes32 keyA, bytes32 keyB, bytes32 checksum) public {
        vm.assume(keyA != bytes32(0) && keyB != bytes32(0));
        vm.assume(keyA != keyB);

        config.setConfig(keyA, checksum);

        assertEq(config.configEpoch(keyB), 0);
        assertEq(config.latestConfigChecksum(keyB), bytes32(0));
    }

    function test_configChecksumAtEpoch_revert_ifEpochOutOfRange(bytes32 key, uint8 writeCount, uint256 overshoot)
        public
    {
        vm.assume(key != bytes32(0));
        uint256 n = bound(writeCount, 1, 10);
        // Valid epochs are 1..n, so the first out-of-range epoch is n + 1.
        overshoot = bound(overshoot, n + 1, n + 1000);

        for (uint256 i = 0; i < n; i++) {
            config.setConfig(key, keccak256(abi.encode(i)));
        }

        vm.expectRevert(IConfig.EpochOutOfRange.selector);
        config.configChecksumAtEpoch(key, overshoot);
    }
}

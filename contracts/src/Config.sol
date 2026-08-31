// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {IConfig} from "./interfaces/IConfig.sol";

/// @title Config
/// @notice Key→checksum directory anchored on-chain so independent protocol
///         components publish authoritative checksums of their off-chain
///         configuration. A single immutable entity that deployed this
///         contract gates every write; for most keys, payload bytes live
///         off-chain and consumers verify against `latestConfigChecksum`,
///         but the registry may instead publish the full payload on-chain
///         via `setConfigWithData` for coordination use cases. The registry
///         is append-only: checksums are never overwritten, only appended.
/// @dev    `Config` can never be orphaned: the binding to its deployer
///         is an immutable address set once at construction. There is no
///         `Ownable` surface, no transfer mechanism, and therefore no
///         renounce path.
contract Config is IConfig {
    address private immutable _owner;

    mapping(bytes32 key => bytes32[]) private _configChecksums;

    modifier onlyOwner() {
        require(msg.sender == _owner, NotConfigOwner());
        _;
    }

    /// @param owner_ Permanently bound — cannot be changed after deployment.
    constructor(address owner_) {
        require(owner_ != address(0), InvalidOwner());
        _owner = owner_;
    }

    /// @inheritdoc IConfig
    function owner() external view returns (address) {
        return _owner;
    }

    /// @inheritdoc IConfig
    function configEpoch(bytes32 key) external view returns (uint64) {
        return uint64(_configChecksums[key].length);
    }

    /// @inheritdoc IConfig
    function latestConfigChecksum(bytes32 key) external view returns (bytes32) {
        uint256 len = _configChecksums[key].length;
        return len == 0 ? bytes32(0) : _configChecksums[key][len - 1];
    }

    /// @inheritdoc IConfig
    function configChecksumAtEpoch(bytes32 key, uint256 epoch) external view returns (bytes32) {
        bytes32[] storage history = _configChecksums[key];
        require(epoch != 0 && epoch <= history.length, EpochOutOfRange());
        return history[epoch - 1];
    }

    /// @inheritdoc IConfig
    function configChecksums(bytes32 key) external view returns (bytes32[] memory) {
        return _configChecksums[key];
    }

    /// @inheritdoc IConfig
    function setConfig(bytes32 key, bytes32 checksum) external onlyOwner {
        uint64 epoch = _appendChecksum(key, checksum);
        emit ConfigSet(key, msg.sender, checksum, epoch);
    }

    /// @inheritdoc IConfig
    function setConfigWithData(bytes32 key, bytes calldata data) external onlyOwner {
        bytes32 checksum = keccak256(data);
        uint64 epoch = _appendChecksum(key, checksum);
        emit ConfigSetWithData(key, msg.sender, checksum, epoch, data);
    }

    /// @dev Shared append step behind both write paths. Rejects the reserved
    ///      zero key. Returns the post-append epoch for the caller to emit.
    function _appendChecksum(bytes32 key, bytes32 checksum) private returns (uint64 epoch) {
        require(key != bytes32(0), EmptyConfigKey());
        bytes32[] storage history = _configChecksums[key];
        history.push(checksum);
        epoch = uint64(history.length);
    }
}

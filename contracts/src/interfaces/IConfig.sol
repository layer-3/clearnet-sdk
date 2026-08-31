// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @title IConfig
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
interface IConfig {
    /// @notice The registry published a new checksum for `key`. `epoch` is the
    ///         per-key write count after this publish (1 on first write).
    event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch);
    /// @notice The registry published a new checksum for `key` together with the
    ///         full config payload it hashes to. `epoch` is the per-key write
    ///         count after this publish, drawn from the same counter as
    ///         `ConfigSet` — `setConfig` and `setConfigWithData` share one
    ///         append-only history per key.
    event ConfigSetWithData(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch, bytes data);

    /// @notice The supplied owner address is the zero address.
    error InvalidOwner();
    /// @notice The zero key is reserved.
    error EmptyConfigKey();
    /// @notice Requested index is beyond the write history for the key.
    error EpochOutOfRange();
    /// @notice Caller is not the owner bound to this `Config`.
    error NotConfigOwner();

    /// @notice Number of times `key` has been written via `setConfig` or
    ///         `setConfigWithData` (== latest epoch). Zero if never written.
    function configEpoch(bytes32 key) external view returns (uint64);

    /// @notice Latest checksum for `key`, or zero if never written.
    function latestConfigChecksum(bytes32 key) external view returns (bytes32);

    /// @notice Checksum written for `key` at 1-based `epoch`. Reverts with
    ///         `EpochOutOfRange` if `epoch` is zero or beyond the history length.
    function configChecksumAtEpoch(bytes32 key, uint256 epoch) external view returns (bytes32);

    /// @notice Full ordered checksum history for `key`, oldest first.
    function configChecksums(bytes32 key) external view returns (bytes32[] memory);

    /// @notice The owner permanently bound to this `Config` at
    ///         deployment — the sole authorized caller of `setConfig` /
    ///         `setConfigWithData`.
    function owner() external view returns (address);

    /// @notice Append a new checksum for `key`. Owner-only.
    function setConfig(bytes32 key, bytes32 checksum) external;

    /// @notice Append a new checksum for `key`, computed on-chain as
    ///         `keccak256(data)`, and emit `data` in the event for off-chain
    ///         consumption. Use this instead of `setConfig` when downstream
    ///         consumers need the raw config value from chain data itself
    ///         (a "coordination" config). Owner-only. Shares the
    ///         same per-key checksum history/epoch counter as `setConfig`.
    function setConfigWithData(bytes32 key, bytes calldata data) external;
}

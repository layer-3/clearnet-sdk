// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @notice Per-node record. Layout is ABI-pinned - do not reorder.
struct NodeRecord {
    address operator;
    uint64 activatedAt;
    uint64 deactivatedAt;
    uint64 vestedAt;
    uint32 index;
    uint32 tokenId;
    uint256 operatorCollateral;
    uint256 sponsorCollateral;
    uint256[2] blsPubkeyG1;
    uint256[4] blsPubkeyG2;
}

/// @title IClearnetRegistry
/// @notice The minimal Registry surface issuers depend on — a BLS
///         validator-pubkey directory.
/// @dev    This is the protocol Registry's surface trimmed to what an issuer
///         touches: no staking economics, NFTs, slashing, bonding-curve
///         pricing, or mutation of any kind.
interface IClearnetRegistry {
    /// @dev Layout pinned as indexed (operator, nodeId, tokenId) +
    /// data (collateral, vestedAt, blsPubkeyG2[4]).
    event NodeActivated(
        address indexed operator,
        bytes32 indexed nodeId,
        uint32 indexed tokenId,
        uint256 collateral,
        uint64 vestedAt,
        uint256[4] blsPubkeyG2
    );

    /// @notice Count of currently-tracked nodes (active + unbonding). Pagination
    ///         cursor for the enumeration methods below. Clients that need just
    ///         the active set filter by `node.deactivatedAt == 0`.
    function totalNodes() external view returns (uint256);

    /// @notice Node ids, paginated. BLS cache warm-up + gateway header verifier.
    function getNodeIds(uint256 offset, uint256 limit) external view returns (bytes32[] memory);

    /// @notice NodeRecords, paginated. Fee distribution; collateral monitor.
    ///         Issuers read `blsPubkeyG2` from each record.
    function getNodes(uint256 offset, uint256 limit) external view returns (NodeRecord[] memory);

    /// @notice Single NodeRecord lookup. Collateral monitor + Slasher signer-attribution.
    function getNodeById(bytes32 nodeId) external view returns (NodeRecord memory);
}

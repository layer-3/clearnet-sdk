// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {NodeRecord, IClearnetRegistry} from "./IClearnetRegistry.sol";

/// @title IClearnetRegistryProtocol
/// @notice Core protocol surface of the Registry — the contract every conforming
///         implementation must honour. Every method here is mandated by a spec or
///         ADR, or is the Slasher's on-chain seam. Removing any of these takes a
///         clearnode, the gateway verifier, the BLS cache, fee distribution, the
///         collateral monitor, or the Slasher offline.
///
///         Sibling interfaces (composed at the contract level, not nested here):
///           IRegistryAdmin — governance (off-protocol)
///           ISlash         — slasher-only `slash` mutator + `Slashed` event
///
///         Dashboard / UI sugar (slot terms, status, supply) is composed
///         off-chain from `NODE_ID()` + the NodeID contract directly — Registry
///         only exposes what protocol callers need on-chain.
///
///         Future protocol obligations land as new sibling interfaces, not new methods on this one.
///
/// Spec anchors:
///   ADR-008 §"Registry (source of truth)"       — BLS pubkey directory
///   ADR-008 §"Cache propagation (event-driven)" — active-set introspection
///   ADR-005 §11                                  — operator lifecycle, dynamic floor
///   ADR-005 §13                                  — slasher-only `slash` mutator
///   protocol/security.md §4.3, §11               — collateral floor (activation-time), fraud enforcement boundary
interface IClearnetRegistryProtocol is IClearnetRegistry {
    // ─── Events ─────────────────────────────────────────────────────────────────
    event NodeUnlocked(address indexed operator, bytes32 indexed nodeId, uint64 availableAt);
    event NodeReleased(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral);
    event NodeFunded(
        address indexed payer, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 amount, uint256 totalCollateral
    );

    // ─── Node introspection ─────────────────────────
    /// @notice tokenId → nodeId mapping. Resolves the slot NFT to its current node.
    function getNodeId(uint32 tokenId) external view returns (bytes32);

    // ─── BLS pubkey reverse lookup ────────────────────
    /// Forward BLS keys live directly on `NodeRecord` (`blsPubkeyG1`, `blsPubkeyG2`):
    /// fetch via `getNodeById` / `getNodes`. Only the reverse-lookup mapping is
    /// surfaced as a standalone method — it answers a different question (hash →
    /// nodeId) that the NodeRecord can't answer.
    ///
    /// @notice Reverse lookup nodeId from hashed G2 pubkey.
    ///         Used by Slasher to attribute fraud signers from attestation validators.
    function getNodeByBlsG2Hash(bytes32 blsG2Hash) external view returns (bytes32);

    // ─── Dynamic activation price ───────────────
    /// @notice Number of currently-active nodes — the input that drives the dynamic
    ///         collateral floor. Equivalently the count of tracked nodes whose
    ///         `deactivatedAt == 0`, maintained as a cached aggregate because
    ///         recomputing it by scanning every NodeRecord on each activation would
    ///         cost O(n) gas on the hot path.
    /// @dev    Pairs with `floorPrice()`: the floor is a pure function of this count,
    ///         so a caller quoting a price and a caller reasoning about the curve read
    ///         both. Deliberately NOT on `IClearnetRegistry` — bonding-curve pricing is
    ///         outside the issuer read subset, whose `totalNodes` already directs
    ///         issuers that need the active set to filter on `deactivatedAt == 0`.
    function activeCount() external view returns (uint32);

    /// @notice What a new activation must pay to satisfy the dynamic floor.
    ///         Operator-facing price quote — `floorPrice` rises monotonically as the
    ///         active node count grows. Existing nodes are not required to track it.
    function floorPrice() external view returns (uint256);

    // ─── Liability accounting ───────────────────────────────────────────────────
    /// @notice Sum of all currently-locked collateral (operator + sponsor) across active
    ///         and unbonding nodes. Foundation-facing solvency input: the asset balance
    ///         held by Registry must be >= liability for every locked obligation to be
    ///         redeemable. Sponsor collateral is backed by the Registry's general grant
    ///         pool: direct ERC20 transfers create surplus headroom until consumed by
    ///         sponsored activations. Does NOT track post-activation floor rises — sponsor
    ///         collateral is locked at activation and unchanged thereafter, so this
    ///         number only grows on activate / fund and shrinks on slash / release.
    function liability() external view returns (uint256);

    // ─── Operator lifecycle ───────────────────────────────────────
    /// @notice Self-funded registration: mint a fresh NodeID directly into
    ///         Registry escrow and activate it in one transaction.
    function register(uint256[2] calldata blsPubkeyG1, uint256[4] calldata blsPubkeyG2, uint256 operatorCollateral)
        external
        returns (uint32 tokenId, bytes32 nodeId);
    /// @notice Lock a held slot NFT into the registry with BLS keys and collateral.
    function activate(
        uint32 tokenId,
        uint256[2] calldata blsPubkeyG1,
        uint256[4] calldata blsPubkeyG2,
        uint256 operatorCollateral
    ) external returns (bytes32 nodeId);
    /// @notice Fund an active slot with additional collateral. Anyone can pay;
    ///         the amount accrues to the slot's operator collateral.
    function fund(uint32 tokenId, uint256 amount) external;
    /// @notice Begin decommission and start the unlock cooldown.
    function unlock(uint32 tokenId) external;
    /// @notice Finalize decommission after cooldown; returns NFT and remaining funds.
    function release(uint32 tokenId) external;

    // ─── Protocol constants ─────────────────────────────────────────────────────
    /// @notice Activation warm-up before a node's signatures are accepted.
    function WARMUP_WINDOW() external view returns (uint64);
    /// @notice Total cooldown after `unlock` before `release` becomes available.
    ///         Spans the signing wind-down and the post-unlock fraud-evidence window.
    function UNBONDING_PERIOD() external view returns (uint64);
    /// @notice Collateral floor at zero active nodes.
    function BASE_PRICE() external view returns (uint256);
    /// @notice Collateral floor at MAX_NODES.
    function TARGET_PRICE() external view returns (uint256);
    /// @notice Address of the NodeID ERC-721 contract that holds slot NFTs.
    function NODE_ID() external view returns (address);
}

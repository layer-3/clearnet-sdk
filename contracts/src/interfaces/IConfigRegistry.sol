// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

/// @title IConfigRegistry
/// @notice Ownerless, permissionless registry of on-chain `Config` issuers.
///         Any quorum of issuer keys can self-register (`registerIssuer`),
///         deploying and owning its own `Config` instance — the deployed
///         `Config`'s address becomes the issuer's `issuerId`. Once
///         registered, the same quorum commits config checksums
///         (`setConfig` / `setConfigWithData`) and rotates its own key set
///         (`updateIssuerSettings`).
interface IConfigRegistry {
    // -------------------------------------------------------------------------
    // Events
    // -------------------------------------------------------------------------

    /// @notice A new issuer self-registered and deployed its `Config`.
    event IssuerRegistered(address indexed issuerId, address[] issuerKeys, uint256 threshold);

    /// @notice A config commit was forwarded to `issuerId`'s `Config`.
    event ConfigCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, uint256 newNonce);

    /// @notice A config with data commit was forwarded to `issuerId`'s `Config`.
    event ConfigWithDataCommitted(
        address indexed issuerId, bytes32 indexed key, bytes32 checksum, bytes data, uint256 newNonce
    );

    /// @notice `issuerId`'s issuer key set was rotated and / or threshold updated.
    event IssuerSettingsUpdated(
        address indexed issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 newNonce
    );

    // -------------------------------------------------------------------------
    // Errors
    // -------------------------------------------------------------------------

    /// @notice Threshold is zero.
    error InvalidThreshold();
    /// @notice Issuer key count is below the threshold.
    error NotEnoughIssuerKeys();
    /// @notice A proposed issuer key is the zero address.
    error ZeroIssuerKey();
    /// @notice Proposed issuer keys are not strictly ascending (also rejects duplicates).
    error IssuerKeysNotSorted();
    /// @notice The supplied nonce does not match the issuer's current shared `nonce`.
    error UnexpectedNonce(uint256 current, uint256 supplied);
    /// @notice Fewer signatures than the threshold were supplied.
    error BelowThreshold();
    /// @notice A recovered signer is not a current issuer key for the issuer.
    error NotAnIssuerKey();
    /// @notice Signatures are not ordered by recovered address (also rejects duplicate counting).
    error SignaturesNotOrdered();
    /// @notice `issuerId` has no registered issuer-key set on this registry.
    error IssuerNotRegistered();
    /// @notice The exact `(issuerKeys, threshold)` tuple is already registered.
    error IssuerAlreadyRegistered();

    // -------------------------------------------------------------------------
    // Read Functions
    // -------------------------------------------------------------------------

    /// @notice The current ordered issuer key list for `issuerId`, ascending by address.
    function issuerKeys(address issuerId) external view returns (address[] memory);

    /// @notice The current signature threshold for `issuerId`.
    function threshold(address issuerId) external view returns (uint256);

    /// @notice The replay-guard nonce for `issuerId`, incremented on every
    ///         successful call to `setConfig`, `setConfigWithData`, or
    ///         `updateIssuerSettings`. A single counter shared across every key
    ///         and across all three entrypoints — not scoped per key or per
    ///         entrypoint — so a successful call to any one of them
    ///         invalidates any other pending pre-signed request of any of
    ///         the three types at that nonce.
    function nonce(address issuerId) external view returns (uint256);

    /// @notice Whether `issuerId` is registered on this registry.
    function isRegistered(address issuerId) external view returns (bool);

    /// @notice All settings for `issuerId` in one call.
    function issuerSettings(address issuerId)
        external
        view
        returns (address[] memory issuerKeys_, uint256 threshold_, uint256 nonce_);

    /// @notice Predicts the `CREATE2` address `registerIssuer` would deploy
    ///         to for the exact `(issuerKeys, threshold)` tuple, without
    ///         requiring the issuer to already be registered.
    /// @param issuerKeys  Issuer key addresses (sorted ascending, no duplicates/zeros).
    /// @param threshold   Number of signatures required (e.g. 5 for 5-of-7).
    /// @return issuerId   The `Config` address `registerIssuer` would deploy for this tuple.
    function computeIssuerId(address[] calldata issuerKeys, uint256 threshold) external view returns (address issuerId);

    // -------------------------------------------------------------------------
    // Registration
    // -------------------------------------------------------------------------

    /// @notice Self-register a new issuer: deploys a fresh `Config` owned by
    ///         this registry and stores `issuerKeys`/`threshold` under the
    ///         deployed address (the new `issuerId`).
    /// @dev    The `Config` is deployed via `CREATE2` with
    ///         `salt = keccak256(abi.encode(issuerKeys, threshold))`, so
    ///         registration is idempotent per exact `(issuerKeys, threshold)`
    ///         tuple: the same tuple always predicts to the same `issuerId`
    ///         (see `computeIssuerId`), and registering it twice reverts with
    ///         `IssuerAlreadyRegistered`.
    /// @param issuerKeys  Issuer key addresses (sorted ascending, no duplicates/zeros).
    /// @param threshold     Number of signatures required (e.g. 5 for 5-of-7).
    /// @param signatures    Issuer signatures over the registration digest,
    ///                      ordered by recovered address ascending, proving
    ///                      control of `issuerKeys`.
    /// @return issuerId     The deployed `Config`'s address.
    function registerIssuer(address[] calldata issuerKeys, uint256 threshold, bytes[] calldata signatures)
        external
        returns (address issuerId);

    // -------------------------------------------------------------------------
    // Config commit (issuer quorum)
    // -------------------------------------------------------------------------

    /// @notice Commit a checksum for `key` on `issuerId`'s `Config` under
    ///         issuer quorum. Verifies `threshold` issuer signatures over
    ///         the domain-separated digest and checks this registry's own
    ///         per-issuer `nonce` equals `expectedNonce` before forwarding to
    ///         `Config.setConfig`.
    /// @dev    `expectedNonce` is the only replay guard, which is owned and
    ///         tracked by this registry itself.
    /// @param issuerId       The issuer whose `Config` is being written.
    /// @param key            `Config` key being written.
    /// @param checksum       Content-addressed checksum of the off-chain payload.
    /// @param expectedNonce  This registry's per-issuer `nonce` the signatures were authorised against.
    /// @param signatures     Issuer signatures, ordered by recovered address ascending.
    function setConfig(
        address issuerId,
        bytes32 key,
        bytes32 checksum,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external;

    /// @notice Commit a config payload for `key` on `issuerId`'s `Config`
    ///         under issuer quorum, publishing the raw `data` on-chain. Use
    ///         this instead of `setConfig` when downstream consumers need the
    ///         raw config value from chain data itself rather than an anchor
    ///         to verify an independently-obtained payload against.
    /// @dev    `expectedNonce` is the only replay guard, which is owned and
    ///         tracked by this registry itself.
    /// @param issuerId       The issuer whose `Config` is being written.
    /// @param key            `Config` key being written.
    /// @param data           Raw config payload; the issuer's `Config` hashes
    ///                       it to the committed checksum.
    /// @param expectedNonce  This registry's per-issuer `nonce` the signatures were authorised against.
    /// @param signatures     Issuer signatures, ordered by recovered address ascending.
    function setConfigWithData(
        address issuerId,
        bytes32 key,
        bytes calldata data,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external;

    // -------------------------------------------------------------------------
    // Issuer settings change
    // -------------------------------------------------------------------------

    /// @notice Rotate `issuerId`'s issuer key set under its current quorum.
    ///         Mirrors `Custody.updateSigners`: signatures are bound to the
    ///         same shared per-issuer `nonce` as `setConfig`/
    ///         `setConfigWithData`, which is consumed on success so a
    ///         signature set authorises exactly one rotation — and a config
    ///         commit landing first invalidates a pending pre-signed
    ///         rotation at that nonce, or vice versa.
    /// @dev    `expectedNonce` is the only replay guard, which is owned and
    ///         tracked by this registry itself.
    /// @param issuerId        The issuer whose issuer key set is rotating.
    /// @param newIssuerKeys New issuer key addresses (sorted ascending, no duplicates/zeros).
    /// @param newThreshold    New signature threshold.
    /// @param expectedNonce   This registry's per-issuer `nonce` the signatures were authorised against.
    /// @param signatures      Current issuer-key signatures, ordered by recovered address ascending.
    function updateIssuerSettings(
        address issuerId,
        address[] calldata newIssuerKeys,
        uint256 newThreshold,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external;
}

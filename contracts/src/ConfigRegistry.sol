// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;

import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {Create2} from "@openzeppelin/contracts/utils/Create2.sol";

import {IConfigRegistry} from "./interfaces/IConfigRegistry.sol";
import {IConfig} from "./interfaces/IConfig.sol";
import {Config} from "./Config.sol";
import {ConfigRegistryDigests} from "./libraries/ConfigRegistryDigests.sol";

/// @title ConfigRegistry
/// @notice Ownerless, permissionless registry of on-chain `Config` issuers
///         (ADR-017). Any quorum of issuer keys self-registers by calling
///         `registerIssuer`, which deploys a fresh `Config` owned by this
///         registry — the deployed `Config`'s address becomes the issuer's
///         `issuerId`. The same quorum then forwards config commits to
///         `Config.setConfig` (checksum-only) and `Config.setConfigWithData`
///         (raw payload published on-chain) after verifying a threshold of
///         signatures over a domain-separated digest bound to a
///         registry-owned, per-issuer `nonce` — a single counter shared across
///         every key and across all three entrypoints (`setConfig`,
///         `setConfigWithData`, `updateIssuerSettings`) for that issuer, consumed
///         on every successful call to any of them. It also implements
///         rotating its own issuer key set via the same threshold idiom and
///         the same shared `nonce`. Key-set size and threshold are issuer-owned
///         governance policy: this generic registry intentionally permits any
///         structurally valid policy, including a single key at threshold one.
contract ConfigRegistry is IConfigRegistry {
    using ECDSA for bytes32;

    // -------------------------------------------------------------------------
    // State
    // -------------------------------------------------------------------------

    struct IssuerSettings {
        address[] issuerKeys;
        uint256 threshold;
        uint256 nonce;
    }

    /// @dev `issuerId` is the address of the issuer's deployed `Config` instance.
    mapping(address issuerId => IssuerSettings settings) private _issuers;

    // -------------------------------------------------------------------------
    // Read Functions
    // -------------------------------------------------------------------------

    /// @inheritdoc IConfigRegistry
    function issuerKeys(address issuerId) external view returns (address[] memory) {
        return _issuers[issuerId].issuerKeys;
    }

    /// @inheritdoc IConfigRegistry
    function threshold(address issuerId) external view returns (uint256) {
        return _issuers[issuerId].threshold;
    }

    /// @inheritdoc IConfigRegistry
    function nonce(address issuerId) external view returns (uint256) {
        return _issuers[issuerId].nonce;
    }

    /// @inheritdoc IConfigRegistry
    function isRegistered(address issuerId) public view returns (bool) {
        return _issuers[issuerId].threshold > 0;
    }

    /// @inheritdoc IConfigRegistry
    function issuerSettings(address issuerId)
        external
        view
        returns (address[] memory issuerKeys_, uint256 threshold_, uint256 nonce_)
    {
        IssuerSettings storage settings = _issuers[issuerId];
        return (settings.issuerKeys, settings.threshold, settings.nonce);
    }

    /// @inheritdoc IConfigRegistry
    function computeIssuerId(address[] calldata issuerKeys_, uint256 threshold_) public view returns (address) {
        bytes32 salt = keccak256(abi.encode(issuerKeys_, threshold_));
        bytes32 bytecodeHash = keccak256(abi.encodePacked(type(Config).creationCode, abi.encode(address(this))));
        return Create2.computeAddress(salt, bytecodeHash);
    }

    // -------------------------------------------------------------------------
    // Registration
    // -------------------------------------------------------------------------

    /// @inheritdoc IConfigRegistry
    /// @dev Self-referential: the signature set is over the claimed key set
    ///      itself, proving control of it. No allowlist, role gate, fee, or
    ///      bond — anti-abuse is off-chain. `CREATE2` with
    ///      `salt = keccak256(abi.encode(issuerKeys_, threshold_))` makes
    ///      registration idempotent per exact tuple: a quorum that wants a
    ///      second issuer must change at least one key or the threshold.
    function registerIssuer(address[] calldata issuerKeys_, uint256 threshold_, bytes[] calldata signatures)
        external
        returns (address issuerId)
    {
        _validateIssuerKeys(issuerKeys_, threshold_);

        issuerId = computeIssuerId(issuerKeys_, threshold_);
        require(!isRegistered(issuerId), IssuerAlreadyRegistered());

        bytes32 digest = ConfigRegistryDigests.registrationDigest(address(this), issuerKeys_, threshold_);
        _verifyQuorum(digest, signatures, issuerKeys_, threshold_);

        bytes32 salt = keccak256(abi.encode(issuerKeys_, threshold_));
        new Config{salt: salt}(address(this));

        IssuerSettings storage settings = _issuers[issuerId];
        settings.issuerKeys = issuerKeys_;
        settings.threshold = threshold_;
        // nonce starts at 0 (default).

        emit IssuerRegistered(issuerId, issuerKeys_, threshold_);
    }

    // -------------------------------------------------------------------------
    // Config commit (issuer quorum)
    // -------------------------------------------------------------------------

    /// @inheritdoc IConfigRegistry
    /// @dev `expectedNonce` is the only replay guard, and it is owned and tracked by
    ///      this registry. It is a single nonce per issuer, shared across every `key`
    ///      and across `setConfig`, `setConfigWithData`, and `updateIssuerSettings`: only
    ///      one call to any of them can be "in flight" per issuer at a time. Pre-signing
    ///      calls to two of these (even of different kinds, e.g. a config commit and a key
    ///      rotation) at the same nonce means whichever lands first invalidates the
    ///      other — the second must be re-signed against the new nonce. The nonce is
    ///      consumed on success (before the external call to `Config`), so the same
    ///      signature set can never be replayed; domain separation (`chainid` +
    ///      `address(this)` + `issuerId` + the entrypoint's own tag string) prevents
    ///      reuse across chains, sibling registries, sibling issuers, or entrypoints.
    function setConfig(
        address issuerId,
        bytes32 key,
        bytes32 checksum,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external {
        require(isRegistered(issuerId), IssuerNotRegistered());

        IssuerSettings storage settings = _issuers[issuerId];
        require(expectedNonce == settings.nonce, UnexpectedNonce(settings.nonce, expectedNonce));

        bytes32 digest = ConfigRegistryDigests.setConfigDigest(address(this), issuerId, key, checksum, expectedNonce);
        _verifyQuorumStored(issuerId, digest, signatures);

        unchecked {
            ++settings.nonce;
        }

        IConfig(issuerId).setConfig(key, checksum);

        emit ConfigCommitted(issuerId, key, checksum, settings.nonce);
    }

    /// @inheritdoc IConfigRegistry
    /// @dev Same replay discipline as `setConfig` — see its dev note.
    function setConfigWithData(
        address issuerId,
        bytes32 key,
        bytes calldata data,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external {
        require(isRegistered(issuerId), IssuerNotRegistered());

        IssuerSettings storage settings = _issuers[issuerId];
        require(expectedNonce == settings.nonce, UnexpectedNonce(settings.nonce, expectedNonce));

        bytes32 digest =
            ConfigRegistryDigests.setConfigWithDataDigest(address(this), issuerId, key, data, expectedNonce);
        _verifyQuorumStored(issuerId, digest, signatures);

        unchecked {
            ++settings.nonce;
        }

        IConfig(issuerId).setConfigWithData(key, data);

        emit ConfigWithDataCommitted(issuerId, key, keccak256(data), data, settings.nonce);
    }

    // -------------------------------------------------------------------------
    // Issuer settings change
    // -------------------------------------------------------------------------

    /// @inheritdoc IConfigRegistry
    /// @dev Mirrors `Custody.updateSigners`: the nonce is bound into the digest and
    ///      consumed on success, so a captured signature set authorises exactly one
    ///      rotation even if the issuer set is later revisited (A -> B -> A). Shares
    ///      the same per-issuer `nonce` as `setConfig`/`setConfigWithData` — see
    ///      `setConfig`'s dev note for the cross-entrypoint invalidation consequence.
    ///      The outgoing quorum may select any structurally valid new policy,
    ///      including fewer keys, a non-majority threshold, or one key at threshold
    ///      one. This is intentional: the registry authenticates each issuer's
    ///      chosen policy; it does not prescribe a decentralization policy.
    function updateIssuerSettings(
        address issuerId,
        address[] calldata newIssuerKeys,
        uint256 newThreshold,
        uint256 expectedNonce,
        bytes[] calldata signatures
    ) external {
        require(isRegistered(issuerId), IssuerNotRegistered());

        IssuerSettings storage settings = _issuers[issuerId];
        require(expectedNonce == settings.nonce, UnexpectedNonce(settings.nonce, expectedNonce));

        _validateIssuerKeys(newIssuerKeys, newThreshold);

        bytes32 digest = ConfigRegistryDigests.updateIssuerSettingsDigest(
            address(this), issuerId, newIssuerKeys, newThreshold, expectedNonce
        );
        _verifyQuorumStored(issuerId, digest, signatures);

        unchecked {
            ++settings.nonce;
        }

        settings.issuerKeys = newIssuerKeys;
        settings.threshold = newThreshold;

        emit IssuerSettingsUpdated(issuerId, newIssuerKeys, newThreshold, settings.nonce);
    }

    // -------------------------------------------------------------------------
    // Internal
    // -------------------------------------------------------------------------

    /// @dev Validate an issuer key set. Strictly ascending order also guarantees
    ///      uniqueness and rejects zeros after the first slot.
    ///      Threshold is issuer-owned governance policy, not a registry-wide safety
    ///      floor. Registration and settings updates intentionally accept every
    ///      structurally valid tuple (`0 < threshold <= keys.length`), including a
    ///      centralized issuer with one key and threshold one. A current issuer
    ///      quorum already has full authority over that issuer's Config, so forcing
    ///      a majority ratio would restrict legitimate issuer policy without reducing
    ///      a compromised quorum's authority.
    function _validateIssuerKeys(address[] memory keys, uint256 threshold_) internal pure {
        require(threshold_ > 0, InvalidThreshold());
        require(keys.length >= threshold_, NotEnoughIssuerKeys());

        for (uint256 i = 0; i < keys.length; i++) {
            require(keys[i] != address(0), ZeroIssuerKey());
            if (i > 0) {
                require(keys[i] > keys[i - 1], IssuerKeysNotSorted());
            }
        }
    }

    /// @dev Verify at least `threshold_` valid signatures over `digest` against an
    ///      in-memory ascending issuer key set. Signatures must be ordered by
    ///      recovered address ascending to prevent duplicate counting; every
    ///      recovered address must be a member of `keys`, checked via a two-pointer
    ///      merge over the two ascending lists (recovered signers, `keys`) — O(n+m).
    function _verifyQuorum(bytes32 digest, bytes[] calldata signatures, address[] memory keys, uint256 threshold_)
        internal
        pure
    {
        require(signatures.length >= threshold_, BelowThreshold());

        uint256 valid = 0;
        uint256 keyIndex = 0;
        address lastSigner = address(0);

        for (uint256 i = 0; i < signatures.length; i++) {
            address recovered = digest.recover(signatures[i]);
            require(recovered > lastSigner, SignaturesNotOrdered());
            lastSigner = recovered;

            while (keyIndex < keys.length && keys[keyIndex] < recovered) {
                unchecked {
                    ++keyIndex;
                }
            }
            require(keyIndex < keys.length && keys[keyIndex] == recovered, NotAnIssuerKey());
            unchecked {
                ++keyIndex;
            }

            valid++;
        }

        require(valid >= threshold_, BelowThreshold());
    }

    /// @dev Loads `_issuers[issuerId]` into memory and delegates to `_verifyQuorum`.
    function _verifyQuorumStored(address issuerId, bytes32 digest, bytes[] calldata signatures) internal view {
        IssuerSettings storage settings = _issuers[issuerId];
        _verifyQuorum(digest, signatures, settings.issuerKeys, settings.threshold);
    }
}

package core

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// URI scheme and default network for Yellow Network resources (ADR-007 §4).
const (
	URIScheme      = "yellow"
	DefaultNetwork = "ynet"
)

// AccountType identifies the entity kind (ADR-007 §3).
type AccountType uint8

const (
	AccountTypeUnknown  AccountType = 0x00 // Sentinel: URI kind not recognised (D-003, 2026-04-21).
	AccountTypeUser     AccountType = 0x01 // Externally owned
	AccountTypeNative   AccountType = 0x02 // On-cluster deterministic handlers (pools, vaults)
	AccountTypeVirtual  AccountType = 0x03 // Off-chain operator, NFT-identified
	AccountTypeContract AccountType = 0x04 // User-deployed bytecode (future)
)

// String returns the human-readable name of the AccountType.
func (t AccountType) String() string {
	switch t {
	case AccountTypeUnknown:
		return "unknown"
	case AccountTypeUser:
		return "user"
	case AccountTypeNative:
		return "native"
	case AccountTypeVirtual:
		return "virtual"
	case AccountTypeContract:
		return "contract"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

// ---------------------------------------------------------------------------
// URI constructors — build canonical yellow:// URIs
//
// ADR-007 §4: "All URI components are lowercase." Every constructor below
// normalises its inputs via strings.ToLower so the same logical address
// produces a byte-identical URI — and therefore a byte-identical
// AccountID — regardless of the case the caller supplied (EIP-55
// checksummed, user wallet, etc.). D-002 (2026-04-21): resolution lives
// inside the constructor, not at the caller.
// ---------------------------------------------------------------------------

// UserURI returns the canonical URI for a user account.
//
//	yellow://ynet/user/0xd8da6bf26964af9d7eed9e03e53415d37aa96045
func UserURI(address string) string {
	return URIScheme + "://" + DefaultNetwork + "/user/" + strings.ToLower(address)
}

// PoolURI returns the canonical URI for a liquidity pool (ADR-007 §4).
// All pools are YELLOW-quoted (ADR-006), so the URI contains only the
// lowercase base asset. Example: yellow://ynet/pool/eth
func PoolURI(base string) string {
	return URIScheme + "://" + DefaultNetwork + "/pool/" + strings.ToLower(base)
}

// TreasuryURI returns the canonical URI for a protocol treasury (ADR-010 §1).
// The name is drawn from the compile-time ValidTreasuries allowlist. Initial
// set: `emission`. Example: yellow://ynet/treasury/emission.
func TreasuryURI(name string) string {
	return URIScheme + "://" + DefaultNetwork + "/treasury/" + strings.ToLower(name)
}

// NodeURI returns the canonical URI for a validator node account.
//
//	yellow://ynet/node/<64-char lowercase hex nodeid>
func NodeURI(id NodeID) string {
	return URIScheme + "://" + DefaultNetwork + "/node/" + hex.EncodeToString(id[:])
}

// URIToAddress maps a canonical yellow:// URI into the 20-byte address slot
// used by EIP-712 schemas that cannot carry a string recipient. The mapping is
// the first 20 bytes of keccak256(uri), paralleling the pool-anchor preimage
// rule while fitting the Transfer(address to, ...) type.
func URIToAddress(uri string) (common.Address, error) {
	if err := ValidateURI(uri); err != nil {
		return common.Address{}, err
	}
	hash := crypto.Keccak256([]byte(uri))
	return common.BytesToAddress(hash[:20]), nil
}

// ServiceURI returns the canonical URI for a named service.
//
//	yellow://ynet/<kind>/<path>
func ServiceURI(kind, path string) string {
	return URIScheme + "://" + DefaultNetwork + "/" + strings.ToLower(kind) + "/" + strings.ToLower(path)
}

// ---------------------------------------------------------------------------
// URI parser
// ---------------------------------------------------------------------------

// ParseURI splits a canonical URI into its components.
//
//	"yellow://ynet/pool/eth" → ("ynet", "pool", "eth", nil)
func ParseURI(uri string) (network, kind, path string, err error) {
	prefix := URIScheme + "://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", "", fmt.Errorf("invalid URI scheme: %q (expected %s://)", uri, URIScheme)
	}
	rest := uri[len(prefix):]

	// Split: network/kind/path...
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid URI: %q (expected yellow://network/kind/path)", uri)
	}

	network = parts[0]
	kind = parts[1]
	if len(parts) > 2 {
		path = parts[2]
	}

	if network == "" || kind == "" {
		return "", "", "", fmt.Errorf("invalid URI: %q (empty network or kind)", uri)
	}

	return network, kind, path, nil
}

// KindToAccountType maps a URI kind segment to its AccountType (ADR-007 §3).
// Returns an error on unknown kinds — this was previously a silent
// User-default which masked malformed URIs (D-003 resolution,
// 2026-04-21).
//
// The returned AccountType on error is AccountTypeUnknown (0), so code
// that ignores the error still surfaces an obviously-bogus type rather
// than a plausible User account.
func KindToAccountType(kind string) (AccountType, error) {
	switch kind {
	case "user":
		return AccountTypeUser, nil
	case "pool", "treasury", "node":
		return AccountTypeNative, nil
	case "dax", "game", "relay":
		return AccountTypeVirtual, nil
	case "contract":
		return AccountTypeContract, nil
	default:
		return AccountTypeUnknown, fmt.Errorf("unknown URI kind %q (ADR-007 §3 as amended by ADR-010 §8 enumerates: user, pool, treasury, node, dax, game, relay, contract)", kind)
	}
}

// ValidateURI enforces ADR-007 §4 URI conventions:
//
//  1. The URI uses the yellow:// scheme.
//  2. Parsing yields a non-empty network and kind.
//  3. The network matches DefaultNetwork ("ynet") — multi-network
//     support is a future extension; every URI in-flight today lives
//     on ynet.
//  4. The kind is one of the enumerated ADR-007 §3 values.
//  5. Every URI component is lowercase (ADR-007 §4: "All URI
//     components are lowercase").
//
// Returns a descriptive error on the first violation; nil on success.
// Callers at system boundaries (gateway request parsing, store loads,
// manifest reads) should call ValidateURI on any URI they receive
// from an external source before trusting it. Internal constructors
// (UserURI, PoolURI, etc.) produce canonical URIs by construction
// and do not need re-validation at their call sites.
func ValidateURI(uri string) error {
	if !IsURI(uri) {
		return fmt.Errorf("URI missing %s:// scheme: %q", URIScheme, uri)
	}
	if uri != strings.ToLower(uri) {
		return fmt.Errorf("URI contains uppercase characters (ADR-007 §4 requires lowercase): %q", uri)
	}
	network, kind, path, err := ParseURI(uri)
	if err != nil {
		return fmt.Errorf("URI parse: %w", err)
	}
	if network != DefaultNetwork {
		return fmt.Errorf("URI network %q unsupported (expected %q): %q", network, DefaultNetwork, uri)
	}
	if _, err := KindToAccountType(kind); err != nil {
		return fmt.Errorf("URI kind check: %w", err)
	}
	// ADR-010 §1: treasury names are drawn from a compile-time allowlist.
	// A URI parsing as kind=treasury but with a name outside ValidTreasuries
	// MUST be rejected by every account-materialization path.
	if kind == "treasury" {
		if !IsValidTreasury(path) {
			return fmt.Errorf("URI treasury name %q not in compile-time allowlist (ADR-010 §1): %q", path, uri)
		}
	}
	if kind == "node" && !isCanonicalNodeIDPath(path) {
		return fmt.Errorf("URI node id must be exactly 64 lowercase hex characters without 0x prefix: %q", uri)
	}
	return nil
}

func isCanonicalNodeIDPath(path string) bool {
	if len(path) != 64 {
		return false
	}
	for _, c := range path {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// IsURI returns true if the string looks like a yellow:// URI.
func IsURI(s string) bool {
	return strings.HasPrefix(s, URIScheme+"://")
}

// URIKind extracts the kind segment from a URI or address.
// For non-URI inputs (raw hex addresses), returns "user".
// Returns "" if parsing fails on an actual URI.
func URIKind(uriOrAddress string) string {
	if !IsURI(uriOrAddress) {
		return "user"
	}
	_, kind, _, err := ParseURI(uriOrAddress)
	if err != nil {
		return ""
	}
	return kind
}

// ComputeAccountID derives the canonical 32-byte AccountID from an account
// URI or raw address (ADR-007 §4). The input is normalized to a canonical
// lowercase before hashing, so a caller that accidentally hands in a
// non-canonical form still produces the canonical AccountID. That
// covers two footguns at once:
//
//  1. Non-URI heterogeneous protocol fields such as BlockEntry.Account
//     (which may carry either a canonical URI for service/pool accounts
//     or a raw user address for user-owned accounts). Non-URI input is
//     auto-wrapped in UserURI() first.
//  2. Mixed-case URI input (e.g. a URI produced by an external tool
//     that didn't enforce the rule). Post-lowercasing eliminates the
//     divergence before the hash is taken.
func ComputeAccountID(uri string) [32]byte {
	if !IsURI(uri) {
		uri = UserURI(uri)
	}
	return crypto.Keccak256Hash([]byte(strings.ToLower(uri)))
}

// ValidTreasuries is the compile-time allowlist of protocol-treasury names
// (ADR-010 §1). Treasuries are consensus-critical state-machine inputs, not
// operator configuration: declaring them in the network manifest would diverge
// SMT roots across operators the first time distribution fires. The set is
// therefore hardcoded here. Adding a new treasury requires a follow-up ADR,
// an entry in this list, a registered minimum balance (ADR-010 §6), and an
// authorized ledger helper for its outbound flow (ADR-010 §3).
var ValidTreasuries = []string{"emission"}

// IsValidTreasury reports whether a treasury name is in the compile-time
// allowlist. A URI that parses as kind=`treasury` but whose name is absent
// from this list MUST be rejected by every account-materialization path
// (ADR-010 §1).
func IsValidTreasury(name string) bool {
	for _, t := range ValidTreasuries {
		if t == name {
			return true
		}
	}
	return false
}

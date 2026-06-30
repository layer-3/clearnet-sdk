package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ChecksumSource yields the latest confirmed content checksum for a registry
// key. The EVM ConfigWatcher satisfies it via LatestChecksum; the interface
// keeps this package decoupled from the blockchain bindings.
type ChecksumSource interface {
	LatestChecksum(key [32]byte) (checksum [32]byte, ok bool)
}

// PayloadStore resolves a content checksum to the bytes whose keccak256 equals
// it. For custody this is the local seeded config_versions store; for an
// external verifier it is wherever the published payload is mirrored. The
// RegistrySignerSource re-verifies the checksum, so a wrong or tampered store
// cannot inject an unauthorised signer set.
type PayloadStore interface {
	Payload(ctx context.Context, checksum [32]byte) ([]byte, error)
}

// RegistrySignerSource is the on-chain-anchored SignerSource (ADR-017): the
// authoritative signer set is whatever payload hashes to the checksum the
// registry has committed under KEY_SIGNERS. It composes a ChecksumSource (the
// confirmed on-chain checksum) with a PayloadStore (the content-addressed
// bytes), verifies keccak256(payload) == checksum, and decodes (signers,
// threshold). It implements SignerSource, so swapping it in for
// StaticSignerSource needs no verifier-side change.
type RegistrySignerSource struct {
	key       [32]byte
	checksums ChecksumSource
	store     PayloadStore
}

var _ SignerSource = (*RegistrySignerSource)(nil)

// NewRegistrySignerSource binds a source to the KEY_SIGNERS registry key.
func NewRegistrySignerSource(key [32]byte, checksums ChecksumSource, store PayloadStore) (*RegistrySignerSource, error) {
	if checksums == nil {
		return nil, fmt.Errorf("registry signer source: nil checksum source")
	}
	if store == nil {
		return nil, fmt.Errorf("registry signer source: nil payload store")
	}
	return &RegistrySignerSource{key: key, checksums: checksums, store: store}, nil
}

// Load resolves the current KEY_SIGNERS checksum, fetches and verifies the
// matching payload, and returns the decoded signer set and threshold. It errors
// (rather than returning a stale set) if no checksum is confirmed yet or the
// payload is missing — the verifier keeps its last good set until a refresh
// succeeds.
func (s *RegistrySignerSource) Load(ctx context.Context) ([]common.Address, int, error) {
	checksum, ok := s.checksums.LatestChecksum(s.key)
	if !ok {
		return nil, 0, fmt.Errorf("registry signer source: no confirmed checksum for key")
	}
	payload, err := s.store.Payload(ctx, checksum)
	if err != nil {
		return nil, 0, fmt.Errorf("registry signer source: load payload: %w", err)
	}
	var got [32]byte
	copy(got[:], crypto.Keccak256(payload))
	if got != checksum {
		return nil, 0, fmt.Errorf("registry signer source: payload checksum %s != on-chain %s",
			"0x"+common.Bytes2Hex(got[:]), "0x"+common.Bytes2Hex(checksum[:]))
	}
	return ParseSignerPayload(payload)
}

// SignerPayloadVersion is bumped whenever the canonical signer-payload wire
// format changes in a way that would alter the checksum for an unchanged signer
// intent. It is part of the hashed payload, so a version skew between the
// producer (custody) and a consumer surfaces as a checksum mismatch rather than
// a silent disagreement.
const SignerPayloadVersion uint32 = 1

// signerPayload is the canonical, content-addressed projection of the custody
// receipt signer set published under the registry's KEY_SIGNERS (ADR-017). The
// threshold travels inside the payload — there is no separate registry entry.
// Both custody (which seeds + commits it) and the receipt verifier (which reads
// it via RegistrySignerSource) marshal/parse through this one definition, so the
// wire format has a single source of truth.
type signerPayload struct {
	Version   uint32   `json:"v"`
	Threshold int      `json:"threshold"`
	Signers   []string `json:"signers"` // ascending, lowercase 0x hex
}

// MarshalSignerPayload returns the deterministic byte encoding of (signers,
// threshold): JSON with stable field order, signers deduplicated, lowercased,
// and sorted ascending, so two callers agreeing on intent produce byte-identical
// output (and therefore an identical keccak256 checksum).
func MarshalSignerPayload(signers []common.Address, threshold int) ([]byte, error) {
	if len(signers) == 0 {
		return nil, fmt.Errorf("signer payload: empty signer set")
	}
	if threshold <= 0 || threshold > len(signers) {
		return nil, fmt.Errorf("signer payload: threshold %d out of range for %d signers", threshold, len(signers))
	}
	seen := make(map[common.Address]struct{}, len(signers))
	hexes := make([]string, 0, len(signers))
	for _, s := range signers {
		if _, dup := seen[s]; dup {
			return nil, fmt.Errorf("signer payload: duplicate signer %s", s.Hex())
		}
		seen[s] = struct{}{}
		hexes = append(hexes, strings.ToLower(s.Hex()))
	}
	sort.Strings(hexes)
	return json.Marshal(signerPayload{Version: SignerPayloadVersion, Threshold: threshold, Signers: hexes})
}

// ParseSignerPayload decodes bytes produced by MarshalSignerPayload back into a
// signer set and threshold, validating the version, the addresses, ascending
// order, and the threshold bound.
func ParseSignerPayload(b []byte) ([]common.Address, int, error) {
	var p signerPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, 0, fmt.Errorf("signer payload: decode: %w", err)
	}
	if p.Version != SignerPayloadVersion {
		return nil, 0, fmt.Errorf("signer payload: unsupported version %d (want %d)", p.Version, SignerPayloadVersion)
	}
	if len(p.Signers) == 0 {
		return nil, 0, fmt.Errorf("signer payload: empty signer set")
	}
	out := make([]common.Address, 0, len(p.Signers))
	var last []byte
	for _, s := range p.Signers {
		if !common.IsHexAddress(s) {
			return nil, 0, fmt.Errorf("signer payload: %q is not a hex address", s)
		}
		a := common.HexToAddress(s)
		if last != nil && bytes.Compare(a[:], last) <= 0 {
			return nil, 0, fmt.Errorf("signer payload: signers not strictly ascending at %s", s)
		}
		last = a[:]
		out = append(out, a)
	}
	if p.Threshold <= 0 || p.Threshold > len(out) {
		return nil, 0, fmt.Errorf("signer payload: threshold %d out of range for %d signers", p.Threshold, len(out))
	}
	return out, p.Threshold, nil
}

package receipt

import (
	"context"
	"fmt"

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

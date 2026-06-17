package core

import (
	"bytes"
	"fmt"
)

// AssetID is a unique identifier for a token (§2).
// Symbols (e.g. "USDT", "ETH") are mapped to L1 (ChainID, TokenAddress) pairs
// via the AssetResolver interface at runtime.
type AssetID string

// FinalizedWithdrawal is the positive clearing-layer finality object consumed
// by custody after the challenge window expires. Attestation signs
// SigningMessage(); Block is carried so custody providers can bind the ID to
// the exact withdrawal payload without maintaining clearing state.
type FinalizedWithdrawal struct {
	WithdrawalID [32]byte    `json:"WithdrawalID"`
	BlockHash    [32]byte    `json:"BlockHash"`
	EntryIndex   uint64      `json:"EntryIndex"`
	FinalizedAt  int64       `json:"FinalizedAt"`
	Attestation  Attestation `json:"Attestation"`
	Block        Block       `json:"Block"`
}

// FinalizedWithdrawalHeader is the deterministic preimage projection for a
// FinalizedWithdrawal. It intentionally excludes Attestation and Block while
// binding the block hash and entry location that custody must verify.
type FinalizedWithdrawalHeader struct {
	WithdrawalID [32]byte
	BlockHash    [32]byte
	EntryIndex   uint64
	FinalizedAt  int64
}

func (fw *FinalizedWithdrawal) Header() FinalizedWithdrawalHeader {
	if fw == nil {
		return FinalizedWithdrawalHeader{}
	}
	return FinalizedWithdrawalHeader{
		WithdrawalID: fw.WithdrawalID,
		BlockHash:    fw.BlockHash,
		EntryIndex:   fw.EntryIndex,
		FinalizedAt:  fw.FinalizedAt,
	}
}

// SigningMessage returns the canonical CBOR preimage covered by the BLS
// finality signature (ADR-009 §4). It is not the same preimage as Block and
// excludes Attestation so the signature cannot be self-referential.
// Returns nil for a nil receiver so callers can distinguish "no message" from
// an all-zero preimage.
//
// Preimage shape: canonical CBOR of FinalizedWithdrawalHeader{WithdrawalID,
// BlockHash, EntryIndex, FinalizedAt}. Frozen by the FinalizedWithdrawalHeader
// case in TestGoldens_Preimages; any change is a schema-family bump.
func (fw *FinalizedWithdrawal) SigningMessage() []byte {
	if fw == nil {
		return nil
	}
	header := fw.Header()
	var buf bytes.Buffer
	if err := (&header).MarshalCBOR(&buf); err != nil {
		// FinalizedWithdrawalHeader's generated codec writes to a bytes.Buffer,
		// which cannot fail; a non-nil error here is a structural regression.
		panic(fmt.Errorf("core: FinalizedWithdrawalHeader.MarshalCBOR: %w", err))
	}
	return buf.Bytes()
}

// BLSPubKeyG2Len is the serialized length of a BN254 G2 pubkey (4×32 bytes).
const BLSPubKeyG2Len = 128

// NodeID is a 256-bit identity in the Kademlia space (§3.1).
// A Node MAY operate multiple NodeIDs, each with its own collateral and BLS key pair.
type NodeID [32]byte

// First8Hex returns the first 8 hex characters of id. Used to synthesize
// human-readable Slot labels when a peer did not supply one
// (docs/plans/logical_node.md §5.4).
func First8Hex(id NodeID) string {
	const hex = "0123456789abcdef"
	var out [8]byte
	for i := 0; i < 4; i++ {
		b := id[i]
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out[:])
}

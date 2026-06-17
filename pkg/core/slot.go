package core

import "math/big"

// SlotState is the lifecycle state of a Registry slot (ADR-005 §3.3).
type SlotState uint8

const (
	// SlotStateWarmup: slot is syncing state, not eligible to sign (§3.3).
	SlotStateWarmup SlotState = iota

	// SlotStateActive: slot is fully operational and may sign blocks.
	SlotStateActive

	// SlotStateUnbonding: slot has unregistered. Cannot sign new blocks but
	// remains in the Kademlia tree for the full UnbondingWindow so fraud
	// proofs may still be submitted against its collateral (ADR-005 §3.3).
	SlotStateUnbonding

	// SlotStateEvicted: slot has been fully removed from the Kademlia tree.
	SlotStateEvicted

	// SlotStateDisabled: slot missed heartbeats and is temporarily disabled (bitmask = 0).
	SlotStateDisabled
)

// Slot is a logical node: one per NFT locked on Registry. Carries the slot's
// public identity (NodeID, Label, BLSPubKey) plus its Registry position and
// economic/lifecycle metadata.
type Slot struct {
	ID        NodeID `json:"id"`
	Label     string `json:"label"`
	BLSPubKey []byte `json:"bls_pubkey,omitempty"`

	// Registry position
	TokenID *big.Int `json:"-"`
	Index   uint64   `json:"-"` // widened from uint32 for cbor-gen compat (Wave 2 pre)

	// Economic weight + lifecycle
	Owner         string    `json:"-"` // EOA (hex) that owns the NFT per Registry.ownerOf(tokenId)
	Collateral    *big.Int  `json:"-"`
	ActivatedAt   uint64    `json:"-"`
	DeactivatedAt uint64    `json:"-"`
	State         SlotState `json:"-"`
	LastHeartbeat int64     `json:"-"`
	MissedBeats   uint64    `json:"-"` // widened from uint32 for cbor-gen compat (Wave 2 pre)
}

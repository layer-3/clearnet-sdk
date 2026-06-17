// Package protocol defines the canonical libp2p wire contract: the stream
// protocol identifiers, the GossipSub topic names, and the framed message
// structs that travel over them.
//
// Every identifier is pinned to a single protocol version (1.0.0). A node only
// speaks to peers that dial the exact same string, so the version is the wire
// compatibility boundary — bump it deliberately, never per-stream.
//
// The package is transport-agnostic: it owns the names and the bytes, never a
// host.Host. The auth/receipt/pubsub helpers under pkg/p2p take a caller-built
// host and register handlers for these identifiers.
package protocol

import "fmt"

// Stream protocol identifiers. Each names a libp2p request/response or one-shot
// stream; the body is a cborx envelope (auth) or frame (receipt) of the typed
// payload. Only the identifiers a shared SDK consumer drives are exported here;
// the cluster-internal streams (transfer, heartbeat, identify, …) stay in the
// node that owns them.
//
// TODO(sdk): revisit whether other streams belong here. The rule today is
// "move a channel iff a consumer of the SDK could legitimately speak it" — so
// only the three shared custody↔clearnet streams (auth, burn/mint receipt)
// moved, and the ~14 cluster-internal streams (transfer, swap, shardsync,
// heartbeat, identify, peerexchange, signature, …) stayed in node.go. If a
// future consumer needs to dial one of those (and its wire body is extracted
// to the SDK), promote its identifier here and pin the version.
const (
	// ProtocolAuth carries the nonce-challenge authentication handshake: the
	// server sends an AuthChallenge, the peer returns a signed AuthResponse.
	ProtocolAuth = "/ynp/auth/1.0.0"

	// ProtocolBurnReceipt carries a signed BurnReceipt — the attestation that
	// the L1 execute() of a finalized withdrawal landed — and a ReceiptAck.
	ProtocolBurnReceipt = "/ynp/burnreceipt/1.0.0"

	// ProtocolMintReceipt carries a signed MintReceipt — the attestation that
	// an L1 deposit confirmed — and a ReceiptAck.
	ProtocolMintReceipt = "/ynp/mintreceipt/1.0.0"
)

// GossipSub topic names. The topic name encodes the payload type; a subscriber
// dispatches on the topic and decodes the body as the cborx envelope of that
// payload directly (no inner message wrapper).
const (
	// TopicBlocks fans out sealed *core.Block values.
	TopicBlocks = "/clearnet/blocks.v1"

	// TopicTransfers fans out a batched []core.Event of non-pool events.
	TopicTransfers = "/clearnet/transfers.v1"

	// TopicWithdrawals fans out *core.FinalizedWithdrawal notifications.
	TopicWithdrawals = "/clearnet/withdrawals.v1"

	// TopicChallenges fans out fraud-challenge submissions.
	TopicChallenges = "/clearnet/challenges.v1"
)

// PoolTopic returns the canonical GossipSub topic for a pool anchor. Format:
// /clearnet/pool/<anchor-hex>.v1. The payload is a batched []core.Event of pool
// events (Swap, LiquidityAdded/Removed, Repeg, PoolCreated).
func PoolTopic(anchor [32]byte) string {
	return fmt.Sprintf("/clearnet/pool/%x.v1", anchor)
}

// PoolTopicHex is the string-anchor variant for call sites that already carry
// the anchor as hex (no 0x prefix, lowercase, 64 chars).
func PoolTopicHex(anchorHex string) string {
	return fmt.Sprintf("/clearnet/pool/%s.v1", anchorHex)
}

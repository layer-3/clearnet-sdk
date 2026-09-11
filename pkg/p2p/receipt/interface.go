package receipt

import (
	"context"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// ReceiptHandler is the business seam a consumer implements to process inbound
// receipts. The Server decodes the wire frame and calls the matching method;
// the returned ReceiptAck is validated and sent back to the peer. A non-nil
// handler error is delivered as temporary_failure; malformed request decode is
// delivered as corrupt when the server can still write an ACK.
//
// Implementations must apply latest-only custody-to-clearnet ingress
// verification before accepting a receipt and be idempotent by receipt logical
// id (BurnReceipt: WithdrawalID; MintReceipt: AssetURI+TxID). The SDK
// ReceiptLogicalID helpers define these identities while excluding proof and
// signer epoch. A handler should return ReceiptAckAlreadyAccepted when that
// logical ID was accepted previously; already-accepted clearnet-internal
// propagation is outside this ingress verifier boundary.
// A consumer that handles only one kind still implements both methods; return a
// reject ACK for the unhandled one.
type ReceiptHandler interface {
	OnBurnReceipt(ctx context.Context, r *core.BurnReceipt) (p2pproto.ReceiptAck, error)
	OnMintReceipt(ctx context.Context, r *core.MintReceipt) (p2pproto.ReceiptAck, error)
}

package receipt

import (
	"context"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// ReceiptHandler is the business seam a consumer implements to process inbound
// receipts. The Server decodes the wire frame and calls the matching method;
// the returned ReceiptAck is sent back to the peer. A non-nil error is
// delivered to the peer as Accepted=false with the error string.
//
// Implementations must be idempotent on the clearing layer's natural de-dupe
// keys (BurnReceipt: BlockHash+EntryIndex; MintReceipt: ChainID+L1TxHash+
// LogIndex) so client retries are safe. A consumer that handles only one kind
// still implements both methods — return a reject ack for the unhandled one.
type ReceiptHandler interface {
	OnBurnReceipt(ctx context.Context, r *core.BurnReceipt) (p2pproto.ReceiptAck, error)
	OnMintReceipt(ctx context.Context, r *core.MintReceipt) (p2pproto.ReceiptAck, error)
}

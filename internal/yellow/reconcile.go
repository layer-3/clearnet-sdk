package yellow

import (
	"context"
	"fmt"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
)

// Reconcile against the saved gateway, which owns this authorization's status.
// Activity also includes transfers submitted through other wallets or gateways.
func reconcileRejectedNonce(ctx context.Context, c *client.Client, op *operation) error {
	status, err := c.Operation(ctx, op.ID)
	if err != nil {
		return err
	}
	if status.Status == "sealed" {
		return observe(op, status)
	}
	entry, err := c.SealedTransferAtNonce(ctx, op.Sender, op.Nonce)
	if err != nil {
		return err
	}
	if entry == nil {
		return fmt.Errorf("no matching outgoing seal is available in the original gateway's recent history")
	}
	if entry.OperationID == op.ID {
		return fmt.Errorf("history reports this payment sealed but its operation status disagrees; the original gateway must reconcile its index")
	}
	op.NonceConsumedBy = entry
	op.Updated = time.Now().UTC()
	return nil
}

package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

// SealedTransferAtNonce searches up to 1,000 recent activity entries for an
// outgoing transfer at the exact sender nonce. Evidence comes from the trusted
// gateway's verified history index, not client-side quorum verification.
// A nil result is missing evidence, never proof of non-execution.
func (c *Client) SealedTransferAtNonce(ctx context.Context, sender string, nonce uint64) (*TransferActivity, error) {
	address, err := normalizeUserAccount(sender)
	if err != nil {
		return nil, err
	}
	cursor := ""
	seen := map[string]bool{"": true}
	for pageNumber := 0; pageNumber < 10; pageNumber++ {
		page, err := c.Transfers(ctx, address, TransferPageOptions{Limit: 100, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		if len(page.Items) > 100 {
			return nil, fmt.Errorf("transfer history exceeded the requested page limit")
		}
		for _, entry := range page.Items {
			from, err := normalizeUserAccount(entry.From)
			if err != nil || !strings.EqualFold(from, address) || entry.Nonce != nonce {
				continue
			}
			if entry.Delivery != "sender_sealed" || (entry.Direction != "out" && entry.Direction != "self") || !eip712.ValidTransferOperationID(entry.OperationID) || !validHash(entry.BlockHash) || !validHash(entry.EntryHash) || entry.SealedAt <= 0 {
				return nil, fmt.Errorf("transfer history has incomplete seal evidence at nonce %d", nonce)
			}
			return &entry, nil
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			return nil, nil
		}
		cursor = *page.NextCursor
		if seen[cursor] {
			return nil, fmt.Errorf("transfer history repeated a pagination cursor")
		}
		seen[cursor] = true
	}
	return nil, fmt.Errorf("no seal found at nonce %d in the latest 1,000 activity entries", nonce)
}

func validHash(raw string) bool {
	return eip712.ValidTransferOperationID(strings.ToLower(raw))
}

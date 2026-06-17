package xrpl

import (
	"context"
	"fmt"
	"sort"

	"github.com/Peersyst/xrpl-go/xrpl/queries/account"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
)

// LedgerTicketProvider is a TicketProvider that hands out Tickets already
// provisioned on the vault account, read live from the ledger via
// account_objects. It does not create Tickets (that needs signing authority
// over the vault, which the caller orchestrates); it surfaces the ones that
// exist.
//
// TicketFor returns the lowest available TicketSequence and is stateless: it
// does not reserve the Ticket it returns, so two concurrent withdrawals can be
// handed the same one (the second submit then fails tefNO_TICKET). Callers that
// run withdrawals concurrently must layer their own reservation/pool on top —
// this is the simple single-flight building block.
type LedgerTicketProvider struct {
	client  *rpc.Client
	account types.Address
}

var _ TicketProvider = (*LedgerTicketProvider)(nil)

// NewLedgerTicketProvider builds a provider reading Tickets owned by
// vaultAddress over the JSON-RPC at rpcURL.
func NewLedgerTicketProvider(rpcURL, vaultAddress string) (*LedgerTicketProvider, error) {
	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	return &LedgerTicketProvider{
		client:  rpc.NewClient(cfg),
		account: types.Address(vaultAddress),
	}, nil
}

// TicketFor returns the lowest TicketSequence currently owned by the vault. The
// withdrawalID is ignored — any of the account's Tickets authorizes any
// withdrawal. Errors if the account owns no Tickets.
func (p *LedgerTicketProvider) TicketFor(_ context.Context, _ [32]byte) (uint32, error) {
	resp, err := p.client.GetAccountObjects(&account.ObjectsRequest{
		Account: p.account,
		Type:    account.TicketObject,
	})
	if err != nil {
		return 0, fmt.Errorf("xrpl: account_objects: %w", err)
	}
	seqs := make([]uint32, 0, len(resp.AccountObjects))
	for _, obj := range resp.AccountObjects {
		if asString(obj["LedgerEntryType"]) != "Ticket" {
			continue
		}
		if seq, ok := uint32Field(obj["TicketSequence"]); ok {
			seqs = append(seqs, seq)
		}
	}
	if len(seqs) == 0 {
		return 0, fmt.Errorf("xrpl: account %s owns no tickets", p.account)
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	return seqs[0], nil
}

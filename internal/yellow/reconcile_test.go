package yellow

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

func TestSendReconcilesExternalNonceAutomatically(t *testing.T) {
	g := newGateway(t)
	g.postCode = 400
	dir, key := setupAgent(t, g)
	code, _, _ := runUnattended(t, dir, agentSend()...)
	if code != ExitRejected {
		t.Fatal("expected rejected first submission", code)
	}
	s := store{dir}
	ac, _ := s.account("bot")
	original := ac.Operations[0]
	otherID := "0x" + strings.Repeat("cd", 32)
	entry := client.TransferActivity{OperationID: otherID, From: core.UserURI(ac.Address), To: recipient, Nonce: 7, Direction: "out", Delivery: "sender_sealed", BlockHash: "0x" + strings.Repeat("ab", 32), EntryHash: "0x" + strings.Repeat("ef", 32), EntryIndex: 2, SealedAt: 42}
	next := "second-page"
	g.nonce, g.postCode = 8, 200
	g.historyPages = map[string]client.TransferPage{"": {NextCursor: &next}, next: {Items: []client.TransferActivity{entry}}}
	code, env, diag := runUnattended(t, dir, agentSend()...)
	if code != 0 || len(g.posted) != 2 || g.historyCalls != 2 || !strings.Contains(diag, "reconciliation saved") {
		t.Fatal("external transfer did not unblock agent send automatically", code, env, diag)
	}
	ac, _ = s.account("bot")
	got := ac.Operations[0]
	if got.State != "rejected" || got.Reason != original.Reason || !bytes.Equal(got.Payload.Data, original.Payload.Data) || got.NonceConsumedBy == nil || got.NonceConsumedBy.OperationID != otherID {
		t.Fatal("reconciliation lost the original record or evidence", got)
	}
	if _, err := decodeOperation(key, got); err != nil {
		t.Fatal("saved authorization no longer decrypts", err)
	}
	// The durable evidence survives a restart and no longer depends on history retention.
	g.nonce, g.historyCode = 9, 503
	if code, env, _ := runUnattended(t, dir, agentSend()...); code != 0 || len(g.posted) != 3 || g.historyCalls != 2 {
		t.Fatal("restart lost nonce evidence", code, env)
	}
}

func TestNonceReconciliationFailsClosed(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	s := store{dir}
	ac, _ := s.account("alice")
	cfg, _ := s.config()
	op := operation{ID: "0x" + strings.Repeat("ab", 32), Sender: ac.Address, Nonce: 6, Profile: cfg.Networks["dev"], State: "rejected", Reason: "not admitted"}
	for _, tt := range []struct {
		name                   string
		status                 client.OperationStatus
		historyCode            int
		wantState              string
		historyReportsOriginal bool
	}{
		{"no history evidence", client.OperationStatus{Status: "rejected", Reason: "not admitted"}, 0, "rejected", false},
		{"history unavailable", client.OperationStatus{Status: "unknown"}, 503, "rejected", false},
		{"original actually sealed", client.OperationStatus{Status: "sealed", Receipt: &client.Receipt{BlockHash: "0x" + strings.Repeat("cd", 32)}}, 0, "sealed", false},
		{"history contradicts rejection", client.OperationStatus{Status: "rejected", Reason: "not admitted"}, 0, "rejected", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g.status, g.historyCode = tt.status, tt.historyCode
			g.historyPages = map[string]client.TransferPage{}
			if tt.historyReportsOriginal {
				g.historyPages[""] = client.TransferPage{Items: []client.TransferActivity{{OperationID: op.ID, From: op.Sender, Nonce: op.Nonce, Direction: "out", Delivery: "sender_sealed", BlockHash: "0x" + strings.Repeat("cd", 32), EntryHash: "0x" + strings.Repeat("ef", 32), SealedAt: 42}}}
			}
			ac.Operations = []operation{op}
			if err := s.saveAccount("alice", ac); err != nil {
				t.Fatal(err)
			}
			a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			if code := a.execute(context.Background(), sendArgs()); code != ExitUnknown || len(g.posted) != 0 {
				t.Fatal("unsafe replacement reached POST", code, out)
			}
			saved, _ := s.account("alice")
			if saved.Operations[0].State != tt.wantState || saved.Operations[0].NonceConsumedBy != nil {
				t.Fatal("unexpected reconciliation", saved.Operations[0])
			}
		})
	}
}

func TestOperationShowReconcilesRejectedNonce(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	s := store{dir}
	ac, _ := s.account("alice")
	cfg, _ := s.config()
	op := operation{ID: "0x" + strings.Repeat("ab", 32), Sender: ac.Address, Nonce: 6, Profile: cfg.Networks["dev"], State: "rejected"}
	ac.Operations = []operation{op}
	if err := s.saveAccount("alice", ac); err != nil {
		t.Fatal(err)
	}
	g.status = client.OperationStatus{Status: "rejected", Reason: "not admitted"}
	g.historyPages = map[string]client.TransferPage{"": {Items: []client.TransferActivity{{OperationID: "0x" + strings.Repeat("cd", 32), From: ac.Address, Nonce: 6, Direction: "out", Delivery: "sender_sealed", BlockHash: "0x" + strings.Repeat("ef", 32), EntryHash: "0x" + strings.Repeat("ab", 32), SealedAt: 42}}}}
	// A changed current profile cannot redirect recovery from its saved gateway.
	other := newGateway(t)
	p := cfg.Networks["dev"]
	p.Gateway = other.server.URL
	cfg.Networks["dev"] = p
	if err := s.saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"--json", "operation", "show", op.ID}); code != ExitRejected || !strings.Contains(out.String(), `"nonceConsumedBy":`) || other.infoCalls != 0 || len(g.posted) != 0 {
		t.Fatal("show failed to preserve and expose original-gateway reconciliation", code, out)
	}
}

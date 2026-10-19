package yellow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
)

func TestCancellationAfterSubmissionPreservesOutcomeExit(t *testing.T) {
	for _, phase := range []string{"post", "wait"} {
		t.Run(phase, func(t *testing.T) {
			g := newGateway(t)
			dir, _ := setup(t, g)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			args, want, state := sendArgs(), ExitUnknown, "unresolved"
			if phase == "post" {
				g.postHook = cancel
			} else {
				g.postCode = 202
				g.status = client.OperationStatus{Status: "pending"}
				g.statusHook = cancel
				args, want, state = sendArgs("--wait", "5s"), ExitPending, "pending"
			}
			a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			if code := a.execute(ctx, args); code != want {
				t.Fatalf("cancel after %s exit %d, want %d: %s", phase, code, want, out)
			}
			assertOneJSON(t, out.String())
			ac, err := a.store.account("alice")
			if err != nil || len(ac.Operations) != 1 || ac.Operations[0].State != state || ac.Operations[0].Submitted == nil {
				t.Fatalf("lost submitted outcome: %+v %v", ac, err)
			}
			if !strings.Contains(out.String(), `"state":"`+state+`"`) {
				t.Fatal("JSON disagrees with the exit code", out)
			}
		})
	}
}

func TestSubmissionReasonSurvivesRestart(t *testing.T) {
	g := newGateway(t)
	g.postCode = 400
	dir, _ := setup(t, g)
	a, out, diag := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != ExitRejected {
		t.Fatal(code, out)
	}
	if !strings.Contains(out.String(), `"reason":`) || !strings.Contains(out.String(), "invalid transfer") || !strings.Contains(diag.String(), "invalid transfer") {
		t.Fatal("submission cause absent from result or diagnostics", out, diag)
	}
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"--json", "operation", "list"}); code != 0 {
		t.Fatal(code, out)
	}
	if !strings.Contains(out.String(), "invalid transfer") {
		t.Fatal("journal lost submission reason on restart", out)
	}
	ac, _ := a.store.account("alice")
	op := ac.Operations[0]
	if err := observe(&op, &client.OperationStatus{Status: "rejected", Reason: "insufficient balance"}); err != nil || op.Reason != "insufficient balance" {
		t.Fatal("status rejection reason was discarded", err, op)
	}
}

func TestRejectedNonceAdvanceNeedsLocalSeal(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	s := store{dir}
	ac, _ := s.account("alice")
	cfg, _ := s.config()
	p := cfg.Networks["dev"]
	for _, tt := range []struct {
		name     string
		rejected uint64
		sealed   *uint64
		want     int
	}{
		{"unexplained advance", 6, nil, ExitUnknown},
		{"same live nonce", 7, nil, 0},
		{"seal at rejected nonce", 6, noncePointer(6), 0},
		{"seal at different nonce", 6, noncePointer(5), ExitUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g.mu.Lock()
			g.posted = nil
			g.mu.Unlock()
			rejected := operation{ID: "0x" + strings.Repeat("ab", 32), Account: "alice", Sender: ac.Address, Nonce: tt.rejected, Profile: p, State: "unresolved"}
			if err := observe(&rejected, &client.OperationStatus{Status: "rejected", Reason: "not admitted"}); err != nil {
				t.Fatal(err)
			}
			ac.Operations = []operation{rejected}
			if tt.sealed != nil {
				ac.Operations = append(ac.Operations, operation{ID: "0x" + strings.Repeat("cd", 32), Profile: p, Nonce: *tt.sealed, State: "sealed"})
			}
			if err := s.saveAccount("alice", ac); err != nil {
				t.Fatal(err)
			}
			a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
			if code := a.execute(context.Background(), sendArgs()); code != tt.want {
				t.Fatalf("exit %d, want %d: %s", code, tt.want, out)
			}
			if tt.want != 0 && (len(g.posted) != 0 || !strings.Contains(out.String(), "advanced past rejected")) {
				t.Fatal("unexplained nonce advance reached POST", out)
			}
		})
	}
}

func noncePointer(n uint64) *uint64 { return &n }

func TestSavedUnsubmittedSendGuardPointsToRetry(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	s := store{dir}
	ac, _ := s.account("alice")
	cfg, _ := s.config()
	op := operation{ID: "0x" + strings.Repeat("ab", 32), Profile: cfg.Networks["dev"], State: "signed_unsubmitted", Created: time.Now()}
	ac.Operations = []operation{op}
	if err := s.saveAccount("alice", ac); err != nil {
		t.Fatal(err)
	}
	a, out, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	if code := a.execute(context.Background(), sendArgs()); code != ExitUnknown || !strings.Contains(out.String(), "yellow operation retry "+op.ID) {
		t.Fatal("unsubmitted guard gives the wrong recovery action", code, out)
	}
	if len(g.posted) != 0 {
		t.Fatal("blocked send submitted")
	}
}

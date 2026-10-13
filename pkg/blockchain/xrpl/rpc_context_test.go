package xrpl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// blockingRippled answers like fakeRippled except for one method, which it
// holds open until the client abandons the request. arrived fires when that
// request is received and cancelled when the server sees it cancelled.
type blockingRippled struct {
	inner     http.Handler
	method    string
	arrived   chan struct{}
	cancelled chan struct{}
}

func newBlockingRippled(t *testing.T, inner *fakeRippled, method string) (*blockingRippled, string) {
	t.Helper()
	b := &blockingRippled{inner: inner, method: method, arrived: make(chan struct{}, 1), cancelled: make(chan struct{}, 1)}
	srv := httptest.NewServer(b)
	t.Cleanup(srv.Close)
	return b, srv.URL
}

func (b *blockingRippled) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &req)
	if req.Method != b.method {
		r.Body = io.NopCloser(bytes.NewReader(body))
		b.inner.ServeHTTP(w, r)
		return
	}
	b.arrived <- struct{}{}
	select {
	case <-r.Context().Done():
		b.cancelled <- struct{}{}
	case <-time.After(10 * time.Second):
	}
}

// TestRPCCallsHonourCancellation cancels ctx while a representative request
// of each kind is in flight and asserts the call returns promptly with the
// context error and the server sees the request abandoned.
func TestRPCCallsHonourCancellation(t *testing.T) {
	signerA, idA := newXRPLSigner(t)
	_, idB := newXRPLSigner(t)
	blob := multisignBlob(t, context.Background(), signerA, idA, idB.ClassicAddress)

	for _, tc := range []struct {
		name   string
		method string
		node   func() *fakeRippled
		call   func(t *testing.T, ctx context.Context, url string) error
	}{
		{"fee floor base fee", "server_state", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorWithdrawalFinalizer(t, url)
			f.SetThresholdResolver(func(context.Context) (int, error) { return floorTestQuorum, nil })
			packed, op, wid := floorWithdrawal(t, floorTestFloor)
			return f.Validate(ctx, packed, op, wid, floorTestBounds)
		}},
		{"fee floor quorum", "account_info", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorRotationFinalizer(t, url)
			return f.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, floorTestFloor), floorNewSigners, 2)
		}},
		{"network policy", "server_info", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorRotationFinalizer(t, url)
			f.SetThresholdResolver(func(context.Context) (int, error) { return floorTestQuorum, nil })
			f.SetBaseFeeResolver(func(context.Context) (uint64, error) { return floorTestBaseFee, nil })
			return f.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, floorTestFloor), floorNewSigners, 2)
		}},
		{"ledger state", "ledger", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorWithdrawalFinalizer(t, url)
			_, err := f.ledgerState(ctx)
			return err
		}},
		{"submit", "submit_multisigned", func() *fakeRippled {
			n := healthyRippled()
			n.quorum, n.signerAccounts = 1, []string{idA.ClassicAddress}
			return n
		}, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorWithdrawalFinalizer(t, url)
			_, err := f.Submit(ctx, nil, [][]byte{[]byte(blob)})
			return err
		}},
		{"verify execution", "account_tx", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			f, _ := newFloorWithdrawalFinalizer(t, url)
			_, _, err := f.VerifyExecution(ctx, [32]byte{1})
			return err
		}},
		{"ticket", "account_objects", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			p, err := NewLedgerTicketProvider(url, floorTestVault)
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.TicketFor(ctx, [32]byte{})
			return err
		}},
		{"deposit status", "tx", healthyRippled, func(t *testing.T, ctx context.Context, url string) error {
			d := newTestDepositor(t, url)
			hash := "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
			_, err := d.ChainDepositStatus(ctx, hash, DepositID(hash), 0)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, url := newBlockingRippled(t, tc.node(), tc.method)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(t, ctx, url) }()

			select {
			case <-b.arrived:
			case err := <-done:
				t.Fatalf("call returned before reaching %s: %v", tc.method, err)
			case <-time.After(3 * time.Second):
				t.Fatalf("%s request never arrived", tc.method)
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
				if errors.Is(err, ErrCandidateRejected) {
					t.Fatalf("cancellation attributed to the candidate: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("call did not return promptly after cancellation")
			}
			select {
			case <-b.cancelled:
			case <-time.After(2 * time.Second):
				t.Fatal("server did not see the request cancelled")
			}
		})
	}
}

// TestRPCCallHonoursDeadline covers a ctx deadline rather than a cancel.
func TestRPCCallHonoursDeadline(t *testing.T) {
	b, url := newBlockingRippled(t, healthyRippled(), "ledger")
	f, _ := newFloorWithdrawalFinalizer(t, url)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := f.ledgerState(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("call took %v past a 200ms deadline", elapsed)
	}
	select {
	case <-b.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not see the request cancelled")
	}
}

// TestRPCCallsKeepResultsAndErrorMapping checks the context-bound transport
// leaves decoding and error mapping as before.
func TestRPCCallsKeepResultsAndErrorMapping(t *testing.T) {
	ctx := context.Background()
	url := newFakeRippled(t, healthyRippled())

	f, _ := newFloorWithdrawalFinalizer(t, url)
	state, err := f.ledgerState(ctx)
	if err != nil || state != (LedgerState{ValidatedIndex: 1000, CloseUnix: 10_000 + rippleEpochOffset}) {
		t.Fatalf("ledgerState = %+v, %v", state, err)
	}
	if q, err := f.LiveQuorum(ctx); err != nil || q != floorTestQuorum {
		t.Fatalf("LiveQuorum = %d, %v; want %d", q, err, floorTestQuorum)
	}

	// An RPC error result (txnNotFound) still maps to DepositAbsent.
	d := newTestDepositor(t, url)
	hash := "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
	if status, err := d.ChainDepositStatus(ctx, hash, DepositID(hash), 0); err != nil || status != core.DepositAbsent {
		t.Fatalf("ChainDepositStatus = %v, %v; want DepositAbsent, nil", status, err)
	}

	// An HTTP error from rippled is still an error, not a context error.
	p, err := NewLedgerTicketProvider(url, floorTestVault)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.TicketFor(ctx, [32]byte{}); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("TicketFor error = %v, want the RPC error", err)
	}

	// The exported policy helper still resolves through a plain client.
	client, err := newRPCClient(url)
	if err != nil {
		t.Fatal(err)
	}
	if policy, err := ResolveNetworkIDPolicy(client); err != nil || policy.Required {
		t.Fatalf("ResolveNetworkIDPolicy = %+v, %v; want not required", policy, err)
	}
}

// TestLiveQuorumIsValidated covers the review fix: LiveQuorum reads the
// validated SignerList, so wiring it as the threshold resolver keeps the fee
// floor fail-closed on an unvalidated response.
func TestLiveQuorumIsValidated(t *testing.T) {
	ctx := context.Background()
	unvalidated := healthyRippled()
	unvalidated.signersValidated = false

	t.Run("withdrawal", func(t *testing.T) {
		f, s := newFloorWithdrawalFinalizer(t, newFakeRippled(t, unvalidated))
		f.SetThresholdResolver(f.LiveQuorum)
		if _, err := f.LiveQuorum(ctx); err == nil {
			t.Fatal("LiveQuorum accepted an unvalidated signer list")
		}
		packed, op, wid := floorWithdrawal(t, floorTestFloor)
		err := f.Validate(ctx, packed, op, wid, floorTestBounds)
		if err == nil || errors.Is(err, ErrCandidateRejected) {
			t.Fatalf("Validate error = %v, want an operational error", err)
		}
		if n := s.calls.Load(); n != 0 {
			t.Fatalf("Validate invoked the signer %d times", n)
		}
	})
	t.Run("rotation", func(t *testing.T) {
		f, s := newFloorRotationFinalizer(t, newFakeRippled(t, unvalidated))
		f.SetThresholdResolver(f.LiveQuorum)
		if _, err := f.LiveQuorum(ctx); err == nil {
			t.Fatal("LiveQuorum accepted an unvalidated signer list")
		}
		err := f.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, floorTestFloor), floorNewSigners, 2)
		if err == nil || errors.Is(err, ErrCandidateRejected) {
			t.Fatalf("Validate error = %v, want an operational error", err)
		}
		if n := s.calls.Load(); n != 0 {
			t.Fatalf("Validate invoked the signer %d times", n)
		}
	})
}

// TestDefaultLedgerStateRequiresValidated covers the default ledger resolver:
// an unvalidated ledger response is an operational error for both finalizers.
func TestDefaultLedgerStateRequiresValidated(t *testing.T) {
	ctx := context.Background()
	node := healthyRippled()
	node.ledgerUnvalidated = true
	url := newFakeRippled(t, node)

	w, s := newFloorWithdrawalFinalizer(t, url)
	w.SetStandaloneLedgerMode(false)
	if _, err := w.ledgerState(ctx); err == nil {
		t.Fatal("withdrawal ledgerState accepted an unvalidated ledger")
	}
	packed, op, wid := floorWithdrawal(t, floorTestFloor)
	if err := w.Validate(ctx, packed, op, wid, floorTestBounds); err == nil || errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("withdrawal Validate error = %v, want an operational error", err)
	}
	if n := s.calls.Load(); n != 0 {
		t.Fatalf("Validate invoked the signer %d times", n)
	}

	r, _ := newFloorRotationFinalizer(t, url)
	r.SetStandaloneLedgerMode(false)
	if _, err := r.ledgerState(ctx); err == nil {
		t.Fatal("rotation ledgerState accepted an unvalidated ledger")
	}
	if err := r.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, floorTestFloor), floorNewSigners, 2); err == nil || errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("rotation Validate error = %v, want an operational error", err)
	}
}

func newTestDepositor(t *testing.T, url string) *Depositor {
	t.Helper()
	s, _ := newXRPLSigner(t)
	d, err := NewDepositor(url, floorTestVault, s, NewAssetResolver(AssetResolverConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

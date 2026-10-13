package xrpl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Peersyst/xrpl-go/xrpl/rpc"

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

// busyRippled answers HTTP 503 to its first busy requests, then serves inner.
// It records every request it receives.
type busyRippled struct {
	inner    http.Handler
	busy     int
	requests chan struct{}
	served   atomic.Int32
}

func newBusyRippled(t *testing.T, busy int) (*busyRippled, string) {
	t.Helper()
	b := &busyRippled{inner: healthyRippled(), busy: busy, requests: make(chan struct{}, 16)}
	srv := httptest.NewServer(b)
	t.Cleanup(srv.Close)
	return b, srv.URL
}

func (b *busyRippled) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.requests <- struct{}{}
	if int(b.served.Add(1)) <= b.busy {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	b.inner.ServeHTTP(w, r)
}

func shortenBusyBackoff(t *testing.T, d time.Duration) {
	t.Helper()
	prev := rpcBusyBackoff
	rpcBusyBackoff = d
	t.Cleanup(func() { rpcBusyBackoff = prev })
}

// TestRPCCallBusyCancellation cancels (or times out) during the backoff after
// an HTTP 503. The call must return the context error promptly, not panic in
// xrpl-go's retry cleanup.
func TestRPCCallBusyCancellation(t *testing.T) {
	for _, tc := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want error
	}{
		{"cancel", func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, context.Canceled},
		{"deadline", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 300*time.Millisecond)
		}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, url := newBusyRippled(t, 100)
			ctx, cancel := tc.ctx()
			defer cancel()
			done := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- fmt.Errorf("panic: %v", r)
					}
				}()
				call, err := rpcCall(ctx, url, nil)
				if err != nil {
					done <- err
					return
				}
				_, err = validatedBaseFee(call)
				done <- err
			}()
			<-b.requests
			if tc.want == context.Canceled {
				time.Sleep(50 * time.Millisecond) // inside the 1s backoff
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, tc.want) {
					t.Fatalf("error = %v, want %v", err, tc.want)
				}
			case <-time.After(900 * time.Millisecond):
				t.Fatal("call did not return before the first retry")
			}
		})
	}
}

// TestRPCCallBusyRetriesKeepSemantics checks the 503 retry still succeeds once
// rippled recovers, resending the request body, and still fails with
// xrpl-go's overload error once the retries are exhausted.
func TestRPCCallBusyRetriesKeepSemantics(t *testing.T) {
	shortenBusyBackoff(t, 5*time.Millisecond)
	ctx := context.Background()

	b, url := newBusyRippled(t, 2)
	call, err := rpcCall(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fee, err := validatedBaseFee(call); err != nil || fee != floorTestBaseFee {
		t.Fatalf("validatedBaseFee after 503s = %d, %v; want %d", fee, err, floorTestBaseFee)
	}
	if b.served.Load() != 3 {
		t.Fatalf("server saw %d requests, want 3", b.served.Load())
	}

	b, url = newBusyRippled(t, 100)
	call, err = rpcCall(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = validatedBaseFee(call)
	var clientErr *rpc.ClientError
	if !errors.As(err, &clientErr) || clientErr.ErrorString != "Server is overloaded, rate limit exceeded" {
		t.Fatalf("exhausted 503 error = %v, want xrpl-go's overload error", err)
	}
	if int(b.served.Load()) != 1+rpcBusyRetries {
		t.Fatalf("server saw %d requests, want %d", b.served.Load(), 1+rpcBusyRetries)
	}
}

// TestConcurrentOperationsShareNetworkIDSafely runs quorum reads and
// validations concurrently on one finalizer of each kind; with -race it
// covers the shared NetworkID cache.
func TestConcurrentOperationsShareNetworkIDSafely(t *testing.T) {
	ctx := context.Background()
	url := newFakeRippled(t, healthyRippled())
	w, _ := newFloorWithdrawalFinalizer(t, url)
	r, _ := newFloorRotationFinalizer(t, url)
	packed, op, wid := floorWithdrawal(t, floorTestFloor)
	rotation := floorRotation(t, floorNewSigners, 2, floorTestFloor)

	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 4; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); _, err := w.LiveQuorum(ctx); errs <- err }()
		go func() { defer wg.Done(); errs <- w.Validate(ctx, packed, op, wid, floorTestBounds) }()
		go func() { defer wg.Done(); _, err := r.LiveQuorum(ctx); errs <- err }()
		go func() { defer wg.Done(); errs <- r.Validate(ctx, [32]byte{}, rotation, floorNewSigners, 2) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent operation failed: %v", err)
		}
	}
}

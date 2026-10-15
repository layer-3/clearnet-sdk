package xrpl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
)

// exportedBaseFee reads the validated base fee through the exported API, as a
// caller outside the SDK would.
func exportedBaseFee(ctx context.Context, url string) (uint64, error) {
	res, err := Request(ctx, url, &server.StateRequest{})
	if err != nil {
		return 0, err
	}
	var resp server.StateResponse
	if err := res.GetResult(&resp); err != nil {
		return 0, err
	}
	return uint64(resp.State.ValidatedLedger.BaseFee), nil
}

// errPanicked marks an error noPanic made from a recovered panic.
var errPanicked = errors.New("panic")

// noPanic runs call and turns a panic into an error wrapping errPanicked, so
// a test can reject it.
func noPanic(call func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", errPanicked, r)
		}
	}()
	return call()
}

// TestExportedRPCHonoursCancellation cancels ctx while a request is in flight,
// through both Request and a NewContextClient typed call.
func TestExportedRPCHonoursCancellation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		call   func(ctx context.Context, url string) error
	}{
		{"Request", "server_state", func(ctx context.Context, url string) error {
			_, err := exportedBaseFee(ctx, url)
			return err
		}},
		{"NewContextClient", "account_info", func(ctx context.Context, url string) error {
			call, err := NewContextClient(ctx, url)
			if err != nil {
				return err
			}
			_, err = fetchValidatedSignerQuorum(call, floorTestVault)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, url := newBlockingRippled(t, healthyRippled(), tc.method)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- noPanic(func() error { return tc.call(ctx, url) }) }()
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

// TestExportedRPCBusyCancellation cancels during the backoff after an HTTP 503.
func TestExportedRPCBusyCancellation(t *testing.T) {
	b, url := newBusyRippled(t, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- noPanic(func() error {
			_, err := exportedBaseFee(ctx, url)
			return err
		})
	}()
	<-b.requests
	time.Sleep(50 * time.Millisecond) // inside the 1s backoff
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(900 * time.Millisecond):
		t.Fatal("call did not return before the first retry")
	}
}

// busyThenDrop answers HTTP 503 to the first request and drops the connection
// of every later one, so the retry fails with a transport error. Through a
// plain xrpl-go client this sequence panics in xrpl-go's retry cleanup.
type busyThenDrop struct{ served atomic.Int32 }

func (b *busyThenDrop) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if b.served.Add(1) == 1 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	conn, _, err := w.(http.Hijacker).Hijack()
	if err == nil {
		_ = conn.Close()
	}
}

// TestExportedRPCBusyThenTransportError: a 503 followed by a transport error
// is an ordinary error, through both entry points, with no panic.
func TestExportedRPCBusyThenTransportError(t *testing.T) {
	shortenBusyBackoff(t, 5*time.Millisecond)
	for _, tc := range []struct {
		name string
		call func(ctx context.Context, url string) error
	}{
		{"Request", func(ctx context.Context, url string) error {
			_, err := exportedBaseFee(ctx, url)
			return err
		}},
		{"NewContextClient", func(ctx context.Context, url string) error {
			call, err := NewContextClient(ctx, url)
			if err != nil {
				return err
			}
			_, err = call.GetServerInfo(&server.InfoRequest{})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &busyThenDrop{}
			srv := httptest.NewServer(b)
			t.Cleanup(srv.Close)
			err := noPanic(func() error { return tc.call(context.Background(), srv.URL) })
			if errors.Is(err, errPanicked) {
				t.Fatalf("call panicked: %v", err)
			}
			if err == nil {
				t.Fatal("call succeeded against a dropped connection")
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want a transport error", err)
			}
			if b.served.Load() != 2 {
				t.Fatalf("server saw %d requests, want the 503 and one retry", b.served.Load())
			}
		})
	}
}

// TestExportedRPCKeepsResultsAndErrors checks that the exported API decodes
// results and maps errors exactly as a plain xrpl-go client does.
func TestExportedRPCKeepsResultsAndErrors(t *testing.T) {
	shortenBusyBackoff(t, 5*time.Millisecond)
	ctx := context.Background()
	url := newFakeRippled(t, healthyRippled())

	if fee, err := exportedBaseFee(ctx, url); err != nil || fee != floorTestBaseFee {
		t.Fatalf("base fee = %d, %v; want %d", fee, err, floorTestBaseFee)
	}
	call, err := NewContextClient(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if q, err := fetchValidatedSignerQuorum(call, floorTestVault); err != nil || q != floorTestQuorum {
		t.Fatalf("quorum = %d, %v; want %d", q, err, floorTestQuorum)
	}

	// An RPC error result keeps xrpl-go's error, through both clients.
	hash := "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
	_, exportedErr := Request(ctx, url, &transactions.TxRequest{Transaction: hash})
	plain, err := newRPCClient(url)
	if err != nil {
		t.Fatal(err)
	}
	_, plainErr := plain.Request(&transactions.TxRequest{Transaction: hash})
	if exportedErr == nil || plainErr == nil || exportedErr.Error() != plainErr.Error() {
		t.Fatalf("tx error = %v, plain client %v; want the same RPC error", exportedErr, plainErr)
	}

	// Exhausted 503 retries fail with xrpl-go's overload error.
	_, busyURL := newBusyRippled(t, 100)
	_, err = Request(ctx, busyURL, &server.StateRequest{})
	var clientErr *rpc.ClientError
	if !errors.As(err, &clientErr) || clientErr.ErrorString != "Server is overloaded, rate limit exceeded" {
		t.Fatalf("exhausted 503 error = %v, want xrpl-go's overload error", err)
	}
}

// TestExportedRPCRequiresContext: there is no way to call without a context,
// and an ended context sends nothing.
func TestExportedRPCRequiresContext(t *testing.T) {
	var nilCtx context.Context
	if _, err := NewContextClient(nilCtx, "http://127.0.0.1:1"); err == nil {
		t.Fatal("NewContextClient accepted a nil context")
	}
	if _, err := Request(nilCtx, "http://127.0.0.1:1", &server.StateRequest{}); err == nil {
		t.Fatal("Request accepted a nil context")
	}
	b, url := newBusyRippled(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Request(ctx, url, &server.StateRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ended context: %v, want context.Canceled", err)
	}
	if b.served.Load() != 0 {
		t.Fatalf("an ended context sent %d requests", b.served.Load())
	}
}

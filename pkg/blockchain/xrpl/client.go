package xrpl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Peersyst/xrpl-go/xrpl/common"
	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
)

const xrplNetworkIDRequiredAbove = 1024

// NetworkIDPolicy is the live rippled network-domain rule for transaction
// signing. Custom networks require an exact uint32 NetworkID; main/test/dev
// networks (IDs <= 1024) require the field to be absent.
type NetworkIDPolicy struct {
	NetworkID uint32
	Required  bool
}

func newRPCClient(rpcURL string) (*rpc.Client, error) {
	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	return rpc.NewClient(cfg), nil
}

// rpcHTTPClient sends every context-bound SDK XRPL request. Its Timeout keeps
// xrpl-go's default request timeout as the upper bound when the caller's ctx
// has no earlier deadline.
var rpcHTTPClient = &http.Client{Timeout: common.DefaultTimeout}

// rpcBusyRetries and rpcBusyBackoff mirror xrpl-go v0.1.18's HTTP 503 retry
// schedule (3 retries after 1s, 2s and 4s), which contextDoer now runs itself.
// rpcBusyBackoff is a variable only so tests can shorten it.
const rpcBusyRetries = 3

var rpcBusyBackoff = time.Second

// errRPCBusy is the error xrpl-go returns once its 503 retries are exhausted.
var errRPCBusy = &rpc.ClientError{ErrorString: "Server is overloaded, rate limit exceeded"}

// contextDoer is an rpc.HTTPClient that sends each request under ctx.
// xrpl-go's rpc.Client.Request derives its request context from
// context.Background, so without this a cancelled caller could not abort an
// in-flight request.
//
// contextDoer also retries HTTP 503 itself, on xrpl-go's schedule but with a
// ctx-aware backoff, and never hands xrpl-go a 503. xrpl-go's own retry loop
// sleeps without observing ctx and, when a retried request fails, dereferences
// a nil response in its deferred Body.Close and panics.
type contextDoer struct {
	ctx context.Context
}

func (d contextDoer) Do(req *http.Request) (*http.Response, error) {
	req = req.WithContext(d.ctx)
	resp, err := rpcHTTPClient.Do(req)
	backoff := rpcBusyBackoff
	for retry := 0; err == nil && resp.StatusCode == http.StatusServiceUnavailable; retry++ {
		_ = resp.Body.Close()
		if retry == rpcBusyRetries {
			return nil, errRPCBusy
		}
		timer := time.NewTimer(backoff)
		select {
		case <-d.ctx.Done():
			timer.Stop()
			return nil, d.ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
		if req, err = rewind(req); err != nil {
			return nil, err
		}
		resp, err = rpcHTTPClient.Do(req)
	}
	return resp, err
}

// rewind returns req with a fresh copy of its body for a retry.
func rewind(req *http.Request) (*http.Request, error) {
	if req.Body == nil || req.GetBody == nil {
		return req, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("xrpl: rewind request body: %w", err)
	}
	retry := req.Clone(req.Context())
	retry.Body = body
	return retry, nil
}

// networkIDCache is the NetworkID an SDK object learned from server_info,
// shared by its concurrent operations. Zero means unknown or not required.
type networkIDCache struct {
	mu sync.Mutex
	id uint32
}

func (c *networkIDCache) load() uint32 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.id
}

func (c *networkIDCache) store(id uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id = id
}

// rpcCall returns an xrpl-go client for one operation against rpcURL. Every
// HTTP request it sends, including those inside Autofill and HTTP 503
// retries, runs under ctx, so cancellation and deadlines abort the request in
// flight. Response decoding and errors stay xrpl-go's own; a cancelled request
// fails with an error wrapping ctx.Err(). The call starts from cache's
// NetworkID (see resolveNetworkIDPolicy); cache may be nil.
func rpcCall(ctx context.Context, rpcURL string, cache *networkIDCache) (*rpc.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg, err := rpc.NewClientConfig(rpcURL, rpc.WithHTTPClient(contextDoer{ctx: ctx}))
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	call := rpc.NewClient(cfg)
	call.NetworkID = cache.load()
	return call, nil
}

// NewContextClient returns an xrpl-go client for rpcURL whose every HTTP
// request runs under ctx: cancellation or a deadline aborts it with an error
// wrapping ctx.Err().Exhausted retries return xrpl-go's overload ClientError.
// Use one client per operation, within ctx's lifetime. ctx must be non-nil;
// an already-ended ctx returns its error.
func NewContextClient(ctx context.Context, rpcURL string) (*rpc.Client, error) {
	if ctx == nil {
		return nil, errors.New("xrpl: context is required")
	}
	return rpcCall(ctx, rpcURL, nil)
}

// Request sends one JSON-RPC request to rpcURL through NewContextClient.
func Request(ctx context.Context, rpcURL string, req rpc.XRPLRequest) (rpc.XRPLResponse, error) {
	call, err := NewContextClient(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return call.Request(req)
}

// ResolveNetworkIDPolicy reads the live network-domain policy through client
// and caches the NetworkID on it. Its request observes a caller context only
// if client does: pass a client from NewContextClient for a context-bound,
// 503-safe read, as the SDK finalizers use their own context-bound client.
func ResolveNetworkIDPolicy(client *rpc.Client) (NetworkIDPolicy, error) {
	policy, id, err := networkIDPolicy(client.NetworkID, client)
	client.NetworkID = id
	return policy, err
}

// resolveNetworkIDPolicy reads server_info through call unless cache already
// holds a NetworkID, and leaves the result on both cache and call so a later
// Autofill through call sees it. The cache lock is not held across the RPC;
// concurrent resolvers store the same live value.
func resolveNetworkIDPolicy(cache *networkIDCache, call *rpc.Client) (NetworkIDPolicy, error) {
	policy, id, err := networkIDPolicy(cache.load(), call)
	cache.store(id)
	call.NetworkID = id
	return policy, err
}

// networkIDPolicy derives the policy from cached, reading server_info through
// call only when cached is zero. It returns the NetworkID to cache.
func networkIDPolicy(cached uint32, call *rpc.Client) (NetworkIDPolicy, uint32, error) {
	if cached != 0 {
		if cached <= xrplNetworkIDRequiredAbove {
			return NetworkIDPolicy{}, 0, nil
		}
		return NetworkIDPolicy{NetworkID: cached, Required: true}, cached, nil
	}
	info, err := call.GetServerInfo(&server.InfoRequest{})
	if err != nil {
		return NetworkIDPolicy{}, cached, fmt.Errorf("xrpl: server_info: %w", err)
	}
	networkID := info.Info.NetworkID
	if networkID <= xrplNetworkIDRequiredAbove {
		return NetworkIDPolicy{}, 0, nil
	}
	if networkID > uint(^uint32(0)) {
		return NetworkIDPolicy{}, cached, fmt.Errorf("xrpl: network_id %d overflows uint32", networkID)
	}
	return NetworkIDPolicy{NetworkID: uint32(networkID), Required: true}, uint32(networkID), nil
}

func ensureNetworkID(client *rpc.Client) error {
	_, err := ResolveNetworkIDPolicy(client)
	return err
}

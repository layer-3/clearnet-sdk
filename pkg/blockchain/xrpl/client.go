package xrpl

import (
	"context"
	"fmt"
	"net/http"

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

// contextDoer is an rpc.HTTPClient that sends each request under ctx.
// xrpl-go's rpc.Client.Request derives its request context from
// context.Background, so without this a cancelled caller could not abort an
// in-flight request.
type contextDoer struct {
	ctx context.Context
}

func (d contextDoer) Do(req *http.Request) (*http.Response, error) {
	return rpcHTTPClient.Do(req.WithContext(d.ctx))
}

// rpcCall returns an xrpl-go client for one operation against rpcURL. Every
// HTTP request it sends, including those inside Autofill and xrpl-go's own
// retries, runs under ctx, so cancellation and deadlines abort the request in
// flight. Response decoding and errors stay xrpl-go's own; a cancelled request
// fails with an error wrapping ctx.Err(). state is the long-lived client whose
// cached NetworkID the call starts from (see resolveNetworkIDPolicy).
func rpcCall(ctx context.Context, rpcURL string, state *rpc.Client) (*rpc.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg, err := rpc.NewClientConfig(rpcURL, rpc.WithHTTPClient(contextDoer{ctx: ctx}))
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	call := rpc.NewClient(cfg)
	if state != nil {
		call.NetworkID = state.NetworkID
	}
	return call, nil
}

// ResolveNetworkIDPolicy reads the live network-domain policy through client
// and caches the NetworkID on it. Its request does not observe a caller
// context; the SDK finalizers resolve the policy through a context-bound
// client instead.
func ResolveNetworkIDPolicy(client *rpc.Client) (NetworkIDPolicy, error) {
	return resolveNetworkIDPolicy(client, client)
}

// resolveNetworkIDPolicy reads server_info through call unless state already
// caches a NetworkID, and leaves the result cached on both state and call so a
// later Autofill through call sees it.
func resolveNetworkIDPolicy(state, call *rpc.Client) (NetworkIDPolicy, error) {
	policy, err := networkIDPolicy(state, call)
	call.NetworkID = state.NetworkID
	return policy, err
}

func networkIDPolicy(state, call *rpc.Client) (NetworkIDPolicy, error) {
	if state.NetworkID != 0 {
		if state.NetworkID <= xrplNetworkIDRequiredAbove {
			state.NetworkID = 0
			return NetworkIDPolicy{}, nil
		}
		return NetworkIDPolicy{NetworkID: state.NetworkID, Required: true}, nil
	}
	info, err := call.GetServerInfo(&server.InfoRequest{})
	if err != nil {
		return NetworkIDPolicy{}, fmt.Errorf("xrpl: server_info: %w", err)
	}
	networkID := info.Info.NetworkID
	if networkID <= xrplNetworkIDRequiredAbove {
		state.NetworkID = 0
		return NetworkIDPolicy{}, nil
	}
	if networkID > uint(^uint32(0)) {
		return NetworkIDPolicy{}, fmt.Errorf("xrpl: network_id %d overflows uint32", networkID)
	}
	state.NetworkID = uint32(networkID)
	return NetworkIDPolicy{NetworkID: uint32(networkID), Required: true}, nil
}

func ensureNetworkID(client *rpc.Client) error {
	_, err := ResolveNetworkIDPolicy(client)
	return err
}

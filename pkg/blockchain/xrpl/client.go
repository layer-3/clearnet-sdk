package xrpl

import (
	"fmt"

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

func ResolveNetworkIDPolicy(client *rpc.Client) (NetworkIDPolicy, error) {
	if client.NetworkID != 0 {
		if client.NetworkID <= xrplNetworkIDRequiredAbove {
			client.NetworkID = 0
			return NetworkIDPolicy{}, nil
		}
		return NetworkIDPolicy{NetworkID: client.NetworkID, Required: true}, nil
	}
	info, err := client.GetServerInfo(&server.InfoRequest{})
	if err != nil {
		return NetworkIDPolicy{}, fmt.Errorf("xrpl: server_info: %w", err)
	}
	networkID := info.Info.NetworkID
	if networkID <= xrplNetworkIDRequiredAbove {
		client.NetworkID = 0
		return NetworkIDPolicy{}, nil
	}
	if networkID > uint(^uint32(0)) {
		return NetworkIDPolicy{}, fmt.Errorf("xrpl: network_id %d overflows uint32", networkID)
	}
	client.NetworkID = uint32(networkID)
	return NetworkIDPolicy{NetworkID: uint32(networkID), Required: true}, nil
}

func ensureNetworkID(client *rpc.Client) error {
	_, err := ResolveNetworkIDPolicy(client)
	return err
}

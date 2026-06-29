package xrpl

import (
	"fmt"

	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
)

const xrplNetworkIDRequiredAbove = 1024

func newRPCClient(rpcURL string) (*rpc.Client, error) {
	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	return rpc.NewClient(cfg), nil
}

func ensureNetworkID(client *rpc.Client) error {
	if client.NetworkID != 0 {
		return nil
	}
	info, err := client.GetServerInfo(&server.InfoRequest{})
	if err != nil {
		return fmt.Errorf("xrpl: server_info: %w", err)
	}
	networkID := info.Info.NetworkID
	if networkID <= xrplNetworkIDRequiredAbove {
		return nil
	}
	if networkID > uint(^uint32(0)) {
		return fmt.Errorf("xrpl: network_id %d overflows uint32", networkID)
	}
	client.NetworkID = uint32(networkID)
	return nil
}

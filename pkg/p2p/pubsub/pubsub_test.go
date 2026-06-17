package pubsub

import (
	"context"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// TestPubSub_FinalizedWithdrawal shows the generic toolset carrying a concrete
// payload: callers name only the value type (core.FinalizedWithdrawal) and
// constraint type inference supplies the *T pointer type to Publisher/Follower.
func TestPubSub_FinalizedWithdrawal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hPub := newHost(t)
	hSub := newHost(t)
	connect(t, hPub, hSub)

	follower, err := NewFollower[core.FinalizedWithdrawal](ctx, hSub, p2pproto.TopicWithdrawals, nil)
	if err != nil {
		t.Fatalf("NewFollower: %v", err)
	}
	defer follower.Close()

	got := make(chan *core.FinalizedWithdrawal, 1)
	follower.SetHandler(func(fw *core.FinalizedWithdrawal) { got <- fw })
	go follower.Run(ctx)

	pub, err := NewPublisher[core.FinalizedWithdrawal](ctx, hPub, p2pproto.TopicWithdrawals, nil)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	if err := pub.WaitForPeers(ctx, 1, 10*time.Second); err != nil {
		t.Fatalf("WaitForPeers: %v", err)
	}

	want := &core.FinalizedWithdrawal{EntryIndex: 7}
	want.WithdrawalID[0], want.WithdrawalID[31] = 0xF1, 0x7A

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(10 * time.Second)
	for {
		if err := pub.Publish(ctx, want); err != nil {
			t.Fatalf("publish: %v", err)
		}
		select {
		case fw := <-got:
			if fw.WithdrawalID != want.WithdrawalID || fw.EntryIndex != want.EntryIndex {
				t.Fatalf("delivered %+v, want %+v", fw.Header(), want.Header())
			}
			if m := follower.Metrics().Snapshot(); m.Delivered != 1 {
				t.Errorf("Delivered = %d, want 1", m.Delivered)
			}
			return
		case <-ticker.C:
			continue
		case <-deadline:
			t.Fatal("withdrawal never delivered")
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func newHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("libp2p new: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func connect(t *testing.T, from, to host.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := from.Connect(ctx, peer.AddrInfo{ID: to.ID(), Addrs: to.Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}
}

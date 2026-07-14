package receipt

import (
	"context"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// testHandler is a ReceiptHandler backed by per-test closures.
type testHandler struct {
	burn func(context.Context, *core.BurnReceipt) (p2pproto.ReceiptAck, error)
	mint func(context.Context, *core.MintReceipt) (p2pproto.ReceiptAck, error)
}

func (h testHandler) OnBurnReceipt(ctx context.Context, r *core.BurnReceipt) (p2pproto.ReceiptAck, error) {
	return h.burn(ctx, r)
}

func (h testHandler) OnMintReceipt(ctx context.Context, r *core.MintReceipt) (p2pproto.ReceiptAck, error) {
	return h.mint(ctx, r)
}

func TestReceipt_BurnRoundTrip(t *testing.T) {
	srv, cli := newPair(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got *core.BurnReceipt
	NewServer(testHandler{
		burn: func(_ context.Context, r *core.BurnReceipt) (p2pproto.ReceiptAck, error) {
			got = r
			return p2pproto.ReceiptAck{Accepted: true}, nil
		},
	}, nil).Register(srv)

	want := &core.BurnReceipt{Signatures: [][]byte{{0x1, 0x2}}}
	want.WithdrawalID[0] = 0xBE
	want.TxID = "tx/ef"

	ack, err := NewClient(cli, srv.ID(), nil).SendBurnReceipt(ctx, want)
	if err != nil {
		t.Fatalf("SendBurnReceipt: %v", err)
	}
	if !ack.Accepted {
		t.Fatalf("ack not accepted: %+v", ack)
	}
	if got == nil || got.WithdrawalID != want.WithdrawalID || got.TxID != want.TxID {
		t.Fatalf("server received %+v, want %+v", got, want)
	}
}

func TestReceipt_MintRejected(t *testing.T) {
	srv, cli := newPair(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	NewServer(testHandler{
		mint: func(_ context.Context, _ *core.MintReceipt) (p2pproto.ReceiptAck, error) {
			return p2pproto.ReceiptAck{Accepted: false, Reason: "duplicate"}, nil
		},
	}, nil).Register(srv)

	ack, err := NewClient(cli, srv.ID(), nil).SendMintReceipt(ctx, &core.MintReceipt{
		TxID: "tx/1", Account: "yellow://x", AssetURI: "yellow://ynet/asset/custody/evm/1/0x0", Amount: decimal.NewFromInt(5),
	})
	if err != nil {
		t.Fatalf("SendMintReceipt: %v", err)
	}
	if ack.Accepted || ack.Reason != "duplicate" {
		t.Fatalf("ack = %+v, want rejected/duplicate", ack)
	}
}

func TestReceipt_HandlerError(t *testing.T) {
	srv, cli := newPair(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	NewServer(testHandler{
		burn: func(_ context.Context, _ *core.BurnReceipt) (p2pproto.ReceiptAck, error) {
			return p2pproto.ReceiptAck{}, context.Canceled // any error
		},
	}, nil).Register(srv)

	ack, err := NewClient(cli, srv.ID(), nil).SendBurnReceipt(ctx, &core.BurnReceipt{})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if ack.Accepted || ack.Reason == "" {
		t.Fatalf("expected Accepted=false with a reason, got %+v", ack)
	}
}

// TestReceipt_HandleBurnReceiptDirect wires a single Handle* method (not via
// Register) to show it is independently registrable — the unit any consumer
// can mount under its own protocol ID or test in isolation.
func TestReceipt_HandleBurnReceiptDirect(t *testing.T) {
	srv, cli := newPair(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	called := make(chan struct{}, 1)
	s := NewServer(testHandler{
		burn: func(_ context.Context, _ *core.BurnReceipt) (p2pproto.ReceiptAck, error) {
			called <- struct{}{}
			return p2pproto.ReceiptAck{Accepted: true}, nil
		},
	}, nil)
	srv.SetStreamHandler(protocol.ID(p2pproto.ProtocolBurnReceipt), s.HandleBurnReceipt)

	if _, err := NewClient(cli, srv.ID(), nil).SendBurnReceipt(ctx, &core.BurnReceipt{}); err != nil {
		t.Fatalf("SendBurnReceipt: %v", err)
	}
	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("HandleBurnReceipt never invoked")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func newPair(t *testing.T) (srv, cli host.Host) {
	t.Helper()
	srv = newHost(t)
	cli = newHost(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Connect(ctx, peer.AddrInfo{ID: srv.ID(), Addrs: srv.Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}
	return srv, cli
}

func newHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("libp2p new: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

package receipt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

const defaultTimeout = 15 * time.Second

// Client submits burn/mint receipts to a single peer over a caller-owned
// host.Host. It is a stateless convenience: the caller is responsible for
// making peerID reachable (adding it to the peerstore and/or dialing) before
// the first send.
type Client struct {
	host    host.Host
	peerID  peer.ID
	timeout time.Duration
	logger  log.Logger
}

// NewClient creates a Client that submits to peerID over h.
func NewClient(h host.Host, peerID peer.ID, logger log.Logger) *Client {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	return &Client{
		host:    h,
		peerID:  peerID,
		timeout: defaultTimeout,
		logger:  logger.WithName("p2p-receipt-client"),
	}
}

// SendBurnReceipt writes r on /ynp/burnreceipt/1.0.0 and returns the ack.
// Transport-level errors (timeout, stream, decode) come back as a non-nil
// error; an Accepted=false ack returns without error so the caller decides
// whether to retry.
func (c *Client) SendBurnReceipt(ctx context.Context, r *core.BurnReceipt) (p2pproto.ReceiptAck, error) {
	return c.submit(ctx, p2pproto.ProtocolBurnReceipt, r)
}

// SendMintReceipt writes r on /ynp/mintreceipt/1.0.0. Same error semantics as
// SendBurnReceipt.
func (c *Client) SendMintReceipt(ctx context.Context, r *core.MintReceipt) (p2pproto.ReceiptAck, error) {
	return c.submit(ctx, p2pproto.ProtocolMintReceipt, r)
}

// submit is the shared transport path for both receipt kinds.
func (c *Client) submit(ctx context.Context, proto string, body cbg.CBORMarshaler) (p2pproto.ReceiptAck, error) {
	if c.peerID == "" {
		return p2pproto.ReceiptAck{}, fmt.Errorf("receipt: no peer configured")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	s, err := c.host.NewStream(ctx, c.peerID, protocol.ID(proto))
	if err != nil {
		return p2pproto.ReceiptAck{}, fmt.Errorf("open stream %s: %w", proto, err)
	}
	defer s.Close()

	var reqBuf bytes.Buffer
	if err := cborx.WriteFrame(&reqBuf, cborx.V1, body); err != nil {
		return p2pproto.ReceiptAck{}, fmt.Errorf("encode receipt: %w", err)
	}
	if _, err := s.Write(reqBuf.Bytes()); err != nil {
		return p2pproto.ReceiptAck{}, fmt.Errorf("write receipt: %w", err)
	}
	if err := s.CloseWrite(); err != nil {
		return p2pproto.ReceiptAck{}, fmt.Errorf("close write: %w", err)
	}

	var ack p2pproto.ReceiptAck
	var v cborx.Version
	if err := cborx.ReadFrame(io.LimitReader(s, maxReceiptBytes), cborx.MaxControlFrame, &v, &ack); err != nil {
		return p2pproto.ReceiptAck{}, fmt.Errorf("decode ack: %w", err)
	}
	return ack, nil
}

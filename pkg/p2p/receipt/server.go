// Package receipt implements the libp2p request/response stream protocols for
// burn/mint receipt submission. A client writes a cborx-framed receipt; the
// Server decodes it, hands it to a ReceiptHandler, and writes back a
// protocol.ReceiptAck.
//
// Two protocols share one shape, differing only in protocol ID and payload:
//
//	/ynp/burnreceipt/1.0.0  — request *core.BurnReceipt, response *protocol.ReceiptAck
//	/ynp/mintreceipt/1.0.0  — request *core.MintReceipt, response *protocol.ReceiptAck
//
// The package never builds a host.Host: Register installs handlers on a
// caller-supplied host (Server satisfies protocol.Registrar), and Client dials
// over one.
package receipt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// Per-stream guards. A CBOR receipt is hundreds of bytes; the cap stops a hung
// or hostile peer from streaming the server out of memory, and the deadline
// bounds a single request end to end.
const (
	maxReceiptBytes    = 64 * 1024
	streamReadDeadline = 10 * time.Second
)

// Server handles inbound burn/mint receipt streams, delegating each decoded
// receipt to a ReceiptHandler.
type Server struct {
	handler ReceiptHandler
	logger  log.Logger
}

var _ p2pproto.Registrar = (*Server)(nil)

// NewServer returns a Server that delegates to handler.
func NewServer(handler ReceiptHandler, logger log.Logger) *Server {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	return &Server{handler: handler, logger: logger.WithName("p2p-receipt-server")}
}

// Register installs both receipt stream handlers on h.
func (s *Server) Register(h host.Host) {
	h.SetStreamHandler(protocol.ID(p2pproto.ProtocolBurnReceipt), s.HandleBurnReceipt)
	h.SetStreamHandler(protocol.ID(p2pproto.ProtocolMintReceipt), s.HandleMintReceipt)
}

// HandleBurnReceipt is the stream handler for /ynp/burnreceipt/1.0.0.
func (s *Server) HandleBurnReceipt(stream network.Stream) {
	s.serve(stream, p2pproto.ProtocolBurnReceipt, func(ctx context.Context, r io.Reader) (p2pproto.ReceiptAck, error) {
		var receipt core.BurnReceipt
		var v cborx.Version
		if err := cborx.ReadFrame(r, cborx.MaxControlFrame, &v, &receipt); err != nil {
			return p2pproto.ReceiptAck{}, fmt.Errorf("decode: %w", err)
		}
		return s.handler.OnBurnReceipt(ctx, &receipt)
	})
}

// HandleMintReceipt is the stream handler for /ynp/mintreceipt/1.0.0.
func (s *Server) HandleMintReceipt(stream network.Stream) {
	s.serve(stream, p2pproto.ProtocolMintReceipt, func(ctx context.Context, r io.Reader) (p2pproto.ReceiptAck, error) {
		var receipt core.MintReceipt
		var v cborx.Version
		if err := cborx.ReadFrame(r, cborx.MaxControlFrame, &v, &receipt); err != nil {
			return p2pproto.ReceiptAck{}, fmt.Errorf("decode: %w", err)
		}
		return s.handler.OnMintReceipt(ctx, &receipt)
	})
}

// serve runs one deadline-bounded request: decode via dispatch, write the ack.
// The per-call context is derived from Background and bounded by the same
// deadline as the stream read — the Server holds no context of its own.
func (s *Server) serve(
	stream network.Stream,
	proto string,
	dispatch func(context.Context, io.Reader) (p2pproto.ReceiptAck, error),
) {
	defer stream.Close()
	lg := s.logger.WithKV("protocol", proto)

	ctx, cancel := context.WithTimeout(context.Background(), streamReadDeadline)
	defer cancel()
	if err := stream.SetReadDeadline(time.Now().Add(streamReadDeadline)); err != nil {
		lg.Warn("set read deadline failed", "error", err)
		return
	}

	ack, err := dispatch(ctx, io.LimitReader(stream, maxReceiptBytes))
	if err != nil {
		lg.Warn("handler error", "error", err)
		writeAck(stream, p2pproto.ReceiptAck{Accepted: false, Reason: err.Error()}, lg)
		return
	}
	writeAck(stream, ack, lg)
}

func writeAck(stream network.Stream, ack p2pproto.ReceiptAck, logger log.Logger) {
	var buf bytes.Buffer
	if err := cborx.WriteFrame(&buf, cborx.V1, &ack); err != nil {
		logger.Warn("encode ack failed", "error", err)
		return
	}
	if _, err := stream.Write(buf.Bytes()); err != nil {
		logger.Warn("write ack failed", "error", err)
	}
}

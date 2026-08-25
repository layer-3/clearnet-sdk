package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// Server handles inbound auth streams: it issues a nonce, verifies the response
// as operator or passive, and reports each success via an onAuth callback.
type Server struct {
	signers             core.ReceiptSignerSource
	trustedIssuerFilter TrustedIssuerFilter
	onAuth              func(network.Conn, Result)
	logger              log.Logger
}

var _ p2pproto.Registrar = (*Server)(nil)

// TrustedIssuerFilter authorizes an issuer id before operator signer lookup.
// A nil filter allows every syntactically valid issuer id.
type TrustedIssuerFilter func(ctx context.Context, issuerID common.Address) error

// ServerOption configures auth server behavior.
type ServerOption func(*Server)

// WithTrustedIssuerFilter installs an issuer authorization hook for operator
// auth. Passing nil is a no-op.
func WithTrustedIssuerFilter(filter TrustedIssuerFilter) ServerOption {
	return func(s *Server) {
		if filter != nil {
			s.trustedIssuerFilter = filter
		}
	}
}

// NewServer returns a Server gated by the current issuer receipt signer source.
// Passing a signer source enables operator auth and disables passive auth; a
// nil signer source enables passive auth and disables operator auth. If no
// trusted issuer filter is configured, operator auth allows any syntactically
// valid issuer id before loading its signer set. onAuth, if non-nil, is invoked
// with the connection and Result after each successful handshake.
func NewServer(signers core.ReceiptSignerSource, onAuth func(network.Conn, Result), logger log.Logger, opts ...ServerOption) *Server {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	lg := logger.WithName("p2p-auth-server").WithKV("protocol", p2pproto.ProtocolAuth)
	s := &Server{signers: signers, onAuth: onAuth, logger: lg}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Register installs the auth stream handler on h.
func (s *Server) Register(h host.Host) {
	h.SetStreamHandler(protocol.ID(p2pproto.ProtocolAuth), s.HandleAuth)
}

// handshakeTimeout bounds one server-side handshake end to end (challenge write
// + response read), so a stalled peer cannot pin the handler goroutine.
const handshakeTimeout = 10 * time.Second

// HandleAuth is the stream handler for /ynp/auth/1.0.0.
func (s *Server) HandleAuth(stream network.Stream) {
	defer stream.Close()
	conn := stream.Conn()
	deadline := time.Now().Add(handshakeTimeout)
	// Bound the whole handshake: the server writes the challenge then reads the
	// response, so the deadline covers both directions (slowloris guard).
	if err := stream.SetDeadline(deadline); err != nil {
		s.logger.Debug("auth set deadline failed", "peer", conn.RemotePeer().ShortString(), "error", err)
		return
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	res, err := s.verify(ctx, stream, conn.RemotePublicKey())
	if err != nil {
		s.logger.Debug("auth handshake failed", "peer", conn.RemotePeer().ShortString(), "error", err)
		return
	}
	s.logger.Info("peer authenticated", "peer", conn.RemotePeer().ShortString(), "address", res.Address, "role", res.Role.String())
	if s.onAuth != nil {
		s.onAuth(conn, res)
	}
}

// verify runs the server side of one handshake on stream: generate a nonce,
// send the challenge, read the response, verify it as operator or passive.
// remotePub is the connection's remote libp2p key, used for passive auth.
func (s *Server) verify(ctx context.Context, stream network.Stream, remotePub libp2pcrypto.PubKey) (Result, error) {
	var challenge p2pproto.AuthChallenge
	if _, err := rand.Read(challenge.Nonce[:]); err != nil {
		return Result{}, fmt.Errorf("generate nonce: %w", err)
	}
	if err := cborx.WriteEnvelope(stream, cborx.V1, &challenge); err != nil {
		return Result{}, fmt.Errorf("send challenge: %w", err)
	}

	var resp p2pproto.AuthResponse
	var v cborx.Version
	if err := cborx.ReadEnvelope(io.LimitReader(stream, maxAuthEnvelope), &v, &resp); err != nil {
		return Result{}, fmt.Errorf("read response: %w", err)
	}
	if v != cborx.V1 {
		return Result{}, fmt.Errorf("unsupported auth wire version: 0x%02x", byte(v))
	}

	// Empty Address ⇒ passive auth proven against the libp2p identity key.
	if resp.Address == "" {
		if s.signers != nil {
			return Result{}, fmt.Errorf("passive auth disabled when signer source is configured")
		}
		if err := verifyPassive(remotePub, challenge.Nonce, resp.Signature); err != nil {
			return Result{}, err
		}
		return Result{Role: RolePassive}, nil
	}

	// Operator auth: recover the signer of keccak256(nonce) and gate it.
	if len(resp.Signature) != 65 {
		return Result{}, fmt.Errorf("operator signature must be 65 bytes, got %d", len(resp.Signature))
	}
	if s.signers == nil {
		return Result{}, fmt.Errorf("operator auth requires signer source")
	}
	if !common.IsHexAddress(resp.IssuerID) {
		return Result{}, fmt.Errorf("operator auth requires issuer id")
	}
	issuerID := common.HexToAddress(resp.IssuerID)
	if s.trustedIssuerFilter != nil {
		if err := s.trustedIssuerFilter(ctx, issuerID); err != nil {
			return Result{}, fmt.Errorf("issuer %s not trusted: %w", issuerID.Hex(), err)
		}
	}
	nonceHash := operatorAuthDigest(challenge.Nonce, issuerID)
	pub, err := ethcrypto.SigToPub(nonceHash, resp.Signature)
	if err != nil {
		return Result{}, fmt.Errorf("ecrecover: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pub)
	if !strings.EqualFold(resp.Address, recovered.Hex()) {
		return Result{}, fmt.Errorf("address mismatch: recovered %s, claimed %s", recovered.Hex(), resp.Address)
	}
	set, err := s.signers.LoadReceiptSigners(ctx, issuerID)
	if err != nil {
		return Result{}, fmt.Errorf("load receipt signers: %w", err)
	}
	if !receiptSignerSetContains(set, recovered) {
		return Result{}, fmt.Errorf("operator %s not in issuer %s signer set", recovered.Hex(), issuerID.Hex())
	}
	return Result{Address: recovered.Hex(), IssuerID: issuerID.Hex(), Role: RoleOperator}, nil
}

func operatorAuthDigest(nonce [32]byte, issuerID common.Address) []byte {
	msg := make([]byte, 0, len(operatorAuthDomain)+len(nonce)+len(issuerID))
	msg = append(msg, operatorAuthDomain...)
	msg = append(msg, nonce[:]...)
	msg = append(msg, issuerID[:]...)
	return ethcrypto.Keccak256(msg)
}

func receiptSignerSetContains(set core.ReceiptSignerSet, addr common.Address) bool {
	for _, s := range set.Signers {
		if s == addr {
			return true
		}
	}
	return false
}

func verifyPassive(pub libp2pcrypto.PubKey, nonce [32]byte, sig []byte) error {
	if pub == nil {
		return fmt.Errorf("missing remote libp2p public key")
	}
	ok, err := pub.Verify(passiveAuthMessage(nonce), sig)
	if err != nil {
		return fmt.Errorf("verify passive auth: %w", err)
	}
	if !ok {
		return fmt.Errorf("passive auth signature mismatch")
	}
	return nil
}

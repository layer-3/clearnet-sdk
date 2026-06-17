package auth

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

// AllowList is the operator allow-list: a set of lowercased hex addresses. A
// nil or empty AllowList disables the operator gate — any well-formed operator
// signature is accepted (useful for early devnet). Passive auth is never gated
// by the allow-list.
type AllowList map[string]struct{}

// ParseAllowListCSV turns "0xabc..,0xdef.." into an AllowList. Empty input
// yields an empty set (gate disabled). Malformed entries are skipped.
func ParseAllowListCSV(s string) AllowList {
	out := AllowList{}
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" || !common.IsHexAddress(raw) {
			continue
		}
		out[strings.ToLower(common.HexToAddress(raw).Hex())] = struct{}{}
	}
	return out
}

func (a AllowList) normalize() AllowList {
	out := make(AllowList, len(a))
	for k := range a {
		if !common.IsHexAddress(k) {
			continue
		}
		out[strings.ToLower(common.HexToAddress(k).Hex())] = struct{}{}
	}
	return out
}

func (a AllowList) permits(addr common.Address) bool {
	if len(a) == 0 {
		return true
	}
	_, ok := a[strings.ToLower(addr.Hex())]
	return ok
}

// Server handles inbound auth streams: it issues a nonce, verifies the response
// as operator or passive, and reports each success via an onAuth callback.
type Server struct {
	allow  AllowList // normalized
	onAuth func(network.Conn, Result)
	logger *slog.Logger
}

var _ p2pproto.Registrar = (*Server)(nil)

// NewServer returns a Server gated by allow (nil/empty disables the operator
// gate). onAuth, if non-nil, is invoked with the connection and Result after
// each successful handshake — the caller binds whatever "authenticated" means
// in its world (e.g. marking the connection so receipt streams pass their
// gate). The connection is passed (not just the peer ID) so the caller can key
// auth state per-connection, matching libp2p's connection lifetime.
func NewServer(allow AllowList, onAuth func(network.Conn, Result), logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	log := logger.With("component", "p2p-auth-server", "protocol", p2pproto.ProtocolAuth)
	clean := allow.normalize()
	if len(allow) > 0 && len(clean) == 0 {
		log.Error("auth: every allow-list entry is malformed; gate is EMPTY and bypassed", "raw_entries", len(allow))
	} else if len(allow) > len(clean) {
		log.Warn("auth: dropped malformed allow-list entries", "raw", len(allow), "accepted", len(clean))
	}
	return &Server{allow: clean, onAuth: onAuth, logger: log}
}

// Register installs the auth stream handler on h.
func (s *Server) Register(h host.Host) {
	h.SetStreamHandler(protocol.ID(p2pproto.ProtocolAuth), s.HandleAuth)
}

// HandleAuth is the stream handler for /ynp/auth/1.0.0.
func (s *Server) HandleAuth(stream network.Stream) {
	defer stream.Close()
	conn := stream.Conn()
	res, err := s.verify(stream, conn.RemotePublicKey())
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
func (s *Server) verify(stream network.Stream, remotePub libp2pcrypto.PubKey) (Result, error) {
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
		if err := verifyPassive(remotePub, challenge.Nonce, resp.Signature); err != nil {
			return Result{}, err
		}
		return Result{Role: RolePassive}, nil
	}

	// Operator auth: recover the signer of keccak256(nonce) and gate it.
	if len(resp.Signature) != 65 {
		return Result{}, fmt.Errorf("operator signature must be 65 bytes, got %d", len(resp.Signature))
	}
	nonceHash := ethcrypto.Keccak256(challenge.Nonce[:])
	pub, err := ethcrypto.SigToPub(nonceHash, resp.Signature)
	if err != nil {
		return Result{}, fmt.Errorf("ecrecover: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pub)
	if !strings.EqualFold(resp.Address, recovered.Hex()) {
		return Result{}, fmt.Errorf("address mismatch: recovered %s, claimed %s", recovered.Hex(), resp.Address)
	}
	if !s.allow.permits(recovered) {
		return Result{}, fmt.Errorf("operator %s not in allow-list", recovered.Hex())
	}
	return Result{Address: recovered.Hex(), Role: RoleOperator}, nil
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

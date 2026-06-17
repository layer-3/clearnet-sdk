package auth

import (
	"context"
	"fmt"
	"io"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// clientTimeout bounds a single Authenticate attempt.
const clientTimeout = 10 * time.Second

// ClientOpts selects the role for Authenticate. Exactly one of Signer
// (operator) or IdentityKey (passive) must be set; Signer takes precedence if
// both are present.
type ClientOpts struct {
	// Signer is a secp256k1 operator key. When set, runs operator auth.
	Signer sign.Signer
	// IdentityKey is the libp2p identity private key. Used for passive auth
	// when Signer is nil.
	IdentityKey libp2pcrypto.PrivKey
}

// Client runs the dialing side of the auth handshake. It mirrors Server:
// Server.HandleAuth verifies what Client.Authenticate produces.
type Client struct {
	opts ClientOpts
}

// NewClient returns a Client configured by opts.
func NewClient(opts ClientOpts) *Client {
	return &Client{opts: opts}
}

// Authenticate runs the handshake against pid over h. With opts.Signer it
// performs operator auth; otherwise passive auth with opts.IdentityKey. The
// remote's HandleAuth marks us authenticated on success.
func (c *Client) Authenticate(ctx context.Context, h host.Host, pid peer.ID) error {
	if pid == "" {
		return fmt.Errorf("auth: empty peer id")
	}
	if c.opts.Signer == nil && c.opts.IdentityKey == nil {
		return fmt.Errorf("auth: ClientOpts needs a Signer or an IdentityKey")
	}

	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()

	s, err := h.NewStream(ctx, pid, protocol.ID(p2pproto.ProtocolAuth))
	if err != nil {
		return fmt.Errorf("open auth stream to %s: %w", pid.ShortString(), err)
	}
	defer s.Close()

	if c.opts.Signer != nil {
		return c.requestOperator(ctx, s)
	}
	return c.requestPassive(s)
}

func (c *Client) requestOperator(ctx context.Context, s network.Stream) error {
	addr, err := sign.EthAddress(c.opts.Signer)
	if err != nil {
		return fmt.Errorf("operator address: %w", err)
	}
	return respond(s, func(nonce [32]byte) (p2pproto.AuthResponse, error) {
		nonceHash := ethcrypto.Keccak256(nonce[:])
		sig, err := sign.SignEthDigest(ctx, c.opts.Signer, nonceHash, addr)
		if err != nil {
			return p2pproto.AuthResponse{}, fmt.Errorf("sign nonce: %w", err)
		}
		return p2pproto.AuthResponse{Signature: sig, Address: addr.Hex()}, nil
	})
}

func (c *Client) requestPassive(s network.Stream) error {
	return respond(s, func(nonce [32]byte) (p2pproto.AuthResponse, error) {
		sig, err := c.opts.IdentityKey.Sign(passiveAuthMessage(nonce))
		if err != nil {
			return p2pproto.AuthResponse{}, fmt.Errorf("sign passive nonce: %w", err)
		}
		return p2pproto.AuthResponse{Signature: sig}, nil
	})
}

// respond reads the challenge, builds the response, and writes it back.
func respond(s network.Stream, build func([32]byte) (p2pproto.AuthResponse, error)) error {
	var challenge p2pproto.AuthChallenge
	var v cborx.Version
	if err := cborx.ReadEnvelope(io.LimitReader(s, maxAuthEnvelope), &v, &challenge); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}
	if v != cborx.V1 {
		return fmt.Errorf("unsupported auth wire version: 0x%02x", byte(v))
	}
	resp, err := build(challenge.Nonce)
	if err != nil {
		return err
	}
	if err := cborx.WriteEnvelope(s, cborx.V1, &resp); err != nil {
		return fmt.Errorf("send response: %w", err)
	}
	return nil
}

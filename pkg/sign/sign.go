// Package sign abstracts signing identities behind a pluggable, algorithm-aware
// interface — raw in-memory keys for clients/CLI/tests, KMS backends in
// production (custody's AWS/GCP signers satisfy this interface unchanged). The
// blockchain adapters take a Signer at construction and apply the chain-specific
// framing (EVM keccak + recovery id, BTC DER + sighash byte, XRPL
// encode-for-multisigning) themselves.
package sign

import (
	"context"
	"errors"
)

// Algorithm names the curve/scheme. The Sign output format is
// algorithm-specific:
//   - AlgSecp256k1: DER ECDSA, low-S normalized; caller passes a 32-byte digest.
//   - AlgEd25519:   raw 64-byte signature; caller passes the message.
type Algorithm string

const (
	AlgSecp256k1 Algorithm = "secp256k1"
	AlgEd25519   Algorithm = "ed25519"
)

// ErrUnsupportedAlgorithm is returned for an algorithm a backend can't handle.
var ErrUnsupportedAlgorithm = errors.New("sign: unsupported algorithm")

// Signer is implemented by every signing backend. Implementations are safe for
// concurrent Sign calls; construct once, share, Close at shutdown.
type Signer interface {
	Algorithm() Algorithm

	// PublicKey returns the raw public key bytes:
	//   - secp256k1: 33-byte compressed SEC1 (0x02/0x03 || X)
	//   - ed25519:   32-byte raw
	PublicKey() []byte

	Sign(ctx context.Context, message []byte) ([]byte, error)
	Close() error
}

package sign

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	decred_secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	decred_ecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// KeySigner is a Signer backed by a raw private key held in memory — for
// clients, the CLI, and tests. Production prefers a KMS-backed Signer.
type KeySigner struct {
	algorithm  Algorithm
	ecdsaKey   *ecdsa.PrivateKey
	ed25519Key ed25519.PrivateKey
	publicKey  []byte
}

// NewKeySignerFromECDSA wraps a secp256k1 private key.
func NewKeySignerFromECDSA(key *ecdsa.PrivateKey) *KeySigner {
	return &KeySigner{
		algorithm: AlgSecp256k1,
		ecdsaKey:  key,
		publicKey: crypto.CompressPubkey(&key.PublicKey),
	}
}

// NewKeySignerFromEd25519 wraps an ed25519 private key.
func NewKeySignerFromEd25519(key ed25519.PrivateKey) (*KeySigner, error) {
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ed25519 private key has wrong length %d (expected %d)", len(key), ed25519.PrivateKeySize)
	}
	pub := make([]byte, ed25519.PublicKeySize)
	copy(pub, key.Public().(ed25519.PublicKey))
	return &KeySigner{
		algorithm:  AlgEd25519,
		ed25519Key: key,
		publicKey:  pub,
	}, nil
}

func (s *KeySigner) Algorithm() Algorithm { return s.algorithm }

func (s *KeySigner) PublicKey() []byte {
	out := make([]byte, len(s.publicKey))
	copy(out, s.publicKey)
	return out
}

func (s *KeySigner) Sign(_ context.Context, message []byte) ([]byte, error) {
	switch s.algorithm {
	case AlgSecp256k1:
		if len(message) != 32 {
			return nil, fmt.Errorf("secp256k1 sign expects 32-byte digest, got %d", len(message))
		}
		// decred's Sign uses RFC6979 deterministic ECDSA and Serialize emits
		// canonical low-S DER — matches the KMS output format.
		priv := decred_secp.PrivKeyFromBytes(crypto.FromECDSA(s.ecdsaKey))
		return decred_ecdsa.Sign(priv, message).Serialize(), nil

	case AlgEd25519:
		return ed25519.Sign(s.ed25519Key, message), nil

	default:
		return nil, ErrUnsupportedAlgorithm
	}
}

func (s *KeySigner) Close() error { return nil }

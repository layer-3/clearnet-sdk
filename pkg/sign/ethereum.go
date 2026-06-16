package sign

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	decred_ecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// EthAddress derives the Ethereum address from a secp256k1 Signer.
func EthAddress(s Signer) (common.Address, error) {
	if s.Algorithm() != AlgSecp256k1 {
		return common.Address{}, fmt.Errorf("sign: EthAddress requires secp256k1, got %s", s.Algorithm())
	}
	pk, err := crypto.DecompressPubkey(s.PublicKey())
	if err != nil {
		return common.Address{}, fmt.Errorf("sign: decompress pubkey: %w", err)
	}
	return crypto.PubkeyToAddress(*pk), nil
}

// SignEthDigest signs a 32-byte digest with a secp256k1 Signer and returns the
// 65-byte Ethereum form R || S || V with V ∈ {0,1} (what crypto.Sign produces
// and crypto.SigToPub expects). Callers that need Solidity's V ∈ {27,28} shift
// at the contract boundary. expectedAddr disambiguates the recovery id.
func SignEthDigest(ctx context.Context, s Signer, digest []byte, expectedAddr common.Address) ([]byte, error) {
	if len(digest) != 32 {
		return nil, fmt.Errorf("sign: SignEthDigest requires 32-byte digest, got %d", len(digest))
	}
	der, err := s.Sign(ctx, digest)
	if err != nil {
		return nil, err
	}
	r, ss, err := derSigRS(der)
	if err != nil {
		return nil, fmt.Errorf("sign: parse DER: %w", err)
	}

	sig := make([]byte, 65)
	copy(sig[:32], r[:])
	copy(sig[32:64], ss[:])

	for v := byte(0); v < 2; v++ {
		sig[64] = v
		recovered, err := crypto.SigToPub(digest, sig)
		if err != nil {
			continue
		}
		if crypto.PubkeyToAddress(*recovered) == expectedAddr {
			return sig, nil
		}
	}
	return nil, errors.New("sign: could not recover V from Ethereum signature")
}

// normalizeLowSDER re-encodes a DER ECDSA signature with S in the lower half of
// the curve order. Required by EVM (EIP-2) and XRPL (canonical signatures); KMS
// backends may return high-S.
func normalizeLowSDER(der []byte) ([]byte, error) {
	sig, err := decred_ecdsa.ParseDERSignature(der)
	if err != nil {
		return nil, fmt.Errorf("parse DER: %w", err)
	}
	return sig.Serialize(), nil
}

// derSigRS extracts R, S as 32-byte big-endian arrays from a DER ECDSA
// signature, normalizing S to the lower half-order in the process.
func derSigRS(der []byte) (r, s [32]byte, err error) {
	canonical, err := normalizeLowSDER(der)
	if err != nil {
		return r, s, err
	}
	if len(canonical) < 8 || canonical[0] != 0x30 {
		return r, s, errors.New("DER: bad SEQUENCE header")
	}
	rest := canonical[2:]
	rBytes, rest, err := parseDERInt(rest)
	if err != nil {
		return r, s, err
	}
	sBytes, _, err := parseDERInt(rest)
	if err != nil {
		return r, s, err
	}
	copy(r[32-len(rBytes):], rBytes)
	copy(s[32-len(sBytes):], sBytes)
	return r, s, nil
}

func parseDERInt(b []byte) (val, rest []byte, err error) {
	if len(b) < 2 || b[0] != 0x02 {
		return nil, nil, errors.New("DER: bad INTEGER tag")
	}
	n := int(b[1])
	if n+2 > len(b) {
		return nil, nil, errors.New("DER: integer length overflow")
	}
	v := b[2 : 2+n]
	if len(v) > 0 && v[0] == 0x00 {
		v = v[1:]
	}
	return v, b[2+n:], nil
}

package bls

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
)

// BN254 field prime P.
var fieldP, _ = new(big.Int).SetString("21888242871839275222246405745257275088696311157297823662689037894645226208583", 10)

// (P + 1) / 4 — exponent for modular square root when P ≡ 3 mod 4.
var sqrtExp = new(big.Int).Rsh(new(big.Int).Add(fieldP, big.NewInt(1)), 2)

// KeyPair holds a BLS key pair on BN254.
type KeyPair struct {
	Secret   fr.Element
	PublicG1 bn254.G1Affine
	PublicG2 bn254.G2Affine
}

// GenerateKeyPair creates a random BLS key pair.
func GenerateKeyPair() (*KeyPair, error) {
	var sk fr.Element
	_, err := sk.SetRandom()
	if err != nil {
		return nil, err
	}
	return keyPairFromScalar(&sk), nil
}

// KeyPairFromSeed derives a deterministic key pair from a seed (for tests).
func KeyPairFromSeed(seed []byte) *KeyPair {
	// Hash seed to get a scalar.
	h := crypto.Keccak256(seed)
	var sk fr.Element
	sk.SetBytes(h)
	return keyPairFromScalar(&sk)
}

// MarshalHex serializes the secret scalar as a 64-character hex string
// (pure, in-memory). File persistence belongs to the caller — see
// `cmd/clearnode` and `client/chain.go` for production I/O.
func (kp *KeyPair) MarshalHex() string {
	var skBytes [32]byte
	sk := kp.Secret.Bytes()
	copy(skBytes[:], sk[:])
	return hex.EncodeToString(skBytes[:])
}

// UnmarshalHexKeyPair parses a hex-encoded 32-byte secret scalar and
// reconstructs the full key pair.
func UnmarshalHexKeyPair(hexStr string) (*KeyPair, error) {
	trimmed := strings.TrimSpace(hexStr)
	skBytes, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, err
	}
	var sk fr.Element
	sk.SetBytes(skBytes)
	return keyPairFromScalar(&sk), nil
}

func keyPairFromScalar(sk *fr.Element) *KeyPair {
	kp := &KeyPair{Secret: *sk}

	// pk_g1 = sk * G1_generator
	var skBigInt big.Int
	sk.BigInt(&skBigInt)

	var g1Gen bn254.G1Affine
	g1Gen.X.SetOne()
	g1Gen.Y.SetUint64(2)

	kp.PublicG1.ScalarMultiplication(&g1Gen, &skBigInt)

	// pk_g2 = sk * G2_generator
	_, _, _, g2Gen := bn254.Generators()
	kp.PublicG2.ScalarMultiplication(&g2Gen, &skBigInt)

	return kp
}

// HashToG1 hashes a 32-byte message to a BN254 G1 point using try-and-increment.
// This MUST match the Solidity BLS.hashToG1 exactly.
func HashToG1(msgHash [32]byte) (bn254.G1Affine, error) {
	h := new(big.Int).SetBytes(msgHash[:])
	h.Mod(h, fieldP)

	three := big.NewInt(3)

	for i := 0; i < 256; i++ {
		// x = (h + i) % P
		x := new(big.Int).Add(h, big.NewInt(int64(i)))
		x.Mod(x, fieldP)

		// y² = x³ + 3
		x2 := new(big.Int).Mul(x, x)
		x2.Mod(x2, fieldP)
		x3 := new(big.Int).Mul(x2, x)
		x3.Mod(x3, fieldP)
		y2 := new(big.Int).Add(x3, three)
		y2.Mod(y2, fieldP)

		// y = y2^((P+1)/4) mod P
		y := new(big.Int).Exp(y2, sqrtExp, fieldP)

		// Verify: y² == y2 mod P
		ySquared := new(big.Int).Mul(y, y)
		ySquared.Mod(ySquared, fieldP)
		if ySquared.Cmp(y2) == 0 {
			var pt bn254.G1Affine
			pt.X.SetBigInt(x)
			pt.Y.SetBigInt(y)
			return pt, nil
		}
	}

	return bn254.G1Affine{}, errors.New("hashToG1: no valid point found in 256 iterations")
}

// Sign produces a BLS signature: sigma = sk * HashToG1(msgHash).
func Sign(sk *fr.Element, msgHash [32]byte) (bn254.G1Affine, error) {
	hm, err := HashToG1(msgHash)
	if err != nil {
		return bn254.G1Affine{}, err
	}

	var skBigInt big.Int
	sk.BigInt(&skBigInt)

	var sigma bn254.G1Affine
	sigma.ScalarMultiplication(&hm, &skBigInt)
	return sigma, nil
}

// ErrEmptyAggregation is returned when AggregateG1 or AggregateG2 is called
// with an empty slice. An empty set of signatures/keys must not produce a
// valid aggregate — doing so would allow an empty cluster to "sign" any message.
var ErrEmptyAggregation = errors.New("cannot aggregate empty point set")

// AggregateG1 sums a slice of G1 points. Returns ErrEmptyAggregation if the
// input is empty — an empty set of signatures must not produce a valid aggregate.
func AggregateG1(points []bn254.G1Affine) (bn254.G1Affine, error) {
	if len(points) == 0 {
		return bn254.G1Affine{}, ErrEmptyAggregation
	}
	var agg bn254.G1Jac
	agg.FromAffine(&points[0])
	for i := 1; i < len(points); i++ {
		var pJac bn254.G1Jac
		pJac.FromAffine(&points[i])
		agg.AddAssign(&pJac)
	}
	var result bn254.G1Affine
	result.FromJacobian(&agg)
	return result, nil
}

// AggregateG2 sums a slice of G2 points. Returns ErrEmptyAggregation if the
// input is empty — an empty set of public keys must not produce a valid aggregate.
func AggregateG2(points []bn254.G2Affine) (bn254.G2Affine, error) {
	if len(points) == 0 {
		return bn254.G2Affine{}, ErrEmptyAggregation
	}
	var agg bn254.G2Jac
	agg.FromAffine(&points[0])
	for i := 1; i < len(points); i++ {
		var pJac bn254.G2Jac
		pJac.FromAffine(&points[i])
		agg.AddAssign(&pJac)
	}
	var result bn254.G2Affine
	result.FromJacobian(&agg)
	return result, nil
}

// Verify checks a BLS signature via pairing: e(sigma, G2gen) == e(H(m), pubG2).
// Equivalently: e(sigma, G2gen) * e(-H(m), pubG2) == 1.
func Verify(sigma bn254.G1Affine, pubG2 bn254.G2Affine, msgHash [32]byte) (bool, error) {
	hm, err := HashToG1(msgHash)
	if err != nil {
		return false, err
	}

	// Negate H(m)
	var negHm bn254.G1Affine
	negHm.Neg(&hm)

	_, _, _, g2Gen := bn254.Generators()

	// PairingCheck verifies: e(g1s[0], g2s[0]) * e(g1s[1], g2s[1]) == 1
	ok, err := bn254.PairingCheck(
		[]bn254.G1Affine{sigma, negHm},
		[]bn254.G2Affine{g2Gen, pubG2},
	)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// EncodeSignatureForContract ABI-encodes a BLS cluster signature for the Solidity contract.
// The Solidity contract decodes: abi.decode(signature, (uint256, uint256[2], uint256[4]))
// where the format is (bitmask, [sigma.X, sigma.Y], [apkG2.X.im, apkG2.X.re, apkG2.Y.im, apkG2.Y.re]).
func EncodeSignatureForContract(bitmask *big.Int, sigma bn254.G1Affine, apkG2 bn254.G2Affine) ([]byte, error) {
	// Extract sigma G1 coordinates
	var sigX, sigY big.Int
	sigma.X.BigInt(&sigX)
	sigma.Y.BigInt(&sigY)

	// Extract apkG2 coordinates.
	// gnark-crypto E2: A0 = real, A1 = imaginary.
	// Solidity expects: [x_im, x_re, y_im, y_re].
	var apkXIm, apkXRe, apkYIm, apkYRe big.Int
	apkG2.X.A1.BigInt(&apkXIm) // imaginary
	apkG2.X.A0.BigInt(&apkXRe) // real
	apkG2.Y.A1.BigInt(&apkYIm) // imaginary
	apkG2.Y.A0.BigInt(&apkYRe) // real

	// ABI encode as (uint256, uint256[2], uint256[4])
	args := abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Uint256Arr2},
		{Type: abiutil.Uint256Arr4},
	}

	return args.Pack(
		bitmask,
		[2]*big.Int{&sigX, &sigY},
		[4]*big.Int{&apkXIm, &apkXRe, &apkYIm, &apkYRe},
	)
}

// G1ToCoords extracts the X, Y coordinates of a G1 point as big.Ints.
func G1ToCoords(p bn254.G1Affine) [2]*big.Int {
	var x, y big.Int
	p.X.BigInt(&x)
	p.Y.BigInt(&y)
	return [2]*big.Int{&x, &y}
}

// SerializeG1 serializes a G1 point as 64 bytes (X || Y, big-endian).
func SerializeG1(p bn254.G1Affine) []byte {
	var x, y big.Int
	p.X.BigInt(&x)
	p.Y.BigInt(&y)
	buf := make([]byte, 64)
	x.FillBytes(buf[:32])
	y.FillBytes(buf[32:])
	return buf
}

// DeserializeG1 reads a G1 point from 64 bytes (X || Y, big-endian). It is an
// acceptance-path decoder for untrusted input, so it fully validates the point:
// each coordinate must be a canonical field element (< P), and the point must be
// on the curve and in the prime-order subgroup. Skipping the subgroup check
// would admit small-subgroup points that break the signature scheme's security.
func DeserializeG1(data []byte) (bn254.G1Affine, error) {
	if len(data) != 64 {
		return bn254.G1Affine{}, errors.New("invalid G1 data length: expected 64 bytes")
	}
	x := new(big.Int).SetBytes(data[:32])
	y := new(big.Int).SetBytes(data[32:])
	if x.Cmp(fieldP) >= 0 || y.Cmp(fieldP) >= 0 {
		return bn254.G1Affine{}, errors.New("bls: G1 coordinate not in field range")
	}
	var pt bn254.G1Affine
	pt.X.SetBigInt(x)
	pt.Y.SetBigInt(y)
	if !pt.IsOnCurve() {
		return bn254.G1Affine{}, errors.New("bls: G1 point not on curve")
	}
	if !pt.IsInSubGroup() {
		return bn254.G1Affine{}, errors.New("bls: G1 point not in prime-order subgroup")
	}
	return pt, nil
}

// G2ToCoords extracts the BN254 G2 coordinates in Solidity order: [x_im, x_re, y_im, y_re].
func G2ToCoords(p bn254.G2Affine) [4]*big.Int {
	var xIm, xRe, yIm, yRe big.Int
	p.X.A1.BigInt(&xIm) // imaginary
	p.X.A0.BigInt(&xRe) // real
	p.Y.A1.BigInt(&yIm) // imaginary
	p.Y.A0.BigInt(&yRe) // real
	return [4]*big.Int{&xIm, &xRe, &yIm, &yRe}
}

// SerializeG2 serializes a G2 point as 128 bytes (X.A1 || X.A0 || Y.A1 || Y.A0, big-endian).
func SerializeG2(p bn254.G2Affine) []byte {
	buf := make([]byte, 128)
	var xi, xr, yi, yr big.Int
	p.X.A1.BigInt(&xi) // imaginary
	p.X.A0.BigInt(&xr) // real
	p.Y.A1.BigInt(&yi) // imaginary
	p.Y.A0.BigInt(&yr) // real
	xi.FillBytes(buf[0:32])
	xr.FillBytes(buf[32:64])
	yi.FillBytes(buf[64:96])
	yr.FillBytes(buf[96:128])
	return buf
}

// DeserializeG2 reads a G2 point from 128 bytes. Like DeserializeG1 it is an
// acceptance-path decoder: it rejects non-canonical coordinates (>= P) and
// points that are off-curve or outside the prime-order subgroup. The off-chain
// verifier must apply the same membership checks the on-chain precompile does,
// or a crafted pubkey/signature could pass off-chain acceptance.
func DeserializeG2(data []byte) (bn254.G2Affine, error) {
	if len(data) != 128 {
		return bn254.G2Affine{}, errors.New("invalid G2 data length: expected 128 bytes")
	}
	xa1 := new(big.Int).SetBytes(data[0:32])
	xa0 := new(big.Int).SetBytes(data[32:64])
	ya1 := new(big.Int).SetBytes(data[64:96])
	ya0 := new(big.Int).SetBytes(data[96:128])
	for _, c := range []*big.Int{xa1, xa0, ya1, ya0} {
		if c.Cmp(fieldP) >= 0 {
			return bn254.G2Affine{}, errors.New("bls: G2 coordinate not in field range")
		}
	}
	var pt bn254.G2Affine
	pt.X.A1.SetBigInt(xa1)
	pt.X.A0.SetBigInt(xa0)
	pt.Y.A1.SetBigInt(ya1)
	pt.Y.A0.SetBigInt(ya0)
	if !pt.IsOnCurve() {
		return bn254.G2Affine{}, errors.New("bls: G2 point not on curve")
	}
	if !pt.IsInSubGroup() {
		return bn254.G2Affine{}, errors.New("bls: G2 point not in prime-order subgroup")
	}
	return pt, nil
}

// ComputePopHash computes the proof-of-possession message hash for a given operator address.
// Matches the Solidity contract: keccak256(abi.encodePacked(msg.sender)).
func ComputePopHash(operator common.Address) [32]byte {
	return crypto.Keccak256Hash(operator.Bytes())
}

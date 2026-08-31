package receipt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// WithdrawalIssuerResolver maps a withdrawal id to the ConfigRegistry issuer
// whose receipt signers authorize the burn receipt.
type WithdrawalIssuerResolver interface {
	IssuerIDByWithdrawalID(ctx context.Context, withdrawalID [32]byte) (common.Address, error)
}

// ReceiptVerifier validates MintReceipts and BurnReceipts against the current
// issuer-scoped receipt signer set.
//
// Verification dispatches on signature length:
//   - 65 bytes → ECDSA secp256k1 (EVM custody chains; ADR-005 §11.1).
//   - 64 bytes → ED25519 (XRPL/Solana custody; not yet implemented; gated on
//     a per-chain ADR per ADR-005 §10).
//   - other     → ignored.
//
// A signer's contribution is counted at most once across the receipt's
// signatures, so a duplicated signature does not satisfy the threshold.
type ReceiptVerifier struct {
	source            core.ReceiptSignerSource
	withdrawalIssuers WithdrawalIssuerResolver
}

// ReceiptSignatureValidator is an immutable snapshot of the signer quorum and
// digest for one receipt. It is safe to use while collecting untrusted mesh
// candidates: rejected signatures never consume a quorum slot.
type ReceiptSignatureValidator struct {
	digest    []byte
	signers   map[common.Address]struct{}
	threshold int
}

const maxCandidatesPerReceiptSigner = 4

var receiptSecp256k1HalfN = new(big.Int).Rsh(new(big.Int).Set(crypto.S256().Params().N), 1)

// Digest returns a defensive copy of the receipt digest.
func (v *ReceiptSignatureValidator) Digest() []byte {
	if v == nil {
		return nil
	}
	return append([]byte(nil), v.digest...)
}

// Threshold returns the frozen quorum threshold.
func (v *ReceiptSignatureValidator) Threshold() int {
	if v == nil {
		return 0
	}
	return v.threshold
}

// MatchesSigningContext reports whether two validators bind the same digest,
// quorum threshold, and authorized signer set. Callers use it to fail closed
// when the live roster changes between collection and persistence.
func (v *ReceiptSignatureValidator) MatchesSigningContext(other *ReceiptSignatureValidator) bool {
	if v == nil || other == nil || v.threshold != other.threshold ||
		!bytes.Equal(v.digest, other.digest) || len(v.signers) != len(other.signers) {
		return false
	}
	for signer := range v.signers {
		if _, ok := other.signers[signer]; !ok {
			return false
		}
	}
	return true
}

// ValidateSignature recovers and authorizes one receipt signature. Legacy
// recovery-ID forms {0,1,27,28} and high-S encodings remain accepted, but
// QuorumSignatures emits their unique low-S, V={0,1} canonical form. Every
// rejected candidate, including an unauthorized recovery, returns zero,false.
func (v *ReceiptSignatureValidator) ValidateSignature(sig []byte) (common.Address, bool) {
	if v == nil {
		return common.Address{}, false
	}
	addr, _, ok, err := recoverCanonicalReceiptSigner(v.digest, sig)
	if err != nil || !ok {
		return common.Address{}, false
	}
	if _, authorized := v.signers[addr]; !authorized {
		return common.Address{}, false
	}
	return addr, true
}

// VerifySignatures verifies a distinct authorized quorum against the frozen
// snapshot used during collection.
func (v *ReceiptSignatureValidator) VerifySignatures(sigs [][]byte) error {
	if v == nil || v.threshold <= 0 {
		return errors.New("receipt signature validator not configured")
	}
	if len(sigs) < v.threshold {
		return fmt.Errorf("insufficient signatures: %d < %d", len(sigs), v.threshold)
	}
	if len(sigs) > maxCandidatesPerReceiptSigner*len(v.signers) {
		return fmt.Errorf("too many signature candidates: %d for %d authorized signers", len(sigs), len(v.signers))
	}
	seen := make(map[common.Address]struct{}, v.threshold)
	var invalid, unauthorized, duplicates int
	for _, sig := range sigs {
		addr, _, ok, err := recoverCanonicalReceiptSigner(v.digest, sig)
		if err != nil || !ok {
			invalid++
			continue
		}
		if _, authorized := v.signers[addr]; !authorized {
			unauthorized++
			continue
		}
		if _, duplicate := seen[addr]; duplicate {
			duplicates++
			continue
		}
		seen[addr] = struct{}{}
		if len(seen) >= v.threshold {
			return nil
		}
	}
	return fmt.Errorf("insufficient distinct signers: %d/%d (invalid=%d unauthorized=%d duplicate=%d)",
		len(seen), v.threshold, invalid, unauthorized, duplicates)
}

// QuorumSignatures filters invalid, unauthorized, and duplicate candidates,
// then returns a deterministic threshold-sized quorum ordered by signer.
func (v *ReceiptSignatureValidator) QuorumSignatures(sigs [][]byte) ([][]byte, error) {
	if v == nil || v.threshold <= 0 {
		return nil, errors.New("receipt signature validator not configured")
	}
	if len(sigs) < v.threshold {
		return nil, fmt.Errorf("insufficient signatures: %d < %d", len(sigs), v.threshold)
	}
	if len(sigs) > maxCandidatesPerReceiptSigner*len(v.signers) {
		return nil, fmt.Errorf("too many signature candidates: %d for %d authorized signers", len(sigs), len(v.signers))
	}
	bySigner := make(map[common.Address][]byte, v.threshold)
	var invalid, unauthorized, duplicates int
	for _, sig := range sigs {
		addr, canonical, ok, err := recoverCanonicalReceiptSigner(v.digest, sig)
		if err != nil || !ok {
			invalid++
			continue
		}
		if _, authorized := v.signers[addr]; !authorized {
			unauthorized++
			continue
		}
		if kept, exists := bySigner[addr]; exists {
			duplicates++
			if bytes.Compare(canonical, kept) < 0 {
				bySigner[addr] = canonical
			}
		} else {
			bySigner[addr] = canonical
		}
	}
	if len(bySigner) < v.threshold {
		return nil, fmt.Errorf("insufficient distinct signers: %d/%d (invalid=%d unauthorized=%d duplicate=%d)",
			len(bySigner), v.threshold, invalid, unauthorized, duplicates)
	}
	addresses := make([]common.Address, 0, len(bySigner))
	for addr := range bySigner {
		addresses = append(addresses, addr)
	}
	sort.Slice(addresses, func(i, j int) bool { return bytes.Compare(addresses[i][:], addresses[j][:]) < 0 })
	out := make([][]byte, v.threshold)
	for i, addr := range addresses[:v.threshold] {
		out[i] = bySigner[addr]
	}
	return out, nil
}

func NewReceiptVerifier(source core.ReceiptSignerSource, withdrawalIssuers WithdrawalIssuerResolver) *ReceiptVerifier {
	return &ReceiptVerifier{source: source, withdrawalIssuers: withdrawalIssuers}
}

// SetSignersForTest seeds the verifier from an explicit signer list. Tests use
// this to drive the verifier without an on-chain reader.
func (rv *ReceiptVerifier) SetSignersForTest(signers []common.Address, threshold int) error {
	if rv == nil {
		return errors.New("receipt verifier not configured")
	}
	src, err := NewStaticSignerSource(signers, threshold)
	if err != nil {
		return err
	}
	rv.source = src
	return nil
}

// VerifyBurnReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over BurnReceiptDigest.
func (rv *ReceiptVerifier) VerifyBurnReceipt(ctx context.Context, v *core.BurnReceipt) error {
	validator, err := rv.PrepareBurnReceiptSignatures(ctx, v)
	if err != nil {
		return err
	}
	return validator.VerifySignatures(v.Signatures)
}

// PrepareBurnReceiptSignatures freezes the current issuer receipt quorum and
// exact BurnReceipt digest for poison-tolerant candidate collection.
func (rv *ReceiptVerifier) PrepareBurnReceiptSignatures(ctx context.Context, v *core.BurnReceipt) (*ReceiptSignatureValidator, error) {
	if v == nil {
		return nil, errors.New("nil burn receipt")
	}
	if rv == nil || rv.withdrawalIssuers == nil {
		return nil, errors.New("receipt verifier has no withdrawal issuer resolver")
	}
	issuerID, err := rv.withdrawalIssuers.IssuerIDByWithdrawalID(ctx, v.WithdrawalID)
	if err != nil {
		return nil, fmt.Errorf("resolve withdrawal issuer: %w", err)
	}
	return rv.prepareSignatures(ctx, issuerID, BurnReceiptDigest(v))
}

// VerifyMintReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over MintReceiptDigest.
func (rv *ReceiptVerifier) VerifyMintReceipt(ctx context.Context, v *core.MintReceipt) error {
	validator, err := rv.PrepareMintReceiptSignatures(ctx, v)
	if err != nil {
		return err
	}
	return validator.VerifySignatures(v.Signatures)
}

// PrepareMintReceiptSignatures freezes the current issuer receipt quorum and
// exact MintReceipt digest for poison-tolerant candidate collection.
func (rv *ReceiptVerifier) PrepareMintReceiptSignatures(ctx context.Context, v *core.MintReceipt) (*ReceiptSignatureValidator, error) {
	if v == nil {
		return nil, errors.New("nil mint receipt")
	}
	if v.Amount.Sign() <= 0 {
		return nil, errors.New("mint receipt amount must be positive")
	}
	issuerID, err := core.IssuerIDFromAssetURI(v.AssetURI)
	if err != nil {
		return nil, fmt.Errorf("mint receipt issuer: %w", err)
	}
	return rv.prepareSignatures(ctx, issuerID, MintReceiptDigest(v))
}

func (rv *ReceiptVerifier) prepareSignatures(ctx context.Context, issuerID common.Address, digest []byte) (*ReceiptSignatureValidator, error) {
	if rv == nil {
		return nil, errors.New("receipt verifier not configured")
	}
	if rv.source == nil {
		return nil, errors.New("receipt verifier has no signer source")
	}
	set, err := rv.source.LoadReceiptSigners(ctx, issuerID)
	if err != nil {
		return nil, fmt.Errorf("load receipt signers: %w", err)
	}
	if set.Threshold <= 0 || set.Threshold > len(set.Signers) {
		return nil, fmt.Errorf("receipt threshold = %d out of range for %d signers", set.Threshold, len(set.Signers))
	}
	signers := make(map[common.Address]struct{}, len(set.Signers))
	for _, s := range set.Signers {
		if s == (common.Address{}) {
			return nil, errors.New("receipt signer set contains zero address")
		}
		signers[s] = struct{}{}
	}
	if set.Threshold > len(signers) {
		return nil, fmt.Errorf("receipt threshold = %d exceeds %d distinct signers", set.Threshold, len(signers))
	}
	return &ReceiptSignatureValidator{
		digest: append([]byte(nil), digest...), signers: signers, threshold: set.Threshold,
	}, nil
}

// recoverCanonicalReceiptSigner dispatches signature verification by length
// and returns both the signer and the canonical wire signature. ED25519 is
// recognised but not yet implemented; unsupported lengths are ignored.
func recoverCanonicalReceiptSigner(digest, sig []byte) (common.Address, []byte, bool, error) {
	switch len(sig) {
	case 65:
		canonical := append([]byte(nil), sig...)
		v := canonical[64]
		if v == 27 || v == 28 {
			v -= 27
		} else if v > 1 {
			return common.Address{}, nil, false, nil
		}
		r := new(big.Int).SetBytes(canonical[:32])
		s := new(big.Int).SetBytes(canonical[32:64])
		if !crypto.ValidateSignatureValues(v, r, s, false) {
			return common.Address{}, nil, false, nil
		}
		if s.Cmp(receiptSecp256k1HalfN) > 0 {
			s.Sub(crypto.S256().Params().N, s)
			s.FillBytes(canonical[32:64])
			v ^= 1
		}
		canonical[64] = v
		pub, err := crypto.SigToPub(digest, canonical)
		if err != nil {
			return common.Address{}, nil, false, err
		}
		return crypto.PubkeyToAddress(*pub), canonical, true, nil
	case 64:
		// XRPL / Solana ED25519 lands when its custody adapter is wired
		// (ADR-005 §10). Until then the receipt path is ECDSA-only.
		return common.Address{}, nil, false, nil
	default:
		return common.Address{}, nil, false, nil
	}
}

// BurnReceiptDigest is the keccak256 digest custody providers sign over for
// withdrawal terminal attestations.
// Format: keccak256(
//
//	WithdrawalID || BlockHash || EntryIndex[uint64be] ||
//	len(TxID)[uint32be] || TxID || Status[byte]).
//
// The trailing Status byte binds the terminal outcome (Executed vs
// Expired) into the signature, so a quorum can never be tricked into swapping
// an executed receipt for an expired one (which would authorize a re-credit).
// Exported so custody-side tooling and the custodytesting package can build
// matching signatures.
func BurnReceiptDigest(v *core.BurnReceipt) []byte {
	txID := []byte(v.TxID)
	buf := make([]byte, 0, 32+32+8+4+len(txID)+1)
	buf = append(buf, v.WithdrawalID[:]...)
	buf = append(buf, v.BlockHash[:]...)
	var index [8]byte
	binary.BigEndian.PutUint64(index[:], v.EntryIndex)
	buf = append(buf, index[:]...)
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(txID)))
	buf = append(buf, u32[:]...)
	buf = append(buf, txID...)
	buf = append(buf, byte(v.Status))
	return crypto.Keccak256(buf)
}

// MintReceiptDigest is the keccak256 digest custody providers sign over for
// deposit confirmation attestations. Exported so custody-side tooling can
// produce matching signatures.
//
// Format: keccak256(
//
//	len(TxID)[uint32be]     || TxID ||
//	len(Account)[uint32be]  || Account ||
//	len(AssetURI)[uint32be] || AssetURI ||
//	len(Amount)[uint32be]   || canonical-CBOR(decimal.Decimal))
//
// The length prefixes prevent boundary-shift collisions between variable-size
// receipt fields. Idempotency is keyed by (AssetURI, TxID), but the digest also
// binds the credited account and protocol amount.
func MintReceiptDigest(v *core.MintReceipt) []byte {
	var amount bytes.Buffer
	if err := v.Amount.MarshalCBOR(&amount); err != nil {
		panic(fmt.Errorf("receipt: decimal MarshalCBOR: %w", err))
	}
	txID := []byte(v.TxID)
	account := []byte(v.Account)
	assetURI := []byte(v.AssetURI)
	amountBytes := amount.Bytes()
	buf := make([]byte, 0, 4+len(txID)+4+len(account)+4+len(assetURI)+4+len(amountBytes))
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(txID)))
	buf = append(buf, u32[:]...)
	buf = append(buf, txID...)
	binary.BigEndian.PutUint32(u32[:], uint32(len(account)))
	buf = append(buf, u32[:]...)
	buf = append(buf, account...)
	binary.BigEndian.PutUint32(u32[:], uint32(len(assetURI)))
	buf = append(buf, u32[:]...)
	buf = append(buf, assetURI...)
	binary.BigEndian.PutUint32(u32[:], uint32(len(amountBytes)))
	buf = append(buf, u32[:]...)
	buf = append(buf, amountBytes...)
	return crypto.Keccak256(buf)
}

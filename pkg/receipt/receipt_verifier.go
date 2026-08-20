package receipt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
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

func NewReceiptVerifier(source core.ReceiptSignerSource, withdrawalIssuers WithdrawalIssuerResolver) *ReceiptVerifier {
	return &ReceiptVerifier{source: source, withdrawalIssuers: withdrawalIssuers}
}

// SetSignersForTest seeds the cache from an explicit signer list. Tests use
// this to drive the verifier without an on-chain reader.
func (rv *ReceiptVerifier) SetSignersForTest(signers []common.Address, threshold int) {
	if rv == nil {
		return
	}
	src, _ := NewStaticSignerSource(signers, threshold)
	rv.source = src
}

// VerifyBurnReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over BurnReceiptDigest.
func (rv *ReceiptVerifier) VerifyBurnReceipt(ctx context.Context, v *core.BurnReceipt) error {
	if v == nil {
		return errors.New("nil burn receipt")
	}
	if rv == nil || rv.withdrawalIssuers == nil {
		return errors.New("receipt verifier has no withdrawal issuer resolver")
	}
	issuerID, err := rv.withdrawalIssuers.IssuerIDByWithdrawalID(ctx, v.WithdrawalID)
	if err != nil {
		return fmt.Errorf("resolve withdrawal issuer: %w", err)
	}
	return rv.verifySignatures(ctx, issuerID, BurnReceiptDigest(v), v.Signatures)
}

// VerifyMintReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over MintReceiptDigest.
func (rv *ReceiptVerifier) VerifyMintReceipt(ctx context.Context, v *core.MintReceipt) error {
	if v == nil {
		return errors.New("nil mint receipt")
	}
	if v.Amount.Sign() <= 0 {
		return errors.New("mint receipt amount must be positive")
	}
	issuerID, err := core.IssuerIDFromAssetURI(v.AssetURI)
	if err != nil {
		return fmt.Errorf("mint receipt issuer: %w", err)
	}
	return rv.verifySignatures(ctx, issuerID, MintReceiptDigest(v), v.Signatures)
}

// verifySignatures is the shared signature-quorum check used by both receipt
// kinds. It enforces the staleness window, the count floor, and distinct-signer
// quorum.
func (rv *ReceiptVerifier) verifySignatures(ctx context.Context, issuerID common.Address, digest []byte, sigs [][]byte) error {
	if rv == nil {
		return errors.New("receipt verifier not configured")
	}
	if rv.source == nil {
		return errors.New("receipt verifier has no signer source")
	}
	set, err := rv.source.LoadReceiptSigners(ctx, issuerID)
	if err != nil {
		return fmt.Errorf("load receipt signers: %w", err)
	}
	if set.Threshold <= 0 || set.Threshold > len(set.Signers) {
		return fmt.Errorf("receipt threshold = %d out of range for %d signers", set.Threshold, len(set.Signers))
	}
	signers := make(map[common.Address]struct{}, len(set.Signers))
	for _, s := range set.Signers {
		signers[s] = struct{}{}
	}
	if len(sigs) < set.Threshold {
		return fmt.Errorf("insufficient signatures: %d < %d", len(sigs), set.Threshold)
	}
	seen := make(map[common.Address]struct{}, set.Threshold)
	for _, sig := range sigs {
		addr, ok, err := recoverReceiptSigner(digest, sig)
		if err != nil {
			continue
		}
		if !ok {
			continue
		}
		if _, isSigner := signers[addr]; !isSigner {
			continue
		}
		if _, dup := seen[addr]; dup {
			continue
		}
		seen[addr] = struct{}{}
		if len(seen) >= set.Threshold {
			return nil
		}
	}
	return fmt.Errorf("insufficient distinct signers: %d/%d", len(seen), set.Threshold)
}

// recoverReceiptSigner dispatches signature verification by length and
// returns the signer address that produced it. ED25519 is recognised but not
// yet implemented; non-canonical lengths are ignored (ok=false, err=nil).
func recoverReceiptSigner(digest, sig []byte) (common.Address, bool, error) {
	switch len(sig) {
	case 65:
		addr, err := eip712.RecoverSigner(digest, sig)
		if err != nil {
			return common.Address{}, false, err
		}
		return addr, true, nil
	case 64:
		// XRPL / Solana ED25519 lands when its custody adapter is wired
		// (ADR-005 §10). Until then the receipt path is ECDSA-only.
		return common.Address{}, false, nil
	default:
		return common.Address{}, false, nil
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

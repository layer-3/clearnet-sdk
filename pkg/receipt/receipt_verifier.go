package receipt

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

// SignerSource supplies the custody signer set and threshold that the
// receipt verifier checks signatures against. It is chain-agnostic: a node
// can serve receipts produced by custody systems on any L1 as long as some
// SignerSource implementation knows the active signer set.
//
// Implementations available today:
//   - StaticSignerSource — manifest-driven; operator-managed list.
//
// Planned: a Registry-backed source that reads a single canonical signer
// directory from the on-chain Registry once that contract grows the right
// method. The interface here is the seam that lets that swap land
// without touching the verifier or its callers. See the ADR-005
// 2026-05-12 receipt-model amendment.
type SignerSource interface {
	// Load returns the current signer set and threshold. Implementations
	// may be cached or pull from a remote source; the verifier calls Load
	// at startup and on a periodic ticker.
	Load(ctx context.Context) (signers []common.Address, threshold int, err error)
}

const defaultReceiptVerifierMaxAge = 15 * time.Minute

// ReceiptVerifier validates MintReceipts and BurnReceipts against a
// SignerSource. It caches signers and threshold and refreshes them
// periodically so verification is a fast in-memory check.
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
	source SignerSource

	mu        sync.RWMutex
	signers   map[common.Address]struct{}
	threshold int
	// refreshedAt/maxAge bound how long a node may trust a cached signer set
	// after refresh failures. Signer rotation must fail closed, not accept an
	// old quorum forever when the source is unreachable.
	refreshedAt time.Time
	maxAge      time.Duration
}

// NewReceiptVerifier returns a verifier backed by the given SignerSource.
// Refresh must be called before either Verify* method; production callers
// do this at startup and on a periodic ticker (see RunReceiptVerifierRefresh).
// maxAge bounds cache staleness; non-positive falls back to the package
// default (15 minutes).
func NewReceiptVerifier(source SignerSource, maxAge time.Duration) *ReceiptVerifier {
	if maxAge <= 0 {
		maxAge = defaultReceiptVerifierMaxAge
	}
	return &ReceiptVerifier{source: source, maxAge: maxAge}
}

// Refresh reads the current signer set and threshold from the SignerSource
// and replaces the cached view atomically.
func (rv *ReceiptVerifier) Refresh(ctx context.Context) error {
	if rv == nil {
		return errors.New("receipt verifier not configured")
	}
	if rv.source == nil {
		return errors.New("receipt verifier has no signer source")
	}
	signers, threshold, err := rv.source.Load(ctx)
	if err != nil {
		return fmt.Errorf("load custody signers: %w", err)
	}
	if threshold <= 0 || threshold > len(signers) {
		return fmt.Errorf("custody threshold = %d out of range for %d signers", threshold, len(signers))
	}
	set := make(map[common.Address]struct{}, len(signers))
	for _, s := range signers {
		set[s] = struct{}{}
	}
	rv.mu.Lock()
	rv.signers = set
	rv.threshold = threshold
	rv.refreshedAt = time.Now()
	rv.mu.Unlock()
	return nil
}

// SetSignersForTest seeds the cache from an explicit signer list. Tests use
// this to drive the verifier without an on-chain reader.
func (rv *ReceiptVerifier) SetSignersForTest(signers []common.Address, threshold int) {
	if rv == nil {
		return
	}
	set := make(map[common.Address]struct{}, len(signers))
	for _, s := range signers {
		set[s] = struct{}{}
	}
	rv.mu.Lock()
	rv.signers = set
	rv.threshold = threshold
	rv.refreshedAt = time.Now()
	rv.mu.Unlock()
}

// VerifyBurnReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over BurnReceiptDigest.
func (rv *ReceiptVerifier) VerifyBurnReceipt(v *core.BurnReceipt) error {
	if v == nil {
		return errors.New("nil burn receipt")
	}
	return rv.verifySignatures(BurnReceiptDigest(v), v.Signatures)
}

// VerifyMintReceipt checks that the receipt carries at least `threshold`
// distinct valid signatures from the cached signer set over MintReceiptDigest.
func (rv *ReceiptVerifier) VerifyMintReceipt(v *core.MintReceipt) error {
	if v == nil {
		return errors.New("nil mint receipt")
	}
	if v.Amount == nil || v.Amount.Sign() <= 0 {
		return errors.New("mint receipt amount must be positive")
	}
	return rv.verifySignatures(MintReceiptDigest(v), v.Signatures)
}

// verifySignatures is the shared signature-quorum check used by both receipt
// kinds. It enforces the staleness window, the count floor, and distinct-signer
// quorum.
func (rv *ReceiptVerifier) verifySignatures(digest []byte, sigs [][]byte) error {
	if rv == nil {
		return errors.New("receipt verifier not configured")
	}
	rv.mu.RLock()
	signers := rv.signers
	threshold := rv.threshold
	refreshedAt := rv.refreshedAt
	maxAge := rv.maxAge
	rv.mu.RUnlock()
	if threshold <= 0 || len(signers) == 0 {
		return errors.New("receipt verifier signer set not initialised")
	}
	if maxAge > 0 {
		age := time.Since(refreshedAt)
		if refreshedAt.IsZero() || age > maxAge {
			return fmt.Errorf("receipt verifier signer set stale: age %s > %s", age, maxAge)
		}
	}
	if len(sigs) < threshold {
		return fmt.Errorf("insufficient signatures: %d < %d", len(sigs), threshold)
	}
	seen := make(map[common.Address]struct{}, threshold)
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
		if len(seen) >= threshold {
			return nil
		}
	}
	return fmt.Errorf("insufficient distinct signers: %d/%d", len(seen), threshold)
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
// Format: keccak256(WithdrawalID || BlockHash || EntryIndex[uint64be] || L1TxHash || Status[byte]).
// The trailing Status byte binds the terminal outcome (Executed vs
// Expired) into the signature, so a quorum can never be tricked into swapping
// an executed receipt for an expired one (which would authorize a re-credit).
// Exported so custody-side tooling and the custodytesting package can build
// matching signatures.
func BurnReceiptDigest(v *core.BurnReceipt) []byte {
	buf := make([]byte, 0, 32+32+8+32+1)
	buf = append(buf, v.WithdrawalID[:]...)
	buf = append(buf, v.BlockHash[:]...)
	var index [8]byte
	binary.BigEndian.PutUint64(index[:], v.EntryIndex)
	buf = append(buf, index[:]...)
	buf = append(buf, v.L1TxHash[:]...)
	buf = append(buf, byte(v.Status))
	return crypto.Keccak256(buf)
}

// MintReceiptDigest is the keccak256 digest custody providers sign over for
// deposit confirmation attestations. Exported so custody-side tooling can
// produce matching signatures.
//
// Format: keccak256(
//
//	ChainID[uint64be] || L1TxHash || LogIndex[uint64be] ||
//	len(Account)[uint32be] || Account ||
//	len(Asset)[uint32be]   || Asset ||
//	Amount[uint256be])
//
// The length prefixes on Account and Asset prevent boundary-shift
// collisions between variable-length string pairs.
func MintReceiptDigest(v *core.MintReceipt) []byte {
	var amountBE [32]byte
	if v.Amount != nil && v.Amount.Sign() > 0 {
		v.Amount.FillBytes(amountBE[:])
	}
	buf := make([]byte, 0, 8+32+8+4+len(v.Account)+4+len(v.Asset)+32)
	var u64 [8]byte
	binary.BigEndian.PutUint64(u64[:], v.ChainID)
	buf = append(buf, u64[:]...)
	buf = append(buf, v.L1TxHash[:]...)
	binary.BigEndian.PutUint64(u64[:], v.LogIndex)
	buf = append(buf, u64[:]...)
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(v.Account)))
	buf = append(buf, u32[:]...)
	buf = append(buf, []byte(v.Account)...)
	binary.BigEndian.PutUint32(u32[:], uint32(len(v.Asset)))
	buf = append(buf, u32[:]...)
	buf = append(buf, []byte(v.Asset)...)
	buf = append(buf, amountBE[:]...)
	return crypto.Keccak256(buf)
}

// SignerCount reports the cached signer-set size for diagnostics.
func (rv *ReceiptVerifier) SignerCount() int {
	if rv == nil {
		return 0
	}
	rv.mu.RLock()
	defer rv.mu.RUnlock()
	return len(rv.signers)
}

// Threshold reports the cached signer threshold for diagnostics.
func (rv *ReceiptVerifier) Threshold() int {
	if rv == nil {
		return 0
	}
	rv.mu.RLock()
	defer rv.mu.RUnlock()
	return rv.threshold
}

// RunReceiptVerifierRefresh periodically refreshes the verifier's cached
// signer set. It returns when ctx is cancelled.
func RunReceiptVerifierRefresh(ctx context.Context, rv *ReceiptVerifier, interval time.Duration, onError func(error)) {
	if rv == nil || interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := rv.Refresh(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

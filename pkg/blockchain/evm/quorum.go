package evm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	internalquorum "github.com/layer-3/clearnet-sdk/internal/quorum"
)

// SignatureValidator is an immutable digest, signer-set, and threshold
// snapshot for one EVM quorum operation.
type SignatureValidator struct {
	snapshot *internalquorum.Snapshot[common.Address]
}

const maxCandidatesPerAuthorizedSigner = internalquorum.MaxCandidatesPerSigner

// NewSignatureValidator prepares candidate validation without any network
// access. Signatures must use the custody wire form V in {0,1}; contract form
// {27,28} is produced only by ContractSignatures.
func NewSignatureValidator(digest common.Hash, signers []common.Address, threshold int) (*SignatureValidator, error) {
	snapshot, err := internalquorum.NewSnapshot([32]byte(digest), signers, threshold, func(signer common.Address) bool {
		return signer == (common.Address{})
	})
	if err != nil {
		return nil, err
	}
	return &SignatureValidator{snapshot: snapshot}, nil
}

// Digest returns the prepared operation digest. A nil validator returns zero.
func (v *SignatureValidator) Digest() common.Hash {
	if v == nil {
		return common.Hash{}
	}
	return common.Hash(v.snapshot.Digest())
}

// Threshold returns the prepared quorum threshold. A nil validator returns zero.
func (v *SignatureValidator) Threshold() int {
	if v == nil {
		return 0
	}
	return v.snapshot.Threshold()
}

// MatchesSigningContext reports whether two validators bind the same digest
// and exact authorized quorum. It lets a collector reject signatures when live
// chain state rotated while the ceremony was in progress.
func (v *SignatureValidator) MatchesSigningContext(other *SignatureValidator) bool {
	return v != nil && other != nil && v.snapshot.Matches(other.snapshot)
}

// ValidateSignature rejects malformed and malleable signatures before signer
// recovery, then returns the canonical authorized signer identity. Every
// rejected candidate, including an unauthorized recovery, returns zero,false.
func (v *SignatureValidator) ValidateSignature(sig []byte) (common.Address, bool) {
	if v == nil {
		return common.Address{}, false
	}
	addr, _, result := v.snapshot.ValidateCandidate(sig, v.decodeSignature, nil)
	if result != internalquorum.Authorized {
		return common.Address{}, false
	}
	return addr, true
}

func (v *SignatureValidator) decodeSignature(sig []byte) (common.Address, []byte, bool) {
	if v == nil || len(sig) != crypto.SignatureLength || sig[64] > 1 {
		return common.Address{}, nil, false
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	if !crypto.ValidateSignatureValues(sig[64], r, s, true) {
		return common.Address{}, nil, false
	}
	digest := v.snapshot.Digest()
	pub, err := crypto.SigToPub(digest[:], sig)
	if err != nil {
		return common.Address{}, nil, false
	}
	addr := crypto.PubkeyToAddress(*pub)
	return addr, sig, true
}

// ContractSignatures filters invalid, unauthorized, and duplicate candidates,
// selects the threshold lowest authorized signer addresses, and converts V from
// wire form {0,1} to Solidity form {27,28}. Selecting after the full validated
// scan is deterministic and intentionally does not preserve arrival order.
func (v *SignatureValidator) ContractSignatures(signatures [][]byte) ([][]byte, error) {
	var snapshot *internalquorum.Snapshot[common.Address]
	if v != nil {
		snapshot = v.snapshot
	}
	entries, err := snapshot.Assemble(signatures, v.decodeSignature, nil, func(a, b common.Address) bool {
		return bytes.Compare(a[:], b[:]) < 0
	})
	if err != nil {
		var limit *internalquorum.CandidateLimitError
		var below *internalquorum.BelowThresholdError
		switch {
		case errors.Is(err, internalquorum.ErrNotConfigured):
			return nil, fmt.Errorf("signature validator not configured")
		case errors.As(err, &limit):
			return nil, fmt.Errorf("too many signature candidates: %d for %d authorized signers", limit.Candidates, limit.Signers)
		case errors.As(err, &below):
			return nil, fmt.Errorf("only %d of %d authorized signatures (invalid=%d unauthorized=%d duplicate=%d)",
				below.Accepted, below.Threshold, below.Stats.Invalid, below.Stats.Unauthorized, below.Stats.Duplicate)
		default:
			return nil, err
		}
	}
	out := make([][]byte, len(entries))
	for i, entry := range entries {
		sig := entry.Payload
		sig[64] += 27
		out[i] = sig
	}
	return out, nil
}

type quorumBlockNumberReader interface {
	BlockNumber(context.Context) (uint64, error)
}

type vaultQuorumReader interface {
	Signers(*bind.CallOpts) ([]common.Address, error)
	Threshold(*bind.CallOpts) (*big.Int, error)
}

// fetchLiveQuorum reads the vault's current authorized signer set and threshold
// at one explicit block. Pinning both calls prevents a rotation between them
// from producing a signer/threshold pair that never existed on chain.
func fetchLiveQuorum(ctx context.Context, chain quorumBlockNumberReader, custody vaultQuorumReader) ([]common.Address, int, error) {
	block, err := chain.BlockNumber(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("read quorum block number: %w", err)
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(block)}
	signers, err := custody.Signers(opts)
	if err != nil {
		return nil, 0, fmt.Errorf("read signers: %w", err)
	}
	thr, err := custody.Threshold(opts)
	if err != nil {
		return nil, 0, fmt.Errorf("read threshold: %w", err)
	}
	if thr == nil || !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(signers)) {
		return nil, 0, fmt.Errorf("on-chain threshold %v out of range for %d signers", thr, len(signers))
	}
	return signers, int(thr.Int64()), nil
}

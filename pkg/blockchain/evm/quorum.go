package evm

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignatureValidator is an immutable digest, signer-set, and threshold
// snapshot for one EVM quorum operation.
type SignatureValidator struct {
	digest     common.Hash
	authorized map[common.Address]struct{}
	threshold  int
}

const maxCandidatesPerAuthorizedSigner = 4

type signatureValidation uint8

const (
	signatureInvalid signatureValidation = iota
	signatureUnauthorized
	signatureAuthorized
)

// NewSignatureValidator prepares candidate validation without any network
// access. Signatures must use the custody wire form V in {0,1}; contract form
// {27,28} is produced only by ContractSignatures.
func NewSignatureValidator(digest common.Hash, signers []common.Address, threshold int) (*SignatureValidator, error) {
	if threshold <= 0 || threshold > len(signers) {
		return nil, fmt.Errorf("threshold %d out of range for %d signers", threshold, len(signers))
	}
	authorized := make(map[common.Address]struct{}, len(signers))
	for _, signer := range signers {
		if signer == (common.Address{}) {
			return nil, fmt.Errorf("zero authorized signer")
		}
		authorized[signer] = struct{}{}
	}
	if threshold > len(authorized) {
		return nil, fmt.Errorf("threshold %d exceeds %d distinct signers", threshold, len(authorized))
	}
	return &SignatureValidator{digest: digest, authorized: authorized, threshold: threshold}, nil
}

// Digest returns the prepared operation digest. A nil validator returns zero.
func (v *SignatureValidator) Digest() common.Hash {
	if v == nil {
		return common.Hash{}
	}
	return v.digest
}

// Threshold returns the prepared quorum threshold. A nil validator returns zero.
func (v *SignatureValidator) Threshold() int {
	if v == nil {
		return 0
	}
	return v.threshold
}

// MatchesSigningContext reports whether two validators bind the same digest
// and exact authorized quorum. It lets a collector reject signatures when live
// chain state rotated while the ceremony was in progress.
func (v *SignatureValidator) MatchesSigningContext(other *SignatureValidator) bool {
	if v == nil || other == nil || v.digest != other.digest ||
		v.threshold != other.threshold || len(v.authorized) != len(other.authorized) {
		return false
	}
	for signer := range v.authorized {
		if _, ok := other.authorized[signer]; !ok {
			return false
		}
	}
	return true
}

// ValidateSignature rejects malformed and malleable signatures before signer
// recovery, then returns the canonical authorized signer identity. Every
// rejected candidate, including an unauthorized recovery, returns zero,false.
func (v *SignatureValidator) ValidateSignature(sig []byte) (common.Address, bool) {
	addr, result := v.validateSignature(sig)
	if result != signatureAuthorized {
		return common.Address{}, false
	}
	return addr, true
}

func (v *SignatureValidator) validateSignature(sig []byte) (common.Address, signatureValidation) {
	if v == nil || len(sig) != crypto.SignatureLength || sig[64] > 1 {
		return common.Address{}, signatureInvalid
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	if !crypto.ValidateSignatureValues(sig[64], r, s, true) {
		return common.Address{}, signatureInvalid
	}
	pub, err := crypto.SigToPub(v.digest[:], sig)
	if err != nil {
		return common.Address{}, signatureInvalid
	}
	addr := crypto.PubkeyToAddress(*pub)
	if _, ok := v.authorized[addr]; !ok {
		return addr, signatureUnauthorized
	}
	return addr, signatureAuthorized
}

// ContractSignatures filters invalid, unauthorized, and duplicate candidates,
// selects the threshold lowest authorized signer addresses, and converts V from
// wire form {0,1} to Solidity form {27,28}. Selecting after the full validated
// scan is deterministic and intentionally does not preserve arrival order.
func (v *SignatureValidator) ContractSignatures(signatures [][]byte) ([][]byte, error) {
	if v == nil || v.threshold <= 0 {
		return nil, fmt.Errorf("signature validator not configured")
	}
	if len(signatures) > maxCandidatesPerAuthorizedSigner*len(v.authorized) {
		return nil, fmt.Errorf("too many signature candidates: %d for %d authorized signers", len(signatures), len(v.authorized))
	}
	bySigner := make(map[common.Address][]byte, v.threshold)
	var invalid, unauthorized, duplicates int
	for _, sig := range signatures {
		addr, result := v.validateSignature(sig)
		switch result {
		case signatureInvalid:
			invalid++
			continue
		case signatureUnauthorized:
			unauthorized++
			continue
		}
		if kept, exists := bySigner[addr]; exists {
			duplicates++
			if bytes.Compare(sig, kept) < 0 {
				bySigner[addr] = append([]byte(nil), sig...)
			}
		} else {
			bySigner[addr] = append([]byte(nil), sig...)
		}
	}
	if len(bySigner) < v.threshold {
		return nil, fmt.Errorf("only %d of %d authorized signatures (invalid=%d unauthorized=%d duplicate=%d)",
			len(bySigner), v.threshold, invalid, unauthorized, duplicates)
	}
	addresses := make([]common.Address, 0, len(bySigner))
	for addr := range bySigner {
		addresses = append(addresses, addr)
	}
	sort.Slice(addresses, func(i, j int) bool { return bytes.Compare(addresses[i][:], addresses[j][:]) < 0 })
	out := make([][]byte, v.threshold)
	for i, addr := range addresses[:v.threshold] {
		sig := bySigner[addr]
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

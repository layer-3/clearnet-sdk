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

func (v *SignatureValidator) Digest() common.Hash {
	if v == nil {
		return common.Hash{}
	}
	return v.digest
}

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
// recovery, then returns the canonical authorized signer identity.
func (v *SignatureValidator) ValidateSignature(sig []byte) (common.Address, bool) {
	if v == nil || len(sig) != crypto.SignatureLength || sig[64] > 1 {
		return common.Address{}, false
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	if !crypto.ValidateSignatureValues(sig[64], r, s, true) {
		return common.Address{}, false
	}
	pub, err := crypto.SigToPub(v.digest[:], sig)
	if err != nil {
		return common.Address{}, false
	}
	addr := crypto.PubkeyToAddress(*pub)
	_, ok := v.authorized[addr]
	return addr, ok
}

// ContractSignatures filters invalid, unauthorized, and duplicate candidates,
// selects a deterministic quorum, sorts it by signer address, and converts V
// from wire form {0,1} to Solidity form {27,28}.
func (v *SignatureValidator) ContractSignatures(signatures [][]byte) ([][]byte, error) {
	if v == nil || v.threshold <= 0 {
		return nil, fmt.Errorf("signature validator not configured")
	}
	bySigner := make(map[common.Address][]byte, v.threshold)
	for _, sig := range signatures {
		addr, ok := v.ValidateSignature(sig)
		if !ok {
			continue
		}
		if kept, exists := bySigner[addr]; !exists || bytes.Compare(sig, kept) < 0 {
			bySigner[addr] = append([]byte(nil), sig...)
		}
	}
	if len(bySigner) < v.threshold {
		return nil, fmt.Errorf("only %d of %d authorized signatures", len(bySigner), v.threshold)
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

// fetchLiveQuorum reads the vault's current authorized signer set and threshold.
// The outgoing/current quorum is what authorizes both execute and updateSigners,
// so withdrawal and rotation both size and filter against it.
func fetchLiveQuorum(ctx context.Context, custody *Custody) ([]common.Address, int, error) {
	signers, err := custody.Signers(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read signers: %w", err)
	}
	thr, err := custody.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(signers)) {
		return nil, 0, fmt.Errorf("on-chain threshold %s out of range for %d signers", thr, len(signers))
	}
	return signers, int(thr.Int64()), nil
}

// mergeQuorumSigs filters the collected signatures over digest against the live
// signer set, drops duplicates and unauthorized recoveries, trims to the live
// threshold, orders by signer address (Custody.sol requires ascending, no
// duplicates), and shifts V to {27,28}. It returns the contract-ready signature
// list. Both Custody.execute (withdrawal) and Custody.updateSigners (rotation)
// share this verification shape, differing only in the digest.
func mergeQuorumSigs(digest common.Hash, signatures [][]byte, liveSigners []common.Address, liveThreshold int) ([][]byte, error) {
	validator, err := NewSignatureValidator(digest, liveSigners, liveThreshold)
	if err != nil {
		return nil, err
	}
	return validator.ContractSignatures(signatures)
}

package finality

import (
	"errors"
	"fmt"

	"github.com/layer-3/clearnet-sdk/pkg/bls"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// ValidatorSetSource provides trusted validator-set snapshots. Implementations
// may use static configuration, a registry cache, or historical storage; this
// package owns the protocol verification performed against the returned set.
type ValidatorSetSource interface {
	ValidatorSetAt(unixTime int64) (TrustedValidatorSet, error)
}

// TrustedValidatorSet is the trusted validator universe for a point in time.
type TrustedValidatorSet interface {
	Contains(pubkey []byte) bool
}

// FinalizedWithdrawalVerifier verifies the complete custody authorization
// envelope for a finalized withdrawal.
type FinalizedWithdrawalVerifier struct {
	ValidatorSets ValidatorSetSource
}

// VerifiedFinalizedWithdrawal is the trusted projection returned after all
// finalized-withdrawal checks pass.
type VerifiedFinalizedWithdrawal struct {
	Op           *core.WithdrawalOp
	Entry        core.BlockEntry
	AccountID    [32]byte
	BlockHash    [32]byte
	WithdrawalID [32]byte
	EntryIndex   uint64
	FinalizedAt  int64
}

// Verify is the complete authorization check for custody execution of a
// FinalizedWithdrawal. Callers must not authorize withdrawals by verifying the
// carried block alone.
func (v *FinalizedWithdrawalVerifier) Verify(fw *core.FinalizedWithdrawal) (*VerifiedFinalizedWithdrawal, error) {
	if fw == nil {
		return nil, errors.New("finality: nil finalized withdrawal")
	}
	if v == nil || v.ValidatorSets == nil {
		return nil, errors.New("finality: validator set source required")
	}
	if fw.Block.K == 0 || fw.Block.K > core.MaxClusterSize {
		return nil, fmt.Errorf("finality: invalid signing quorum k=%d (want 1..%d)", fw.Block.K, core.MaxClusterSize)
	}
	if fw.FinalizedAt < fw.Block.SealedAt {
		return nil, fmt.Errorf("finality: finalized_at %d before block sealed_at %d", fw.FinalizedAt, fw.Block.SealedAt)
	}

	if err := fw.Block.ValidateEntriesDigest(); err != nil {
		return nil, fmt.Errorf("finality: entries digest: %w", err)
	}
	blockHash := fw.Block.Hash()
	if blockHash != fw.BlockHash {
		return nil, errors.New("finality: block hash mismatch")
	}

	// TODO: Decide whether finalized-withdrawal verification should enforce
	// core.ValidateEntryOrder. It is protocol hardening, but may reject
	// historical producers if canonical ordering was not always enforced.

	blockSet, err := v.ValidatorSets.ValidatorSetAt(fw.Block.SealedAt)
	if err != nil {
		return nil, fmt.Errorf("finality: block validator set: %w", err)
	}
	if err := verifyAttestationWithSet(fw.Block.SigningMessage(), fw.Block.Attestation, fw.Block.K, blockSet); err != nil {
		return nil, fmt.Errorf("finality: block attestation: %w", err)
	}

	finalitySet, err := v.ValidatorSets.ValidatorSetAt(fw.FinalizedAt)
	if err != nil {
		return nil, fmt.Errorf("finality: finalized withdrawal validator set: %w", err)
	}
	if err := verifyAttestationWithSet(fw.SigningMessage(), fw.Attestation, fw.Block.K, finalitySet); err != nil {
		return nil, fmt.Errorf("finality: finalized withdrawal attestation: %w", err)
	}

	if fw.EntryIndex >= uint64(len(fw.Block.Entries)) {
		return nil, fmt.Errorf("finality: entry index %d out of range (block has %d entries)", fw.EntryIndex, len(fw.Block.Entries))
	}
	entry := fw.Block.Entries[fw.EntryIndex]
	if entry.Type != core.OpWithdrawal {
		return nil, fmt.Errorf("finality: entry type is %d, want %d (OpWithdrawal)", entry.Type, core.OpWithdrawal)
	}
	op, err := entry.DecodeWithdrawalOp()
	if err != nil {
		return nil, fmt.Errorf("finality: decode withdrawal payload: %w", err)
	}

	accountID := core.ComputeAccountID(entry.Account)
	withdrawalID := bls.ComputeVaultWithdrawalID(
		accountID,
		fw.BlockHash,
		fw.EntryIndex,
		op.AssetURI,
		op.Amount,
		op.Recipient,
		entry.Nonce,
	)
	if withdrawalID != fw.WithdrawalID {
		return nil, errors.New("finality: withdrawal id mismatch")
	}

	return &VerifiedFinalizedWithdrawal{
		Op:           op,
		Entry:        entry,
		AccountID:    accountID,
		BlockHash:    blockHash,
		WithdrawalID: withdrawalID,
		EntryIndex:   fw.EntryIndex,
		FinalizedAt:  fw.FinalizedAt,
	}, nil
}

func verifyAttestationWithSet(message []byte, att core.Attestation, k uint64, set TrustedValidatorSet) error {
	if k == 0 || k > core.MaxClusterSize {
		return fmt.Errorf("invalid signing quorum k=%d (want 1..%d)", k, core.MaxClusterSize)
	}
	if err := verifyRosterAuthorized(att.Validators, set); err != nil {
		return err
	}
	ok, err := bls.VerifyClusterSignature(
		message,
		att.ThresholdSig,
		att.Bitmask,
		uint16(k),
		att.Validators,
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("signature did not verify")
	}
	return nil
}

func verifyRosterAuthorized(validators [][]byte, set TrustedValidatorSet) error {
	if set == nil {
		return errors.New("validator set required")
	}
	if len(validators) == 0 {
		return errors.New("empty validator roster")
	}
	if len(validators) > core.MaxClusterSize {
		return fmt.Errorf("validator roster has %d entries (maximum %d)", len(validators), core.MaxClusterSize)
	}
	seen := make(map[string]int, len(validators))
	for i, pubkey := range validators {
		if len(pubkey) != core.BLSPubKeyG2Len {
			return fmt.Errorf("validator[%d] has wrong length: got %d, want %d", i, len(pubkey), core.BLSPubKeyG2Len)
		}
		key := string(pubkey)
		if first, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate validator pubkey at indices %d and %d", first, i)
		}
		seen[key] = i
		if !set.Contains(pubkey) {
			return fmt.Errorf("validator[%d] pubkey not authorized", i)
		}
	}
	return nil
}

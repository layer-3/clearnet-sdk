package finality

import (
	"errors"
	"fmt"

	"github.com/layer-3/clearnet-sdk/pkg/bls"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// TrustedValidatorChecker checks whether a BLS validator pubkey belongs to the
// current trusted validator universe. Implementations may use static
// configuration or a registry cache; this package owns the protocol
// verification performed against that trusted data.
type TrustedValidatorChecker interface {
	IsTrustedValidator(pubkey []byte) bool
}

// FinalizedWithdrawalVerifier verifies the complete custody authorization
// envelope for a finalized withdrawal.
type FinalizedWithdrawalVerifier struct {
	TrustedValidators TrustedValidatorChecker
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
	if v == nil || v.TrustedValidators == nil {
		return nil, errors.New("finality: trusted validators required")
	}
	if fw.Block.K != core.BlockSigningClusterSize {
		return nil, fmt.Errorf("finality: invalid signing quorum k=%d (want %d)", fw.Block.K, core.BlockSigningClusterSize)
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

	if err := verifyAttestationWithChecker(fw.Block.SigningMessage(), fw.Block.Attestation, fw.Block.K, v.TrustedValidators); err != nil {
		return nil, fmt.Errorf("finality: block attestation: %w", err)
	}

	if err := verifyAttestationWithChecker(fw.SigningMessage(), fw.Attestation, fw.Block.K, v.TrustedValidators); err != nil {
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

func verifyAttestationWithChecker(message []byte, att core.Attestation, k uint64, checker TrustedValidatorChecker) error {
	if k == 0 || k > core.MaxClusterSize {
		return fmt.Errorf("invalid signing quorum k=%d (want 1..%d)", k, core.MaxClusterSize)
	}
	if err := verifyRosterTrusted(att.Validators, checker); err != nil {
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

func verifyRosterTrusted(validators [][]byte, checker TrustedValidatorChecker) error {
	if checker == nil {
		return errors.New("trusted validator checker required")
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
		if !checker.IsTrustedValidator(pubkey) {
			return fmt.Errorf("validator[%d] pubkey not authorized", i)
		}
	}
	return nil
}

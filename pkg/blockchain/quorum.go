package blockchain

import "fmt"

// MajorityThresholdError reports a threshold that is not a strict majority of
// its signer set.
type MajorityThresholdError struct {
	Threshold   int
	SignerCount int
}

func (e *MajorityThresholdError) Error() string {
	return fmt.Sprintf("threshold %d must be a strict majority of %d signers", e.Threshold, e.SignerCount)
}

// ValidateMajorityThreshold applies a reusable strict-majority policy check.
// It does not imply that every chain or SDK consumer uses this policy; callers
// with a stronger or chain-specific quorum policy enforce that separately.
// Division keeps the check overflow-safe for every int value.
func ValidateMajorityThreshold(threshold, signerCount int) error {
	if signerCount <= 0 || threshold <= signerCount/2 || threshold > signerCount {
		return &MajorityThresholdError{Threshold: threshold, SignerCount: signerCount}
	}
	return nil
}

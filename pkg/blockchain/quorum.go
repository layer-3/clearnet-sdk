package blockchain

import "fmt"

// ValidateMajorityThreshold requires threshold to be a strict majority of the
// signer set. Division keeps the check overflow-safe for every int value.
func ValidateMajorityThreshold(threshold, signerCount int) error {
	if signerCount <= 0 || threshold <= signerCount/2 || threshold > signerCount {
		return fmt.Errorf("threshold %d must be a strict majority of %d signers", threshold, signerCount)
	}
	return nil
}

package bls

import (
	"errors"
	"fmt"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// VerifyBlock checks the BLS threshold signature on a sealed block.
// validators must be 128-byte serialized G2 pubkeys.
// Authorization against the on-chain NodeRegistry is the caller's
// responsibility — see pkg/blockchain/evm.BlockVerifier.authorizeValidators.
func VerifyBlock(block *core.Block, validators [][]byte) error {
	if block == nil {
		return errors.New("bls: nil block")
	}
	if block.K == 0 || block.K > core.MaxClusterSize {
		return fmt.Errorf("bls: invalid signing quorum k=%d (want 1..%d)", block.K, core.MaxClusterSize)
	}
	if len(validators) > core.MaxClusterSize {
		return fmt.Errorf("bls: validator roster has %d entries (maximum %d)", len(validators), core.MaxClusterSize)
	}
	ok, err := VerifyClusterSignature(
		block.SigningMessage(),
		block.Attestation.ThresholdSig,
		block.Attestation.Bitmask,
		uint16(block.K),
		validators,
	)
	if err != nil {
		return fmt.Errorf("bls: %w", err)
	}
	if !ok {
		return errors.New("bls: signature did not verify")
	}
	return nil
}

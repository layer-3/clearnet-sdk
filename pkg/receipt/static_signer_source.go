package receipt

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// StaticSignerSource returns a fixed signer set and threshold supplied at
// construction time. It is the production SignerSource implementation for
// the receipt verifier today: operators publish the custody signer set in
// the manifest, and every node loads the same list.
//
// When the on-chain Registry grows a custody-signers view, a
// RegistrySignerSource will replace this implementation at the call site;
// no verifier-side code changes are needed.
type StaticSignerSource struct {
	signers   []common.Address
	threshold int
}

// NewStaticSignerSource validates the inputs and returns a source that
// hands the same (signers, threshold) pair to every Load call.
func NewStaticSignerSource(signers []common.Address, threshold int) (*StaticSignerSource, error) {
	if len(signers) == 0 {
		return nil, errors.New("custody signer set is empty")
	}
	if threshold <= 0 || threshold > len(signers) {
		return nil, fmt.Errorf("custody threshold %d out of range for %d signers", threshold, len(signers))
	}
	seen := make(map[common.Address]struct{}, len(signers))
	out := make([]common.Address, 0, len(signers))
	for _, s := range signers {
		if _, dup := seen[s]; dup {
			return nil, fmt.Errorf("duplicate custody signer %s", s.Hex())
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return &StaticSignerSource{signers: out, threshold: threshold}, nil
}

// Load returns the configured signers and threshold. The slice is copied
// so callers can mutate it without affecting the source.
func (s *StaticSignerSource) Load(_ context.Context) ([]common.Address, int, error) {
	out := make([]common.Address, len(s.signers))
	copy(out, s.signers)
	return out, s.threshold, nil
}

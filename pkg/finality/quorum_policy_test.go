package finality

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

func TestExplicitVerifierPolicyIsImmutable(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	checker := newMapValidatorChecker(validators)
	if _, err := NewFinalizedWithdrawalVerifier(nil, 5); err == nil {
		t.Fatal("accepted missing trusted validators")
	}
	for _, k := range []uint64{0, core.MaxClusterSize + 1} {
		if _, err := NewFinalizedWithdrawalVerifier(checker, k); !errors.Is(err, ErrSigningClusterSize) {
			t.Fatalf("invalid policy %d: %v", k, err)
		}
	}
	v, err := NewFinalizedWithdrawalVerifier(checker, 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(fw); err != nil {
		t.Fatal(err)
	}
	one, validators := finalizedWithdrawalFixtureForK(t, 1)
	v.TrustedValidators = newMapValidatorChecker(validators)
	if _, err := v.Verify(one); !errors.Is(err, ErrSigningClusterSize) {
		t.Fatalf("K=1 accepted by immutable K=5 verifier: %v", err)
	}
}

func TestDeploymentQuorumPolicies(t *testing.T) {
	for _, k := range []uint64{1, 2, 5, 7, 64, 256} {
		t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
			fw, validators := finalizedWithdrawalFixtureForK(t, k)
			v, err := NewFinalizedWithdrawalVerifier(newMapValidatorChecker(validators), k)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := v.Verify(fw); err != nil {
				t.Fatal(err)
			}
			otherK := uint64(5)
			if k == 5 {
				otherK = 1
			}
			other, err := NewFinalizedWithdrawalVerifier(newMapValidatorChecker(validators), otherK)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := other.Verify(fw); err == nil || !strings.Contains(err.Error(), "invalid signing cluster size") {
				t.Fatalf("valid K=%d envelope accepted by K=%d policy: %v", k, otherK, err)
			}
			v.TrustedValidators = mapValidatorChecker{}
			if _, err := v.Verify(fw); err == nil || !strings.Contains(err.Error(), "not authorized") {
				t.Fatalf("untrusted validators accepted: %v", err)
			}
		})
	}
}

func TestVerifierRejectsMissingPolicy(t *testing.T) {
	for _, k := range []uint64{1, 5} {
		t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
			fw, validators := finalizedWithdrawalFixtureForK(t, k)
			v := &FinalizedWithdrawalVerifier{TrustedValidators: newMapValidatorChecker(validators)}
			if _, err := v.Verify(fw); !errors.Is(err, ErrSigningClusterSize) {
				t.Fatalf("unconfigured verifier accepted K=%d: %v", k, err)
			}
		})
	}
}

func TestFiveValidatorPolicyRequiresFourSignaturesOnBothAttestations(t *testing.T) {
	for _, target := range []string{"block", "finality"} {
		t.Run(target, func(t *testing.T) {
			fw, validators := finalizedWithdrawalFixture(t)
			if target == "block" {
				fw.Block.Attestation.Bitmask = [32]byte{}
				for i := 0; i < 3; i++ {
					core.SetBitmaskBit(&fw.Block.Attestation.Bitmask, i)
				}
				// Attestations are not part of the block hash or finality preimage.
				resignBlock(t, &fw.Block)
			} else {
				fw.Attestation.Bitmask = [32]byte{}
				for i := 0; i < 3; i++ {
					core.SetBitmaskBit(&fw.Attestation.Bitmask, i)
				}
				resignFinalizedWithdrawal(t, fw)
			}
			if err := verifyFixtureFails(fw, validators); err == nil {
				t.Fatal("three valid distinct signatures accepted for K=5")
			}
		})
	}
}

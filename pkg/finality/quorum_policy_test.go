package finality

import (
	"fmt"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

func TestDeploymentQuorumPolicies(t *testing.T) {
	for _, k := range []uint64{1, 5} {
		t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
			fw, validators := finalizedWithdrawalFixtureForK(t, k)
			v := &FinalizedWithdrawalVerifier{TrustedValidators: newMapValidatorChecker(validators), ExpectedSigningClusterSize: k}
			if _, err := v.Verify(fw); err != nil {
				t.Fatal(err)
			}
			otherK := uint64(5)
			if k == 5 {
				otherK = 1
			}
			v.ExpectedSigningClusterSize = otherK
			if _, err := v.Verify(fw); err == nil || !strings.Contains(err.Error(), "invalid signing cluster size") {
				t.Fatalf("valid K=%d envelope accepted by K=%d policy: %v", k, otherK, err)
			}
			v.ExpectedSigningClusterSize = k
			v.TrustedValidators = mapValidatorChecker{}
			if _, err := v.Verify(fw); err == nil || !strings.Contains(err.Error(), "not authorized") {
				t.Fatalf("untrusted validators accepted: %v", err)
			}
		})
	}
}

func TestLegacyVerifierDefaultsToSingleValidator(t *testing.T) {
	fw, validators := finalizedWithdrawalFixtureForK(t, 1)
	v := &FinalizedWithdrawalVerifier{TrustedValidators: newMapValidatorChecker(validators)}
	if _, err := v.Verify(fw); err != nil {
		t.Fatal(err)
	}
	fw, validators = finalizedWithdrawalFixtureForK(t, 5)
	v.TrustedValidators = newMapValidatorChecker(validators)
	if _, err := v.Verify(fw); err == nil || !strings.Contains(err.Error(), "invalid signing cluster size") {
		t.Fatalf("legacy policy accepted K=5: %v", err)
	}
}

func TestVerifierRejectsInvalidConfiguredQuorum(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	v := &FinalizedWithdrawalVerifier{TrustedValidators: newMapValidatorChecker(validators), ExpectedSigningClusterSize: core.MaxClusterSize + 1}
	if _, err := v.Verify(fw); err == nil || !strings.Contains(err.Error(), "invalid expected signing cluster size") {
		t.Fatalf("invalid configuration accepted: %v", err)
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

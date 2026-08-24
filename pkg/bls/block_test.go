package bls

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// blockFixture seals a small Block with a real BLS threshold signature over
// `block.SigningMessage()`. Returns the block and the validator pubkey set
// expected by VerifyBlock. signerIdx selects which validators sign; their
// bits are set in block.Attestation.Bitmask. The caller picks K explicitly
// (block.K — the DQE-computed signing quorum); threshold = (2k/3)+1.
func blockFixture(t *testing.T, n int, signerIdx []int, k uint64) (*core.Block, [][]byte) {
	t.Helper()

	kps := make([]*KeyPair, n)
	validators := make([][]byte, n)
	for i := range n {
		kps[i] = KeyPairFromSeed(fmt.Appendf(nil, "signer-%d", i))
		validators[i] = SerializeG2(kps[i].PublicG2)
	}

	var bitmask [32]byte
	for _, i := range signerIdx {
		core.SetBitmaskBit(&bitmask, i)
	}

	block := &core.Block{
		Anchor:   [32]byte{0xAA},
		SealedAt: 1_700_000_000,
		K:        k,
		Attestation: core.Attestation{
			Validators: validators,
			Bitmask:    bitmask,
		},
	}

	msgHash := crypto.Keccak256Hash(block.SigningMessage())

	sigmas := make([]bn254.G1Affine, 0, len(signerIdx))
	pubs := make([]bn254.G2Affine, 0, len(signerIdx))
	for _, i := range signerIdx {
		s, err := Sign(&kps[i].Secret, msgHash)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}
		sigmas = append(sigmas, s)
		pubs = append(pubs, kps[i].PublicG2)
	}

	aggSig, err := AggregateG1(sigmas)
	if err != nil {
		t.Fatalf("AggregateG1: %v", err)
	}
	apkG2, err := AggregateG2(pubs)
	if err != nil {
		t.Fatalf("AggregateG2: %v", err)
	}

	// Sealers pack tupleBitmask=0; outer Attestation.Bitmask is authoritative.
	encoded, err := EncodeSignatureForContract(new(big.Int), aggSig, apkG2)
	if err != nil {
		t.Fatalf("EncodeSignatureForContract: %v", err)
	}
	block.Attestation.ThresholdSig = encoded

	return block, validators
}

// fixedKLighthouse returns the canonical Lighthouse-stage deployment shape:
// 5 validators, K=5 fixed signing quorum, 4 signers (threshold = floor(2*5/3)+1 = 4).
// Use this for new tests that care about real-world topology rather than corner cases.
func fixedKLighthouse(t *testing.T) (*core.Block, [][]byte) {
	t.Helper()
	return blockFixture(t, 5, []int{0, 1, 2, 3}, 5)
}

// TestVerifyBlock_Valid is the end-to-end pass on the canonical Lighthouse
// shape — real signature over block.SigningMessage().
func TestVerifyBlock_Valid(t *testing.T) {
	block, validators := fixedKLighthouse(t)
	if err := VerifyBlock(block, validators); err != nil {
		t.Fatalf("expected valid block to verify: %v", err)
	}
}

// TestVerifyBlock_NilBlock pins the nil-block guard owned by this wrapper.
func TestVerifyBlock_NilBlock(t *testing.T) {
	if err := VerifyBlock(nil, nil); err == nil {
		t.Fatal("expected error on nil block")
	}
}

// TestVerifyBlock_TamperedSigningMessage flips a field that participates in
// SigningMessage(); the pairing must fail.
func TestVerifyBlock_TamperedSigningMessage(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)
	block.SealedAt++
	if err := VerifyBlock(block, validators); err == nil {
		t.Fatal("expected verification to fail after SealedAt mutation")
	}
}

// TestVerifyBlock_RejectsKZero pins the signing-quorum lower bound.
func TestVerifyBlock_RejectsKZero(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)
	block.K = 0
	err := VerifyBlock(block, validators)
	if err == nil {
		t.Fatal("F-CONSENSUS-001: VerifyBlock must reject K=0")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("k=0")) {
		t.Fatalf("expected k=0 in error, got %v", err)
	}
}

// TestVerifyBlock_RejectsKAboveBitmaskCapacity prevents the uint64 Block.K
// value from truncating when passed to VerifyClusterSignature's uint16 k.
func TestVerifyBlock_RejectsKAboveBitmaskCapacity(t *testing.T) {
	const oversizedK = 1<<16 + 1 // previously truncated to one at the uint16 conversion
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, oversizedK)

	err := VerifyBlock(block, validators)
	if err == nil {
		t.Fatal("expected rejection when K exceeds the 256-validator bitmask capacity")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("k=65537")) {
		t.Fatalf("expected rejected K value in error, got %v", err)
	}
}

// TestVerifyBlock_RejectsOversizedValidatorRoster makes the [32]byte bitmask
// contract explicit: entries above index 255 cannot be selected or bound to
// the aggregate public key.
func TestVerifyBlock_RejectsOversizedValidatorRoster(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)
	for len(validators) <= core.MaxClusterSize {
		validators = append(validators, validators[0])
	}

	err := VerifyBlock(block, validators)
	if err == nil {
		t.Fatal("expected rejection of validator roster with more than 256 entries")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("257 entries")) {
		t.Fatalf("expected roster size in error, got %v", err)
	}
}

// TestVerifyBlock_RejectsNodeIDPlaceholder substitutes a 32-byte NodeID at
// validator index 0; clearnet's verifier rejects non-128-byte entries
func TestVerifyBlock_RejectsNodeIDPlaceholder(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)
	validators[0] = bytes.Repeat([]byte{0x11}, 32)
	err := VerifyBlock(block, validators)
	if err == nil {
		t.Fatal("expected rejection of 32-byte validator entry")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("wrong length")) {
		t.Fatalf("expected length error, got %v", err)
	}
}

// TestVerifyBlock_InsufficientSigners — popcount(bitmask)=3 below threshold
// (2*8/3)+1 = 6 for k=8.
func TestVerifyBlock_InsufficientSigners(t *testing.T) {
	block, validators := blockFixture(t, 8, []int{0, 1, 2}, 8)
	if err := VerifyBlock(block, validators); err == nil {
		t.Fatal("expected rejection: 3 signers below k=8 threshold")
	}
}

// TestVerifyBlock_MultiSlotThreshold — ISSUE-035 WS-1. Shard replication
// r=8 but signing quorum k=4; three signers satisfy the threshold
// (2*4/3)+1 = 3. Pre-alignment custody rejected this when r > k.
func TestVerifyBlock_MultiSlotThreshold(t *testing.T) {
	block, validators := blockFixture(t, 8, []int{0, 1, 2}, 4)
	if err := VerifyBlock(block, validators); err != nil {
		t.Fatalf("ISSUE-035 WS-1: r=8 k=4 with 3 signers must verify: %v", err)
	}
}

// TestVerifyBlock_HighBitSigners is the regression test for the [32]byte
// bitmask alignment. n=128 validators with bits 0..127 all signing — every
// signer above index 63 was silently dropped by pre-alignment custody and
// the block rejected via ErrBitmaskTruncated.
func TestVerifyBlock_HighBitSigners(t *testing.T) {
	const n = 128
	signers := make([]int, n)
	for i := range signers {
		signers[i] = i
	}
	block, validators := blockFixture(t, n, signers, n)
	if err := VerifyBlock(block, validators); err != nil {
		t.Fatalf("128-validator cluster with signers above bit 63 must verify: %v", err)
	}
}

// TestVerifyBlock_RejectsMissingValidators — clearnet rejects len(validators)==0.
func TestVerifyBlock_RejectsMissingValidators(t *testing.T) {
	block, _ := blockFixture(t, 4, []int{0, 1, 2}, 4)
	if err := VerifyBlock(block, nil); err == nil {
		t.Fatal("expected rejection with no validators")
	}
}

// TestVerifyBlock_TamperedTupleBitmask — ISSUE-043-01. The ABI tuple's
// internal bitmask must be 0 or equal to the outer bitmask.
func TestVerifyBlock_TamperedTupleBitmask(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)

	kps := make([]*KeyPair, 4)
	for i := range 4 {
		kps[i] = KeyPairFromSeed(fmt.Appendf(nil, "signer-%d", i))
	}
	msgHash := crypto.Keccak256Hash(block.SigningMessage())
	sigmas := make([]bn254.G1Affine, 0, 3)
	pubs := make([]bn254.G2Affine, 0, 3)
	for _, i := range []int{0, 1, 2} {
		s, _ := Sign(&kps[i].Secret, msgHash)
		sigmas = append(sigmas, s)
		pubs = append(pubs, kps[i].PublicG2)
	}
	aggSig, _ := AggregateG1(sigmas)
	apkG2, _ := AggregateG2(pubs)

	var poisonedBM [32]byte
	core.SetBitmaskBit(&poisonedBM, 0)
	core.SetBitmaskBit(&poisonedBM, 1)
	core.SetBitmaskBit(&poisonedBM, 3)
	poisoned, _ := EncodeSignatureForContract(core.BitmaskToBigInt(poisonedBM), aggSig, apkG2)
	block.Attestation.ThresholdSig = poisoned

	if err := VerifyBlock(block, validators); err == nil {
		t.Fatal("expected rejection: tupleBitmask != outer bitmask")
	}
}

// TestVerifyBlock_ApkG2Mismatch swaps validator[0] for an unrelated key.
// The reconstructed apkG2 = Σ validators[i | bit set] will no longer match
// the embedded apkG2.
func TestVerifyBlock_ApkG2Mismatch(t *testing.T) {
	block, validators := blockFixture(t, 4, []int{0, 1, 2}, 4)
	forged := KeyPairFromSeed([]byte("not-a-signer"))
	validators[0] = SerializeG2(forged.PublicG2)
	if err := VerifyBlock(block, validators); err == nil {
		t.Fatal("expected apkG2-mismatch rejection after validator substitution")
	}
}

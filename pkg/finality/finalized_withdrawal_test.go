package finality

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/bls"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

type mapValidatorChecker map[string]struct{}

func newMapValidatorChecker(validators [][]byte) mapValidatorChecker {
	set := make(mapValidatorChecker, len(validators))
	for _, validator := range validators {
		set[string(validator)] = struct{}{}
	}
	return set
}

func (s mapValidatorChecker) IsTrustedValidator(pubkey []byte) bool {
	_, ok := s[string(pubkey)]
	return ok
}

func TestFinalizedWithdrawalVerifier_Valid(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	verifier := &FinalizedWithdrawalVerifier{
		TrustedValidators: newMapValidatorChecker(validators),
	}

	got, err := verifier.Verify(fw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Op == nil {
		t.Fatal("expected decoded withdrawal op")
	}
	if got.WithdrawalID != fw.WithdrawalID {
		t.Fatal("returned withdrawal ID mismatch")
	}
	if got.BlockHash != fw.BlockHash {
		t.Fatal("returned block hash mismatch")
	}
	if got.EntryIndex != fw.EntryIndex {
		t.Fatal("returned entry index mismatch")
	}
	wantEntry := fw.Block.Entries[fw.EntryIndex]
	if got.Entry.Type != wantEntry.Type ||
		got.Entry.Account != wantEntry.Account ||
		got.Entry.Nonce != wantEntry.Nonce ||
		!bytes.Equal(got.Entry.Payload, wantEntry.Payload) {
		t.Fatal("returned selected entry mismatch")
	}
}

func TestFinalizedWithdrawalVerifier_RejectsNilFinalizedWithdrawal(t *testing.T) {
	verifier := &FinalizedWithdrawalVerifier{
		TrustedValidators: mapValidatorChecker{},
	}
	if _, err := verifier.Verify(nil); err == nil {
		t.Fatal("expected nil finalized withdrawal rejection")
	}
}

func TestFinalizedWithdrawalVerifier_RejectsMissingTrustedValidators(t *testing.T) {
	fw, _ := finalizedWithdrawalFixture(t)
	if _, err := (&FinalizedWithdrawalVerifier{}).Verify(fw); err == nil {
		t.Fatal("expected missing trusted validators rejection")
	}
}

func TestFinalizedWithdrawalVerifier_RejectsEntriesDigestMismatch(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.Entries[fw.EntryIndex].Payload = (&core.WithdrawalOp{
		AssetURI:  "clearnet:evm/eip155:1/erc20:0x0000000000000000000000000000000000000002",
		Amount:    decimal.NewFromInt(2),
		Recipient: "0x2222222222222222222222222222222222222222",
	}).Encode()

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "entries digest") {
		t.Fatalf("expected entries digest rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsBlockHashMismatch(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.BlockHash[0] ^= 0x01

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "block hash mismatch") {
		t.Fatalf("expected block hash rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsBlockAttestationMismatch(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.SealedAt++
	fw.Block.EntriesDigest = core.ComputeEntriesDigest(fw.Block.Entries)
	fw.BlockHash = fw.Block.Hash()

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "block attestation") {
		t.Fatalf("expected block attestation rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsUnexpectedSigningClusterSize(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.K = core.BlockSigningClusterSize + 1
	resignBlock(t, &fw.Block)
	fw.BlockHash = fw.Block.Hash()
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "invalid signing quorum") {
		t.Fatalf("expected signing cluster size rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsFinalizedWithdrawalAttestationMismatch(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.FinalizedAt++

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "finalized withdrawal attestation") {
		t.Fatalf("expected finalized withdrawal attestation rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsUnauthorizedValidator(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	trusted := newMapValidatorChecker(validators[:len(validators)-1])
	verifier := &FinalizedWithdrawalVerifier{
		TrustedValidators: trusted,
	}

	err := verifierErr(verifier, fw)
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected unauthorized validator rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsDuplicateValidator(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Attestation.Validators[3] = append([]byte(nil), fw.Attestation.Validators[0]...)
	verifier := &FinalizedWithdrawalVerifier{
		TrustedValidators: newMapValidatorChecker(validators),
	}

	err := verifierErr(verifier, fw)
	if err == nil || !strings.Contains(err.Error(), "duplicate validator") {
		t.Fatalf("expected duplicate validator rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsEntryIndexOutOfRange(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.EntryIndex = uint64(len(fw.Block.Entries))
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected entry index rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsNonWithdrawalEntry(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.Entries[fw.EntryIndex] = core.BlockEntry{
		Type:    core.OpTransfer,
		Account: "alice",
		Nonce:   1,
		Payload: (&core.TransferOp{}).Encode(),
	}
	fw.Block.EntriesDigest = core.ComputeEntriesDigest(fw.Block.Entries)
	resignBlock(t, &fw.Block)
	fw.BlockHash = fw.Block.Hash()
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "entry type") {
		t.Fatalf("expected non-withdrawal entry rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsMalformedWithdrawalPayload(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.Entries[fw.EntryIndex].Payload = []byte{0xff}
	fw.Block.EntriesDigest = core.ComputeEntriesDigest(fw.Block.Entries)
	resignBlock(t, &fw.Block)
	fw.BlockHash = fw.Block.Hash()
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "decode withdrawal payload") {
		t.Fatalf("expected malformed payload rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsWithdrawalIDMismatch(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.WithdrawalID[0] ^= 0x01
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "withdrawal id mismatch") {
		t.Fatalf("expected withdrawal ID rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsReSignedPayloadWithStaleWithdrawalID(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.Block.Entries[fw.EntryIndex].Payload = (&core.WithdrawalOp{
		AssetURI:      "clearnet:evm/eip155:1/erc20:0x0000000000000000000000000000000000000001",
		Amount:        decimal.NewFromInt(456),
		Recipient:     "0x1111111111111111111111111111111111111111",
		UserSignature: []byte("user-signature"),
	}).Encode()
	fw.Block.EntriesDigest = core.ComputeEntriesDigest(fw.Block.Entries)
	resignBlock(t, &fw.Block)
	fw.BlockHash = fw.Block.Hash()
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "withdrawal id mismatch") {
		t.Fatalf("expected stale withdrawal ID rejection, got %v", err)
	}
}

func TestFinalizedWithdrawalVerifier_RejectsFinalizedBeforeSealed(t *testing.T) {
	fw, validators := finalizedWithdrawalFixture(t)
	fw.FinalizedAt = fw.Block.SealedAt - 1
	resignFinalizedWithdrawal(t, fw)

	err := verifyFixtureFails(fw, validators)
	if err == nil || !strings.Contains(err.Error(), "before block sealed_at") {
		t.Fatalf("expected finalized-before-sealed rejection, got %v", err)
	}
}

func verifyFixtureFails(fw *core.FinalizedWithdrawal, validators [][]byte) error {
	verifier := &FinalizedWithdrawalVerifier{
		TrustedValidators: newMapValidatorChecker(validators),
	}
	return verifierErr(verifier, fw)
}

func verifierErr(verifier *FinalizedWithdrawalVerifier, fw *core.FinalizedWithdrawal) error {
	_, err := verifier.Verify(fw)
	return err
}

var fixtureKeyPairs []*bls.KeyPair

func finalizedWithdrawalFixture(t *testing.T) (*core.FinalizedWithdrawal, [][]byte) {
	t.Helper()

	const n = 5
	fixtureKeyPairs = make([]*bls.KeyPair, n)
	validators := make([][]byte, n)
	for i := range n {
		fixtureKeyPairs[i] = bls.KeyPairFromSeed(fmt.Appendf(nil, "finality-signer-%d", i))
		validators[i] = bls.SerializeG2(fixtureKeyPairs[i].PublicG2)
	}

	op := &core.WithdrawalOp{
		AssetURI:      "clearnet:evm/eip155:1/erc20:0x0000000000000000000000000000000000000001",
		Amount:        decimal.NewFromInt(123),
		Recipient:     "0x1111111111111111111111111111111111111111",
		UserSignature: []byte("user-signature"),
	}
	entry := core.BlockEntry{
		Type:    core.OpWithdrawal,
		Account: "did:pkh:eip155:1:0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Nonce:   42,
		Payload: op.Encode(),
	}
	block := core.Block{
		Anchor:        [32]byte{0xaa},
		SealedAt:      1_700_000_000,
		Entries:       []core.BlockEntry{entry},
		EntriesDigest: core.ComputeEntriesDigest([]core.BlockEntry{entry}),
		StateRoot:     [32]byte{0xbb},
		K:             core.BlockSigningClusterSize,
		Attestation: core.Attestation{
			Validators: cloneValidators(validators),
		},
	}
	setFixtureBitmask(&block.Attestation)
	resignBlock(t, &block)

	blockHash := block.Hash()
	withdrawalID := bls.ComputeVaultWithdrawalID(
		core.ComputeAccountID(entry.Account),
		blockHash,
		0,
		op.AssetURI,
		op.Amount,
		op.Recipient,
		entry.Nonce,
	)
	fw := &core.FinalizedWithdrawal{
		WithdrawalID: withdrawalID,
		BlockHash:    blockHash,
		EntryIndex:   0,
		FinalizedAt:  block.SealedAt + 60,
		Block:        block,
		Attestation: core.Attestation{
			Validators: cloneValidators(validators),
		},
	}
	setFixtureBitmask(&fw.Attestation)
	resignFinalizedWithdrawal(t, fw)

	return fw, validators
}

func setFixtureBitmask(att *core.Attestation) {
	att.Bitmask = [32]byte{}
	for _, i := range []int{0, 1, 2, 3} {
		core.SetBitmaskBit(&att.Bitmask, i)
	}
}

func resignBlock(t *testing.T, block *core.Block) {
	t.Helper()
	signAttestation(t, block.SigningMessage(), &block.Attestation)
}

func resignFinalizedWithdrawal(t *testing.T, fw *core.FinalizedWithdrawal) {
	t.Helper()
	signAttestation(t, fw.SigningMessage(), &fw.Attestation)
}

func signAttestation(t *testing.T, message []byte, att *core.Attestation) {
	t.Helper()
	msgHash := crypto.Keccak256Hash(message)
	sigmas := make([]bn254.G1Affine, 0, 4)
	pubs := make([]bn254.G2Affine, 0, 4)
	for _, i := range []int{0, 1, 2, 3} {
		sigma, err := bls.Sign(&fixtureKeyPairs[i].Secret, msgHash)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}
		sigmas = append(sigmas, sigma)
		pubs = append(pubs, fixtureKeyPairs[i].PublicG2)
	}
	aggSig, err := bls.AggregateG1(sigmas)
	if err != nil {
		t.Fatalf("AggregateG1: %v", err)
	}
	aggPub, err := bls.AggregateG2(pubs)
	if err != nil {
		t.Fatalf("AggregateG2: %v", err)
	}
	encoded, err := bls.EncodeSignatureForContract(new(big.Int), aggSig, aggPub)
	if err != nil {
		t.Fatalf("EncodeSignatureForContract: %v", err)
	}
	att.ThresholdSig = encoded
}

func cloneValidators(validators [][]byte) [][]byte {
	out := make([][]byte, len(validators))
	for i := range validators {
		out[i] = append([]byte(nil), validators[i]...)
	}
	return out
}

func TestVerifyRosterAuthorized_RejectsWrongLength(t *testing.T) {
	err := verifyRosterTrusted([][]byte{bytes.Repeat([]byte{0x11}, 32)}, mapValidatorChecker{})
	if err == nil || !strings.Contains(err.Error(), "wrong length") {
		t.Fatalf("expected wrong-length rejection, got %v", err)
	}
}

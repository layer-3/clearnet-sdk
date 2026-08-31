package evm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNewSignatureValidatorGuards(t *testing.T) {
	a1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	a2 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	tests := []struct {
		name      string
		signers   []common.Address
		threshold int
		wantErr   bool
	}{
		{name: "zero threshold", signers: []common.Address{a1}, threshold: 0, wantErr: true},
		{name: "threshold equals signer count", signers: []common.Address{a1, a2}, threshold: 2},
		{name: "threshold exceeds signer count", signers: []common.Address{a1, a2}, threshold: 3, wantErr: true},
		{name: "zero signer", signers: []common.Address{a1, {}}, threshold: 1, wantErr: true},
		{name: "dedup collapses below threshold", signers: []common.Address{a1, a1}, threshold: 2, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSignatureValidator(common.Hash{1}, tc.signers, tc.threshold)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewSignatureValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSignatureValidatorRejectsPoisonBeforeSignerDedup(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := crypto.Keccak256Hash([]byte("iss-050"))
	low, err := crypto.Sign(digest[:], key)
	if err != nil {
		t.Fatal(err)
	}
	high := append([]byte(nil), low...)
	s := new(big.Int).SetBytes(high[32:64])
	s.Sub(crypto.S256().Params().N, s)
	s.FillBytes(high[32:64])
	high[64] ^= 1

	addr := crypto.PubkeyToAddress(key.PublicKey)
	v, err := NewSignatureValidator(digest, []common.Address{addr}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.ValidateSignature(high); ok {
		t.Fatal("high-S malleated signature accepted")
	}
	if got, ok := v.ValidateSignature(low); !ok || got != addr {
		t.Fatalf("canonical signature rejected: got %s ok=%v", got, ok)
	}
	contract, err := v.ContractSignatures([][]byte{high, low})
	if err != nil {
		t.Fatal(err)
	}
	if len(contract) != 1 || contract[0][64] != low[64]+27 {
		t.Fatalf("contract signatures = %#v", contract)
	}
}

func TestSignatureValidatorRejectsWrongWireVAndWrongDigest(t *testing.T) {
	key, _ := crypto.GenerateKey()
	digest := crypto.Keccak256Hash([]byte("right"))
	sig, _ := crypto.Sign(crypto.Keccak256([]byte("wrong")), key)
	v, err := NewSignatureValidator(digest, []common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := v.ValidateSignature(sig); ok || got != (common.Address{}) {
		t.Fatalf("wrong-digest signature returned %s, %v", got, ok)
	}
	sig[64] = 27
	if _, ok := v.ValidateSignature(sig); ok {
		t.Fatal("contract-form V accepted on mesh")
	}
}

func TestSignatureValidatorPoisonBelowThresholdReturnsDiagnosticError(t *testing.T) {
	k1, _ := crypto.GenerateKey()
	k2, _ := crypto.GenerateKey()
	digest := crypto.Keccak256Hash([]byte("below-threshold"))
	valid, _ := crypto.Sign(digest[:], k1)
	v, _ := NewSignatureValidator(digest, []common.Address{
		crypto.PubkeyToAddress(k1.PublicKey), crypto.PubkeyToAddress(k2.PublicKey),
	}, 2)
	got, err := v.ContractSignatures([][]byte{valid, make([]byte, 65)})
	if err == nil || got != nil {
		t.Fatalf("ContractSignatures() = %x, %v; want nil error result", got, err)
	}
	if !strings.Contains(err.Error(), "only 1 of 2") || !strings.Contains(err.Error(), "invalid=1") {
		t.Fatalf("diagnostic error = %v", err)
	}
}

func TestSignatureValidatorSnapshotEqualityIncludesRosterThresholdAndDigest(t *testing.T) {
	k1, _ := crypto.GenerateKey()
	k2, _ := crypto.GenerateKey()
	k3, _ := crypto.GenerateKey()
	a1 := crypto.PubkeyToAddress(k1.PublicKey)
	a2 := crypto.PubkeyToAddress(k2.PublicKey)
	a3 := crypto.PubkeyToAddress(k3.PublicKey)
	digest := crypto.Keccak256Hash([]byte("snapshot"))
	base, _ := NewSignatureValidator(digest, []common.Address{a1, a2}, 1)
	reordered, _ := NewSignatureValidator(digest, []common.Address{a2, a1}, 1)
	changedThreshold, _ := NewSignatureValidator(digest, []common.Address{a1, a2}, 2)
	changedDigest, _ := NewSignatureValidator(crypto.Keccak256Hash([]byte("other")), []common.Address{a1, a2}, 1)
	changedSameSizeRoster, _ := NewSignatureValidator(digest, []common.Address{a1, a3}, 1)
	if !base.MatchesSigningContext(reordered) {
		t.Fatal("equivalent signer set did not match")
	}
	if base.MatchesSigningContext(changedThreshold) || base.MatchesSigningContext(changedDigest) ||
		base.MatchesSigningContext(changedSameSizeRoster) {
		t.Fatal("changed quorum or digest matched snapshot")
	}
}

func TestSignatureValidatorNilAndCandidateLimitGuards(t *testing.T) {
	var nilValidator *SignatureValidator
	if _, err := nilValidator.ContractSignatures(nil); err == nil {
		t.Fatal("nil validator assembled signatures")
	}
	if nilValidator.Digest() != (common.Hash{}) || nilValidator.Threshold() != 0 ||
		nilValidator.MatchesSigningContext(nil) {
		t.Fatal("nil receiver contract changed")
	}
	a1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	v, err := NewSignatureValidator(common.Hash{1}, []common.Address{a1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.ContractSignatures(make([][]byte, maxCandidatesPerAuthorizedSigner+1)); err == nil {
		t.Fatal("oversized candidate set accepted")
	}
}

func randomRecoverableSignature(t *testing.T, digest []byte, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	r, s, err := ecdsa.Sign(rand.Reader, key, digest)
	if err != nil {
		t.Fatal(err)
	}
	if s.Cmp(new(big.Int).Rsh(new(big.Int).Set(crypto.S256().Params().N), 1)) > 0 {
		s.Sub(crypto.S256().Params().N, s)
	}
	sig := make([]byte, crypto.SignatureLength)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:64])
	want := crypto.PubkeyToAddress(key.PublicKey)
	for recoveryID := byte(0); recoveryID <= 1; recoveryID++ {
		sig[64] = recoveryID
		pub, recoverErr := crypto.SigToPub(digest, sig)
		if recoverErr == nil && crypto.PubkeyToAddress(*pub) == want {
			return append([]byte(nil), sig...)
		}
	}
	t.Fatal("could not determine recovery ID")
	return nil
}

func TestSignatureValidatorDuplicateTieBreakIsOrderIndependent(t *testing.T) {
	key, _ := crypto.GenerateKey()
	digest := crypto.Keccak256Hash([]byte("duplicate-tie-break"))
	s1 := randomRecoverableSignature(t, digest[:], key)
	s2 := randomRecoverableSignature(t, digest[:], key)
	for bytes.Equal(s1, s2) {
		s2 = randomRecoverableSignature(t, digest[:], key)
	}
	v, _ := NewSignatureValidator(digest, []common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	forward, err := v.ContractSignatures([][]byte{s1, s2})
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := v.ContractSignatures([][]byte{s2, s1})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(forward[0], reverse[0]) {
		t.Fatal("duplicate tie-break depends on arrival order")
	}
}

type fixedBlockNumber uint64

func (n fixedBlockNumber) BlockNumber(context.Context) (uint64, error) { return uint64(n), nil }

type recordingVaultQuorum struct {
	signersBlock   *big.Int
	thresholdBlock *big.Int
}

func (r *recordingVaultQuorum) Signers(opts *bind.CallOpts) ([]common.Address, error) {
	r.signersBlock = new(big.Int).Set(opts.BlockNumber)
	return []common.Address{common.HexToAddress("0x0000000000000000000000000000000000000001")}, nil
}

func (r *recordingVaultQuorum) Threshold(opts *bind.CallOpts) (*big.Int, error) {
	r.thresholdBlock = new(big.Int).Set(opts.BlockNumber)
	return big.NewInt(1), nil
}

func TestFetchLiveQuorumPinsBothReadsToOneBlock(t *testing.T) {
	reader := &recordingVaultQuorum{}
	if _, _, err := fetchLiveQuorum(context.Background(), fixedBlockNumber(42), reader); err != nil {
		t.Fatal(err)
	}
	if reader.signersBlock.Cmp(big.NewInt(42)) != 0 || reader.thresholdBlock.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("reads used blocks %v and %v, want 42", reader.signersBlock, reader.thresholdBlock)
	}
}

func TestSignatureValidatorAssemblyIsArrivalOrderIndependent(t *testing.T) {
	k1, _ := crypto.GenerateKey()
	k2, _ := crypto.GenerateKey()
	digest := crypto.Keccak256Hash([]byte("deterministic"))
	s1, _ := crypto.Sign(digest[:], k1)
	s2, _ := crypto.Sign(digest[:], k2)
	v, _ := NewSignatureValidator(digest, []common.Address{
		crypto.PubkeyToAddress(k1.PublicKey), crypto.PubkeyToAddress(k2.PublicKey),
	}, 2)
	forward, err := v.ContractSignatures([][]byte{s1, s2})
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := v.ContractSignatures([][]byte{s2, s1})
	if err != nil {
		t.Fatal(err)
	}
	if len(forward) != 2 || !bytes.Equal(forward[0], reverse[0]) || !bytes.Equal(forward[1], reverse[1]) {
		t.Fatal("EVM quorum assembly depends on arrival order")
	}
}

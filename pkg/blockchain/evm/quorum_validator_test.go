package evm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

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
	if _, ok := v.ValidateSignature(sig); ok {
		t.Fatal("wrong-digest signature accepted")
	}
	sig[64] = 27
	if _, ok := v.ValidateSignature(sig); ok {
		t.Fatal("contract-form V accepted on mesh")
	}
}

func TestSignatureValidatorSnapshotEqualityIncludesRosterThresholdAndDigest(t *testing.T) {
	k1, _ := crypto.GenerateKey()
	k2, _ := crypto.GenerateKey()
	a1 := crypto.PubkeyToAddress(k1.PublicKey)
	a2 := crypto.PubkeyToAddress(k2.PublicKey)
	digest := crypto.Keccak256Hash([]byte("snapshot"))
	base, _ := NewSignatureValidator(digest, []common.Address{a1, a2}, 1)
	reordered, _ := NewSignatureValidator(digest, []common.Address{a2, a1}, 1)
	changedThreshold, _ := NewSignatureValidator(digest, []common.Address{a1, a2}, 2)
	changedDigest, _ := NewSignatureValidator(crypto.Keccak256Hash([]byte("other")), []common.Address{a1, a2}, 1)
	if !base.MatchesSigningContext(reordered) {
		t.Fatal("equivalent signer set did not match")
	}
	if base.MatchesSigningContext(changedThreshold) || base.MatchesSigningContext(changedDigest) {
		t.Fatal("changed quorum or digest matched snapshot")
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

package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

type preparedOperationFixture struct {
	rpc            *preparedTestRPC
	rotations      []*RotationFinalizer
	consolidations []*ConsolidationFinalizer
	newSigners     []string
	store          *stubVaultStore
}

type countingPreparedSigner struct {
	sign.Signer
	calls int
}

func (s *countingPreparedSigner) Sign(ctx context.Context, message []byte) ([]byte, error) {
	s.calls++
	return s.Signer.Sign(ctx, message)
}

func newPreparedOperationFixture(t *testing.T) *preparedOperationFixture {
	t.Helper()
	net := &chaincfg.RegressionNetParams
	signers := []*sign.KeySigner{
		p2wpkhTestKeySigner(t, 11),
		p2wpkhTestKeySigner(t, 12),
		p2wpkhTestKeySigner(t, 13),
	}
	pubkeys := make([][]byte, len(signers))
	for i := range signers {
		pubkeys[i] = signers[i].PublicKey()
	}
	redeem, err := RedeemScript(2, pubkeys)
	if err != nil {
		t.Fatal(err)
	}
	vault, err := VaultAddress(redeem, net)
	if err != nil {
		t.Fatal(err)
	}
	vaultScript, err := PkScript(vault)
	if err != nil {
		t.Fatal(err)
	}
	vaultScriptHex := hex.EncodeToString(vaultScript)

	unspent := make([]Unspent, 3)
	rpc := &preparedTestRPC{
		unspent:    unspent,
		txOuts:     make(map[string]*TxOut, len(unspent)),
		txOutsLive: true,
		feeRate:    2,
	}
	for i := range unspent {
		var hash chainhash.Hash
		hash[0], hash[31] = byte(0x30+i), byte(0xa0+i)
		unspent[i] = Unspent{
			TxID: hash.String(), Vout: uint32(i), AmountSats: int64(80_000 + i*10_000),
			Confirmations: 10, ScriptPubKey: vaultScriptHex,
		}
		rpc.txOuts[preparedOutpointKey(unspent[i].TxID, unspent[i].Vout)] = &TxOut{
			AmountSats: unspent[i].AmountSats, ScriptPubKey: vaultScriptHex, Confirmations: unspent[i].Confirmations,
		}
	}
	rpc.unspent = unspent
	store := &stubVaultStore{pubkeys: pubkeys, threshold: 2}
	cfg := Config{ConfirmationDepth: 2, FeeConfTarget: 6, FallbackFeeRate: 2, FeeCapSatPerVByte: 10}

	fixture := &preparedOperationFixture{
		rpc:            rpc,
		rotations:      make([]*RotationFinalizer, len(signers)),
		consolidations: make([]*ConsolidationFinalizer, len(signers)),
		store:          store,
	}
	for i := range signers {
		fixture.rotations[i], err = NewRotationFinalizer(net, rpc, signers[i], store, cfg, NewAssetResolver())
		if err != nil {
			t.Fatalf("NewRotationFinalizer[%d]: %v", i, err)
		}
		fixture.consolidations[i], err = NewConsolidationFinalizer(net, rpc, signers[i], store, cfg, NewAssetResolver())
		if err != nil {
			t.Fatalf("NewConsolidationFinalizer[%d]: %v", i, err)
		}
	}
	for _, keyByte := range []byte{21, 22, 23} {
		fixture.newSigners = append(fixture.newSigners, hex.EncodeToString(p2wpkhTestKeySigner(t, keyByte).PublicKey()))
	}
	return fixture
}

func TestPreparedOperationTypesAreStructurallyDistinct(t *testing.T) {
	withdrawal := reflect.TypeFor[PreparedWithdrawal]().Field(1).Name
	rotation := reflect.TypeFor[PreparedRotation]().Field(1).Name
	consolidation := reflect.TypeFor[PreparedConsolidation]().Field(1).Name
	if withdrawal == rotation || withdrawal == consolidation || rotation == consolidation {
		t.Fatal("prepared operation types have identical structures")
	}
}

func TestPreparedOperationsRejectWrongAuthorizationBeforeSigner(t *testing.T) {
	t.Run("withdrawal", func(t *testing.T) {
		fixture := newPreparedFixture(t)
		prepared := fixture.prepare(t)
		counter := &countingPreparedSigner{Signer: fixture.finalizers[0].signer}
		fixture.finalizers[0].signer = counter
		wrong := fixture.authorization()
		wrong.WithdrawalID[0] ^= 1
		if _, err := fixture.finalizers[0].SignPrepared(context.Background(), prepared, wrong); err == nil {
			t.Fatal("wrong authorization signed")
		}
		if counter.calls != 0 {
			t.Fatalf("signer calls = %d, want 0", counter.calls)
		}
	})

	t.Run("rotation", func(t *testing.T) {
		fixture := newPreparedOperationFixture(t)
		auth := RotationAuthorization{OperationID: [32]byte{1}, Signers: fixture.newSigners, Threshold: 2}
		packed, err := fixture.rotations[0].Pack(context.Background(), auth.OperationID, auth.Signers, auth.Threshold)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := fixture.rotations[0].Prepare(context.Background(), packed, auth)
		if err != nil {
			t.Fatal(err)
		}
		counter := &countingPreparedSigner{Signer: fixture.rotations[0].signer}
		fixture.rotations[0].signer = counter
		wrong := auth
		wrong.OperationID[0] ^= 1
		if _, err := fixture.rotations[0].SignPrepared(context.Background(), prepared, wrong); err == nil {
			t.Fatal("wrong authorization signed")
		}
		if counter.calls != 0 {
			t.Fatalf("signer calls = %d, want 0", counter.calls)
		}
	})

	t.Run("consolidation", func(t *testing.T) {
		fixture := newPreparedOperationFixture(t)
		target, err := fixture.consolidations[0].TargetVaultIdentity(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		auth := ConsolidationAuthorization{OperationID: [32]byte{2}, TargetVaultIdentity: target}
		packed, err := fixture.consolidations[0].Pack(context.Background(), auth.OperationID)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := fixture.consolidations[0].Prepare(context.Background(), packed, auth)
		if err != nil {
			t.Fatal(err)
		}
		counter := &countingPreparedSigner{Signer: fixture.consolidations[0].signer}
		fixture.consolidations[0].signer = counter
		wrong := auth
		wrong.OperationID[0] ^= 1
		if _, err := fixture.consolidations[0].SignPrepared(context.Background(), prepared, wrong); err == nil {
			t.Fatal("wrong authorization signed")
		}
		if counter.calls != 0 {
			t.Fatalf("signer calls = %d, want 0", counter.calls)
		}
	})
}

func TestPreparedWithdrawalRejectsSignerOutsideSnapshottedVault(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	outsider := p2wpkhTestKeySigner(t, 40)
	counter := &countingPreparedSigner{Signer: outsider}
	outsiderKeys := [][]byte{outsider.PublicKey(), p2wpkhTestKeySigner(t, 41).PublicKey()}
	finalizer, err := NewWithdrawalFinalizer(
		fixture.finalizers[0].net, fixture.rpc, counter, outsiderKeys, 2,
		fixture.finalizers[0].cfg, fixture.finalizers[0].assets,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := finalizer.SignPrepared(context.Background(), prepared, fixture.authorization()); err == nil {
		t.Fatal("signer outside snapshotted vault signed prepared withdrawal")
	}
	if counter.calls != 0 {
		t.Fatalf("outsider signer calls = %d, want 0", counter.calls)
	}
}

type preparedOperationFinalizer[P, A any] interface {
	SignPrepared(context.Context, *P, A) ([]byte, error)
	VerifySharePrepared(*P, A, []byte) error
	FinalizePrepared(*P, A, [][]byte) ([]byte, string, error)
	VerifyFinalizedPrepared(*P, A, []byte) error
	BroadcastPrepared(context.Context, *P, A, []byte) (string, error)
}

func testPreparedOperationOfflineFlow[P, A any](t *testing.T, rpc *preparedTestRPC, finalizers []preparedOperationFinalizer[P, A], prepared, wrongPrepared *P, auth A, transaction func(*P) *preparedTransaction) {
	t.Helper()
	ctx := context.Background()
	getTxOutCalls := len(rpc.getTxOutCalls)

	mutatePreparedTransactionTx(t, transaction(wrongPrepared), func(tx *wire.MsgTx) { tx.TxOut[0].PkScript[2] ^= 1 })
	resealPreparedTransaction(t, transaction(wrongPrepared))
	if _, err := finalizers[0].SignPrepared(ctx, wrongPrepared, auth); err == nil {
		t.Fatal("SignPrepared accepted an operation-invalid body")
	}

	rpc.txOutsLive = false
	shares := make([][]byte, len(finalizers))
	for i := range finalizers {
		share, err := finalizers[i].SignPrepared(ctx, prepared, auth)
		if err != nil {
			t.Fatalf("SignPrepared[%d] after inputs disappeared: %v", i, err)
		}
		shares[i] = share
		if err := finalizers[0].VerifySharePrepared(prepared, auth, shares[i]); err != nil {
			t.Fatalf("VerifySharePrepared[%d] after inputs disappeared: %v", i, err)
		}
	}

	invalid := decodePreparedShare(t, shares[0])
	invalidSignature, err := hex.DecodeString(invalid.Sigs[0])
	if err != nil {
		t.Fatal(err)
	}
	invalidSignature[len(invalidSignature)/2] ^= 1
	invalid.Sigs[0] = hex.EncodeToString(invalidSignature)
	invalidRaw := encodePreparedShare(t, invalid)
	if err := finalizers[0].VerifySharePrepared(prepared, auth, invalidRaw); err == nil {
		t.Fatal("VerifySharePrepared accepted an invalid signature")
	}
	if _, _, err := finalizers[0].FinalizePrepared(prepared, auth, [][]byte{shares[0], shares[0], invalidRaw}); err == nil {
		t.Fatal("FinalizePrepared counted invalid or duplicate shares")
	}

	rawA, txID, err := finalizers[0].FinalizePrepared(prepared, auth, [][]byte{invalidRaw, shares[2], shares[0], shares[0], shares[1]})
	if err != nil {
		t.Fatalf("FinalizePrepared after inputs disappeared: %v", err)
	}
	rawB, txIDB, err := finalizers[0].FinalizePrepared(prepared, auth, [][]byte{shares[1], shares[0], shares[2], shares[2]})
	if err != nil {
		t.Fatalf("FinalizePrepared permutation: %v", err)
	}
	if txID != transaction(prepared).expectedTxID || txIDB != txID || !bytes.Equal(rawA, rawB) {
		t.Fatal("FinalizePrepared was not deterministic")
	}
	if err := finalizers[0].VerifyFinalizedPrepared(prepared, auth, rawA); err != nil {
		t.Fatalf("VerifyFinalizedPrepared: %v", err)
	}
	if got, err := finalizers[0].BroadcastPrepared(ctx, prepared, auth, rawA); err != nil || got != txID {
		t.Fatalf("BroadcastPrepared = %q, %v", got, err)
	}
	corruptWitness := append([]byte(nil), rawA...)
	corruptWitness[len(corruptWitness)-10] ^= 1
	sends := len(rpc.sentRaw)
	if _, err := finalizers[0].BroadcastPrepared(ctx, prepared, auth, corruptWitness); err == nil {
		t.Fatal("BroadcastPrepared accepted witness corruption with the same txid")
	}
	if len(rpc.sentRaw) != sends {
		t.Fatal("BroadcastPrepared sent witness corruption to RPC")
	}
	if len(rpc.getTxOutCalls) != getTxOutCalls {
		t.Fatalf("prepared operations called GetTxOut after capture: %d -> %d", getTxOutCalls, len(rpc.getTxOutCalls))
	}
}

func TestRotationPreparedOperation(t *testing.T) {
	fixture := newPreparedOperationFixture(t)
	ctx := context.Background()
	opID := [32]byte{0x72, 0x6f, 0x74}
	auth := RotationAuthorization{OperationID: opID, Signers: fixture.newSigners, Threshold: 2}
	packed, err := fixture.rotations[0].Pack(ctx, opID, fixture.newSigners, 2)
	if err != nil {
		t.Fatal(err)
	}

	wrongTarget, err := deserializeTx(packed)
	if err != nil {
		t.Fatal(err)
	}
	wrongTarget.TxOut[0].PkScript = append([]byte(nil), wrongTarget.TxOut[1].PkScript...)
	wrongTargetRaw, err := serializeTx(wrongTarget)
	if err != nil {
		t.Fatal(err)
	}
	if prepared, err := fixture.rotations[0].Prepare(ctx, wrongTargetRaw, auth); err == nil || prepared != nil {
		t.Fatal("Prepare accepted a rotation with the wrong target")
	}
	if len(fixture.rpc.getTxOutCalls) != 0 {
		t.Fatal("wrong rotation target reached prevout capture")
	}

	incomplete, err := deserializeTx(packed)
	if err != nil {
		t.Fatal(err)
	}
	incomplete.TxIn = incomplete.TxIn[:len(incomplete.TxIn)-1]
	incompleteRaw, err := serializeTx(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	if prepared, err := fixture.rotations[0].Prepare(ctx, incompleteRaw, auth); err == nil || prepared != nil {
		t.Fatal("Prepare accepted an incomplete rotation")
	}
	if len(fixture.rpc.getTxOutCalls) != 0 {
		t.Fatal("incomplete rotation reached prevout capture")
	}

	prepared, err := fixture.rotations[0].Prepare(ctx, packed, auth)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(fixture.rpc.getTxOutCalls) != len(rotationTransaction(prepared).inputs) {
		t.Fatalf("GetTxOut calls = %d, want %d", len(fixture.rpc.getTxOutCalls), len(rotationTransaction(prepared).inputs))
	}
	encoded, err := prepared.MarshalBinary()
	if err != nil {
		t.Fatalf("rotation MarshalBinary: %v", err)
	}
	var decodedRotation PreparedRotation
	if err := decodedRotation.UnmarshalBinary(encoded); err != nil {
		t.Fatalf("rotation UnmarshalBinary: %v", err)
	}
	var wrongType PreparedConsolidation
	if err := wrongType.UnmarshalBinary(encoded); err == nil {
		t.Fatal("consolidation accepted a rotation prepared context")
	}
	finalizers := make([]preparedOperationFinalizer[PreparedRotation, RotationAuthorization], len(fixture.rotations))
	for i := range fixture.rotations {
		finalizers[i] = fixture.rotations[i]
	}
	wrongPrepared := &PreparedRotation{preparedTransaction: clonePreparedTransaction(rotationTransaction(prepared))}
	fixture.store.pubkeys = [][]byte{p2wpkhTestKeySigner(t, 31).PublicKey(), p2wpkhTestKeySigner(t, 32).PublicKey()}
	fixture.store.threshold = 2
	testPreparedOperationOfflineFlow(t, fixture.rpc, finalizers, prepared, wrongPrepared, auth, rotationTransaction)
}

func TestConsolidationPreparedOperation(t *testing.T) {
	fixture := newPreparedOperationFixture(t)
	ctx := context.Background()
	consolidationID := [32]byte{0x66, 0x6f, 0x6c, 0x64}
	target, err := fixture.consolidations[0].TargetVaultIdentity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	auth := ConsolidationAuthorization{OperationID: consolidationID, TargetVaultIdentity: target}
	packed, err := fixture.consolidations[0].Pack(ctx, consolidationID)
	if err != nil {
		t.Fatal(err)
	}

	wrongBody, err := deserializeTx(packed)
	if err != nil {
		t.Fatal(err)
	}
	wrongBody.TxOut[0].PkScript = append([]byte(nil), wrongBody.TxOut[1].PkScript...)
	wrongBodyRaw, err := serializeTx(wrongBody)
	if err != nil {
		t.Fatal(err)
	}
	if prepared, err := fixture.consolidations[0].Prepare(ctx, wrongBodyRaw, auth); err == nil || prepared != nil {
		t.Fatal("Prepare accepted a consolidation with the wrong body")
	}
	if len(fixture.rpc.getTxOutCalls) != 0 {
		t.Fatal("wrong consolidation body reached prevout capture")
	}

	prepared, err := fixture.consolidations[0].Prepare(ctx, packed, auth)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(fixture.rpc.getTxOutCalls) != len(consolidationTransaction(prepared).inputs) {
		t.Fatalf("GetTxOut calls = %d, want %d", len(fixture.rpc.getTxOutCalls), len(consolidationTransaction(prepared).inputs))
	}
	encoded, err := prepared.MarshalBinary()
	if err != nil {
		t.Fatalf("consolidation MarshalBinary: %v", err)
	}
	var decodedConsolidation PreparedConsolidation
	if err := decodedConsolidation.UnmarshalBinary(encoded); err != nil {
		t.Fatalf("consolidation UnmarshalBinary: %v", err)
	}
	finalizers := make([]preparedOperationFinalizer[PreparedConsolidation, ConsolidationAuthorization], len(fixture.consolidations))
	for i := range fixture.consolidations {
		finalizers[i] = fixture.consolidations[i]
	}
	wrongPrepared := &PreparedConsolidation{preparedTransaction: clonePreparedTransaction(consolidationTransaction(prepared))}
	fixture.store.pubkeys = [][]byte{p2wpkhTestKeySigner(t, 31).PublicKey(), p2wpkhTestKeySigner(t, 32).PublicKey()}
	fixture.store.threshold = 2
	testPreparedOperationOfflineFlow(t, fixture.rpc, finalizers, prepared, wrongPrepared, auth, consolidationTransaction)
}

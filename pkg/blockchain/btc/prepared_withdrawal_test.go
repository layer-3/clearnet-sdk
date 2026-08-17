package btc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

type preparedGetTxOutCall struct {
	txID           string
	vout           uint32
	includeMempool bool
}

type preparedTestRPC struct {
	unspent       []Unspent
	txOuts        map[string]*TxOut
	txOutsLive    bool
	getTxOutCalls []preparedGetTxOutCall
	feeRate       int64

	sendErr    error
	sendResult *string
	sentRaw    [][]byte

	rawTx      *RawTx
	rawErr     error
	rawLookups []string
}

func (r *preparedTestRPC) ListUnspent(context.Context, int, []string) ([]Unspent, error) {
	return append([]Unspent(nil), r.unspent...), nil
}

func (r *preparedTestRPC) GetTxOut(_ context.Context, txID string, vout uint32, includeMempool bool) (*TxOut, error) {
	r.getTxOutCalls = append(r.getTxOutCalls, preparedGetTxOutCall{txID: txID, vout: vout, includeMempool: includeMempool})
	if !r.txOutsLive {
		return nil, nil
	}
	out := r.txOuts[preparedOutpointKey(txID, vout)]
	if out == nil {
		return nil, nil
	}
	copy := *out
	return &copy, nil
}

func (r *preparedTestRPC) SendRawTransaction(_ context.Context, rawHex string) (string, error) {
	raw, err := hex.DecodeString(rawHex)
	if err != nil {
		return "", err
	}
	r.sentRaw = append(r.sentRaw, append([]byte(nil), raw...))
	if r.sendErr != nil {
		return "", r.sendErr
	}
	if r.sendResult != nil {
		return *r.sendResult, nil
	}
	tx, err := deserializeExactTx(raw)
	if err != nil {
		return "", err
	}
	return tx.TxHash().String(), nil
}

func (r *preparedTestRPC) EstimateSmartFeeSatPerVByte(context.Context, int, int64) (int64, error) {
	return r.feeRate, nil
}

func (r *preparedTestRPC) GetBlockCount(context.Context) (int64, error) { return 0, nil }
func (r *preparedTestRPC) GetBlockHash(context.Context, int64) (string, error) {
	return "", nil
}
func (r *preparedTestRPC) GetBlockTxids(context.Context, string) ([]string, error) {
	return nil, nil
}
func (r *preparedTestRPC) GetRawTransaction(_ context.Context, txID string) (*RawTx, error) {
	r.rawLookups = append(r.rawLookups, txID)
	if r.rawTx == nil {
		return nil, r.rawErr
	}
	copy := *r.rawTx
	return &copy, r.rawErr
}

func preparedOutpointKey(txID string, vout uint32) string {
	return fmt.Sprintf("%s:%d", txID, vout)
}

type preparedFixture struct {
	rpc          *preparedTestRPC
	finalizers   []*WithdrawalFinalizer
	signers      []*sign.KeySigner
	pubkeys      [][]byte
	op           *core.WithdrawalOp
	withdrawalID [32]byte
	deadline     int64
	canonical    []byte
}

func newPreparedFixture(t *testing.T) *preparedFixture {
	t.Helper()
	net := &chaincfg.RegressionNetParams
	signers := []*sign.KeySigner{
		p2wpkhTestKeySigner(t, 1),
		p2wpkhTestKeySigner(t, 2),
		p2wpkhTestKeySigner(t, 3),
	}
	pubkeys := make([][]byte, len(signers))
	for i := range signers {
		pubkeys[i] = signers[i].PublicKey()
	}
	redeem, err := RedeemScript(2, pubkeys)
	if err != nil {
		t.Fatalf("RedeemScript: %v", err)
	}
	vault, err := VaultAddress(redeem, net)
	if err != nil {
		t.Fatalf("VaultAddress: %v", err)
	}
	vaultScript, err := PkScript(vault)
	if err != nil {
		t.Fatalf("vault PkScript: %v", err)
	}
	recipientKey := p2wpkhTestKeySigner(t, 9)
	recipient, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(recipientKey.PublicKey()), net)
	if err != nil {
		t.Fatalf("recipient address: %v", err)
	}

	var firstHash, secondHash chainhash.Hash
	firstHash[0], firstHash[31] = 0x11, 0xa1
	secondHash[0], secondHash[31] = 0x22, 0xb2
	unspent := []Unspent{
		{TxID: firstHash.String(), Vout: 3, AmountSats: 80_000, Confirmations: 12, ScriptPubKey: hex.EncodeToString(vaultScript)},
		{TxID: secondHash.String(), Vout: 7, AmountSats: 70_000, Confirmations: 9, ScriptPubKey: hex.EncodeToString(vaultScript)},
	}
	rpc := &preparedTestRPC{
		unspent:    unspent,
		txOuts:     make(map[string]*TxOut, len(unspent)),
		txOutsLive: true,
		feeRate:    2,
	}
	for _, u := range unspent {
		rpc.txOuts[preparedOutpointKey(u.TxID, u.Vout)] = &TxOut{
			AmountSats: u.AmountSats, ScriptPubKey: u.ScriptPubKey, Confirmations: u.Confirmations,
		}
	}
	cfg := Config{
		ConfirmationDepth: 2,
		FeeConfTarget:     6,
		FallbackFeeRate:   2,
		FeeCapSatPerVByte: 10,
	}
	finalizers := make([]*WithdrawalFinalizer, len(signers))
	for i := range signers {
		finalizers[i], err = NewWithdrawalFinalizer(net, rpc, signers[i], pubkeys, 2, cfg, NewAssetResolver())
		if err != nil {
			t.Fatalf("NewWithdrawalFinalizer[%d]: %v", i, err)
		}
	}
	fixture := &preparedFixture{
		rpc:          rpc,
		finalizers:   finalizers,
		signers:      signers,
		pubkeys:      pubkeys,
		op:           &core.WithdrawalOp{Recipient: recipient.EncodeAddress(), AssetURI: "yellow://ynet/asset/custody/btc/0/0", Amount: decimal.New(100_000, -8)},
		withdrawalID: [32]byte{0xba, 0xdd, 0xca, 0xfe},
		deadline:     2_000_000_000,
	}
	fixture.canonical, err = finalizers[0].Pack(context.Background(), fixture.op, fixture.withdrawalID, fixture.deadline)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	return fixture
}

func (f *preparedFixture) prepare(t *testing.T) *PreparedWithdrawal {
	t.Helper()
	prepared, err := f.finalizers[0].Prepare(context.Background(), f.canonical, f.authorization())
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	return prepared
}

func (f *preparedFixture) authorization() WithdrawalAuthorization {
	return WithdrawalAuthorization{Operation: f.op, WithdrawalID: f.withdrawalID, Deadline: f.deadline}
}

func clonePrepared(prepared *PreparedWithdrawal) *PreparedWithdrawal {
	return &PreparedWithdrawal{preparedTransaction: clonePreparedTransaction(withdrawalTransaction(prepared))}
}

func clonePreparedTransaction(prepared *preparedTransaction) preparedTransaction {
	clone := *prepared
	clone.canonicalTx = append([]byte(nil), prepared.canonicalTx...)
	clone.inputs = clonePreparedInputs(prepared.inputs)
	return clone
}

func mutatePreparedTx(t *testing.T, prepared *PreparedWithdrawal, mutate func(*wire.MsgTx)) {
	mutatePreparedTransactionTx(t, withdrawalTransaction(prepared), mutate)
}

func mutatePreparedTransactionTx(t *testing.T, prepared *preparedTransaction, mutate func(*wire.MsgTx)) {
	t.Helper()
	tx, err := deserializeTx(prepared.canonicalTx)
	if err != nil {
		t.Fatalf("deserialize prepared tx: %v", err)
	}
	mutate(tx)
	prepared.canonicalTx, err = serializeTx(tx)
	if err != nil {
		t.Fatalf("serialize mutated prepared tx: %v", err)
	}
	prepared.expectedTxID = tx.TxHash().String()
}

func resealPrepared(t *testing.T, prepared *PreparedWithdrawal) {
	resealPreparedTransaction(t, withdrawalTransaction(prepared))
}

func resealPreparedTransaction(t *testing.T, prepared *preparedTransaction) {
	t.Helper()
	if err := sealPreparedTransaction(prepared); err != nil {
		t.Fatalf("seal prepared withdrawal: %v", err)
	}
}

func decodePreparedShare(t *testing.T, raw []byte) SigShare {
	t.Helper()
	var share SigShare
	if err := json.Unmarshal(raw, &share); err != nil {
		t.Fatalf("decode share: %v", err)
	}
	return share
}

func encodePreparedShare(t *testing.T, share SigShare) []byte {
	t.Helper()
	raw, err := json.Marshal(share)
	if err != nil {
		t.Fatalf("encode share: %v", err)
	}
	return raw
}

func TestPreparedWithdrawalPrepareAndValidate(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	tx, err := deserializeTx(fixture.canonical)
	if err != nil {
		t.Fatal(err)
	}

	if prepared.FormatVersion() != PreparedWithdrawalFormatVersion {
		t.Fatalf("FormatVersion = %d, want %d", prepared.FormatVersion(), PreparedWithdrawalFormatVersion)
	}
	if !bytes.Equal(prepared.CanonicalBytes(), fixture.canonical) {
		t.Fatal("Prepare did not preserve the exact canonical bytes")
	}
	if prepared.ExpectedTxID() != tx.TxHash().String() {
		t.Fatalf("ExpectedTxID = %s, want %s", prepared.ExpectedTxID(), tx.TxHash())
	}
	inputs := prepared.Inputs()
	if len(inputs) != len(tx.TxIn) {
		t.Fatalf("prepared inputs = %d, want %d", len(inputs), len(tx.TxIn))
	}
	if len(fixture.rpc.getTxOutCalls) != len(tx.TxIn) {
		t.Fatalf("GetTxOut calls = %d, want exactly %d", len(fixture.rpc.getTxOutCalls), len(tx.TxIn))
	}
	for i, txIn := range tx.TxIn {
		call := fixture.rpc.getTxOutCalls[i]
		if call.txID != txIn.PreviousOutPoint.Hash.String() || call.vout != txIn.PreviousOutPoint.Index || !call.includeMempool {
			t.Errorf("GetTxOut call %d = %#v, want %s:%d includeMempool=true", i, call, txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index)
		}
		input := inputs[i]
		want := fixture.rpc.txOuts[preparedOutpointKey(call.txID, call.vout)]
		if input.PreviousTxID != call.txID || input.PreviousVout != call.vout || input.AmountSats != want.AmountSats || input.Confirmations != want.Confirmations {
			t.Errorf("prepared input %d does not contain the complete ordered prevout: %#v", i, input)
		}
		wantScript, _ := hex.DecodeString(want.ScriptPubKey)
		if !bytes.Equal(input.PreviousOutputScript, wantScript) {
			t.Errorf("prepared input %d previous-output script mismatch", i)
		}
		if len(input.WitnessScript) == 0 {
			t.Errorf("prepared input %d has no witness script", i)
		}
	}
	body, err := marshalPreparedTransactionBody(withdrawalTransaction(prepared))
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256.Sum256(body); got != prepared.IntegrityHash() {
		t.Fatalf("IntegrityHash = %x, want SHA-256 %x", prepared.IntegrityHash(), got)
	}

	calls := len(fixture.rpc.getTxOutCalls)
	if err := fixture.finalizers[0].ValidatePrepared(prepared, fixture.authorization()); err != nil {
		t.Fatalf("ValidatePrepared: %v", err)
	}
	wrongDeadline := fixture.authorization()
	wrongDeadline.Deadline += 12345
	if err := fixture.finalizers[0].ValidatePrepared(prepared, wrongDeadline); err == nil {
		t.Fatal("ValidatePrepared accepted a different authorization deadline")
	}
	if len(fixture.rpc.getTxOutCalls) != calls {
		t.Fatalf("ValidatePrepared made RPC calls: %d -> %d", calls, len(fixture.rpc.getTxOutCalls))
	}

	wrongRecipient := *fixture.op
	wrongRecipient.Recipient = p2wpkhTestRecipient(t, &chaincfg.RegressionNetParams)
	wrongAmount := *fixture.op
	wrongAmount.Amount = decimal.New(100_001, -8)
	wrongAsset := *fixture.op
	wrongAsset.AssetURI = "yellow://ynet/asset/custody/btc/0/1"
	wrongID := fixture.withdrawalID
	wrongID[31] ^= 1
	checks := []struct {
		name string
		op   *core.WithdrawalOp
		id   [32]byte
	}{
		{name: "recipient", op: &wrongRecipient, id: fixture.withdrawalID},
		{name: "amount", op: &wrongAmount, id: fixture.withdrawalID},
		{name: "asset", op: &wrongAsset, id: fixture.withdrawalID},
		{name: "withdrawal marker", op: fixture.op, id: wrongID},
	}
	for _, check := range checks {
		t.Run("authorization_"+check.name, func(t *testing.T) {
			if err := fixture.finalizers[0].ValidatePrepared(prepared, WithdrawalAuthorization{Operation: check.op, WithdrawalID: check.id, Deadline: fixture.deadline}); err == nil {
				t.Fatal("ValidatePrepared accepted mismatched authorization")
			}
		})
	}
}

func TestPreparedWithdrawalRejectsMutations(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	recipientScript := append([]byte(nil), mustPreparedTx(t, prepared).TxOut[0].PkScript...)

	tests := []struct {
		name   string
		mutate func(*PreparedWithdrawal)
		reseal bool
	}{
		{name: "integrity hash", mutate: func(p *PreparedWithdrawal) { p.inputs[0].AmountSats++ }},
		{name: "format version", mutate: func(p *PreparedWithdrawal) { p.formatVersion++ }},
		{name: "expected txid", mutate: func(p *PreparedWithdrawal) { p.expectedTxID = strings.Repeat("0", 64) }, reseal: true},
		{name: "ordered outpoint", mutate: func(p *PreparedWithdrawal) { p.inputs[0].PreviousVout++ }, reseal: true},
		{name: "incomplete input records", mutate: func(p *PreparedWithdrawal) { p.inputs = p.inputs[:1] }, reseal: true},
		{name: "duplicate input outpoint", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.TxIn[1].PreviousOutPoint = tx.TxIn[0].PreviousOutPoint })
			p.inputs[1].PreviousTxID = p.inputs[0].PreviousTxID
			p.inputs[1].PreviousVout = p.inputs[0].PreviousVout
		}, reseal: true},
		{name: "p2wsh relationship", mutate: func(p *PreparedWithdrawal) { p.inputs[0].PreviousOutputScript[2] ^= 1 }, reseal: true},
		{name: "witness script", mutate: func(p *PreparedWithdrawal) { p.inputs[0].WitnessScript[0] ^= 1 }, reseal: true},
		{name: "non-positive input value", mutate: func(p *PreparedWithdrawal) { p.inputs[0].AmountSats = 0 }},
		{name: "confirmations", mutate: func(p *PreparedWithdrawal) { p.inputs[0].Confirmations = 1 }, reseal: true},
		{name: "fixed version", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.Version++ })
		}, reseal: true},
		{name: "fixed locktime", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.LockTime = 1 })
		}, reseal: true},
		{name: "fixed sequence", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.TxIn[0].Sequence-- })
		}, reseal: true},
		{name: "recipient", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) {
				tx.TxOut[0].PkScript = append([]byte(nil), tx.TxOut[len(tx.TxOut)-1].PkScript...)
			})
		}, reseal: true},
		{name: "amount", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.TxOut[0].Value++ })
		}, reseal: true},
		{name: "marker", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) {
				wrong := fixture.withdrawalID
				wrong[1] ^= 1
				tx.TxOut[len(tx.TxOut)-1].PkScript, _ = txscript.NullDataScript(wrong[:])
			})
		}, reseal: true},
		{name: "change", mutate: func(p *PreparedWithdrawal) {
			mutatePreparedTx(t, p, func(tx *wire.MsgTx) { tx.TxOut[1].PkScript = append([]byte(nil), recipientScript...) })
		}, reseal: true},
		{name: "fee ceiling", mutate: func(p *PreparedWithdrawal) { p.inputs[0].AmountSats += 10_000 }, reseal: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := clonePrepared(prepared)
			test.mutate(mutated)
			if test.reseal {
				resealPrepared(t, mutated)
			}
			if err := fixture.finalizers[0].ValidatePrepared(mutated, fixture.authorization()); err == nil {
				t.Fatal("ValidatePrepared accepted mutated context")
			}
		})
	}
}

func mustPreparedTx(t *testing.T, prepared *PreparedWithdrawal) *wire.MsgTx {
	t.Helper()
	tx, err := deserializeTx(prepared.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestPreparedWithdrawalBinaryRoundTripAndMalformed(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	encoded, err := prepared.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// This golden fingerprint is intentionally independent of the decoder. It
	// catches an encoder/decoder pair drifting together and silently making
	// persisted prepared envelopes unreadable across binary upgrades.
	const goldenLength = 1079
	wantGoldenHash, err := hex.DecodeString("c43b9bf982496b46a47fd2177369a04f2f2e2dab0dca620fd47860f9d6496ae8")
	if err != nil {
		t.Fatal(err)
	}
	gotGoldenHash := sha256.Sum256(encoded)
	if len(encoded) != goldenLength || !bytes.Equal(gotGoldenHash[:], wantGoldenHash) {
		t.Fatalf("prepared binary drift: len=%d sha256=%x", len(encoded), gotGoldenHash)
	}
	if string(encoded[:len(preparedWithdrawalMagic)]) != preparedWithdrawalMagic {
		t.Fatalf("binary magic = %q", encoded[:len(preparedWithdrawalMagic)])
	}
	if version := binary.BigEndian.Uint16(encoded[len(preparedWithdrawalMagic):]); version != PreparedWithdrawalFormatVersion {
		t.Fatalf("binary version = %d", version)
	}
	if !bytes.Contains(encoded, prepared.CanonicalBytes()) {
		t.Fatal("binary form does not contain the exact canonical transaction")
	}

	var decoded PreparedWithdrawal
	if err := decoded.UnmarshalBinary(encoded); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !reflect.DeepEqual(decoded, *prepared) {
		t.Fatalf("binary round trip mismatch\ngot:  %#v\nwant: %#v", decoded, *prepared)
	}
	reencoded, err := decoded.MarshalBinary()
	if err != nil {
		t.Fatalf("round-trip MarshalBinary: %v", err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatal("binary encoding is not stable across a round trip")
	}

	corruptHash := append([]byte(nil), encoded...)
	corruptHash[len(corruptHash)-1] ^= 1
	badMagic := append([]byte(nil), encoded...)
	badMagic[0] ^= 1
	rehashPreparedEnvelope(badMagic)
	badVersion := append([]byte(nil), encoded...)
	binary.BigEndian.PutUint16(badVersion[len(preparedWithdrawalMagic):], PreparedWithdrawalFormatVersion+1)
	rehashPreparedEnvelope(badVersion)
	badLength := append([]byte(nil), encoded...)
	binary.BigEndian.PutUint32(badLength[preparedWithdrawalHeaderBytes:], maxPreparedCanonicalBytes+1)
	rehashPreparedEnvelope(badLength)
	bodyWithTrailing := append([]byte(nil), encoded[:len(encoded)-preparedWithdrawalHashBytes]...)
	bodyWithTrailing = append(bodyWithTrailing, 0)
	trailing := append(bodyWithTrailing, make([]byte, preparedWithdrawalHashBytes)...)
	rehashPreparedEnvelope(trailing)

	malformed := []struct {
		name string
		data []byte
	}{
		{name: "nil", data: nil},
		{name: "truncated", data: encoded[:len(encoded)-1]},
		{name: "corrupt integrity", data: corruptHash},
		{name: "bad magic", data: badMagic},
		{name: "unsupported version", data: badVersion},
		{name: "oversized canonical length", data: badLength},
		{name: "trailing body data", data: trailing},
	}
	for _, test := range malformed {
		t.Run(test.name, func(t *testing.T) {
			before := decoded
			if err := decoded.UnmarshalBinary(test.data); err == nil {
				t.Fatal("UnmarshalBinary accepted malformed data")
			}
			if !reflect.DeepEqual(decoded, before) {
				t.Fatal("failed UnmarshalBinary modified its receiver")
			}
		})
	}

	mutated := clonePrepared(prepared)
	mutated.inputs[0].AmountSats++
	if _, err := mutated.MarshalBinary(); err == nil {
		t.Fatal("MarshalBinary accepted a context with a stale integrity hash")
	}
}

func rehashPreparedEnvelope(encoded []byte) {
	body := encoded[:len(encoded)-preparedWithdrawalHashBytes]
	hash := sha256.Sum256(body)
	copy(encoded[len(body):], hash[:])
}

func TestPreparedWithdrawalOfflineSharesAndDeterministicFinalization(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	getTxOutCalls := len(fixture.rpc.getTxOutCalls)

	wrongPrepared := clonePrepared(prepared)
	mutatePreparedTx(t, wrongPrepared, func(tx *wire.MsgTx) {
		tx.TxOut[0].Value--
		tx.TxOut[1].Value++
	})
	resealPrepared(t, wrongPrepared)
	if _, err := fixture.finalizers[0].SignPrepared(context.Background(), wrongPrepared, fixture.authorization()); err == nil {
		t.Fatal("SignPrepared accepted an operation-invalid body")
	}

	fixture.rpc.txOutsLive = false
	shares := make([][]byte, len(fixture.finalizers))
	for i, finalizer := range fixture.finalizers {
		shareRaw, err := finalizer.SignPrepared(context.Background(), prepared, fixture.authorization())
		if err != nil {
			t.Fatalf("SignPrepared[%d] after UTXOs disappeared: %v", i, err)
		}
		shares[i] = shareRaw
		share := decodePreparedShare(t, shares[i])
		for inputIndex, encoded := range share.Sigs {
			withHashType, err := hex.DecodeString(encoded)
			if err != nil {
				t.Fatalf("decode SignPrepared[%d] input %d: %v", i, inputIndex, err)
			}
			if withHashType[len(withHashType)-1] != byte(txscript.SigHashAll) {
				t.Fatalf("SignPrepared[%d] input %d did not append SIGHASH_ALL", i, inputIndex)
			}
			parsed, err := ecdsa.ParseDERSignature(withHashType[:len(withHashType)-1])
			if err != nil || !bytes.Equal(parsed.Serialize(), withHashType[:len(withHashType)-1]) {
				t.Fatalf("SignPrepared[%d] input %d did not emit canonical DER: %v", i, inputIndex, err)
			}
			s := parsed.S()
			if s.IsOverHalfOrder() {
				t.Fatalf("SignPrepared[%d] input %d emitted high-S", i, inputIndex)
			}
		}
		if err := fixture.finalizers[0].VerifySharePrepared(prepared, fixture.authorization(), shares[i]); err != nil {
			t.Fatalf("VerifySharePrepared[%d]: %v", i, err)
		}
	}
	if len(fixture.rpc.getTxOutCalls) != getTxOutCalls {
		t.Fatalf("offline signing/verification called GetTxOut: %d -> %d", getTxOutCalls, len(fixture.rpc.getTxOutCalls))
	}

	invalid := decodePreparedShare(t, shares[0])
	invalidSig, _ := hex.DecodeString(invalid.Sigs[0])
	invalidSig[8] ^= 1
	invalid.Sigs[0] = hex.EncodeToString(invalidSig)
	invalidRaw := encodePreparedShare(t, invalid)
	invalidSecond := decodePreparedShare(t, shares[0])
	invalidSecondSig, _ := hex.DecodeString(invalidSecond.Sigs[1])
	invalidSecondSig[len(invalidSecondSig)-3] ^= 1
	invalidSecond.Sigs[1] = hex.EncodeToString(invalidSecondSig)
	invalidSecondRaw := encodePreparedShare(t, invalidSecond)
	unknown := decodePreparedShare(t, shares[0])
	unknown.PubKey = hex.EncodeToString(p2wpkhTestKeySigner(t, 8).PublicKey())
	unknownRaw := encodePreparedShare(t, unknown)
	wrongCount := decodePreparedShare(t, shares[0])
	wrongCount.Sigs = wrongCount.Sigs[:1]
	wrongCountRaw := encodePreparedShare(t, wrongCount)
	wrongSighash := decodePreparedShare(t, shares[0])
	wrongSighashBytes, _ := hex.DecodeString(wrongSighash.Sigs[0])
	wrongSighashBytes[len(wrongSighashBytes)-1] = byte(txscript.SigHashNone)
	wrongSighash.Sigs[0] = hex.EncodeToString(wrongSighashBytes)
	wrongSighashRaw := encodePreparedShare(t, wrongSighash)
	badDER := decodePreparedShare(t, shares[0])
	badDER.Sigs[0] = "0001"
	badDERRaw := encodePreparedShare(t, badDER)
	upperHex := decodePreparedShare(t, shares[0])
	upperHex.Sigs[0] = strings.ToUpper(upperHex.Sigs[0])
	upperHexRaw := encodePreparedShare(t, upperHex)
	valid := decodePreparedShare(t, shares[0])
	extraFieldRaw := []byte(fmt.Sprintf(`{"pubkey":%q,"sigs":%s,"extra":true}`, valid.PubKey, mustPreparedJSON(t, valid.Sigs)))
	malformed := []byte(`{"pubkey":`)

	for name, share := range map[string][]byte{
		"invalid first-input signature":  invalidRaw,
		"invalid second-input signature": invalidSecondRaw,
		"unknown signer":                 unknownRaw,
		"wrong signature count":          wrongCountRaw,
		"wrong sighash byte":             wrongSighashRaw,
		"invalid DER":                    badDERRaw,
		"non-canonical hex":              upperHexRaw,
		"unknown JSON field":             extraFieldRaw,
		"malformed":                      malformed,
	} {
		t.Run("verify rejects "+name, func(t *testing.T) {
			if err := fixture.finalizers[0].VerifySharePrepared(prepared, fixture.authorization(), share); err == nil {
				t.Fatal("VerifySharePrepared accepted invalid share")
			}
		})
	}
	if _, _, err := fixture.finalizers[0].FinalizePrepared(prepared, fixture.authorization(), [][]byte{shares[0], shares[0], invalidRaw, unknownRaw}); err == nil {
		t.Fatal("FinalizePrepared counted duplicate or invalid shares toward threshold")
	}

	noisyShares := [][]byte{malformed, shares[2], shares[0], invalidRaw, shares[1], shares[0], unknownRaw}
	rawA, txID, err := fixture.finalizers[0].FinalizePrepared(prepared, fixture.authorization(), noisyShares)
	if err != nil {
		t.Fatalf("FinalizePrepared after UTXOs disappeared: %v", err)
	}
	if err := fixture.finalizers[0].VerifyFinalizedPrepared(prepared, fixture.authorization(), rawA); err != nil {
		t.Fatalf("VerifyFinalizedPrepared: %v", err)
	}
	corrupt := append([]byte(nil), rawA...)
	corrupt[len(corrupt)-10] ^= 1
	if err := fixture.finalizers[0].VerifyFinalizedPrepared(prepared, fixture.authorization(), corrupt); err == nil {
		t.Fatal("VerifyFinalizedPrepared accepted corrupted witness bytes")
	}
	rawB, txIDB, err := fixture.finalizers[0].FinalizePrepared(prepared, fixture.authorization(), [][]byte{shares[1], shares[0], shares[2], shares[2], malformed})
	if err != nil {
		t.Fatalf("FinalizePrepared permutation: %v", err)
	}
	if txID != prepared.ExpectedTxID() || txIDB != prepared.ExpectedTxID() {
		t.Fatalf("final txids = %s, %s; want %s", txID, txIDB, prepared.ExpectedTxID())
	}
	if !bytes.Equal(rawA, rawB) {
		t.Fatal("FinalizePrepared witness depends on share arrival order")
	}
	if len(fixture.rpc.getTxOutCalls) != getTxOutCalls {
		t.Fatalf("offline finalization called GetTxOut: %d -> %d", getTxOutCalls, len(fixture.rpc.getTxOutCalls))
	}

	finalTx, err := deserializeTx(rawA)
	if err != nil {
		t.Fatal(err)
	}
	unsignedTx := mustPreparedTx(t, prepared)
	if finalTx.TxHash() != unsignedTx.TxHash() || finalTx.TxHash().String() != prepared.ExpectedTxID() {
		t.Fatalf("SegWit txid changed after witness assembly: unsigned=%s finalized=%s", unsignedTx.TxHash(), finalTx.TxHash())
	}
	if finalTx.WitnessHash() == finalTx.TxHash() {
		t.Fatal("finalized SegWit wtxid unexpectedly equals txid")
	}

	orderedShares := append([][]byte(nil), shares...)
	sort.Slice(orderedShares, func(i, j int) bool {
		left := decodePreparedShare(t, orderedShares[i])
		right := decodePreparedShare(t, orderedShares[j])
		return fixture.finalizers[0].pubkeyPos[left.PubKey] < fixture.finalizers[0].pubkeyPos[right.PubKey]
	})
	for inputIndex, txIn := range finalTx.TxIn {
		if len(txIn.Witness) != fixture.finalizers[0].threshold+2 || len(txIn.Witness[0]) != 0 || !bytes.Equal(txIn.Witness[len(txIn.Witness)-1], prepared.Inputs()[inputIndex].WitnessScript) {
			t.Fatalf("input %d malformed deterministic witness: %#v", inputIndex, txIn.Witness)
		}
		for signerIndex := 0; signerIndex < fixture.finalizers[0].threshold; signerIndex++ {
			share := decodePreparedShare(t, orderedShares[signerIndex])
			want, _ := hex.DecodeString(share.Sigs[inputIndex])
			if !bytes.Equal(txIn.Witness[signerIndex+1], want) {
				t.Errorf("input %d witness signature %d is not in redeem-script key order", inputIndex, signerIndex)
			}
		}
	}

	_, prevFetcher, err := validatePreparedLocal(withdrawalTransaction(prepared))
	if err != nil {
		t.Fatal(err)
	}
	sigHashes := txscript.NewTxSigHashes(finalTx, prevFetcher)
	for i, input := range prepared.Inputs() {
		engine, err := txscript.NewEngine(input.PreviousOutputScript, finalTx, i, txscript.StandardVerifyFlags, nil, sigHashes, input.AmountSats, prevFetcher)
		if err != nil {
			t.Fatalf("NewEngine input %d: %v", i, err)
		}
		if err := engine.Execute(); err != nil {
			t.Fatalf("assembled witness input %d failed script validation: %v", i, err)
		}
	}
}

func mustPreparedJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test JSON: %v", err)
	}
	return string(raw)
}

func TestPreparedWithdrawalRejectsHighSSharesAndSignerOutput(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	shareRaw, err := fixture.finalizers[0].SignPrepared(context.Background(), prepared, fixture.authorization())
	if err != nil {
		t.Fatal(err)
	}
	share := decodePreparedShare(t, shareRaw)
	withHashType, _ := hex.DecodeString(share.Sigs[0])
	parsed, err := ecdsa.ParseDERSignature(withHashType[:len(withHashType)-1])
	if err != nil {
		t.Fatal(err)
	}
	r, s := parsed.R(), parsed.S()
	s.Negate()
	if !s.IsOverHalfOrder() {
		t.Fatal("test failed to construct a high-S signature")
	}
	highS := marshalPreparedDERSignature(t, r.Bytes(), s.Bytes())
	share.Sigs[0] = hex.EncodeToString(append(highS, byte(txscript.SigHashAll)))
	if err := fixture.finalizers[0].VerifySharePrepared(prepared, fixture.authorization(), encodePreparedShare(t, share)); err == nil || !strings.Contains(err.Error(), "high-S") {
		t.Fatalf("VerifySharePrepared high-S error = %v", err)
	}

	highSigner := &preparedHighSSigner{Signer: fixture.signers[0]}
	highFinalizer, err := NewWithdrawalFinalizer(&chaincfg.RegressionNetParams, fixture.rpc, highSigner, fixture.pubkeys, 2, fixture.finalizers[0].cfg, NewAssetResolver())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := highFinalizer.SignPrepared(context.Background(), prepared, fixture.authorization()); err == nil || !strings.Contains(err.Error(), "high-S") {
		t.Fatalf("SignPrepared high-S signer error = %v", err)
	}
}

type preparedHighSSigner struct {
	sign.Signer
}

func (s *preparedHighSSigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	der, err := s.Signer.Sign(ctx, digest)
	if err != nil {
		return nil, err
	}
	parsed, err := ecdsa.ParseDERSignature(der)
	if err != nil {
		return nil, err
	}
	r, scalar := parsed.R(), parsed.S()
	scalar.Negate()
	return marshalPreparedDERSignatureValue(r.Bytes(), scalar.Bytes())
}

func marshalPreparedDERSignature(t *testing.T, r, s [32]byte) []byte {
	t.Helper()
	der, err := marshalPreparedDERSignatureValue(r, s)
	if err != nil {
		t.Fatalf("marshal DER signature: %v", err)
	}
	return der
}

func marshalPreparedDERSignatureValue(r, s [32]byte) ([]byte, error) {
	return asn1.Marshal(struct {
		R *big.Int
		S *big.Int
	}{R: new(big.Int).SetBytes(r[:]), S: new(big.Int).SetBytes(s[:])})
}

func TestPreparedWithdrawalBroadcastPrepared(t *testing.T) {
	fixture := newPreparedFixture(t)
	prepared := fixture.prepare(t)
	shares := make([][]byte, 2)
	var err error
	for i := range shares {
		shares[i], err = fixture.finalizers[i].SignPrepared(context.Background(), prepared, fixture.authorization())
		if err != nil {
			t.Fatal(err)
		}
	}
	raw, expectedTxID, err := fixture.finalizers[0].FinalizePrepared(prepared, fixture.authorization(), shares)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), raw); err != nil || got != expectedTxID {
		t.Fatalf("BroadcastPrepared success = %q, %v", got, err)
	}
	if len(fixture.rpc.sentRaw) != 1 || !bytes.Equal(fixture.rpc.sentRaw[0], raw) {
		t.Fatal("BroadcastPrepared did not submit the exact raw bytes")
	}

	corruptWitness := append([]byte(nil), raw...)
	corruptWitness[len(corruptWitness)-10] ^= 1
	sends := len(fixture.rpc.sentRaw)
	if _, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), corruptWitness); err == nil {
		t.Fatal("BroadcastPrepared accepted witness corruption with the same txid")
	}
	if len(fixture.rpc.sentRaw) != sends {
		t.Fatal("BroadcastPrepared sent witness corruption to RPC")
	}

	mismatchedTx := mustPreparedTx(t, prepared)
	mismatchedTx.TxOut[0].Value--
	mismatchedRaw, err := serializeTx(mismatchedTx)
	if err != nil {
		t.Fatal(err)
	}
	sends = len(fixture.rpc.sentRaw)
	if _, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), mismatchedRaw); err == nil {
		t.Fatal("BroadcastPrepared accepted a mismatched non-witness body")
	}
	if len(fixture.rpc.sentRaw) != sends {
		t.Fatal("BroadcastPrepared submitted a body with the wrong txid")
	}
	if _, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), append(raw, 0)); err == nil {
		t.Fatal("BroadcastPrepared accepted raw bytes with trailing data")
	}
	wrongPrepared := clonePrepared(prepared)
	wrongPrepared.expectedTxID = strings.Repeat("0", 64)
	resealPrepared(t, wrongPrepared)
	if _, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), wrongPrepared, fixture.authorization(), raw); err == nil {
		t.Fatal("BroadcastPrepared accepted the wrong expected txid")
	}

	tests := []struct {
		name        string
		sendErr     error
		lookup      *RawTx
		wantSuccess bool
		wantLookup  bool
	}{
		{name: "ambiguous exact lookup", sendErr: &RPCError{Code: -25, Message: "Missing inputs"}, lookup: &RawTx{TxID: expectedTxID}, wantSuccess: true, wantLookup: true},
		{name: "ambiguous missing lookup", sendErr: &RPCError{Code: -25, Message: "Missing inputs"}, wantLookup: true},
		{name: "ambiguous wrong lookup", sendErr: &RPCError{Code: -25, Message: "Missing inputs"}, lookup: &RawTx{TxID: strings.Repeat("1", 64)}, wantLookup: true},
		{name: "already known exact lookup", sendErr: &RPCError{Code: -27, Message: "Transaction already in block chain"}, lookup: &RawTx{TxID: expectedTxID}, wantSuccess: true, wantLookup: true},
		{name: "already known missing lookup", sendErr: &RPCError{Code: -27, Message: "Transaction already in block chain"}, wantLookup: true},
		{name: "timeout after accepted send", sendErr: context.DeadlineExceeded, lookup: &RawTx{TxID: expectedTxID}, wantSuccess: true, wantLookup: true},
		{name: "EOF after accepted send", sendErr: io.EOF, lookup: &RawTx{TxID: expectedTxID}, wantSuccess: true, wantLookup: true},
		{name: "connection reset after accepted send", sendErr: errors.New("connection reset by peer"), lookup: &RawTx{TxID: expectedTxID}, wantSuccess: true, wantLookup: true},
		{name: "unrelated missing lookup", sendErr: errors.New("connection refused"), wantLookup: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture.rpc.sendErr = test.sendErr
			fixture.rpc.sendResult = nil
			fixture.rpc.rawTx = test.lookup
			fixture.rpc.rawErr = nil
			fixture.rpc.rawLookups = nil
			got, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), raw)
			if test.wantSuccess {
				if err != nil || got != expectedTxID {
					t.Fatalf("BroadcastPrepared = %q, %v", got, err)
				}
			} else if err == nil {
				t.Fatal("BroadcastPrepared accepted an unconfirmed broadcast error")
			}
			if gotLookup := len(fixture.rpc.rawLookups) == 1 && fixture.rpc.rawLookups[0] == expectedTxID; gotLookup != test.wantLookup {
				t.Fatalf("exact lookup = %v (%v), want %v", gotLookup, fixture.rpc.rawLookups, test.wantLookup)
			}
		})
	}

	fixture.rpc.sendErr = nil
	wrongResult := strings.Repeat("2", 64)
	fixture.rpc.sendResult = &wrongResult
	fixture.rpc.rawLookups = nil
	if _, err := fixture.finalizers[0].BroadcastPrepared(context.Background(), prepared, fixture.authorization(), raw); err == nil {
		t.Fatal("BroadcastPrepared accepted a successful response with the wrong txid")
	}
	if len(fixture.rpc.rawLookups) != 0 {
		t.Fatal("BroadcastPrepared looked up a non-error txid mismatch")
	}
}

func TestPreparedWithdrawalLegacyCompatibility(t *testing.T) {
	fixture := newPreparedFixture(t)
	ctx := context.Background()

	if err := fixture.finalizers[0].Validate(ctx, fixture.canonical, fixture.op, fixture.withdrawalID, fixture.deadline); err != nil {
		t.Fatalf("legacy Validate: %v", err)
	}
	if len(fixture.rpc.getTxOutCalls) != 2 {
		t.Fatalf("legacy Validate GetTxOut calls = %d, want 2", len(fixture.rpc.getTxOutCalls))
	}
	legacyShares := make([][]byte, 2)
	var err error
	for i := range legacyShares {
		legacyShares[i], err = fixture.finalizers[i].Sign(ctx, fixture.canonical)
		if err != nil {
			t.Fatalf("legacy Sign[%d]: %v", i, err)
		}
	}
	legacyTxID, err := fixture.finalizers[0].Submit(ctx, fixture.canonical, legacyShares)
	if err != nil {
		t.Fatalf("legacy Submit: %v", err)
	}
	legacyRaw := fixture.rpc.sentRaw[len(fixture.rpc.sentRaw)-1]
	if legacyTxID != mustPreparedTx(t, &PreparedWithdrawal{preparedTransaction: preparedTransaction{canonicalTx: fixture.canonical}}).TxHash().String() {
		t.Fatalf("legacy Submit txid = %s", legacyTxID)
	}

	prepared, err := fixture.finalizers[0].Prepare(ctx, fixture.canonical, fixture.authorization())
	if err != nil {
		t.Fatalf("Prepare after legacy flow: %v", err)
	}
	preparedShares := make([][]byte, 2)
	for i := range preparedShares {
		preparedShares[i], err = fixture.finalizers[i].SignPrepared(ctx, prepared, fixture.authorization())
		if err != nil {
			t.Fatalf("SignPrepared[%d]: %v", i, err)
		}
	}
	preparedRaw, preparedTxID, err := fixture.finalizers[0].FinalizePrepared(prepared, fixture.authorization(), preparedShares)
	if err != nil {
		t.Fatalf("FinalizePrepared: %v", err)
	}
	if preparedTxID != legacyTxID || !bytes.Equal(preparedRaw, legacyRaw) {
		t.Fatal("prepared API changed the legacy deterministic transaction or txid")
	}

	wrong := *fixture.op
	wrong.Amount = decimal.New(100_001, -8)
	if err := fixture.finalizers[0].Validate(ctx, fixture.canonical, &wrong, fixture.withdrawalID, fixture.deadline); err == nil {
		t.Fatal("legacy Validate accepted a mismatched operation")
	}
}

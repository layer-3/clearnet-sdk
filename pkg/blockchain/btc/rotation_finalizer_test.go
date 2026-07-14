package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// TestValidateFixedTxFields covers the ISS-006(b) griefing guard shared by the
// withdrawal and rotation validators: the canonical tx must use the canonical
// version, a zero locktime, and final (non-RBF) input sequences.
func TestValidateFixedTxFields(t *testing.T) {
	good := func() *wire.MsgTx {
		tx := wire.NewMsgTx(wire.TxVersion)
		in := wire.NewTxIn(&wire.OutPoint{Index: 0}, nil, nil)
		in.Sequence = wire.MaxTxInSequenceNum
		tx.AddTxIn(in)
		return tx
	}
	if err := validateFixedTxFields(good()); err != nil {
		t.Fatalf("canonical tx rejected: %v", err)
	}

	badVersion := good()
	badVersion.Version = wire.TxVersion + 1
	if err := validateFixedTxFields(badVersion); err == nil {
		t.Error("non-canonical version accepted")
	}

	badLock := good()
	badLock.LockTime = 1
	if err := validateFixedTxFields(badLock); err == nil {
		t.Error("non-zero locktime accepted")
	}

	rbf := good()
	rbf.TxIn[0].Sequence = wire.MaxTxInSequenceNum - 2 // RBF-signalling
	if err := validateFixedTxFields(rbf); err == nil {
		t.Error("RBF-signalling sequence accepted")
	}
}

// TestBuildSweepTx_OpReturnMarker pins the rotation sweep wire a watcher matches
// on: output 0 pays the new vault, the final output is a zero-value OP_RETURN
// carrying opID. (Devnet-free; the full flow is exercised by the integration
// test.)
func TestBuildSweepTx_OpReturnMarker(t *testing.T) {
	net := &chaincfg.RegressionNetParams

	pubs := make([][]byte, 2)
	for i := range pubs {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen key: %v", err)
		}
		pubs[i] = sign.NewKeySignerFromECDSA(k).PublicKey()
	}
	redeem, err := RedeemScript(2, pubs)
	if err != nil {
		t.Fatalf("RedeemScript: %v", err)
	}
	vault, err := VaultAddress(redeem, net)
	if err != nil {
		t.Fatalf("VaultAddress: %v", err)
	}

	var txid chainhash.Hash
	txid[0] = 0x11
	utxos := []UTXO{{TxID: txid, Vout: 0, Amount: 1_000_000}}

	var opID [32]byte
	opID[0], opID[31] = 0xAB, 0xCD

	tx, err := buildSweepTx(utxos, vault, opID, 5)
	if err != nil {
		t.Fatalf("buildSweepTx: %v", err)
	}

	if len(tx.TxOut) != 2 {
		t.Fatalf("outputs = %d, want 2 (vault + OP_RETURN)", len(tx.TxOut))
	}

	// Output 0: the new vault, total minus fee.
	vaultScript, err := txscript.PayToAddrScript(vault)
	if err != nil {
		t.Fatalf("vault script: %v", err)
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, vaultScript) {
		t.Error("output 0 is not the new vault script")
	}
	if tx.TxOut[0].Value <= 0 || tx.TxOut[0].Value >= 1_000_000 {
		t.Errorf("output 0 value = %d, want 0 < v < total", tx.TxOut[0].Value)
	}

	// Output 1: zero-value OP_RETURN(opID).
	marker, err := txscript.NullDataScript(opID[:])
	if err != nil {
		t.Fatalf("marker script: %v", err)
	}
	if tx.TxOut[1].Value != 0 {
		t.Errorf("OP_RETURN value = %d, want 0", tx.TxOut[1].Value)
	}
	if !bytes.Equal(tx.TxOut[1].PkScript, marker) {
		t.Error("output 1 is not OP_RETURN(opID)")
	}

	// Inputs: every UTXO consumed.
	if len(tx.TxIn) != len(utxos) {
		t.Errorf("inputs = %d, want %d", len(tx.TxIn), len(utxos))
	}
}

// stubRotationRPC is a minimal RPC seam for exercising Validate without a node.
// ListUnspent reports the owned UTXO set; GetTxOut answers each input as a
// confirmed output of vaultScript so sumValidatedInputs accepts it. The other
// methods are unused by Validate.
type stubRotationRPC struct {
	unspent     []Unspent
	vaultScript string // hex pkScript every owned UTXO pays to
	confs       int64
}

func (s *stubRotationRPC) ListUnspent(_ context.Context, _ int, _ []string) ([]Unspent, error) {
	return s.unspent, nil
}

func (s *stubRotationRPC) GetTxOut(_ context.Context, txid string, vout uint32, _ bool) (*TxOut, error) {
	for _, u := range s.unspent {
		if u.TxID == txid && u.Vout == vout {
			return &TxOut{AmountSats: u.AmountSats, ScriptPubKey: s.vaultScript, Confirmations: s.confs}, nil
		}
	}
	return nil, nil
}

func (s *stubRotationRPC) SendRawTransaction(context.Context, string) (string, error) {
	return "", nil
}
func (s *stubRotationRPC) EstimateSmartFeeSatPerVByte(context.Context, int, int64) (int64, error) {
	return 5, nil
}
func (s *stubRotationRPC) GetBlockCount(context.Context) (int64, error)        { return 0, nil }
func (s *stubRotationRPC) GetBlockHash(context.Context, int64) (string, error) { return "", nil }
func (s *stubRotationRPC) GetBlockTxids(context.Context, string) ([]string, error) {
	return nil, nil
}
func (s *stubRotationRPC) GetRawTransaction(context.Context, string) (*RawTx, error) {
	return nil, nil
}

// stubVaultStore returns a fixed current vault and accepts any pivot.
type stubVaultStore struct {
	pubkeys   [][]byte
	threshold int
}

func (s *stubVaultStore) Current(context.Context) ([][]byte, int, error) {
	return s.pubkeys, s.threshold, nil
}
func (s *stubVaultStore) Pivot(context.Context, [][]byte, int) error { return nil }

// TestRotationValidateRejectsPartialSweep proves the completeness guard: a sweep
// that omits an owned UTXO is rejected (it would strand that UTXO at the old
// vault), while the full sweep over the same owned set validates.
func TestRotationValidateRejectsPartialSweep(t *testing.T) {
	net := &chaincfg.RegressionNetParams
	ctx := context.Background()

	// Current vault: a 2-of-2 P2WSH whose first key is this node's signer.
	keys := make([]*sign.KeySigner, 2)
	pubs := make([][]byte, 2)
	for i := range keys {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen key: %v", err)
		}
		keys[i] = sign.NewKeySignerFromECDSA(k)
		pubs[i] = keys[i].PublicKey()
	}
	signer := keys[0]

	redeem, err := RedeemScript(2, pubs)
	if err != nil {
		t.Fatalf("RedeemScript: %v", err)
	}
	vaultAddr, err := VaultAddress(redeem, net)
	if err != nil {
		t.Fatalf("VaultAddress: %v", err)
	}
	vaultScript, err := PkScript(vaultAddr)
	if err != nil {
		t.Fatalf("PkScript: %v", err)
	}

	// New vault: a distinct 2-of-2 set the sweep pays into.
	newPubs := make([]string, 2)
	for i := range newPubs {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen new key: %v", err)
		}
		newPubs[i] = hex.EncodeToString(sign.NewKeySignerFromECDSA(k).PublicKey())
	}

	// Three owned UTXOs.
	owned := make([]UTXO, 3)
	for i := range owned {
		var h chainhash.Hash
		h[0] = byte(0x10 + i)
		owned[i] = UTXO{TxID: h, Vout: uint32(i), Amount: 1_000_000}
	}
	unspent := make([]Unspent, len(owned))
	for i, u := range owned {
		unspent[i] = Unspent{TxID: u.TxID.String(), Vout: u.Vout, AmountSats: u.Amount, Confirmations: 100}
	}

	vaultScriptHex := strings.ToLower(hex.EncodeToString(vaultScript))
	for i := range unspent {
		unspent[i].ScriptPubKey = vaultScriptHex // so toUTXOs resolves them as owned
	}

	cfg := Config{ConfirmationDepth: 1, FeeConfTarget: 6, FallbackFeeRate: 5, FeeCapSatPerVByte: 100}
	rpc := &stubRotationRPC{
		unspent:     unspent,
		vaultScript: vaultScriptHex,
		confs:       100,
	}
	store := &stubVaultStore{pubkeys: pubs, threshold: 2}

	rf, err := NewRotationFinalizer(net, rpc, signer, store, cfg, NewAssetResolver())
	if err != nil {
		t.Fatalf("NewRotationFinalizer: %v", err)
	}

	var opID [32]byte
	opID[0], opID[31] = 0xAB, 0xCD

	newVaultAddr, _, _, err := rf.newVaultAddress(newPubs, 2)
	if err != nil {
		t.Fatalf("newVaultAddress: %v", err)
	}

	// A full sweep over all owned UTXOs validates.
	full, err := buildSweepTx(owned, newVaultAddr, opID, 5)
	if err != nil {
		t.Fatalf("buildSweepTx (full): %v", err)
	}
	fullBytes, err := serializeTx(full)
	if err != nil {
		t.Fatalf("serialize full: %v", err)
	}
	if err := rf.Validate(ctx, opID, fullBytes, newPubs, 2); err != nil {
		t.Fatalf("full sweep rejected: %v", err)
	}

	// Drop one input: the sweep now omits an owned UTXO and must be rejected.
	dropped := owned[len(owned)-1]
	partial, err := buildSweepTx(owned[:len(owned)-1], newVaultAddr, opID, 5)
	if err != nil {
		t.Fatalf("buildSweepTx (partial): %v", err)
	}
	partialBytes, err := serializeTx(partial)
	if err != nil {
		t.Fatalf("serialize partial: %v", err)
	}
	err = rf.Validate(ctx, opID, partialBytes, newPubs, 2)
	if err == nil {
		t.Fatal("partial sweep accepted, want rejection")
	}
	missing := dropped.TxID.String()
	if !strings.Contains(err.Error(), "owned utxo") && !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not mention the omitted owned utxo %s", err, missing)
	}
}

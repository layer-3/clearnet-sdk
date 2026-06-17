package btc

import (
	"bytes"
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

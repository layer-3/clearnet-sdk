package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// TestSelectUTXOsRespectsMaxInputs proves the fragmentation bound: when covering
// the amount would need more than maxInputs inputs, SelectUTXOs returns
// ErrTooFragmented instead of an oversized selection; within the bound (or
// unbounded) it selects normally.
func TestSelectUTXOsRespectsMaxInputs(t *testing.T) {
	// Ten 100-sat UTXOs; an 800-sat withdrawal needs many inputs once the fee
	// (which grows per input) is folded in.
	utxos := make([]UTXO, 10)
	for i := range utxos {
		var h chainhash.Hash
		h[0] = byte(i + 1)
		utxos[i] = UTXO{TxID: h, Vout: 0, Amount: 100}
	}

	// Bounded at 3 inputs: coverage can't be reached, so it must signal fragmentation.
	if _, _, err := SelectUTXOs(utxos, 800, 1, 2, 3); !errors.Is(err, ErrTooFragmented) {
		t.Fatalf("bounded selection: got %v, want ErrTooFragmented", err)
	}

	// Unbounded (maxInputs=0) with a tiny amount + zero-ish fee selects fine.
	sel, _, err := SelectUTXOs(utxos, 150, 0, 2, 0)
	if err != nil {
		t.Fatalf("unbounded selection: unexpected error %v", err)
	}
	if len(sel) == 0 {
		t.Fatal("unbounded selection returned no inputs")
	}
}

// consolidationFixture builds a current 2-of-2 P2WSH vault, a ConsolidationFinalizer
// over a stub RPC seeded with `n` owned UTXOs, and returns the pieces a test needs.
func consolidationFixture(t *testing.T, n int) (*ConsolidationFinalizer, []UTXO, []byte) {
	t.Helper()
	net := &chaincfg.RegressionNetParams

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
	vaultScriptHex := strings.ToLower(hex.EncodeToString(vaultScript))

	owned := make([]UTXO, n)
	unspent := make([]Unspent, n)
	for i := range owned {
		var h chainhash.Hash
		h[0] = byte(0x20 + i)
		owned[i] = UTXO{TxID: h, Vout: uint32(i), Amount: 1_000_000}
		unspent[i] = Unspent{
			TxID: h.String(), Vout: uint32(i), AmountSats: 1_000_000,
			Confirmations: 100, ScriptPubKey: vaultScriptHex,
		}
	}

	cfg := Config{ConfirmationDepth: 1, FeeConfTarget: 6, FallbackFeeRate: 5, FeeCapSatPerVByte: 100}
	rpc := &stubRotationRPC{unspent: unspent, vaultScript: vaultScriptHex, confs: 100}
	store := &stubVaultStore{pubkeys: pubs, threshold: 2}

	cf, err := NewConsolidationFinalizer(net, rpc, keys[0], store, cfg, NewAssetResolver())
	if err != nil {
		t.Fatalf("NewConsolidationFinalizer: %v", err)
	}
	return cf, owned, vaultScript
}

// TestConsolidationPackValidate covers the happy path (Pack builds the two-output
// self-fold, Validate accepts it) and the two follower trust-boundary rejections:
// an output 0 not paying the base vault, and an input that is not an owned UTXO.
func TestConsolidationPackValidate(t *testing.T) {
	ctx := context.Background()
	cf, _, vaultScript := consolidationFixture(t, 3)

	var cid [32]byte
	cid[0], cid[31] = 0xC0, 0xDE

	packed, err := cf.Pack(ctx, cid)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	tx, err := deserializeTx(packed)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if len(tx.TxOut) != 2 {
		t.Fatalf("outputs = %d, want 2 (base vault + OP_RETURN)", len(tx.TxOut))
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, vaultScript) {
		t.Error("output 0 is not the base vault")
	}
	marker, err := txscript.NullDataScript(cid[:])
	if err != nil {
		t.Fatalf("marker: %v", err)
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, marker) {
		t.Error("output 1 is not OP_RETURN(consolidationID)")
	}

	// Honest fold validates.
	if err := cf.Validate(ctx, cid, packed); err != nil {
		t.Fatalf("honest fold rejected: %v", err)
	}

	// Tamper: output 0 redirected away from the base vault.
	bad := tx.Copy()
	bad.TxOut[0].PkScript = marker // anything that isn't the vault script
	badBytes, err := serializeTx(bad)
	if err != nil {
		t.Fatalf("serialize tampered: %v", err)
	}
	if err := cf.Validate(ctx, cid, badBytes); err == nil {
		t.Error("fold with output 0 not the base vault was accepted")
	}

	// Tamper: splice in a foreign (non-owned) input.
	foreign := tx.Copy()
	var fh chainhash.Hash
	fh[0] = 0xFF
	foreign.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&fh, 0), nil, nil)) // NewTxIn defaults to a final sequence
	foreignBytes, err := serializeTx(foreign)
	if err != nil {
		t.Fatalf("serialize foreign: %v", err)
	}
	err = cf.Validate(ctx, cid, foreignBytes)
	if err == nil {
		t.Error("fold with a foreign input was accepted")
	} else if !strings.Contains(err.Error(), "owned utxo") {
		t.Errorf("error %q does not flag the non-owned input", err)
	}
}

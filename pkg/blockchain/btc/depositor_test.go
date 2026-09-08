package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/btc/marker"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

type depositorTestRPC struct {
	unspent      []Unspent
	listErr      error
	feeRate      int64
	feeErr       error
	sendErr      error
	rawTx        *RawTx
	rawErr       error
	rawFunc      func(string) (*RawTx, error)
	rawTxID      string
	listMin      int
	listAddrs    []string
	feeTarget    int
	feeFallback  int64
	broadcastHex string
}

func (r *depositorTestRPC) ListUnspent(_ context.Context, min int, addrs []string) ([]Unspent, error) {
	r.listMin = min
	r.listAddrs = append([]string(nil), addrs...)
	return append([]Unspent(nil), r.unspent...), r.listErr
}

func (r *depositorTestRPC) GetTxOut(context.Context, string, uint32, bool) (*TxOut, error) {
	return nil, nil
}

func (r *depositorTestRPC) SendRawTransaction(_ context.Context, rawHex string) (string, error) {
	r.broadcastHex = rawHex
	if r.sendErr != nil {
		return "", r.sendErr
	}
	raw, err := hex.DecodeString(rawHex)
	if err != nil {
		return "", err
	}
	tx, err := decodeP2WPKHTestTx(raw)
	if err != nil {
		return "", err
	}
	return tx.TxHash().String(), nil
}

func (r *depositorTestRPC) EstimateSmartFeeSatPerVByte(_ context.Context, target int, fallback int64) (int64, error) {
	r.feeTarget, r.feeFallback = target, fallback
	return r.feeRate, r.feeErr
}

func (r *depositorTestRPC) GetBlockCount(context.Context) (int64, error)        { return 0, nil }
func (r *depositorTestRPC) GetBlockHash(context.Context, int64) (string, error) { return "", nil }
func (r *depositorTestRPC) GetBlockTxids(context.Context, string) ([]string, error) {
	return nil, nil
}
func (r *depositorTestRPC) GetRawTransaction(_ context.Context, txID string) (*RawTx, error) {
	r.rawTxID = txID
	if r.rawFunc != nil {
		return r.rawFunc(txID)
	}
	return r.rawTx, r.rawErr
}

func depositorTestVaultKeys(t *testing.T) (sign.Signer, [][]byte) {
	t.Helper()
	depositorSigner := p2wpkhTestKeySigner(t, 1)
	vaultKeys := [][]byte{
		p2wpkhTestKeySigner(t, 2).PublicKey(),
		p2wpkhTestKeySigner(t, 3).PublicKey(),
		p2wpkhTestKeySigner(t, 4).PublicKey(),
	}
	return depositorSigner, vaultKeys
}

// depositorTestAccount returns a 20-byte clearnet account plus its hex encoding,
// the shape SubmitDeposit's dest.Account requires.
func depositorTestAccount(b byte) ([20]byte, string) {
	var addr [20]byte
	addr[0], addr[19] = b, b^0xff
	return addr, hex.EncodeToString(addr[:])
}

// TestParseClearnetAccount pins the exact accepted/rejected input set for
// parseClearnetAccount: bare hex, an optional case-insensitive "0x" prefix,
// a yellow://.../user/<hex> URI's last segment, and surrounding whitespace.
// It also pins that an ADR-015 sub-account URI (yellow://.../user/<addr>/tag/<32-byte-ref>)
// is rejected rather than silently parsed as the trailing 32-byte reference.
func TestParseClearnetAccount(t *testing.T) {
	const addrHex = "000102030405060708090a0b0c0d0e0f10111213"
	const refHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	want, err := hex.DecodeString(addrHex)
	if err != nil {
		t.Fatal(err)
	}
	var wantAddr [20]byte
	copy(wantAddr[:], want)

	accept := []string{
		addrHex,
		"0x" + addrHex,
		"0X" + addrHex,
		strings.ToUpper(addrHex),
		"  " + addrHex + "  ",
		"\t" + addrHex + "\n",
		"yellow://ynet/user/" + addrHex,
		"  yellow://ynet/user/" + addrHex + "  ",
	}
	for _, in := range accept {
		t.Run("accept/"+in, func(t *testing.T) {
			got, err := parseClearnetAccount(in)
			if err != nil {
				t.Fatalf("parseClearnetAccount(%q) error = %v, want nil", in, err)
			}
			if got != wantAddr {
				t.Fatalf("parseClearnetAccount(%q) = %x, want %x", in, got, wantAddr)
			}
		})
	}

	reject := []string{
		"",
		"not-hex",
		addrHex[:38],   // 19 bytes
		addrHex + "00", // 21 bytes
		"yellow://ynet/user/" + addrHex + "/tag/" + refHex, // ADR-015 sub-account URI: last segment is the 32-byte reference, not the address.
	}
	for _, in := range reject {
		t.Run("reject/"+in, func(t *testing.T) {
			if _, err := parseClearnetAccount(in); err == nil {
				t.Fatalf("parseClearnetAccount(%q) error = nil, want error", in)
			}
		})
	}
}

// genericDepositTestAddress derives the single generic P2WSH deposit address
// independently of the depositor's own genericDepositTarget helper: it starts
// from marker.GenericDepositTagHex and rebuilds the address through the same
// shared primitives DepositAddress uses.
func genericDepositTestAddress(t *testing.T, threshold int, vaultKeys [][]byte, net *chaincfg.Params) (btcutil.Address, []byte) {
	t.Helper()
	tag, err := hex.DecodeString(marker.GenericDepositTagHex)
	if err != nil || len(tag) != 32 {
		t.Fatalf("decode marker.GenericDepositTagHex: %v", err)
	}
	redeem, err := TaggedRedeemScript(tag, threshold, vaultKeys)
	if err != nil {
		t.Fatalf("TaggedRedeemScript: %v", err)
	}
	addr, err := VaultAddress(redeem, net)
	if err != nil {
		t.Fatalf("VaultAddress: %v", err)
	}
	script, err := PkScript(addr)
	if err != nil {
		t.Fatalf("PkScript: %v", err)
	}
	return addr, script
}

// TestDepositorSendsToGenericAddressWithVersion1Marker: the build transaction
// pays the single generic deposit address and carries a zero-value marker output.
func TestDepositorSendsToGenericAddressWithVersion1Marker(t *testing.T) {
	ctx := context.Background()
	net := &chaincfg.RegressionNetParams
	rpc := &depositorTestRPC{feeRate: defaultDepositorFallbackFeeRate}
	depositorSigner, vaultKeys := depositorTestVaultKeys(t)
	depositor, err := NewDepositor(net, rpc, depositorSigner, vaultKeys, 2, Config{}, NewAssetResolver())
	if err != nil {
		t.Fatalf("NewDepositor with legacy zero config: %v", err)
	}
	if depositor.DepositorAddress() != depositor.sender.Address() {
		t.Fatalf("DepositorAddress = %q, sender = %q", depositor.DepositorAddress(), depositor.sender.Address())
	}

	var fundingHash chainhash.Hash
	fundingHash[0] = 0x42
	rpc.unspent = []Unspent{{
		TxID:          fundingHash.String(),
		Vout:          7,
		AmountSats:    50_000,
		Confirmations: int64(defaultDepositorMinConfirmations),
		ScriptPubKey:  hex.EncodeToString(depositor.sender.sourceScript),
	}}
	accountAddr, account := depositorTestAccount(0xaa)
	txID, err := depositor.SubmitDeposit(ctx, "", decimal.RequireFromString("0.0001"), core.DepositDestination{Account: account})
	if err != nil {
		t.Fatalf("SubmitDeposit: %v", err)
	}
	raw, err := hex.DecodeString(rpc.broadcastHex)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := decodeP2WPKHTestTx(raw)
	if err != nil {
		t.Fatal(err)
	}
	if txID != tx.TxHash().String() {
		t.Fatalf("SubmitDeposit txid = %s, local = %s", txID, tx.TxHash())
	}

	wantAddress, wantScript := genericDepositTestAddress(t, 2, vaultKeys, net)
	wantMarkerScript, err := marker.EncodeScript(marker.Marker{Version: marker.Version1, Address: accountAddr})
	if err != nil {
		t.Fatal(err)
	}
	if len(wantMarkerScript) != 27 {
		t.Fatalf("v1 marker script length = %d, want 27", len(wantMarkerScript))
	}

	if len(tx.TxOut) != 3 {
		t.Fatalf("outputs = %d, want 3 (value, marker, change)", len(tx.TxOut))
	}
	if tx.TxOut[0].Value != 10_000 || !bytes.Equal(tx.TxOut[0].PkScript, wantScript) {
		t.Fatalf("deposit output = %#v, want 10000 sats to generic %s", tx.TxOut[0], wantAddress)
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, wantMarkerScript) {
		t.Fatalf("marker output = %#v, want zero-value %x", tx.TxOut[1], wantMarkerScript)
	}

	// The exact fee is asserted against p2wpkhFee itself rather than a
	// hand-derived constant, so this test also pins that the marker output
	// reached the fee path and not just the builder.
	wantFee, err := p2wpkhFee(1, [][]byte{wantScript, wantMarkerScript, depositor.sender.sourceScript}, defaultDepositorFallbackFeeRate)
	if err != nil {
		t.Fatal(err)
	}
	wantChange := int64(50_000) - int64(10_000) - wantFee
	if tx.TxOut[2].Value != wantChange || !bytes.Equal(tx.TxOut[2].PkScript, depositor.sender.sourceScript) {
		t.Fatalf("change output = (%d,%x), want (%d,%x)", tx.TxOut[2].Value, tx.TxOut[2].PkScript, wantChange, depositor.sender.sourceScript)
	}
	if got := int64(50_000) - tx.TxOut[0].Value - tx.TxOut[1].Value - tx.TxOut[2].Value; got != wantFee {
		t.Fatalf("absolute deposit fee = %d, want %d", got, wantFee)
	}
	if len(tx.TxIn) != 1 || tx.TxIn[0].PreviousOutPoint != (wire.OutPoint{Hash: fundingHash, Index: 7}) {
		t.Fatalf("deposit inputs = %#v, want funding outpoint", tx.TxIn)
	}
	if len(tx.TxIn[0].Witness) != 2 || !bytes.Equal(tx.TxIn[0].Witness[1], depositorSigner.PublicKey()) {
		t.Fatal("deposit was not signed by depositor P2WPKH key")
	}
	if rpc.listMin != int(defaultDepositorMinConfirmations) || len(rpc.listAddrs) != 1 || rpc.listAddrs[0] != depositor.DepositorAddress() {
		t.Fatalf("legacy ListUnspent args = (%d,%v)", rpc.listMin, rpc.listAddrs)
	}
	if rpc.feeTarget != defaultDepositorFeeConfirmationTarget || rpc.feeFallback != defaultDepositorFallbackFeeRate {
		t.Fatalf("legacy fee args = (%d,%d)", rpc.feeTarget, rpc.feeFallback)
	}
}

// TestDepositorMarkerRoundTripsThroughMarkerPackage: for both marker versions,
// the script the depositor builds decodes back through marker.DecodeScript /
// marker.ScanOutputs to the same marker.Marker, and a non-zero dest.Ref selects
// the 59-byte 0x02 shape.
func TestDepositorMarkerRoundTripsThroughMarkerPackage(t *testing.T) {
	ctx := context.Background()
	net := &chaincfg.RegressionNetParams
	rpc := &depositorTestRPC{feeRate: defaultDepositorFallbackFeeRate}
	signer, vaultKeys := depositorTestVaultKeys(t)
	depositor, err := NewDepositor(net, rpc, signer, vaultKeys, 2, Config{}, NewAssetResolver())
	if err != nil {
		t.Fatal(err)
	}
	var fundingHash chainhash.Hash
	fundingHash[0] = 0x45
	rpc.unspent = []Unspent{{
		TxID:          fundingHash.String(),
		AmountSats:    50_000,
		Confirmations: int64(defaultDepositorMinConfirmations),
		ScriptPubKey:  hex.EncodeToString(depositor.sender.sourceScript),
	}}
	accountAddr, account := depositorTestAccount(0xcc)

	tests := []struct {
		name       string
		ref        [32]byte
		wantVer    byte
		wantLength int
	}{
		{"zero ref selects v0x01", [32]byte{}, marker.Version1, 27},
		{"non-zero ref selects v0x02", [32]byte{0x01}, marker.Version2, 59},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpc.broadcastHex = ""
			if _, err := depositor.SubmitDeposit(ctx, "", decimal.RequireFromString("0.0001"), core.DepositDestination{Account: account, Ref: tc.ref}); err != nil {
				t.Fatalf("SubmitDeposit: %v", err)
			}
			raw, err := hex.DecodeString(rpc.broadcastHex)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := decodeP2WPKHTestTx(raw)
			if err != nil {
				t.Fatal(err)
			}

			var scriptPubKeys [][]byte
			var markerScript []byte
			var markerValue int64 = -1
			for _, out := range tx.TxOut {
				scriptPubKeys = append(scriptPubKeys, out.PkScript)
				if _, decodeErr := marker.DecodeScript(out.PkScript); decodeErr == nil {
					markerScript, markerValue = out.PkScript, out.Value
				}
			}
			if markerScript == nil {
				t.Fatal("built transaction carries no marker candidate")
			}
			if markerValue != 0 {
				t.Fatalf("marker output value = %d, want 0", markerValue)
			}
			if len(markerScript) != tc.wantLength {
				t.Fatalf("marker script length = %d, want %d", len(markerScript), tc.wantLength)
			}

			want := marker.Marker{Version: tc.wantVer, Address: accountAddr, Reference: tc.ref}
			decoded, err := marker.DecodeScript(markerScript)
			if err != nil {
				t.Fatalf("marker.DecodeScript: %v", err)
			}
			if decoded != want {
				t.Fatalf("DecodeScript = %+v, want %+v", decoded, want)
			}
			scanned, err := marker.ScanOutputs(scriptPubKeys)
			if err != nil {
				t.Fatalf("marker.ScanOutputs: %v", err)
			}
			if scanned != want {
				t.Fatalf("ScanOutputs = %+v, want %+v", scanned, want)
			}
		})
	}
}

func TestDepositorBroadcastAlreadyAcceptedIsIdempotent(t *testing.T) {
	tests := []struct {
		name       string
		sendErr    error
		lookup     func(string) (*RawTx, error)
		wantOK     bool
		wantLookup bool
	}{
		{
			name:    "already in chain needs no lookup",
			sendErr: &RPCError{Code: -27, Message: "Transaction already in block chain"},
			wantOK:  true,
		},
		{
			name:    "verify error with exact transaction",
			sendErr: &RPCError{Code: -25, Message: "Missing inputs"},
			lookup: func(txID string) (*RawTx, error) {
				return &RawTx{TxID: txID}, nil
			},
			wantOK:     true,
			wantLookup: true,
		},
		{
			name:       "verify error with absent transaction",
			sendErr:    &RPCError{Code: -25, Message: "Missing inputs"},
			lookup:     func(string) (*RawTx, error) { return nil, nil },
			wantLookup: true,
		},
		{
			name:       "typed verify error cannot bypass lookup by message",
			sendErr:    &RPCError{Code: -25, Message: "Transaction already in block chain"},
			lookup:     func(string) (*RawTx, error) { return nil, nil },
			wantLookup: true,
		},
		{
			name:    "verify error with unknown transaction",
			sendErr: &RPCError{Code: -25, Message: "Missing inputs"},
			lookup: func(string) (*RawTx, error) {
				return nil, &RPCError{Code: -5, Message: "No such mempool or blockchain transaction"}
			},
			wantLookup: true,
		},
		{
			name:    "verify error with lookup failure",
			sendErr: &RPCError{Code: -25, Message: "Missing inputs"},
			lookup: func(string) (*RawTx, error) {
				return nil, errP2WPKHTestBackend
			},
			wantLookup: true,
		},
		{
			name:    "verify error with different transaction",
			sendErr: &RPCError{Code: -25, Message: "Missing inputs"},
			lookup: func(string) (*RawTx, error) {
				var other chainhash.Hash
				other[0] = 0xff
				return &RawTx{TxID: other.String()}, nil
			},
			wantLookup: true,
		},
		{
			name:       "max fee rejection remains an error",
			sendErr:    &RPCError{Code: -25, Message: "Fee exceeds maximum configured by user"},
			lookup:     func(string) (*RawTx, error) { return nil, nil },
			wantLookup: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &depositorTestRPC{
				feeRate: defaultDepositorFallbackFeeRate,
				sendErr: tc.sendErr,
				rawFunc: tc.lookup,
			}
			signer, vaultKeys := depositorTestVaultKeys(t)
			depositor, err := NewDepositor(&chaincfg.RegressionNetParams, rpc, signer, vaultKeys, 2, Config{}, NewAssetResolver())
			if err != nil {
				t.Fatal(err)
			}
			var fundingHash chainhash.Hash
			fundingHash[0] = 0x43
			rpc.unspent = []Unspent{{
				TxID:          fundingHash.String(),
				AmountSats:    50_000,
				Confirmations: int64(defaultDepositorMinConfirmations),
				ScriptPubKey:  hex.EncodeToString(depositor.sender.sourceScript),
			}}

			_, account := depositorTestAccount(0x99)
			got, err := depositor.SubmitDeposit(
				context.Background(),
				"",
				decimal.RequireFromString("0.0001"),
				core.DepositDestination{Account: account},
			)
			raw, decodeErr := hex.DecodeString(rpc.broadcastHex)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			tx, decodeErr := decodeP2WPKHTestTx(raw)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			localTxID := tx.TxHash().String()

			if tc.wantOK {
				if err != nil {
					t.Fatalf("SubmitDeposit: %v", err)
				}
				if got != localTxID {
					t.Fatalf("SubmitDeposit txid = %q, want locally calculated %q", got, localTxID)
				}
			} else {
				if err == nil {
					t.Fatal("SubmitDeposit unexpectedly succeeded")
				}
				if !errors.Is(err, tc.sendErr) {
					t.Fatalf("SubmitDeposit error %v does not wrap original broadcast error %v", err, tc.sendErr)
				}
			}
			if tc.wantLookup {
				if rpc.rawTxID != localTxID {
					t.Fatalf("GetRawTransaction txid = %q, want local %q", rpc.rawTxID, localTxID)
				}
			} else if rpc.rawTxID != "" {
				t.Fatalf("unexpected GetRawTransaction lookup for %q", rpc.rawTxID)
			}
		})
	}
}

func TestDepositorSubmitValidationRemainsDepositSpecific(t *testing.T) {
	rpc := &depositorTestRPC{feeRate: 1}
	signer, vaultKeys := depositorTestVaultKeys(t)
	depositor, err := NewDepositor(&chaincfg.RegressionNetParams, rpc, signer, vaultKeys, 2, Config{}, NewAssetResolver())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, account := depositorTestAccount(0xbb)
	dest := core.DepositDestination{Account: account}
	tests := []struct {
		name   string
		asset  string
		amount decimal.Decimal
		dest   core.DepositDestination
		text   string
	}{
		{"invalid account", "", decimal.New(1, 0), core.DepositDestination{Account: "not-hex"}, "hex address"},
		{"non-native asset", "not-btc", decimal.New(1, 0), dest, "native BTC"},
		{"zero amount", "", decimal.New(0, 0), dest, "not positive"},
		{"negative amount", "", decimal.New(-1, 0), dest, "not positive"},
		{"fractional satoshi", "", decimal.RequireFromString("0.000000001"), dest, "amount"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := depositor.SubmitDeposit(ctx, tc.asset, tc.amount, tc.dest); err == nil || !strings.Contains(err.Error(), tc.text) {
				t.Fatalf("SubmitDeposit error = %v, want text %q", err, tc.text)
			}
		})
	}
	if rpc.broadcastHex != "" {
		t.Fatal("invalid deposit reached broadcast")
	}
}

func TestDepositorVerifyDepositStatusCompatibility(t *testing.T) {
	txID := strings.Repeat("a", 64)
	rpc := &depositorTestRPC{}
	signer, vaultKeys := depositorTestVaultKeys(t)
	depositor, err := NewDepositor(&chaincfg.RegressionNetParams, rpc, signer, vaultKeys, 2, Config{}, NewAssetResolver())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		raw     *RawTx
		err     error
		minConf uint64
		want    core.DepositStatus
		wantErr bool
	}{
		{"unknown typed RPC error", nil, &RPCError{Code: -5, Message: "not found"}, 1, core.DepositAbsent, false},
		{"nil raw", nil, nil, 1, core.DepositAbsent, false},
		{"mempool", &RawTx{Confirmations: 0}, nil, 1, core.DepositPending, false},
		{"zero depth still requires a block", &RawTx{Confirmations: 0}, nil, 0, core.DepositPending, false},
		{"zero depth confirmed on chain", &RawTx{Confirmations: 1}, nil, 0, core.DepositConfirmed, false},
		{"below depth", &RawTx{Confirmations: 1}, nil, 2, core.DepositPending, false},
		{"confirmed", &RawTx{Confirmations: 2}, nil, 2, core.DepositConfirmed, false},
		{"transport error", nil, errP2WPKHTestBackend, 1, core.DepositAbsent, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpc.rawTx, rpc.rawErr = tc.raw, tc.err
			got, err := depositor.VerifyDeposit(context.Background(), txID, tc.minConf)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("VerifyDeposit = (%v,%v), want (%v,error=%v)", got, err, tc.want, tc.wantErr)
			}
			if tc.err != nil && tc.wantErr && !errors.Is(err, tc.err) {
				t.Fatalf("error %v does not wrap %v", err, tc.err)
			}
		})
	}
}

func TestDepositorRPCBackendRejectsMalformedLegacyUTXOs(t *testing.T) {
	tests := []struct {
		name string
		u    Unspent
	}{
		{"negative confirmations", Unspent{Confirmations: -1, ScriptPubKey: "00"}},
		{"malformed script hex", Unspent{Confirmations: 1, ScriptPubKey: "xyz"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &depositorTestRPC{unspent: []Unspent{tc.u}}
			backend := newLegacyCoreP2WPKHBackend(rpc, 1)
			if _, err := backend.ListUnspent(context.Background(), "address", 1); err == nil {
				t.Fatal("adapter accepted malformed legacy UTXO")
			}
		})
	}
	backend := newLegacyCoreP2WPKHBackend(&depositorTestRPC{}, 1)
	if _, err := backend.ListUnspent(context.Background(), "address", uint64(math.MaxInt)); err != nil {
		t.Fatalf("adapter rejected maximum int confirmations: %v", err)
	}
}

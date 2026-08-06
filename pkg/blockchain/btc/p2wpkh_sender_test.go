package btc

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

var errP2WPKHTestBackend = errors.New("test backend failure")

type p2wpkhTestBackend struct {
	mu           sync.Mutex
	utxos        []UnspentOutput
	feeRate      int64
	listErr      error
	feeErr       error
	broadcastErr error
	listHook     func(context.Context)
	feeHook      func(context.Context)
	broadcast    func(context.Context, []byte) (string, error)
	listedAddr   string
	listedMin    uint64
	feeTarget    int
	listCalls    int
	raw          []byte
}

func (b *p2wpkhTestBackend) ListUnspent(ctx context.Context, address string, min uint64) ([]UnspentOutput, error) {
	if b.listHook != nil {
		b.listHook(ctx)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listCalls++
	b.listedAddr, b.listedMin = address, min
	if b.listErr != nil {
		return nil, b.listErr
	}
	return append([]UnspentOutput(nil), b.utxos...), nil
}

func (b *p2wpkhTestBackend) FeeRateSatPerVByte(ctx context.Context, target int) (int64, error) {
	if b.feeHook != nil {
		b.feeHook(ctx)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.feeTarget = target
	return b.feeRate, b.feeErr
}

func (b *p2wpkhTestBackend) Broadcast(ctx context.Context, raw []byte) (string, error) {
	b.mu.Lock()
	b.raw = append([]byte(nil), raw...)
	b.mu.Unlock()
	if b.broadcast != nil {
		return b.broadcast(ctx, raw)
	}
	if b.broadcastErr != nil {
		return "", b.broadcastErr
	}
	tx, err := decodeP2WPKHTestTx(raw)
	if err != nil {
		return "", err
	}
	return tx.TxHash().String(), nil
}

type configurableTestSigner struct {
	algorithm sign.Algorithm
	publicKey []byte
	signFunc  func(context.Context, []byte) ([]byte, error)
}

func (s *configurableTestSigner) Algorithm() sign.Algorithm { return s.algorithm }
func (s *configurableTestSigner) PublicKey() []byte         { return s.publicKey }
func (s *configurableTestSigner) Close() error              { return nil }
func (s *configurableTestSigner) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	if s.signFunc == nil {
		return nil, errors.New("test signer has no Sign implementation")
	}
	return s.signFunc(ctx, digest)
}

func p2wpkhTestKeySigner(t *testing.T, scalar byte) *sign.KeySigner {
	t.Helper()
	raw := make([]byte, 32)
	raw[31] = scalar
	key, err := crypto.ToECDSA(raw)
	if err != nil {
		t.Fatalf("make secp256k1 key: %v", err)
	}
	return sign.NewKeySignerFromECDSA(key)
}

func p2wpkhTestConfig() P2WPKHConfig {
	return P2WPKHConfig{
		MinConfirmations:           2,
		FeeConfirmationTarget:      6,
		FallbackFeeRateSatPerVByte: 3,
		FeeCapSatPerVByte:          100,
		DustThresholdSats:          330,
		MaxInputs:                  10,
	}
}

func p2wpkhTestSender(t *testing.T, backend *p2wpkhTestBackend) (*P2WPKHSender, *sign.KeySigner) {
	t.Helper()
	signer := p2wpkhTestKeySigner(t, 1)
	sender, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, signer, p2wpkhTestConfig())
	if err != nil {
		t.Fatalf("NewP2WPKHSender: %v", err)
	}
	return sender, signer
}

func p2wpkhTestRecipient(t *testing.T, net *chaincfg.Params) string {
	t.Helper()
	pub := p2wpkhTestKeySigner(t, 2).PublicKey()
	addr, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pub), net)
	if err != nil {
		t.Fatalf("recipient address: %v", err)
	}
	return addr.EncodeAddress()
}

func p2wpkhTestUTXO(sender *P2WPKHSender, id byte, vout uint32, amount int64) UnspentOutput {
	var hash chainhash.Hash
	hash[0], hash[31] = id, id^0xff
	return UnspentOutput{
		TxID:          hash.String(),
		Vout:          vout,
		AmountSats:    amount,
		ScriptPubKey:  append([]byte(nil), sender.sourceScript...),
		Confirmations: sender.config.MinConfirmations,
	}
}

func decodeP2WPKHTestTx(raw []byte) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	if err := tx.Deserialize(bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	return tx, nil
}

func TestNewP2WPKHSenderDerivesAddressAndValidatesDependencies(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	signer := p2wpkhTestKeySigner(t, 1)
	cfg := p2wpkhTestConfig()
	sender, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, signer, cfg)
	if err != nil {
		t.Fatalf("NewP2WPKHSender: %v", err)
	}
	want, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(signer.PublicKey()), &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if sender.Address() != want.EncodeAddress() {
		t.Fatalf("Address() = %q, want %q", sender.Address(), want.EncodeAddress())
	}

	var nilBackend *p2wpkhTestBackend
	var nilSigner *configurableTestSigner
	tests := []struct {
		name    string
		net     *chaincfg.Params
		backend P2WPKHBackend
		signer  sign.Signer
	}{
		{"nil network", nil, backend, signer},
		{"nil backend", &chaincfg.RegressionNetParams, nil, signer},
		{"typed nil backend", &chaincfg.RegressionNetParams, nilBackend, signer},
		{"nil signer", &chaincfg.RegressionNetParams, backend, nil},
		{"typed nil signer", &chaincfg.RegressionNetParams, backend, nilSigner},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewP2WPKHSender(tc.net, tc.backend, tc.signer, cfg); err == nil {
				t.Fatal("constructor accepted invalid dependency")
			}
		})
	}
}

func TestNewP2WPKHSenderRejectsSignerAndConfig(t *testing.T) {
	backend := &p2wpkhTestBackend{}
	valid := p2wpkhTestKeySigner(t, 1)
	cfg := p2wpkhTestConfig()

	edSigner, err := sign.NewKeySignerFromEd25519(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, edSigner, cfg); err == nil {
		t.Fatal("constructor accepted ed25519 signer")
	}
	for name, pub := range map[string][]byte{
		"empty":        nil,
		"uncompressed": crypto.FromECDSAPub(&crypto.ToECDSAUnsafe(append(make([]byte, 31), 1)).PublicKey),
		"bad prefix":   append([]byte{4}, make([]byte, 32)...),
		"bad point":    append([]byte{2}, make([]byte, 32)...),
	} {
		t.Run(name, func(t *testing.T) {
			bad := &configurableTestSigner{algorithm: sign.AlgSecp256k1, publicKey: pub}
			if _, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, bad, cfg); err == nil {
				t.Fatal("constructor accepted malformed public key")
			}
		})
	}

	configTests := map[string]func(*P2WPKHConfig){
		"zero confirmations": func(c *P2WPKHConfig) { c.MinConfirmations = 0 },
		"zero target":        func(c *P2WPKHConfig) { c.FeeConfirmationTarget = 0 },
		"zero fallback":      func(c *P2WPKHConfig) { c.FallbackFeeRateSatPerVByte = 0 },
		"zero cap":           func(c *P2WPKHConfig) { c.FeeCapSatPerVByte = 0 },
		"fallback over cap":  func(c *P2WPKHConfig) { c.FallbackFeeRateSatPerVByte = c.FeeCapSatPerVByte + 1 },
		"zero dust":          func(c *P2WPKHConfig) { c.DustThresholdSats = 0 },
		"zero max inputs":    func(c *P2WPKHConfig) { c.MaxInputs = 0 },
	}
	for name, mutate := range configTests {
		t.Run(name, func(t *testing.T) {
			bad := cfg
			mutate(&bad)
			if _, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, valid, bad); err == nil {
				t.Fatal("constructor accepted invalid config")
			}
		})
	}
}

func TestP2WPKHSenderRejectsAmountAndRecipientBeforeBackend(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	ctx := context.Background()
	for _, amount := range []int64{0, -1, math.MinInt64} {
		if _, err := sender.Send(ctx, p2wpkhTestRecipient(t, &chaincfg.RegressionNetParams), amount); err == nil {
			t.Fatalf("accepted amount %d", amount)
		}
	}
	if _, err := sender.Send(ctx, "not-an-address", 1); err == nil {
		t.Fatal("accepted malformed recipient")
	}
	wrongNet := p2wpkhTestRecipient(t, &chaincfg.MainNetParams)
	if _, err := sender.Send(ctx, wrongNet, 1); err == nil || !strings.Contains(err.Error(), "network") {
		t.Fatalf("wrong-network recipient error = %v", err)
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.listCalls != 0 {
		t.Fatalf("ListUnspent called %d times for invalid requests", backend.listCalls)
	}
}

func TestP2WPKHSenderValidatesEveryBackendUTXO(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	good := p2wpkhTestUTXO(sender, 1, 0, 20_000)
	recipient := p2wpkhTestRecipient(t, sender.net)

	tests := map[string]func([]UnspentOutput) []UnspentOutput{
		"malformed txid":  func(u []UnspentOutput) []UnspentOutput { u[0].TxID = "xyz"; return u },
		"zero amount":     func(u []UnspentOutput) []UnspentOutput { u[0].AmountSats = 0; return u },
		"negative amount": func(u []UnspentOutput) []UnspentOutput { u[0].AmountSats = -1; return u },
		"insufficient confirmations": func(u []UnspentOutput) []UnspentOutput {
			u[0].Confirmations = sender.config.MinConfirmations - 1
			return u
		},
		"empty script":       func(u []UnspentOutput) []UnspentOutput { u[0].ScriptPubKey = nil; return u },
		"foreign script":     func(u []UnspentOutput) []UnspentOutput { u[0].ScriptPubKey = []byte{txscript.OP_TRUE}; return u },
		"duplicate outpoint": func(u []UnspentOutput) []UnspentOutput { return append(u, u[0]) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			backend.utxos = mutate([]UnspentOutput{good})
			backend.raw = nil
			if _, err := sender.Send(context.Background(), recipient, 10_000); err == nil {
				t.Fatal("sender accepted untrusted UTXO")
			}
			if backend.raw != nil {
				t.Fatal("invalid UTXO reached broadcast")
			}
		})
	}
	backend.utxos = []UnspentOutput{good}
	if _, err := sender.Send(context.Background(), recipient, 10_000); err != nil {
		t.Fatalf("valid UTXO rejected: %v", err)
	}
	if backend.listedAddr != sender.Address() || backend.listedMin != sender.config.MinConfirmations {
		t.Fatalf("ListUnspent arguments = (%q,%d), want (%q,%d)", backend.listedAddr, backend.listedMin, sender.Address(), sender.config.MinConfirmations)
	}
}

func TestP2WPKHFeeUsesExactScriptsAndCheckedArithmetic(t *testing.T) {
	p2wpkh := make([]byte, 22)
	fee, err := p2wpkhFee(1, [][]byte{p2wpkh}, 1)
	if err != nil || fee != 110 {
		t.Fatalf("one-input/one-output fee = %d, %v; want 110", fee, err)
	}
	fee, err = p2wpkhFee(1, [][]byte{p2wpkh, p2wpkh}, 2)
	if err != nil || fee != 282 {
		t.Fatalf("one-input/two-output fee = %d, %v; want 282", fee, err)
	}
	longer := make([]byte, 34)
	longFee, err := p2wpkhFee(1, [][]byte{longer}, 1)
	if err != nil || longFee <= 110 {
		t.Fatalf("long recipient fee = %d, %v; want > 110", longFee, err)
	}
	fee252, err := p2wpkhFee(252, [][]byte{p2wpkh}, 1)
	if err != nil || fee252 != 17_241 {
		t.Fatalf("252-input fee = %d, %v; want 17241", fee252, err)
	}
	fee253, err := p2wpkhFee(253, [][]byte{p2wpkh}, 1)
	if err != nil || fee253 != 17_311 {
		t.Fatalf("253-input fee = %d, %v; want 17311", fee253, err)
	}
	if fee253-fee252 != 70 {
		t.Fatalf("252/253 varint boundary delta = %d, want 70 vbytes", fee253-fee252)
	}
	for _, tc := range []struct {
		inputs int
		outs   [][]byte
		rate   int64
	}{
		{0, [][]byte{p2wpkh}, 1},
		{1, nil, 1},
		{1, [][]byte{p2wpkh}, 0},
		{math.MaxInt, [][]byte{p2wpkh}, 1},
		{1, [][]byte{p2wpkh}, math.MaxInt64},
	} {
		if _, err := p2wpkhFee(tc.inputs, tc.outs, tc.rate); err == nil {
			t.Fatalf("fee estimator accepted overflowing/invalid parameters: %+v", tc)
		}
	}
}

func TestP2WPKHSenderDefensivelyCopiesSignerKeyAndBackendScript(t *testing.T) {
	keySigner := p2wpkhTestKeySigner(t, 1)
	originalPub := keySigner.PublicKey()
	exposedPub := append([]byte(nil), originalPub...)
	signer := &configurableTestSigner{
		algorithm: sign.AlgSecp256k1,
		publicKey: exposedPub,
		signFunc:  keySigner.Sign,
	}
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, signer, p2wpkhTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	originalAddress := sender.Address()
	candidate := p2wpkhTestUTXO(sender, 1, 0, 20_000)
	backend.utxos = []UnspentOutput{candidate}

	// Mutating the signer-owned slice after construction must not change the
	// retained address or the public key placed in a witness.
	for i := range exposedPub {
		exposedPub[i] ^= 0xff
	}
	if sender.Address() != originalAddress {
		t.Fatalf("signer pubkey mutation changed address from %s to %s", originalAddress, sender.Address())
	}

	// ListUnspent returns a shallow slice copy in this fake. Mutate the
	// backend-owned ScriptPubKey after candidate validation, while fee lookup
	// is in progress. Signing must use the sender's retained candidate copy.
	backend.feeHook = func(context.Context) {
		for i := range backend.utxos[0].ScriptPubKey {
			backend.utxos[0].ScriptPubKey[i] ^= 0xff
		}
	}
	if _, err := sender.Send(context.Background(), p2wpkhTestRecipient(t, sender.net), 10_000); err != nil {
		t.Fatalf("Send after caller/backend mutations: %v", err)
	}
	tx, err := decodeP2WPKHTestTx(backend.raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tx.TxIn[0].Witness[1], originalPub) {
		t.Fatalf("witness pubkey = %x, want retained %x", tx.TxIn[0].Witness[1], originalPub)
	}
	prevouts := txscript.NewCannedPrevOutputFetcher(sender.sourceScript, candidate.AmountSats)
	sigHashes := txscript.NewTxSigHashes(tx, prevouts)
	engine, err := txscript.NewEngine(sender.sourceScript, tx, 0, txscript.StandardVerifyFlags, nil, sigHashes, candidate.AmountSats, prevouts)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Execute(); err != nil {
		t.Fatalf("witness no longer verifies against retained script: %v", err)
	}
}

func TestP2WPKHSenderChangeDustAndAbsoluteFee(t *testing.T) {
	const amount = int64(10_000)
	tests := []struct {
		name       string
		remainder  int64
		wantOuts   int
		wantChange int64
		wantFee    int64
	}{
		{"exact no-change minimum", 110, 1, 0, 110},
		{"dust remainder becomes fee", 470, 1, 0, 470},
		{"change at dust boundary", 471, 2, 330, 141},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := &p2wpkhTestBackend{feeRate: 1}
			sender, _ := p2wpkhTestSender(t, backend)
			backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, amount+tc.remainder)}
			if _, err := sender.Send(context.Background(), p2wpkhTestRecipient(t, sender.net), amount); err != nil {
				t.Fatalf("Send: %v", err)
			}
			tx, err := decodeP2WPKHTestTx(backend.raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(tx.TxOut) != tc.wantOuts {
				t.Fatalf("outputs = %d, want %d", len(tx.TxOut), tc.wantOuts)
			}
			if tc.wantOuts == 2 && (tx.TxOut[1].Value != tc.wantChange || !bytes.Equal(tx.TxOut[1].PkScript, sender.sourceScript)) {
				t.Fatalf("change output = (%d,%x), want (%d,%x)", tx.TxOut[1].Value, tx.TxOut[1].PkScript, tc.wantChange, sender.sourceScript)
			}
			var outputs int64
			for _, out := range tx.TxOut {
				outputs += out.Value
			}
			if got := amount + tc.remainder - outputs; got != tc.wantFee {
				t.Fatalf("absolute fee = %d, want %d", got, tc.wantFee)
			}
		})
	}
}

func TestP2WPKHSenderDeterministicSelectionAndWireOrder(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	utxos := []UnspentOutput{
		p2wpkhTestUTXO(sender, 3, 2, 7_000),
		p2wpkhTestUTXO(sender, 1, 1, 9_000),
		p2wpkhTestUTXO(sender, 2, 0, 9_000),
	}
	recipient := p2wpkhTestRecipient(t, sender.net)
	raws := make([][]byte, 0, 3)
	orders := [][]int{{0, 1, 2}, {2, 0, 1}, {1, 2, 0}}
	for _, order := range orders {
		backend.utxos = nil
		for _, i := range order {
			backend.utxos = append(backend.utxos, utxos[i])
		}
		if _, err := sender.Send(context.Background(), recipient, 15_000); err != nil {
			t.Fatalf("Send order %v: %v", order, err)
		}
		raws = append(raws, append([]byte(nil), backend.raw...))
	}
	for i := 1; i < len(raws); i++ {
		if !bytes.Equal(raws[0], raws[i]) {
			t.Fatalf("backend ordering changed signed transaction (%x != %x)", raws[0], raws[i])
		}
	}
	tx, err := decodeP2WPKHTestTx(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.TxIn) != 2 {
		t.Fatalf("selected %d inputs, want two largest", len(tx.TxIn))
	}
	if bytes.Compare(tx.TxIn[0].PreviousOutPoint.Hash[:], tx.TxIn[1].PreviousOutPoint.Hash[:]) >= 0 {
		t.Fatal("final inputs are not in deterministic outpoint order")
	}
}

func TestP2WPKHSenderInsufficientMaxInputsAndOverflow(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	recipient := p2wpkhTestRecipient(t, sender.net)
	backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 100)}
	if _, err := sender.Send(context.Background(), recipient, 100); !errors.Is(err, ErrP2WPKHInsufficientFunds) {
		t.Fatalf("insufficient error = %v", err)
	}

	sender.config.MaxInputs = 1
	backend.utxos = []UnspentOutput{
		p2wpkhTestUTXO(sender, 1, 0, 600),
		p2wpkhTestUTXO(sender, 2, 0, 600),
	}
	if _, err := sender.Send(context.Background(), recipient, 1_000); !errors.Is(err, ErrP2WPKHMaxInputsExceeded) {
		t.Fatalf("max-input error = %v", err)
	}

	sender.config.MaxInputs = 10
	backend.utxos = []UnspentOutput{
		p2wpkhTestUTXO(sender, 1, 0, math.MaxInt64),
		p2wpkhTestUTXO(sender, 2, 0, 1),
	}
	if _, err := sender.Send(context.Background(), recipient, math.MaxInt64); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("input-total overflow error = %v", err)
	}
}

func TestP2WPKHSenderFeeBackendFallbackCapAndErrors(t *testing.T) {
	recipient := p2wpkhTestRecipient(t, &chaincfg.RegressionNetParams)
	tests := []struct {
		name         string
		feeRate      int64
		feeErr       error
		listErr      error
		broadcastErr error
		wantErr      error
		wantText     string
		wantSuccess  bool
	}{
		{name: "fallback sentinel", feeErr: fmt.Errorf("wrapped: %w", ErrP2WPKHFeeEstimateUnavailable), wantSuccess: true},
		{name: "zero rate", feeRate: 0, wantText: "invalid"},
		{name: "negative rate", feeRate: -1, wantText: "invalid"},
		{name: "over cap", feeRate: 101, wantText: "exceeds cap"},
		{name: "list error", listErr: errP2WPKHTestBackend, wantErr: errP2WPKHTestBackend},
		{name: "fee transport error", feeErr: errP2WPKHTestBackend, wantErr: errP2WPKHTestBackend},
		{name: "broadcast error", feeRate: 1, broadcastErr: errP2WPKHTestBackend, wantErr: errP2WPKHTestBackend},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := &p2wpkhTestBackend{feeRate: tc.feeRate, feeErr: tc.feeErr, listErr: tc.listErr, broadcastErr: tc.broadcastErr}
			sender, _ := p2wpkhTestSender(t, backend)
			backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 20_000)}
			_, err := sender.Send(context.Background(), recipient, 10_000)
			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("fallback Send: %v", err)
				}
				if backend.feeTarget != sender.config.FeeConfirmationTarget {
					t.Fatalf("fee target = %d, want %d", backend.feeTarget, sender.config.FeeConfirmationTarget)
				}
				return
			}
			if err == nil {
				t.Fatal("Send unexpectedly succeeded")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("error %v does not wrap %v", err, tc.wantErr)
			}
			if tc.wantText != "" && !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("error %q does not contain %q", err, tc.wantText)
			}
		})
	}
}

func TestP2WPKHSenderTransactionAndWitnesses(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 2}
	sender, signer := p2wpkhTestSender(t, backend)
	backend.utxos = []UnspentOutput{
		p2wpkhTestUTXO(sender, 3, 1, 8_000),
		p2wpkhTestUTXO(sender, 1, 0, 8_000),
	}
	recipient := p2wpkhTestRecipient(t, sender.net)
	txID, err := sender.Send(context.Background(), recipient, 12_000)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	tx, err := decodeP2WPKHTestTx(backend.raw)
	if err != nil {
		t.Fatal(err)
	}
	if txID != tx.TxHash().String() {
		t.Fatalf("returned txid = %s, local = %s", txID, tx.TxHash())
	}
	if tx.Version != wire.TxVersion || tx.LockTime != 0 {
		t.Fatalf("fixed fields = version %d locktime %d", tx.Version, tx.LockTime)
	}
	if len(tx.TxIn) != 2 || len(tx.TxOut) != 2 {
		t.Fatalf("shape = %d inputs/%d outputs, want 2/2", len(tx.TxIn), len(tx.TxOut))
	}
	for i, in := range tx.TxIn {
		if in.Sequence != wire.MaxTxInSequenceNum {
			t.Errorf("input %d sequence = %d, want final", i, in.Sequence)
		}
		if len(in.SignatureScript) != 0 || len(in.Witness) != 2 {
			t.Errorf("input %d scriptSig/witness shape = %x/%d", i, in.SignatureScript, len(in.Witness))
			continue
		}
		if got := in.Witness[0][len(in.Witness[0])-1]; got != byte(txscript.SigHashAll) {
			t.Errorf("input %d sighash byte = %x", i, got)
		}
		if !bytes.Equal(in.Witness[1], signer.PublicKey()) {
			t.Errorf("input %d witness pubkey mismatch", i)
		}
	}
	recipientAddr, _ := btcutil.DecodeAddress(recipient, sender.net)
	recipientScript, _ := txscript.PayToAddrScript(recipientAddr)
	if tx.TxOut[0].Value != 12_000 || !bytes.Equal(tx.TxOut[0].PkScript, recipientScript) {
		t.Fatalf("recipient output = (%d,%x)", tx.TxOut[0].Value, tx.TxOut[0].PkScript)
	}
	if !bytes.Equal(tx.TxOut[1].PkScript, sender.sourceScript) || tx.TxOut[1].Value < sender.config.DustThresholdSats {
		t.Fatalf("change output = (%d,%x)", tx.TxOut[1].Value, tx.TxOut[1].PkScript)
	}

	amounts := make(map[wire.OutPoint]int64)
	for _, u := range backend.utxos {
		h, _ := chainhash.NewHashFromStr(u.TxID)
		amounts[wire.OutPoint{Hash: *h, Index: u.Vout}] = u.AmountSats
	}
	prevouts := txscript.NewMultiPrevOutFetcher(nil)
	for _, in := range tx.TxIn {
		prevouts.AddPrevOut(in.PreviousOutPoint, wire.NewTxOut(amounts[in.PreviousOutPoint], sender.sourceScript))
	}
	sigHashes := txscript.NewTxSigHashes(tx, prevouts)
	for i, in := range tx.TxIn {
		amount := amounts[in.PreviousOutPoint]
		engine, err := txscript.NewEngine(sender.sourceScript, tx, i, txscript.StandardVerifyFlags, nil, sigHashes, amount, prevouts)
		if err != nil {
			t.Fatalf("NewEngine input %d: %v", i, err)
		}
		if err := engine.Execute(); err != nil {
			t.Fatalf("witness input %d failed script verification: %v", i, err)
		}
	}
}

func TestP2WPKHSenderSignerAndBroadcastValidation(t *testing.T) {
	keySigner := p2wpkhTestKeySigner(t, 1)
	cfg := p2wpkhTestConfig()
	recipient := p2wpkhTestRecipient(t, &chaincfg.RegressionNetParams)
	tests := []struct {
		name      string
		signFunc  func(context.Context, []byte) ([]byte, error)
		broadcast func(context.Context, []byte) (string, error)
		wantErr   error
		wantText  string
	}{
		{name: "sign error", signFunc: func(context.Context, []byte) ([]byte, error) { return nil, errP2WPKHTestBackend }, wantErr: errP2WPKHTestBackend},
		{name: "invalid DER", signFunc: func(context.Context, []byte) ([]byte, error) { return []byte{1, 2, 3}, nil }, wantText: "signature"},
		{name: "malformed broadcast txid", signFunc: keySigner.Sign, broadcast: func(context.Context, []byte) (string, error) { return "not-a-txid", nil }, wantText: "invalid transaction ID"},
		{name: "broadcast txid mismatch", signFunc: keySigner.Sign, broadcast: func(context.Context, []byte) (string, error) { var h chainhash.Hash; h[0] = 9; return h.String(), nil }, wantText: "mismatch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := &p2wpkhTestBackend{feeRate: 1, broadcast: tc.broadcast}
			signer := &configurableTestSigner{algorithm: sign.AlgSecp256k1, publicKey: keySigner.PublicKey(), signFunc: tc.signFunc}
			sender, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, signer, cfg)
			if err != nil {
				t.Fatal(err)
			}
			backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 20_000)}
			_, err = sender.Send(context.Background(), recipient, 10_000)
			if err == nil {
				t.Fatal("Send unexpectedly succeeded")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("error %v does not wrap %v", err, tc.wantErr)
			}
			if tc.wantText != "" && !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("error %q does not contain %q", err, tc.wantText)
			}
		})
	}
}

func TestP2WPKHSenderAcceptsUppercaseEquivalentBroadcastTxID(t *testing.T) {
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 20_000)}
	backend.broadcast = func(_ context.Context, raw []byte) (string, error) {
		tx, err := decodeP2WPKHTestTx(raw)
		if err != nil {
			return "", err
		}
		return strings.ToUpper(tx.TxHash().String()), nil
	}
	got, err := sender.Send(context.Background(), p2wpkhTestRecipient(t, sender.net), 10_000)
	if err != nil {
		t.Fatalf("uppercase equivalent broadcast txid rejected: %v", err)
	}
	tx, err := decodeP2WPKHTestTx(backend.raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != tx.TxHash().String() {
		t.Fatalf("Send returned %q, want canonical local %q", got, tx.TxHash())
	}
}

func TestP2WPKHSenderContextCancellationAndSerializedSend(t *testing.T) {
	enteredBroadcast := make(chan struct{})
	releaseBroadcast := make(chan struct{})
	backend := &p2wpkhTestBackend{feeRate: 1}
	sender, _ := p2wpkhTestSender(t, backend)
	backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 20_000)}
	backend.broadcast = func(ctx context.Context, raw []byte) (string, error) {
		select {
		case <-enteredBroadcast:
		default:
			close(enteredBroadcast)
		}
		select {
		case <-releaseBroadcast:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		tx, err := decodeP2WPKHTestTx(raw)
		if err != nil {
			return "", err
		}
		return tx.TxHash().String(), nil
	}
	recipient := p2wpkhTestRecipient(t, sender.net)
	firstDone := make(chan error, 1)
	go func() {
		_, err := sender.Send(context.Background(), recipient, 10_000)
		firstDone <- err
	}()
	select {
	case <-enteredBroadcast:
	case <-time.After(2 * time.Second):
		t.Fatal("first send did not reach broadcast")
	}

	ctx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() {
		_, err := sender.Send(ctx, recipient, 10_000)
		secondDone <- err
	}()
	cancel()
	select {
	case err := <-secondDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiting send error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled send waited indefinitely for serialization gate")
	}
	backend.mu.Lock()
	listCalls := backend.listCalls
	backend.mu.Unlock()
	if listCalls != 1 {
		t.Fatalf("cancelled waiting send reached discovery; list calls = %d", listCalls)
	}
	close(releaseBroadcast)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Send: %v", err)
	}

	preCancelled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if _, err := sender.Send(preCancelled, recipient, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled Send error = %v", err)
	}
	if _, err := sender.Send(nil, recipient, 1); err == nil {
		t.Fatal("nil context accepted")
	}
}

func TestP2WPKHSenderCancellationDuringStages(t *testing.T) {
	for _, stage := range []string{"list", "fee", "sign", "broadcast"} {
		t.Run(stage, func(t *testing.T) {
			entered := make(chan struct{})
			backend := &p2wpkhTestBackend{feeRate: 1}
			keySigner := p2wpkhTestKeySigner(t, 1)
			signer := sign.Signer(keySigner)
			waitForCancellation := func(ctx context.Context) {
				close(entered)
				<-ctx.Done()
			}
			switch stage {
			case "list":
				backend.listHook = waitForCancellation
			case "fee":
				backend.feeHook = waitForCancellation
			case "sign":
				signer = &configurableTestSigner{
					algorithm: sign.AlgSecp256k1,
					publicKey: keySigner.PublicKey(),
					signFunc: func(ctx context.Context, _ []byte) ([]byte, error) {
						waitForCancellation(ctx)
						return nil, errP2WPKHTestBackend
					},
				}
			case "broadcast":
				backend.broadcast = func(ctx context.Context, _ []byte) (string, error) {
					waitForCancellation(ctx)
					return "", errP2WPKHTestBackend
				}
			}
			sender, err := NewP2WPKHSender(&chaincfg.RegressionNetParams, backend, signer, p2wpkhTestConfig())
			if err != nil {
				t.Fatal(err)
			}
			backend.utxos = []UnspentOutput{p2wpkhTestUTXO(sender, 1, 0, 20_000)}
			recipient := p2wpkhTestRecipient(t, sender.net)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				_, err := sender.Send(ctx, recipient, 10_000)
				done <- err
			}()
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatalf("Send did not reach %s stage", stage)
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("%s cancellation error = %v, want context.Canceled", stage, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("Send did not return after cancellation during %s", stage)
			}
		})
	}
}

func TestClientSendRawTransactionReturnsRPCResult(t *testing.T) {
	const want = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"result":%q,"error":null,"id":"sdk"}`, want)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	client := NewClient("http://bitcoind.invalid", "", "", "", WithHTTPClient(httpClient))
	got, err := client.SendRawTransaction(context.Background(), "00")
	if err != nil {
		t.Fatalf("SendRawTransaction: %v", err)
	}
	if got != want {
		t.Fatalf("SendRawTransaction = %q, want %q", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

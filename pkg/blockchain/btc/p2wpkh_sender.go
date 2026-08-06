package btc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

var (
	// ErrP2WPKHFeeEstimateUnavailable is the only backend error for which a
	// sender uses its configured fallback fee rate. Backends should wrap this
	// error only when a valid backend response says no estimate is available.
	ErrP2WPKHFeeEstimateUnavailable = errors.New("btc: P2WPKH fee estimate unavailable")
	// ErrP2WPKHInsufficientFunds reports that every confirmed candidate output
	// together is unable to fund the amount and its minimum fee.
	ErrP2WPKHInsufficientFunds = errors.New("btc: insufficient P2WPKH funds")
	// ErrP2WPKHMaxInputsExceeded reports that a payable transaction would need
	// more inputs than the sender's configured bound.
	ErrP2WPKHMaxInputsExceeded = errors.New("btc: P2WPKH maximum input count exceeded")
)

// UnspentOutput is a candidate output supplied by a chain-data backend.
// AmountSats is an exact integer amount. ScriptPubKey identifies the output
// type and owner; senders must validate it before signing a spend.
type UnspentOutput struct {
	TxID          string
	Vout          uint32
	AmountSats    int64
	ScriptPubKey  []byte
	Confirmations uint64
}

// P2WPKHBackend supplies chain data and broadcast without prescribing a
// Bitcoin Core, Esplora, wallet, or transport implementation.
type P2WPKHBackend interface {
	ListUnspent(ctx context.Context, address string, minConfirmations uint64) ([]UnspentOutput, error)
	FeeRateSatPerVByte(ctx context.Context, confirmationTarget int) (int64, error)
	Broadcast(ctx context.Context, rawTx []byte) (txID string, err error)
}

// P2WPKHConfig is the policy for one sender. All fields are required and are
// validated by NewP2WPKHSender; this reusable API has no implicit defaults.
type P2WPKHConfig struct {
	MinConfirmations           uint64
	FeeConfirmationTarget      int
	FallbackFeeRateSatPerVByte int64
	FeeCapSatPerVByte          int64
	DustThresholdSats          int64
	MaxInputs                  int
}

// P2WPKHSender spends confirmed P2WPKH outputs controlled by one compressed
// secp256k1 key. Send calls on one instance are serialized to reduce accidental
// reuse of the same UTXO snapshot. This is not a durable reservation: callers
// whose backend omits mempool spends (for example, scantxoutset) must wait for
// confirmation before sending again from the same source.
type P2WPKHSender struct {
	net          *chaincfg.Params
	backend      P2WPKHBackend
	signer       sign.Signer
	config       P2WPKHConfig
	publicKey    []byte
	address      btcutil.Address
	addressText  string
	sourceScript []byte
	scriptCode   []byte
	sendGate     chan struct{}
}

// NewP2WPKHSender constructs a reusable, single-key native SegWit sender.
//
// Callers should pass one of btcsuite's predefined network parameters rather
// than constructing chaincfg.Params themselves, for example:
//
//	&chaincfg.MainNetParams
//	&chaincfg.TestNet3Params
//	&chaincfg.RegressionNetParams
//	&chaincfg.SigNetParams
func NewP2WPKHSender(net *chaincfg.Params, backend P2WPKHBackend, signer sign.Signer, cfg P2WPKHConfig) (*P2WPKHSender, error) {
	if net == nil {
		return nil, fmt.Errorf("btc: P2WPKH network is required")
	}
	if nilInterface(backend) {
		return nil, fmt.Errorf("btc: P2WPKH backend is required")
	}
	if nilInterface(signer) {
		return nil, fmt.Errorf("btc: P2WPKH signer is required")
	}
	if err := validateP2WPKHConfig(cfg); err != nil {
		return nil, err
	}
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: P2WPKH signer must be secp256k1, got %s", signer.Algorithm())
	}

	publicKey := append([]byte(nil), signer.PublicKey()...)
	if len(publicKey) != btcec.PubKeyBytesLenCompressed || (publicKey[0] != 0x02 && publicKey[0] != 0x03) {
		return nil, fmt.Errorf("btc: P2WPKH signer public key must be 33-byte compressed secp256k1")
	}
	parsedKey, err := btcec.ParsePubKey(publicKey)
	if err != nil || !bytes.Equal(parsedKey.SerializeCompressed(), publicKey) {
		return nil, fmt.Errorf("btc: invalid compressed secp256k1 public key")
	}

	address, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(publicKey), net)
	if err != nil {
		return nil, fmt.Errorf("btc: derive P2WPKH sender address: %w", err)
	}
	sourceScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, fmt.Errorf("btc: derive P2WPKH source script: %w", err)
	}
	scriptCode, err := p2wpkhWitnessScriptCode(btcutil.Hash160(publicKey))
	if err != nil {
		return nil, fmt.Errorf("btc: build P2WPKH scriptCode: %w", err)
	}

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &P2WPKHSender{
		net:          net,
		backend:      backend,
		signer:       signer,
		config:       cfg,
		publicKey:    publicKey,
		address:      address,
		addressText:  address.EncodeAddress(),
		sourceScript: append([]byte(nil), sourceScript...),
		scriptCode:   append([]byte(nil), scriptCode...),
		sendGate:     gate,
	}, nil
}

// p2wpkhWitnessScriptCode returns the scriptCode required by BIP-143 when
// signing a native P2WPKH input. It deliberately has the traditional P2PKH
// template (DUP HASH160 <pubKeyHash> EQUALVERIFY CHECKSIG); the witness
// prevout's scriptPubKey, OP_0 <pubKeyHash>, is not the scriptCode committed to
// by a P2WPKH signature digest.
//
// We build it directly because txscript's higher-level signing helper requires
// an in-memory private key, while P2WPKHSender supports external sign.Signer
// implementations such as KMS-backed signers.
func p2wpkhWitnessScriptCode(publicKeyHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(publicKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

// Address returns the sender's P2WPKH source and change address.
func (s *P2WPKHSender) Address() string { return s.addressText }

// Send builds, signs, and broadcasts a final, non-RBF P2WPKH transaction.
func (s *P2WPKHSender) Send(ctx context.Context, recipient string, amountSats int64) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("btc: P2WPKH send context is required")
	}
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("btc: P2WPKH send: %w", ctx.Err())
	case <-s.sendGate:
	}
	defer func() { s.sendGate <- struct{}{} }()

	if amountSats <= 0 {
		return "", fmt.Errorf("btc: P2WPKH amount must be positive, got %d", amountSats)
	}
	recipientAddr, err := btcutil.DecodeAddress(recipient, s.net)
	if err != nil {
		return "", fmt.Errorf("btc: decode P2WPKH recipient: %w", err)
	}
	if !recipientAddr.IsForNet(s.net) {
		return "", fmt.Errorf("btc: recipient %q is not for network %s", recipient, s.net.Name)
	}
	recipientScript, err := txscript.PayToAddrScript(recipientAddr)
	if err != nil {
		return "", fmt.Errorf("btc: build recipient script: %w", err)
	}

	candidates, err := s.backend.ListUnspent(ctx, s.addressText, s.config.MinConfirmations)
	if err != nil {
		return "", backendStageError(ctx, "list P2WPKH UTXOs", err)
	}
	validated, err := s.validateCandidates(candidates)
	if err != nil {
		return "", err
	}
	if err := contextError(ctx, "after listing P2WPKH UTXOs"); err != nil {
		return "", err
	}

	feeRate, err := s.backend.FeeRateSatPerVByte(ctx, s.config.FeeConfirmationTarget)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("btc: estimate P2WPKH fee: %w", ctx.Err())
		}
		if errors.Is(err, ErrP2WPKHFeeEstimateUnavailable) {
			feeRate = s.config.FallbackFeeRateSatPerVByte
		} else {
			return "", fmt.Errorf("btc: estimate P2WPKH fee: %w", err)
		}
	}
	if feeRate <= 0 {
		return "", fmt.Errorf("btc: invalid P2WPKH fee rate %d sat/vB", feeRate)
	}
	if feeRate > s.config.FeeCapSatPerVByte {
		return "", fmt.Errorf("btc: P2WPKH fee rate %d sat/vB exceeds cap %d", feeRate, s.config.FeeCapSatPerVByte)
	}
	if err := contextError(ctx, "after estimating P2WPKH fee"); err != nil {
		return "", err
	}

	selection, err := selectP2WPKH(validated, amountSats, feeRate, recipientScript, s.sourceScript, s.config)
	if err != nil {
		return "", err
	}
	tx, prevouts, err := s.buildTransaction(selection, recipientScript, amountSats, feeRate)
	if err != nil {
		return "", err
	}
	if err := s.signTransaction(ctx, tx, prevouts); err != nil {
		return "", err
	}
	raw, err := serializeTx(tx)
	if err != nil {
		return "", fmt.Errorf("btc: serialize P2WPKH transaction: %w", err)
	}
	localHash := tx.TxHash()
	returnedID, err := s.backend.Broadcast(ctx, append([]byte(nil), raw...))
	if err != nil {
		return "", backendStageError(ctx, "broadcast P2WPKH transaction", err)
	}
	returnedHash, err := chainhash.NewHashFromStr(returnedID)
	if err != nil {
		return "", fmt.Errorf("btc: broadcast returned invalid transaction ID %q: %w", returnedID, err)
	}
	if !returnedHash.IsEqual(&localHash) {
		return "", fmt.Errorf("btc: broadcast transaction ID mismatch: local %s, backend %s", localHash.String(), returnedID)
	}
	return localHash.String(), nil
}

func validateP2WPKHConfig(cfg P2WPKHConfig) error {
	if cfg.MinConfirmations < 1 {
		return fmt.Errorf("btc: P2WPKH minimum confirmations must be at least 1")
	}
	if cfg.FeeConfirmationTarget <= 0 {
		return fmt.Errorf("btc: P2WPKH fee confirmation target must be positive")
	}
	if cfg.FallbackFeeRateSatPerVByte <= 0 {
		return fmt.Errorf("btc: P2WPKH fallback fee rate must be positive")
	}
	if cfg.FeeCapSatPerVByte <= 0 {
		return fmt.Errorf("btc: P2WPKH fee cap must be positive")
	}
	if cfg.FallbackFeeRateSatPerVByte > cfg.FeeCapSatPerVByte {
		return fmt.Errorf("btc: P2WPKH fallback fee rate %d exceeds cap %d", cfg.FallbackFeeRateSatPerVByte, cfg.FeeCapSatPerVByte)
	}
	if cfg.DustThresholdSats <= 0 {
		return fmt.Errorf("btc: P2WPKH dust threshold must be positive")
	}
	if cfg.MaxInputs <= 0 {
		return fmt.Errorf("btc: P2WPKH maximum input count must be positive")
	}
	return nil
}

type validatedUTXO struct {
	outpoint     wire.OutPoint
	amountSats   int64
	scriptPubKey []byte
}

func (s *P2WPKHSender) validateCandidates(candidates []UnspentOutput) ([]validatedUTXO, error) {
	validated := make([]validatedUTXO, 0, len(candidates))
	seen := make(map[wire.OutPoint]struct{}, len(candidates))
	for i, candidate := range candidates {
		hash, err := chainhash.NewHashFromStr(candidate.TxID)
		if err != nil {
			return nil, fmt.Errorf("btc: invalid P2WPKH UTXO %d txid %q: %w", i, candidate.TxID, err)
		}
		if candidate.AmountSats <= 0 {
			return nil, fmt.Errorf("btc: invalid P2WPKH UTXO %s:%d amount %d", candidate.TxID, candidate.Vout, candidate.AmountSats)
		}
		if candidate.Confirmations < s.config.MinConfirmations {
			return nil, fmt.Errorf("btc: P2WPKH UTXO %s:%d has %d confirmations, need %d", candidate.TxID, candidate.Vout, candidate.Confirmations, s.config.MinConfirmations)
		}
		if !bytes.Equal(candidate.ScriptPubKey, s.sourceScript) {
			return nil, fmt.Errorf("btc: P2WPKH UTXO %s:%d script does not match sender address", candidate.TxID, candidate.Vout)
		}
		outpoint := wire.OutPoint{Hash: *hash, Index: candidate.Vout}
		if _, exists := seen[outpoint]; exists {
			return nil, fmt.Errorf("btc: duplicate P2WPKH UTXO %s:%d", candidate.TxID, candidate.Vout)
		}
		seen[outpoint] = struct{}{}
		validated = append(validated, validatedUTXO{
			outpoint:     outpoint,
			amountSats:   candidate.AmountSats,
			scriptPubKey: append([]byte(nil), candidate.ScriptPubKey...),
		})
	}
	return validated, nil
}

type p2wpkhSelection struct {
	inputs     []validatedUTXO
	total      int64
	minimumFee int64
	withChange bool
}

func selectP2WPKH(candidates []validatedUTXO, amount, feeRate int64, recipientScript, changeScript []byte, cfg P2WPKHConfig) (p2wpkhSelection, error) {
	pool := append([]validatedUTXO(nil), candidates...)
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].amountSats != pool[j].amountSats {
			return pool[i].amountSats > pool[j].amountSats
		}
		iTxID, jTxID := pool[i].outpoint.Hash.String(), pool[j].outpoint.Hash.String()
		if iTxID != jTxID {
			return iTxID < jTxID
		}
		return pool[i].outpoint.Index < pool[j].outpoint.Index
	})

	var total int64
	for i := range pool {
		var ok bool
		total, ok = checkedAddInt64(total, pool[i].amountSats)
		if !ok {
			return p2wpkhSelection{}, fmt.Errorf("btc: P2WPKH input total overflows int64")
		}
		n := i + 1
		selection, payable, err := evaluateP2WPKHPrefix(pool[:n], total, amount, feeRate, recipientScript, changeScript, cfg.DustThresholdSats)
		if err != nil {
			return p2wpkhSelection{}, err
		}
		if !payable {
			continue
		}
		if n > cfg.MaxInputs {
			return p2wpkhSelection{}, fmt.Errorf("%w: need %d inputs, configured maximum is %d", ErrP2WPKHMaxInputsExceeded, n, cfg.MaxInputs)
		}
		return selection, nil
	}
	return p2wpkhSelection{}, fmt.Errorf("%w: have %d sats, need %d sats plus fee at %d sat/vB", ErrP2WPKHInsufficientFunds, total, amount, feeRate)
}

func evaluateP2WPKHPrefix(inputs []validatedUTXO, total, amount, feeRate int64, recipientScript, changeScript []byte, dust int64) (p2wpkhSelection, bool, error) {
	if total < amount {
		return p2wpkhSelection{}, false, nil
	}
	remainder := total - amount
	changeFee, err := p2wpkhFee(len(inputs), [][]byte{recipientScript, changeScript}, feeRate)
	if err != nil {
		return p2wpkhSelection{}, false, err
	}
	if remainder >= changeFee {
		change := remainder - changeFee
		if change >= dust {
			return p2wpkhSelection{inputs: append([]validatedUTXO(nil), inputs...), total: total, minimumFee: changeFee, withChange: true}, true, nil
		}
	}
	noChangeFee, err := p2wpkhFee(len(inputs), [][]byte{recipientScript}, feeRate)
	if err != nil {
		return p2wpkhSelection{}, false, err
	}
	if remainder >= noChangeFee {
		return p2wpkhSelection{inputs: append([]validatedUTXO(nil), inputs...), total: total, minimumFee: noChangeFee}, true, nil
	}
	return p2wpkhSelection{}, false, nil
}

// p2wpkhFee estimates the final native-SegWit transaction using exact output
// script lengths and a conservative 72-byte DER signature plus sighash byte.
func p2wpkhFee(numInputs int, outputScripts [][]byte, feeRate int64) (int64, error) {
	if numInputs <= 0 || len(outputScripts) == 0 || feeRate <= 0 {
		return 0, fmt.Errorf("btc: invalid P2WPKH fee estimate parameters")
	}
	baseSize := int64(4 + wire.VarIntSerializeSize(uint64(numInputs)) + wire.VarIntSerializeSize(uint64(len(outputScripts))) + 4)
	inputBytes, ok := checkedMulInt64(int64(numInputs), 41)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH fee size overflow")
	}
	baseSize, ok = checkedAddInt64(baseSize, inputBytes)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH fee size overflow")
	}
	for _, script := range outputScripts {
		outputSize := int64(8 + wire.VarIntSerializeSize(uint64(len(script))))
		outputSize, ok = checkedAddInt64(outputSize, int64(len(script)))
		if !ok {
			return 0, fmt.Errorf("btc: P2WPKH output size overflow")
		}
		baseSize, ok = checkedAddInt64(baseSize, outputSize)
		if !ok {
			return 0, fmt.Errorf("btc: P2WPKH fee size overflow")
		}
	}
	weight, ok := checkedMulInt64(baseSize, 4)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH transaction weight overflow")
	}
	// Two weight units for marker+flag, and 109 witness bytes per input:
	// stack count + (length + DER + sighash) + (length + compressed pubkey).
	witnessSize, ok := checkedMulInt64(int64(numInputs), 109)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH witness size overflow")
	}
	weight, ok = checkedAddInt64(weight, 2)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH transaction weight overflow")
	}
	weight, ok = checkedAddInt64(weight, witnessSize)
	if !ok || weight > math.MaxInt64-3 {
		return 0, fmt.Errorf("btc: P2WPKH transaction weight overflow")
	}
	vsize := (weight + 3) / 4
	fee, ok := checkedMulInt64(vsize, feeRate)
	if !ok {
		return 0, fmt.Errorf("btc: P2WPKH fee overflows int64")
	}
	return fee, nil
}

func (s *P2WPKHSender) buildTransaction(selection p2wpkhSelection, recipientScript []byte, amount, feeRate int64) (*wire.MsgTx, *txscript.MultiPrevOutFetcher, error) {
	ordered := append([]validatedUTXO(nil), selection.inputs...)
	sort.Slice(ordered, func(i, j int) bool {
		if cmp := bytes.Compare(ordered[i].outpoint.Hash[:], ordered[j].outpoint.Hash[:]); cmp != 0 {
			return cmp < 0
		}
		return ordered[i].outpoint.Index < ordered[j].outpoint.Index
	})

	tx := wire.NewMsgTx(wire.TxVersion)
	prevouts := txscript.NewMultiPrevOutFetcher(nil)
	for _, input := range ordered {
		txIn := wire.NewTxIn(&input.outpoint, nil, nil)
		txIn.Sequence = wire.MaxTxInSequenceNum
		tx.AddTxIn(txIn)
		prevouts.AddPrevOut(input.outpoint, wire.NewTxOut(input.amountSats, append([]byte(nil), input.scriptPubKey...)))
	}
	tx.AddTxOut(wire.NewTxOut(amount, append([]byte(nil), recipientScript...)))

	actualFee := selection.total - amount
	if selection.withChange {
		change := actualFee - selection.minimumFee
		if change < s.config.DustThresholdSats {
			return nil, nil, fmt.Errorf("btc: selected P2WPKH change %d below dust threshold %d", change, s.config.DustThresholdSats)
		}
		tx.AddTxOut(wire.NewTxOut(change, append([]byte(nil), s.sourceScript...)))
		actualFee = selection.minimumFee
	}
	minimumFee, err := p2wpkhFee(len(tx.TxIn), outputScripts(tx.TxOut), feeRate)
	if err != nil {
		return nil, nil, fmt.Errorf("btc: recheck P2WPKH transaction fee: %w", err)
	}
	if actualFee < minimumFee {
		return nil, nil, fmt.Errorf("btc: P2WPKH transaction fee %d below required %d", actualFee, minimumFee)
	}
	return tx, prevouts, nil
}

func (s *P2WPKHSender) signTransaction(ctx context.Context, tx *wire.MsgTx, prevouts *txscript.MultiPrevOutFetcher) error {
	sigHashes := txscript.NewTxSigHashes(tx, prevouts)
	for i, txIn := range tx.TxIn {
		if err := contextError(ctx, fmt.Sprintf("before signing P2WPKH input %d", i)); err != nil {
			return err
		}
		prevout := prevouts.FetchPrevOutput(txIn.PreviousOutPoint)
		if prevout == nil {
			return fmt.Errorf("btc: missing P2WPKH prevout for input %d", i)
		}
		digest, err := txscript.CalcWitnessSigHash(s.scriptCode, sigHashes, txscript.SigHashAll, tx, i, prevout.Value)
		if err != nil {
			return fmt.Errorf("btc: calculate P2WPKH sighash for input %d: %w", i, err)
		}
		der, err := s.signer.Sign(ctx, digest)
		if err != nil {
			return backendStageError(ctx, fmt.Sprintf("sign P2WPKH input %d", i), err)
		}
		signature, err := btcecdsa.ParseDERSignature(der)
		if err != nil || !bytes.Equal(signature.Serialize(), der) {
			if err == nil {
				err = fmt.Errorf("non-canonical DER encoding")
			}
			return fmt.Errorf("btc: invalid P2WPKH signature for input %d: %w", i, err)
		}
		sigWithHashType := make([]byte, len(der)+1)
		copy(sigWithHashType, der)
		sigWithHashType[len(der)] = byte(txscript.SigHashAll)
		tx.TxIn[i].Witness = wire.TxWitness{sigWithHashType, append([]byte(nil), s.publicKey...)}
	}
	return nil
}

func outputScripts(outputs []*wire.TxOut) [][]byte {
	scripts := make([][]byte, len(outputs))
	for i := range outputs {
		scripts[i] = outputs[i].PkScript
	}
	return scripts
}

func checkedAddInt64(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, false
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, false
	}
	return a + b, true
}

func checkedMulInt64(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a != 0 && b > math.MaxInt64/a {
		return 0, false
	}
	return a * b, true
}

func contextError(ctx context.Context, stage string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("btc: %s: %w", stage, err)
	}
	return nil
}

func backendStageError(ctx context.Context, stage string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("btc: %s: %w", stage, ctxErr)
	}
	return fmt.Errorf("btc: %s: %w", stage, err)
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	kind := reflect.ValueOf(value).Kind()
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflect.ValueOf(value).IsNil()
	default:
		return false
	}
}

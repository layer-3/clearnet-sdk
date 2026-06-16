package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// executionScanBlocks bounds how many recent blocks VerifyExecution scans for
// the OP_RETURN <withdrawalID> marker.
const executionScanBlocks = int64(100)

// Config carries the per-chain tunables the finalizer needs.
type Config struct {
	ConfirmationDepth uint64 // min confirmations for a vault UTXO to be spendable
	FeeConfTarget     int    // estimatesmartfee confirmation target (blocks)
	FallbackFeeRate   int64  // sat/vByte used when the node can't estimate
	FeeCapSatPerVByte int64  // ceiling Validate accepts on a canonical tx
}

// WithdrawalFinalizer is the Bitcoin m-of-n P2WSH vault withdrawal path. It
// owns this node's signer (one of the vault keys); the quorum's shares are
// merged off-mesh by the caller. It implements core.VaultWithdrawalFinalizer.
type WithdrawalFinalizer struct {
	net       *chaincfg.Params
	rpc       RPC
	signer    sign.Signer
	signerPub []byte
	pubkeys   [][]byte
	threshold int
	cfg       Config

	vaultAddr   btcutil.Address
	vaultScript []byte
	pubkeyPos   map[string]int

	mu           sync.RWMutex
	spendScripts map[string][]byte
	watchAddrs   []string
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer builds the vault finalizer. pubkeys are the providers'
// 33-byte compressed keys; signer is this node's identity and its public key
// must be one of pubkeys.
func NewWithdrawalFinalizer(net *chaincfg.Params, rpc RPC, signer sign.Signer, pubkeys [][]byte, threshold int, cfg Config) (*WithdrawalFinalizer, error) {
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: signer must be secp256k1, got %s", signer.Algorithm())
	}
	redeem, err := RedeemScript(threshold, pubkeys)
	if err != nil {
		return nil, err
	}
	vaultAddr, err := VaultAddress(redeem, net)
	if err != nil {
		return nil, fmt.Errorf("btc: derive vault address: %w", err)
	}
	vaultScript, err := PkScript(vaultAddr)
	if err != nil {
		return nil, fmt.Errorf("btc: vault pkScript: %w", err)
	}
	pos := redeemKeyPositions(pubkeys)
	pub := signer.PublicKey()
	if _, ok := pos[hex.EncodeToString(pub)]; !ok {
		return nil, fmt.Errorf("btc: signer pubkey not in the provided key set")
	}
	return &WithdrawalFinalizer{
		net:          net,
		rpc:          rpc,
		signer:       signer,
		signerPub:    pub,
		pubkeys:      pubkeys,
		threshold:    threshold,
		cfg:          cfg,
		vaultAddr:    vaultAddr,
		vaultScript:  vaultScript,
		pubkeyPos:    pos,
		spendScripts: map[string][]byte{strings.ToLower(hex.EncodeToString(vaultScript)): redeem},
		watchAddrs:   []string{vaultAddr.EncodeAddress()},
	}, nil
}

// RegisterDepositAccounts adds the tagged deposit addresses for the given
// account URIs to the spendable set, so withdrawals can select and sign UTXOs
// that landed at per-account deposit addresses.
func (f *WithdrawalFinalizer) RegisterDepositAccounts(accountURIs ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, acct := range accountURIs {
		redeem, err := TaggedRedeemScript(AccountTag(acct), f.threshold, f.pubkeys)
		if err != nil {
			return err
		}
		addr, err := VaultAddress(redeem, f.net)
		if err != nil {
			return err
		}
		pk, err := PkScript(addr)
		if err != nil {
			return err
		}
		key := strings.ToLower(hex.EncodeToString(pk))
		if _, ok := f.spendScripts[key]; !ok {
			f.spendScripts[key] = redeem
			f.watchAddrs = append(f.watchAddrs, addr.EncodeAddress())
		}
	}
	return nil
}

func (f *WithdrawalFinalizer) watchAddresses() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]string, len(f.watchAddrs))
	copy(out, f.watchAddrs)
	return out
}

func (f *WithdrawalFinalizer) resolveScript(pkScriptHex string) ([]byte, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	s, ok := f.spendScripts[strings.ToLower(pkScriptHex)]
	return s, ok
}

// Pack selects vault UTXOs, sizes the fee, and builds the canonical unsigned
// transaction (recipient, optional change, OP_RETURN <withdrawalID>).
func (f *WithdrawalFinalizer) Pack(ctx context.Context, op *core.WithdrawalOp, withdrawalID [32]byte) ([]byte, error) {
	recipient, amount, err := f.parseOp(op)
	if err != nil {
		return nil, err
	}
	unspent, err := f.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), f.watchAddresses())
	if err != nil {
		return nil, fmt.Errorf("btc: list vault utxos: %w", err)
	}
	utxos, err := f.toUTXOs(unspent)
	if err != nil {
		return nil, err
	}
	feeRate, err := f.rpc.EstimateSmartFeeSatPerVByte(ctx, f.cfg.FeeConfTarget, f.cfg.FallbackFeeRate)
	if err != nil {
		return nil, fmt.Errorf("btc: estimate fee: %w", err)
	}
	selected, feeSats, err := SelectUTXOs(utxos, amount, feeRate, 2)
	if err != nil {
		return nil, err
	}
	tx, err := BuildUnsignedTx(selected, recipient, amount, f.vaultAddr, withdrawalID, feeSats)
	if err != nil {
		return nil, err
	}
	return serializeTx(tx)
}

// Validate re-derives the trust-bound shape from the op and asserts the packed
// tx matches: output 0 pays the exact recipient/amount, the final output is
// OP_RETURN <withdrawalID>, any middle output is change to the vault, every
// input is a confirmed vault UTXO, and the implied fee is within the ceiling.
func (f *WithdrawalFinalizer) Validate(ctx context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte) error {
	recipient, amount, err := f.parseOp(op)
	if err != nil {
		return err
	}
	tx, err := deserializeTx(packed)
	if err != nil {
		return fmt.Errorf("btc validate: %w", err)
	}
	if n := len(tx.TxOut); n != 2 && n != 3 {
		return fmt.Errorf("btc validate: expected 2 or 3 outputs, got %d", n)
	}
	recipientScript, err := txscript.PayToAddrScript(recipient)
	if err != nil {
		return fmt.Errorf("btc validate: recipient script: %w", err)
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, recipientScript) {
		return fmt.Errorf("btc validate: output 0 not the op recipient")
	}
	if tx.TxOut[0].Value != amount {
		return fmt.Errorf("btc validate: output 0 value %d != op amount %d", tx.TxOut[0].Value, amount)
	}
	wantOpReturn, err := txscript.NullDataScript(withdrawalID[:])
	if err != nil {
		return fmt.Errorf("btc validate: opreturn script: %w", err)
	}
	last := tx.TxOut[len(tx.TxOut)-1]
	if last.Value != 0 || !bytes.Equal(last.PkScript, wantOpReturn) {
		return fmt.Errorf("btc validate: final output is not OP_RETURN <withdrawalID>")
	}
	if len(tx.TxOut) == 3 {
		change := tx.TxOut[1]
		if !bytes.Equal(change.PkScript, f.vaultScript) {
			return fmt.Errorf("btc validate: change output not paid to the vault")
		}
		if change.Value < dustThresholdSats {
			return fmt.Errorf("btc validate: change output %d below dust", change.Value)
		}
	}
	totalIn, err := f.sumValidatedInputs(ctx, tx)
	if err != nil {
		return err
	}
	var totalOut int64
	for _, o := range tx.TxOut {
		totalOut += o.Value
	}
	fee := totalIn - totalOut
	if fee < 0 {
		return fmt.Errorf("btc validate: outputs exceed inputs (fee %d)", fee)
	}
	if cap := EstimateFeeSats(len(tx.TxIn), len(tx.TxOut), f.cfg.FeeCapSatPerVByte); f.cfg.FeeCapSatPerVByte > 0 && fee > cap {
		return fmt.Errorf("btc validate: fee %d exceeds ceiling %d", fee, cap)
	}
	return nil
}

// SigShare is one signer's contribution: a DER+sighash signature per input, in
// input order, plus the signer's 33-byte compressed pubkey (hex).
type SigShare struct {
	PubKey string   `json:"pubkey"`
	Sigs   []string `json:"sigs"`
}

// Sign produces this node's signature over every input of the packed tx, each
// under its own witness script. Returns a JSON SigShare.
func (f *WithdrawalFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	tx, err := deserializeTx(packed)
	if err != nil {
		return nil, fmt.Errorf("btc sign: %w", err)
	}
	prevFetcher, amounts, redeems, err := f.prevOutputs(ctx, tx)
	if err != nil {
		return nil, err
	}
	sigs := make([]string, len(tx.TxIn))
	for idx := range tx.TxIn {
		sighash, err := SighashAll(tx, idx, redeems[idx], amounts[idx], prevFetcher)
		if err != nil {
			return nil, fmt.Errorf("btc sign: sighash input %d: %w", idx, err)
		}
		der, err := f.signer.Sign(ctx, sighash)
		if err != nil {
			return nil, fmt.Errorf("btc sign: input %d: %w", idx, err)
		}
		sigs[idx] = hex.EncodeToString(append(der, byte(txscript.SigHashAll)))
	}
	return json.Marshal(SigShare{PubKey: hex.EncodeToString(f.signerPub), Sigs: sigs})
}

// merge assembles the witness for every input from the collected shares (the
// threshold lowest by redeem-script key position) and returns the fully-signed
// tx serialization.
func (f *WithdrawalFinalizer) merge(ctx context.Context, packed []byte, shares [][]byte) ([]byte, error) {
	tx, err := deserializeTx(packed)
	if err != nil {
		return nil, fmt.Errorf("btc merge: %w", err)
	}
	_, _, redeems, err := f.prevOutputs(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("btc merge: %w", err)
	}

	parsed := make([]SigShare, 0, len(shares))
	for _, raw := range shares {
		var s SigShare
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("btc merge: decode share: %w", err)
		}
		if _, ok := f.pubkeyPos[strings.ToLower(s.PubKey)]; !ok {
			return nil, fmt.Errorf("btc merge: share from unknown signer %s", s.PubKey)
		}
		if len(s.Sigs) != len(tx.TxIn) {
			return nil, fmt.Errorf("btc merge: share has %d sigs, tx has %d inputs", len(s.Sigs), len(tx.TxIn))
		}
		parsed = append(parsed, s)
	}

	for idx := range tx.TxIn {
		type posSig struct {
			pos int
			sig []byte
		}
		collected := make([]posSig, 0, len(parsed))
		for _, s := range parsed {
			sig, err := hex.DecodeString(s.Sigs[idx])
			if err != nil {
				return nil, fmt.Errorf("btc merge: decode sig: %w", err)
			}
			collected = append(collected, posSig{pos: f.pubkeyPos[strings.ToLower(s.PubKey)], sig: sig})
		}
		if len(collected) < f.threshold {
			return nil, fmt.Errorf("btc merge: input %d has %d sigs, need %d", idx, len(collected), f.threshold)
		}
		sort.Slice(collected, func(i, j int) bool { return collected[i].pos < collected[j].pos })
		ordered := make([][]byte, f.threshold)
		for i := 0; i < f.threshold; i++ {
			ordered[i] = collected[i].sig
		}
		tx.TxIn[idx].Witness = AssembleWitness(redeems[idx], ordered)
	}
	return serializeTx(tx)
}

// Submit assembles the witnesses from the collected shares and broadcasts the
// signed tx, returning its hash. Idempotent on an already-known/spent reply.
func (f *WithdrawalFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (core.TxRef, error) {
	merged, err := f.merge(ctx, packed, shares)
	if err != nil {
		return core.TxRef{}, err
	}
	tx, err := deserializeTx(merged)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("btc submit: %w", err)
	}
	hash := [32]byte(tx.TxHash())
	txid := hashToTxid(hash)
	if _, err := f.rpc.SendRawTransaction(ctx, hex.EncodeToString(merged)); err != nil {
		if isAlreadyKnown(err) {
			return core.TxRef{Hash: hash, Raw: txid}, nil
		}
		return core.TxRef{}, fmt.Errorf("btc submit: sendrawtransaction: %w", err)
	}
	return core.TxRef{Hash: hash, Raw: txid}, nil
}

// VerifyExecution scans the most recent blocks for a tx carrying
// OP_RETURN <withdrawalID>. Returns the tx hash + true on a hit. Bounded by
// executionScanBlocks; a withdrawal older than that window reads as not-found.
func (f *WithdrawalFinalizer) VerifyExecution(ctx context.Context, withdrawalID [32]byte) ([32]byte, bool, error) {
	marker, err := txscript.NullDataScript(withdrawalID[:])
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("btc verify: opreturn script: %w", err)
	}
	markerHex := hex.EncodeToString(marker)

	head, err := f.rpc.GetBlockCount(ctx)
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("btc verify: block count: %w", err)
	}
	for h := head; h >= 0 && h > head-executionScanBlocks; h-- {
		blockHash, err := f.rpc.GetBlockHash(ctx, h)
		if err != nil {
			return [32]byte{}, false, fmt.Errorf("btc verify: block hash %d: %w", h, err)
		}
		txids, err := f.rpc.GetBlockTxids(ctx, blockHash)
		if err != nil {
			return [32]byte{}, false, fmt.Errorf("btc verify: block txids: %w", err)
		}
		for _, txid := range txids {
			raw, err := f.rpc.GetRawTransaction(ctx, txid)
			if err != nil || raw == nil {
				continue
			}
			for _, vo := range raw.Vouts {
				if strings.EqualFold(vo.ScriptPubKeyHex, markerHex) {
					return txidToHash(txid), true, nil
				}
			}
		}
	}
	return [32]byte{}, false, nil
}

// --- helpers ---

func (f *WithdrawalFinalizer) parseOp(op *core.WithdrawalOp) (btcutil.Address, int64, error) {
	addr, err := btcutil.DecodeAddress(op.Recipient, f.net)
	if err != nil {
		return nil, 0, fmt.Errorf("btc: decode recipient %q: %w", op.Recipient, err)
	}
	if !addr.IsForNet(f.net) {
		return nil, 0, fmt.Errorf("btc: recipient %q not valid for %s", op.Recipient, f.net.Name)
	}
	amt := op.Amount.BigInt()
	if !amt.IsInt64() || amt.Int64() <= 0 {
		return nil, 0, fmt.Errorf("btc: amount %s not a positive int64 satoshi value", op.Amount.String())
	}
	return addr, amt.Int64(), nil
}

func (f *WithdrawalFinalizer) toUTXOs(unspent []Unspent) ([]UTXO, error) {
	out := make([]UTXO, 0, len(unspent))
	for _, u := range unspent {
		if _, ok := f.resolveScript(u.ScriptPubKey); !ok {
			continue
		}
		h, err := chainhash.NewHashFromStr(u.TxID)
		if err != nil {
			return nil, fmt.Errorf("btc: bad utxo txid %q: %w", u.TxID, err)
		}
		out = append(out, UTXO{TxID: *h, Vout: u.Vout, Amount: u.AmountSats})
	}
	return out, nil
}

func (f *WithdrawalFinalizer) prevOutputs(ctx context.Context, tx *wire.MsgTx) (txscript.PrevOutputFetcher, []int64, [][]byte, error) {
	fetcher := txscript.NewMultiPrevOutFetcher(nil)
	amounts := make([]int64, len(tx.TxIn))
	redeems := make([][]byte, len(tx.TxIn))
	for i, in := range tx.TxIn {
		out, err := f.rpc.GetTxOut(ctx, in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index, true)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("btc: gettxout input %d: %w", i, err)
		}
		if out == nil {
			return nil, nil, nil, fmt.Errorf("btc: input %d references a spent or unknown output", i)
		}
		redeem, ok := f.resolveScript(out.ScriptPubKey)
		if !ok {
			return nil, nil, nil, fmt.Errorf("btc: input %d not a vault output", i)
		}
		pkScript, err := hex.DecodeString(out.ScriptPubKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("btc: input %d bad scriptPubKey %q: %w", i, out.ScriptPubKey, err)
		}
		amounts[i] = out.AmountSats
		redeems[i] = redeem
		fetcher.AddPrevOut(in.PreviousOutPoint, wire.NewTxOut(out.AmountSats, pkScript))
	}
	return fetcher, amounts, redeems, nil
}

func (f *WithdrawalFinalizer) sumValidatedInputs(ctx context.Context, tx *wire.MsgTx) (int64, error) {
	var total int64
	for i, in := range tx.TxIn {
		out, err := f.rpc.GetTxOut(ctx, in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index, true)
		if err != nil {
			return 0, fmt.Errorf("btc validate: gettxout input %d: %w", i, err)
		}
		if out == nil {
			return 0, fmt.Errorf("btc validate: input %d spent or unknown", i)
		}
		if _, ok := f.resolveScript(out.ScriptPubKey); !ok {
			return 0, fmt.Errorf("btc validate: input %d not a vault output", i)
		}
		if out.Confirmations < int64(f.cfg.ConfirmationDepth) {
			return 0, fmt.Errorf("btc validate: input %d has %d confs, need %d", i, out.Confirmations, f.cfg.ConfirmationDepth)
		}
		total += out.AmountSats
	}
	return total, nil
}

func serializeTx(tx *wire.MsgTx) ([]byte, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, fmt.Errorf("serialize tx: %w", err)
	}
	return buf.Bytes(), nil
}

func deserializeTx(b []byte) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	if err := tx.Deserialize(bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("deserialize tx: %w", err)
	}
	return tx, nil
}

func redeemKeyPositions(pubkeys [][]byte) map[string]int {
	sorted := make([][]byte, len(pubkeys))
	for i, pk := range pubkeys {
		cp := make([]byte, len(pk))
		copy(cp, pk)
		sorted[i] = cp
	}
	sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i], sorted[j]) < 0 })
	pos := make(map[string]int, len(sorted))
	for i, pk := range sorted {
		pos[hex.EncodeToString(pk)] = i
	}
	return pos
}

// hashToTxid reverses a 32-byte tx hash into the big-endian hex txid bitcoind
// displays; txidToHash is the inverse.
func hashToTxid(h [32]byte) string {
	var r [32]byte
	for i := 0; i < 32; i++ {
		r[i] = h[31-i]
	}
	return hex.EncodeToString(r[:])
}

func txidToHash(txid string) [32]byte {
	b, err := hex.DecodeString(txid)
	var out [32]byte
	if err != nil || len(b) != 32 {
		return out
	}
	for i := 0; i < 32; i++ {
		out[i] = b[31-i]
	}
	return out
}

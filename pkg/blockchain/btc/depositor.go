package btc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor funds a per-account deposit address from the depositor's own
// P2WPKH wallet (the key the supplied sign.Signer holds). It implements
// core.VaultDepositor. The deposit address is derived from the vault's pubkeys
// + threshold (the same address the withdrawal finalizer can later spend).
type Depositor struct {
	net          *chaincfg.Params
	rpc          RPC
	signer       sign.Signer
	signerPub    []byte
	depositAddr  btcutil.Address // depositor's own P2WPKH address (funding source)
	vaultPubkeys [][]byte
	threshold    int
	cfg          Config
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor builds the BTC depositor. signer is the depositor's secp256k1
// key; vaultPubkeys + threshold define the vault whose per-account deposit
// addresses funds are sent to.
func NewDepositor(net *chaincfg.Params, rpc RPC, signer sign.Signer, vaultPubkeys [][]byte, threshold int, cfg Config) (*Depositor, error) {
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: depositor signer must be secp256k1, got %s", signer.Algorithm())
	}
	pub := signer.PublicKey()
	addr, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pub), net)
	if err != nil {
		return nil, fmt.Errorf("btc: derive depositor address: %w", err)
	}
	return &Depositor{
		net:          net,
		rpc:          rpc,
		signer:       signer,
		signerPub:    pub,
		depositAddr:  addr,
		vaultPubkeys: vaultPubkeys,
		threshold:    threshold,
		cfg:          cfg,
	}, nil
}

// DepositorAddress returns the depositor's own P2WPKH funding address.
func (d *Depositor) DepositorAddress() string { return d.depositAddr.EncodeAddress() }

// SubmitDeposit sends `amount` satoshis from the depositor's wallet to the
// per-account deposit address for dest.Account. asset must be native BTC ("" or
// "BTC"). Builds, signs (P2WPKH), and broadcasts the funding tx. A non-zero
// dest.Ref is rejected: the account is encoded in the deposit address and a
// plain BTC send has no side-data channel for a sub-account (ADR-015 has no BTC
// reference).
func (d *Depositor) SubmitDeposit(ctx context.Context, asset string, amount decimal.Decimal, dest core.DepositDestination) (core.TxRef, error) {
	if a := strings.ToUpper(strings.TrimSpace(asset)); a != "" && a != "BTC" {
		return core.TxRef{}, fmt.Errorf("btc: only native BTC deposits supported, got asset %q", asset)
	}
	if dest.Ref != ([32]byte{}) {
		return core.TxRef{}, fmt.Errorf("btc: deposit reference not supported")
	}
	amt := amount.BigInt()
	if !amt.IsInt64() || amt.Int64() <= 0 {
		return core.TxRef{}, fmt.Errorf("btc: amount %s not a positive int64 satoshi value", amount.String())
	}
	sats := amt.Int64()

	depositAddr, _, err := DepositAddress(dest.Account, d.threshold, d.vaultPubkeys, d.net)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("btc: derive deposit address: %w", err)
	}

	myAddr := d.depositAddr.EncodeAddress()
	unspent, err := d.rpc.ListUnspent(ctx, int(d.cfg.ConfirmationDepth), []string{myAddr})
	if err != nil {
		return core.TxRef{}, fmt.Errorf("btc: list depositor utxos: %w", err)
	}
	utxos, scripts, err := depositorUTXOs(unspent, myAddr, d.net)
	if err != nil {
		return core.TxRef{}, err
	}
	feeRate, err := d.rpc.EstimateSmartFeeSatPerVByte(ctx, d.cfg.FeeConfTarget, d.cfg.FallbackFeeRate)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("btc: estimate fee: %w", err)
	}
	// numFixedOutputs = recipient (deposit address); change is sized in.
	selected, feeSats, err := SelectUTXOs(utxos, sats, feeRate, 1, 0)
	if err != nil {
		return core.TxRef{}, err
	}

	tx, err := buildDepositTx(selected, depositAddr, sats, d.depositAddr, feeSats)
	if err != nil {
		return core.TxRef{}, err
	}
	if err := d.signP2WPKH(ctx, tx, selected, scripts); err != nil {
		return core.TxRef{}, err
	}

	raw, err := serializeTx(tx)
	if err != nil {
		return core.TxRef{}, err
	}
	hash := [32]byte(tx.TxHash())
	txid := hashToTxid(hash)
	if _, err := d.rpc.SendRawTransaction(ctx, hex.EncodeToString(raw)); err != nil {
		if isAlreadyKnown(err) {
			return core.TxRef{Hash: hash, Raw: txid}, nil
		}
		return core.TxRef{}, fmt.Errorf("btc: sendrawtransaction: %w", err)
	}
	return core.TxRef{Hash: hash, Raw: txid}, nil
}

// VerifyDeposit reports the on-chain status of the deposit tx in ref (matched
// by txid, ref.Raw). Requires the node to resolve the tx (txindex=1, or the tx
// unspent / in the mempool). A tx the node has never seen — or one reorged out
// and dropped — reads as DepositAbsent; a mempool tx (0 confs) is DepositPending
// until it is mined with at least max(1, minConf) confirmations (a deposit is
// only Confirmed once on chain, consistent with the other chains).
func (d *Depositor) VerifyDeposit(ctx context.Context, ref core.TxRef, minConf uint64) (core.DepositStatus, error) {
	raw, err := d.rpc.GetRawTransaction(ctx, ref.Raw)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -5 { // RPC_INVALID_ADDRESS_OR_KEY: unknown tx
			return core.DepositAbsent, nil
		}
		return core.DepositAbsent, fmt.Errorf("btc: getrawtransaction: %w", err)
	}
	if raw == nil {
		return core.DepositAbsent, nil
	}
	if raw.Confirmations > 0 && raw.Confirmations >= int64(minConf) {
		return core.DepositConfirmed, nil
	}
	return core.DepositPending, nil
}

// depositorUTXOs filters unspent outputs to the depositor's own address and
// returns the UTXO set plus a per-outpoint amount/script index for signing.
func depositorUTXOs(unspent []Unspent, myAddr string, net *chaincfg.Params) ([]UTXO, map[wire.OutPoint]int64, error) {
	addr, err := btcutil.DecodeAddress(myAddr, net)
	if err != nil {
		return nil, nil, fmt.Errorf("btc: decode depositor addr: %w", err)
	}
	myScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, nil, fmt.Errorf("btc: depositor pkScript: %w", err)
	}
	myScriptHex := strings.ToLower(hex.EncodeToString(myScript))

	utxos := make([]UTXO, 0, len(unspent))
	amounts := make(map[wire.OutPoint]int64)
	for _, u := range unspent {
		if strings.ToLower(u.ScriptPubKey) != myScriptHex {
			continue
		}
		h, err := chainhash.NewHashFromStr(u.TxID)
		if err != nil {
			return nil, nil, fmt.Errorf("btc: bad txid %q: %w", u.TxID, err)
		}
		utxos = append(utxos, UTXO{TxID: *h, Vout: u.Vout, Amount: u.AmountSats})
		amounts[wire.OutPoint{Hash: *h, Index: u.Vout}] = u.AmountSats
	}
	return utxos, amounts, nil
}

// buildDepositTx builds the unsigned funding tx: output 0 pays the deposit
// address `sats`, with change back to the depositor above dust.
func buildDepositTx(utxos []UTXO, depositAddr btcutil.Address, sats int64, change btcutil.Address, feeSats int64) (*wire.MsgTx, error) {
	if len(utxos) == 0 {
		return nil, fmt.Errorf("btc: no depositor UTXOs selected")
	}
	ordered := make([]UTXO, len(utxos))
	copy(ordered, utxos)
	sort.Slice(ordered, func(i, j int) bool {
		if c := compareHash(ordered[i].TxID[:], ordered[j].TxID[:]); c != 0 {
			return c < 0
		}
		return ordered[i].Vout < ordered[j].Vout
	})

	var inTotal int64
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, u := range ordered {
		op := wire.NewOutPoint(&u.TxID, u.Vout)
		tx.AddTxIn(wire.NewTxIn(op, nil, nil))
		inTotal += u.Amount
	}
	depScript, err := txscript.PayToAddrScript(depositAddr)
	if err != nil {
		return nil, fmt.Errorf("btc: deposit script: %w", err)
	}
	tx.AddTxOut(wire.NewTxOut(sats, depScript))

	rem := inTotal - sats - feeSats
	if rem < 0 {
		return nil, fmt.Errorf("btc: depositor inputs %d below amount %d + fee %d", inTotal, sats, feeSats)
	}
	if rem >= dustThresholdSats {
		changeScript, err := txscript.PayToAddrScript(change)
		if err != nil {
			return nil, fmt.Errorf("btc: change script: %w", err)
		}
		tx.AddTxOut(wire.NewTxOut(rem, changeScript))
	}
	return tx, nil
}

// signP2WPKH signs every input as a P2WPKH spend with the depositor key and
// installs the [sig, pubkey] witness.
func (d *Depositor) signP2WPKH(ctx context.Context, tx *wire.MsgTx, utxos []UTXO, amounts map[wire.OutPoint]int64) error {
	pkh := btcutil.Hash160(d.signerPub)
	// BIP-143 scriptCode for P2WPKH is the corresponding P2PKH script.
	scriptCode, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).AddOp(txscript.OP_HASH160).
		AddData(pkh).
		AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).Script()
	if err != nil {
		return fmt.Errorf("btc: build scriptCode: %w", err)
	}
	myScript, err := txscript.PayToAddrScript(d.depositAddr)
	if err != nil {
		return fmt.Errorf("btc: depositor pkScript: %w", err)
	}
	fetcher := txscript.NewMultiPrevOutFetcher(nil)
	for op, amt := range amounts {
		fetcher.AddPrevOut(op, wire.NewTxOut(amt, myScript))
	}
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	for idx, in := range tx.TxIn {
		amt := amounts[in.PreviousOutPoint]
		sighash, err := txscript.CalcWitnessSigHash(scriptCode, sigHashes, txscript.SigHashAll, tx, idx, amt)
		if err != nil {
			return fmt.Errorf("btc: sighash input %d: %w", idx, err)
		}
		der, err := d.signer.Sign(ctx, sighash)
		if err != nil {
			return fmt.Errorf("btc: sign input %d: %w", idx, err)
		}
		tx.TxIn[idx].Witness = wire.TxWitness{append(der, byte(txscript.SigHashAll)), d.signerPub}
	}
	return nil
}

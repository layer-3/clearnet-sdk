package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// defaultConsolidationBatchMax bounds how many of the vault's smallest UTXOs a
// single consolidation fold spends when Config.ConsolidationBatchMax is unset.
// Kept well under the ~800-input standard-tx ceiling so every fold is relayable.
const defaultConsolidationBatchMax = 200

// ConsolidationFinalizer folds a bounded batch of the vault's smallest UTXOs
// back into a single base-vault output, shrinking the UTXO count so withdrawals
// keep fitting a standard-size tx (the counterpart to SelectUTXOs' maxInputs
// bound / ErrTooFragmented).
//
// It is mechanically a withdrawal-to-self: same vault, current keys, no pivot.
// Unlike the rotation sweep it is partial by design (a bounded batch, not the
// full set), so it carries no completeness rule. Pack/Validate are the
// fold-specific mechanics; Sign/Submit reuse the current-vault withdrawal
// machinery, exactly as RotationFinalizer does.
type ConsolidationFinalizer struct {
	net      *chaincfg.Params
	rpc      RPC
	signer   sign.Signer
	store    VaultStore
	cfg      Config
	assets   blockchain.AssetResolver
	accounts []string // per-account deposit URIs whose UTXOs are also foldable
}

// NewConsolidationFinalizer builds the BTC consolidation finalizer. signer is
// this node's vault key; store supplies the current vault (consolidation never
// pivots, so Pivot is unused). accountURIs are the per-account deposit accounts
// whose tagged-address UTXOs are eligible to fold alongside the base vault.
func NewConsolidationFinalizer(net *chaincfg.Params, rpc RPC, signer sign.Signer, store VaultStore, cfg Config, assets blockchain.AssetResolver, accountURIs ...string) (*ConsolidationFinalizer, error) {
	if assets == nil {
		return nil, fmt.Errorf("btc: asset resolver is required")
	}
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: consolidation signer must be secp256k1, got %s", signer.Algorithm())
	}
	return &ConsolidationFinalizer{
		net:      net,
		rpc:      rpc,
		signer:   signer,
		store:    store,
		cfg:      cfg,
		assets:   assets,
		accounts: accountURIs,
	}, nil
}

// currentVault builds a withdrawal finalizer over the current vault, registering
// the deposit accounts so the spend-script set covers the base vault plus every
// tagged deposit address whose UTXOs may be folded. It provides the shared
// UTXO/sign/merge machinery.
func (f *ConsolidationFinalizer) currentVault(ctx context.Context) (*WithdrawalFinalizer, error) {
	pubkeys, threshold, err := f.store.Current(ctx)
	if err != nil {
		return nil, fmt.Errorf("btc: read current vault: %w", err)
	}
	cur, err := NewWithdrawalFinalizer(f.net, f.rpc, f.signer, pubkeys, threshold, f.cfg, f.assets)
	if err != nil {
		return nil, fmt.Errorf("btc: build current vault: %w", err)
	}
	if err := cur.RegisterDepositAccounts(f.accounts...); err != nil {
		return nil, err
	}
	return cur, nil
}

func (f *ConsolidationFinalizer) batchMax() int {
	if f.cfg.ConsolidationBatchMax > 0 {
		return f.cfg.ConsolidationBatchMax
	}
	return defaultConsolidationBatchMax
}

// listOwned returns every currently-spendable owned UTXO (base vault + each
// registered deposit address) at the configured confirmation depth.
func (f *ConsolidationFinalizer) listOwned(ctx context.Context, cur *WithdrawalFinalizer) ([]UTXO, error) {
	unspent, err := f.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), cur.watchAddresses())
	if err != nil {
		return nil, fmt.Errorf("btc consolidate: list vault utxos: %w", err)
	}
	return cur.toUTXOs(unspent)
}

// OwnedUTXOCount reports the number of currently-spendable owned UTXOs. The
// consolidation trigger uses it to decide whether a fold is warranted.
func (f *ConsolidationFinalizer) OwnedUTXOCount(ctx context.Context) (int, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return 0, err
	}
	owned, err := f.listOwned(ctx, cur)
	if err != nil {
		return 0, err
	}
	return len(owned), nil
}

// Due reports whether a periodic fold should run now: the owned UTXO count
// exceeds target AND the current fee rate is at or below feeCeilingSatVb (a
// low-fee window). feeCeilingSatVb <= 0 disables the fee gate.
func (f *ConsolidationFinalizer) Due(ctx context.Context, target int, feeCeilingSatVb int64) (bool, error) {
	count, err := f.OwnedUTXOCount(ctx)
	if err != nil {
		return false, err
	}
	if count <= target {
		return false, nil
	}
	feeRate, err := f.rpc.EstimateSmartFeeSatPerVByte(ctx, f.cfg.FeeConfTarget, f.cfg.FallbackFeeRate)
	if err != nil {
		return false, fmt.Errorf("btc consolidate: estimate fee: %w", err)
	}
	if feeCeilingSatVb > 0 && feeRate > feeCeilingSatVb {
		return false, nil
	}
	return true, nil
}

// Pack selects the vault's smallest spendable UTXOs (up to the batch max) and
// folds them into a single base-vault output minus fee, with consolidationID in
// an OP_RETURN. Smallest-first keeps the large coins intact for largest-first
// withdrawal selection and shrinks the count fastest. The change computes to
// zero (recipient == vault), so the result is the two-output form
// [baseVault(total-fee), OP_RETURN(consolidationID)].
func (f *ConsolidationFinalizer) Pack(ctx context.Context, consolidationID [32]byte) ([]byte, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}
	utxos, err := f.listOwned(ctx, cur)
	if err != nil {
		return nil, err
	}
	if len(utxos) < 2 {
		return nil, errors.New("btc consolidate: fewer than 2 spendable utxos; nothing to fold")
	}
	// Smallest-amount-first, BIP-69 tiebreak for a deterministic batch.
	sort.Slice(utxos, func(i, j int) bool {
		if utxos[i].Amount != utxos[j].Amount {
			return utxos[i].Amount < utxos[j].Amount
		}
		if c := bytes.Compare(utxos[i].TxID[:], utxos[j].TxID[:]); c != 0 {
			return c < 0
		}
		return utxos[i].Vout < utxos[j].Vout
	})
	if max := f.batchMax(); len(utxos) > max {
		utxos = utxos[:max]
	}

	feeRate, err := f.rpc.EstimateSmartFeeSatPerVByte(ctx, f.cfg.FeeConfTarget, f.cfg.FallbackFeeRate)
	if err != nil {
		return nil, fmt.Errorf("btc consolidate: estimate fee: %w", err)
	}
	var total int64
	for _, u := range utxos {
		total += u.Amount
	}
	// Two outputs: the base vault + the OP_RETURN marker. No change.
	fee := EstimateFeeSats(len(utxos), 2, feeRate)
	amount := total - fee
	if amount < dustThresholdSats {
		return nil, fmt.Errorf("btc consolidate: post-fee amount %d below dust (total %d, fee %d); batch not worth folding", amount, total, fee)
	}
	tx, err := BuildUnsignedTx(utxos, cur.vaultAddr, amount, cur.vaultAddr, consolidationID, fee)
	if err != nil {
		return nil, err
	}
	return serializeTx(tx)
}

// Validate is the follower-side trust boundary for a fold. A vault→vault fold
// cannot move funds out of custody, so validation is deliberately lenient (no
// completeness rule, no byte-identical build): exactly two outputs, output 0
// paying the base vault, output 1 an OP_RETURN(consolidationID); every input a
// confirmed owned UTXO (nothing foreign dragged in); the batch within the size
// bound; and the implied fee within the griefing ceiling.
func (f *ConsolidationFinalizer) Validate(ctx context.Context, consolidationID [32]byte, packed []byte) error {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return err
	}
	tx, err := deserializeTx(packed)
	if err != nil {
		return fmt.Errorf("btc consolidate validate: %w", err)
	}
	if err := validateFixedTxFields(tx); err != nil {
		return fmt.Errorf("btc consolidate validate: %w", err)
	}
	if n := len(tx.TxOut); n != 2 {
		return fmt.Errorf("btc consolidate validate: expected 2 outputs, got %d", n)
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, cur.vaultScript) {
		return errors.New("btc consolidate validate: output 0 is not the base vault")
	}
	wantOpReturn, err := txscript.NullDataScript(consolidationID[:])
	if err != nil {
		return fmt.Errorf("btc consolidate validate: opreturn script: %w", err)
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, wantOpReturn) {
		return errors.New("btc consolidate validate: output 1 is not OP_RETURN <consolidationID>")
	}
	if len(tx.TxIn) < 2 {
		return fmt.Errorf("btc consolidate validate: %d inputs; a fold spends at least 2", len(tx.TxIn))
	}
	if max := f.batchMax(); len(tx.TxIn) > max {
		return fmt.Errorf("btc consolidate validate: %d inputs exceed batch max %d", len(tx.TxIn), max)
	}

	// Every input must be a confirmed, owned UTXO. Re-list the owned set and
	// require the inputs to be a subset of it (lenient: no full-set match).
	owned, err := f.listOwned(ctx, cur)
	if err != nil {
		return err
	}
	byOutpoint := make(map[string]int64, len(owned))
	for _, u := range owned {
		byOutpoint[fmt.Sprintf("%s:%d", u.TxID.String(), u.Vout)] = u.Amount
	}
	var totalIn int64
	for _, in := range tx.TxIn {
		key := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
		amt, ok := byOutpoint[key]
		if !ok {
			return fmt.Errorf("btc consolidate validate: input %s is not a confirmed owned utxo", key)
		}
		totalIn += amt
	}

	fee := totalIn - tx.TxOut[0].Value // output 1 is the zero-value OP_RETURN
	if fee < 0 {
		return fmt.Errorf("btc consolidate validate: outputs exceed inputs (fee %d)", fee)
	}
	if cap := EstimateFeeSats(len(tx.TxIn), 2, f.cfg.FeeCapSatPerVByte); f.cfg.FeeCapSatPerVByte > 0 && fee > cap {
		return fmt.Errorf("btc consolidate validate: fee %d exceeds ceiling %d", fee, cap)
	}
	return nil
}

// Sign produces this node's per-input signatures over the fold, delegating to
// the current-vault signing machinery (the fold spends base-vault and deposit
// inputs under the current redeem scripts, exactly as a withdrawal).
func (f *ConsolidationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}
	return cur.Sign(ctx, packed)
}

// Submit assembles the witnesses from the collected shares and broadcasts the
// fold, returning its hash. Idempotent on an already-known/spent reply (the
// UTXO-model analogue of a re-submit guard).
func (f *ConsolidationFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (core.TxRef, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return core.TxRef{}, err
	}
	merged, err := cur.merge(ctx, packed, shares)
	if err != nil {
		return core.TxRef{}, err
	}
	tx, err := deserializeTx(merged)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("btc consolidate submit: %w", err)
	}
	hash := [32]byte(tx.TxHash())
	txid := hashToTxid(hash)
	if _, err := f.rpc.SendRawTransaction(ctx, hex.EncodeToString(merged)); err != nil {
		if isAlreadyKnown(err) {
			return core.TxRef{Hash: hash, Raw: txid}, nil
		}
		return core.TxRef{}, fmt.Errorf("btc consolidate submit: sendrawtransaction: %w", err)
	}
	return core.TxRef{Hash: hash, Raw: txid}, nil
}

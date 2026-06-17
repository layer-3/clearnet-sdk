package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// VaultStore is the seam that lets BTC rotation fit the in-place
// SignerRotationFinalizer interface. A P2WSH vault's address is a function of
// its signer set, so rotation is a sweep into a newly-derived vault, after which
// the daemon must pivot to that vault. Current supplies the active vault (the
// source of UTXOs to sweep and the set that authorizes the sweep); Pivot adopts
// the new vault once the sweep confirms.
//
// Pivot must be idempotent: it is called by every node from VerifyRotation when
// the sweep is observed, and re-adopting the already-current vault is a no-op.
type VaultStore interface {
	Current(ctx context.Context) (pubkeys [][]byte, threshold int, err error)
	Pivot(ctx context.Context, pubkeys [][]byte, threshold int) error
}

// RotationFinalizer rotates a BTC P2WSH vault by sweeping every old-vault UTXO
// into the vault derived from the new signer set. It implements
// core.SignerRotationFinalizer. The signing/merge/UTXO machinery is the
// withdrawal path (it is mechanically a withdrawal whose inputs are all
// old-vault UTXOs and whose single output is the new vault); the extra pieces
// are the sweep build and the post-confirmation pivot via the VaultStore.
type RotationFinalizer struct {
	net      *chaincfg.Params
	rpc      RPC
	signer   sign.Signer
	store    VaultStore
	cfg      Config
	accounts []string // per-account deposit URIs whose UTXOs must also be swept
}

var _ core.SignerRotationFinalizer = (*RotationFinalizer)(nil)

// NewRotationFinalizer builds the BTC rotation finalizer. signer is this node's
// vault key; store supplies the current vault and receives the pivot. accountURIs
// are the per-account deposit accounts whose tagged-address UTXOs must be
// included in the sweep (the base vault is always swept) — undeclared accounts'
// UTXOs would be stranded under the old vault.
func NewRotationFinalizer(net *chaincfg.Params, rpc RPC, signer sign.Signer, store VaultStore, cfg Config, accountURIs ...string) (*RotationFinalizer, error) {
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: rotation signer must be secp256k1, got %s", signer.Algorithm())
	}
	return &RotationFinalizer{
		net:      net,
		rpc:      rpc,
		signer:   signer,
		store:    store,
		cfg:      cfg,
		accounts: accountURIs,
	}, nil
}

// currentVault builds a withdrawal finalizer over the current vault (from the
// store), registering the deposit accounts so its spend-script set covers the
// base vault plus every tagged deposit address to sweep. It provides the shared
// UTXO/sign/merge machinery.
func (f *RotationFinalizer) currentVault(ctx context.Context) (*WithdrawalFinalizer, error) {
	pubkeys, threshold, err := f.store.Current(ctx)
	if err != nil {
		return nil, fmt.Errorf("btc: read current vault: %w", err)
	}
	cur, err := NewWithdrawalFinalizer(f.net, f.rpc, f.signer, pubkeys, threshold, f.cfg)
	if err != nil {
		return nil, fmt.Errorf("btc: build current vault: %w", err)
	}
	if err := cur.RegisterDepositAccounts(f.accounts...); err != nil {
		return nil, err
	}
	return cur, nil
}

// newVaultAddress derives the destination vault address + pkScript from the
// incoming signer set.
func (f *RotationFinalizer) newVaultAddress(newSigners []string, newThreshold int) (btcutil.Address, []byte, [][]byte, error) {
	pubkeys, err := parseVaultPubkeys(newSigners)
	if err != nil {
		return nil, nil, nil, err
	}
	redeem, err := RedeemScript(newThreshold, pubkeys)
	if err != nil {
		return nil, nil, nil, err
	}
	addr, err := VaultAddress(redeem, f.net)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("btc: derive new vault address: %w", err)
	}
	pk, err := PkScript(addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("btc: new vault pkScript: %w", err)
	}
	return addr, pk, pubkeys, nil
}

// Pack lists every current-vault UTXO and builds the unsigned sweep: all of them
// as inputs, output 0 paying the new vault the total minus fee, and a final
// zero-value OP_RETURN carrying opID so an external watcher can attribute the
// landed sweep to this rotation (and pivot the vault).
func (f *RotationFinalizer) Pack(ctx context.Context, opID [32]byte, newSigners []string, newThreshold int) ([]byte, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}
	newVault, _, _, err := f.newVaultAddress(newSigners, newThreshold)
	if err != nil {
		return nil, err
	}
	unspent, err := f.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), cur.watchAddresses())
	if err != nil {
		return nil, fmt.Errorf("btc: list vault utxos: %w", err)
	}
	utxos, err := cur.toUTXOs(unspent)
	if err != nil {
		return nil, err
	}
	if len(utxos) == 0 {
		return nil, fmt.Errorf("btc: vault has no UTXOs to sweep")
	}
	feeRate, err := f.rpc.EstimateSmartFeeSatPerVByte(ctx, f.cfg.FeeConfTarget, f.cfg.FallbackFeeRate)
	if err != nil {
		return nil, fmt.Errorf("btc: estimate fee: %w", err)
	}
	tx, err := buildSweepTx(utxos, newVault, opID, feeRate)
	if err != nil {
		return nil, err
	}
	return serializeTx(tx)
}

// Validate re-derives the new vault and asserts the packed sweep pays exactly it
// from output 0, carries the OP_RETURN(opID) marker as its final output,
// consumes only current-vault UTXOs, and keeps the implied fee within the
// ceiling.
func (f *RotationFinalizer) Validate(ctx context.Context, opID [32]byte, packed []byte, newSigners []string, newThreshold int) error {
	_, newVaultScript, _, err := f.newVaultAddress(newSigners, newThreshold)
	if err != nil {
		return err
	}
	tx, err := deserializeTx(packed)
	if err != nil {
		return fmt.Errorf("btc rotation validate: %w", err)
	}
	if len(tx.TxOut) != 2 {
		return fmt.Errorf("btc rotation validate: expected 2 outputs (new vault + OP_RETURN), got %d", len(tx.TxOut))
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, newVaultScript) {
		return fmt.Errorf("btc rotation validate: output 0 not paid to the new vault")
	}
	wantMarker, err := txscript.NullDataScript(opID[:])
	if err != nil {
		return fmt.Errorf("btc rotation validate: opID marker script: %w", err)
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, wantMarker) {
		return fmt.Errorf("btc rotation validate: final output is not OP_RETURN(opID)")
	}
	cur, err := f.currentVault(ctx)
	if err != nil {
		return err
	}
	totalIn, err := cur.sumValidatedInputs(ctx, tx)
	if err != nil {
		return err
	}
	fee := totalIn - tx.TxOut[0].Value
	if fee < 0 {
		return fmt.Errorf("btc rotation validate: output exceeds inputs (fee %d)", fee)
	}
	if cap := EstimateFeeSats(len(tx.TxIn), 2, f.cfg.FeeCapSatPerVByte); f.cfg.FeeCapSatPerVByte > 0 && fee > cap {
		return fmt.Errorf("btc rotation validate: fee %d exceeds ceiling %d", fee, cap)
	}
	return nil
}

// Sign produces this node's per-input signatures over the sweep, delegating to
// the current-vault signing machinery.
func (f *RotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}
	return cur.Sign(ctx, packed)
}

// Submit assembles the witnesses from the collected shares and broadcasts the
// sweep. Idempotent by the UTXO model: if the sweep already landed, its inputs
// are spent and the rebroadcast is rejected as already-known/missing-inputs, in
// which case the original tx hash is returned.
func (f *RotationFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (core.TxRef, error) {
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
		return core.TxRef{}, fmt.Errorf("btc rotation submit: %w", err)
	}
	hash := [32]byte(tx.TxHash())
	txid := hashToTxid(hash)
	if _, err := f.rpc.SendRawTransaction(ctx, hex.EncodeToString(merged)); err != nil {
		if isAlreadyKnown(err) {
			return core.TxRef{Hash: hash, Raw: txid}, nil
		}
		return core.TxRef{}, fmt.Errorf("btc rotation submit: sendrawtransaction: %w", err)
	}
	return core.TxRef{Hash: hash, Raw: txid}, nil
}

// VerifyRotation reports whether the sweep landed — the new vault holds at least
// one confirmed UTXO — and, when so, pivots the store to the new vault. Binary;
// the sweep tx hash is not recovered here, so a zero hash is returned with
// done=true. Note: a vault with nothing to sweep cannot be observed as rotated.
func (f *RotationFinalizer) VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) ([32]byte, bool, error) {
	newVault, _, newPubkeys, err := f.newVaultAddress(newSigners, newThreshold)
	if err != nil {
		return [32]byte{}, false, err
	}
	unspent, err := f.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), []string{newVault.EncodeAddress()})
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("btc rotation verify: list new vault utxos: %w", err)
	}
	if len(unspent) == 0 {
		return [32]byte{}, false, nil
	}
	if err := f.store.Pivot(ctx, newPubkeys, newThreshold); err != nil {
		return [32]byte{}, false, fmt.Errorf("btc rotation verify: pivot: %w", err)
	}
	return [32]byte{}, true, nil
}

// buildSweepTx builds the unsigned sweep: every UTXO as an input, output 0
// paying newVault the total minus the estimated fee, and a final zero-value
// OP_RETURN carrying opID (the rotation marker). The two-output shape — vault as
// output 0 plus the OP_RETURN — is what a watcher matches to attribute the
// landed sweep to this rotation.
func buildSweepTx(utxos []UTXO, newVault btcutil.Address, opID [32]byte, feeRate int64) (*wire.MsgTx, error) {
	ordered := make([]UTXO, len(utxos))
	copy(ordered, utxos)
	sort.Slice(ordered, func(i, j int) bool {
		if c := compareHash(ordered[i].TxID[:], ordered[j].TxID[:]); c != 0 {
			return c < 0
		}
		return ordered[i].Vout < ordered[j].Vout
	})

	var total int64
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, u := range ordered {
		op := wire.NewOutPoint(&u.TxID, u.Vout)
		tx.AddTxIn(wire.NewTxIn(op, nil, nil))
		total += u.Amount
	}
	fee := EstimateFeeSats(len(ordered), 2, feeRate)
	out := total - fee
	if out < dustThresholdSats {
		return nil, fmt.Errorf("btc: sweep output %d below dust after fee %d (total %d)", out, fee, total)
	}
	script, err := txscript.PayToAddrScript(newVault)
	if err != nil {
		return nil, fmt.Errorf("btc: new vault script: %w", err)
	}
	tx.AddTxOut(wire.NewTxOut(out, script))

	marker, err := txscript.NullDataScript(opID[:])
	if err != nil {
		return nil, fmt.Errorf("btc: opID OP_RETURN script: %w", err)
	}
	tx.AddTxOut(wire.NewTxOut(0, marker))
	return tx, nil
}

// parseVaultPubkeys decodes the incoming signer set (33-byte compressed pubkey
// hex) for vault derivation. Ordering is handled by RedeemScript (BIP-67).
func parseVaultPubkeys(newSigners []string) ([][]byte, error) {
	if len(newSigners) == 0 {
		return nil, fmt.Errorf("btc: empty new signer set")
	}
	out := make([][]byte, 0, len(newSigners))
	for _, s := range newSigners {
		b, err := hex.DecodeString(s)
		if err != nil || len(b) != 33 {
			return nil, fmt.Errorf("btc: signer %q must be a 33-byte compressed pubkey hex", s)
		}
		out = append(out, b)
	}
	return out, nil
}

package btc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
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
	assets   blockchain.AssetResolver
	accounts []string // per-account deposit URIs whose UTXOs must also be swept
}

var _ core.SignerRotationFinalizer = (*RotationFinalizer)(nil)

// NewRotationFinalizer builds the BTC rotation finalizer. signer is this node's
// vault key; store supplies the current vault and receives the pivot. accountURIs
// are the per-account deposit accounts whose tagged-address UTXOs must be
// included in the sweep (the base vault is always swept) — undeclared accounts'
// UTXOs would be stranded under the old vault.
func NewRotationFinalizer(net *chaincfg.Params, rpc RPC, signer sign.Signer, store VaultStore, cfg Config, assets blockchain.AssetResolver, accountURIs ...string) (*RotationFinalizer, error) {
	if assets == nil {
		return nil, fmt.Errorf("btc: asset resolver is required")
	}
	if signer.Algorithm() != sign.AlgSecp256k1 {
		return nil, fmt.Errorf("btc: rotation signer must be secp256k1, got %s", signer.Algorithm())
	}
	return &RotationFinalizer{
		net:      net,
		rpc:      rpc,
		signer:   signer,
		store:    store,
		cfg:      cfg,
		assets:   assets,
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
	cur, err := NewWithdrawalFinalizer(f.net, f.rpc, f.signer, pubkeys, threshold, f.cfg, f.assets)
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
	_, err := f.Prepare(ctx, packed, RotationAuthorization{OperationID: opID, Signers: newSigners, Threshold: newThreshold})
	return err
}

// Prepare performs the full rotation validation, including sweep completeness,
// then captures the exact previous outputs needed for offline share validation,
// signing, and finalization.
func (f *RotationFinalizer) Prepare(ctx context.Context, packed []byte, auth RotationAuthorization) (*PreparedRotation, error) {
	return f.prepare(ctx, packed, auth)
}

func (f *RotationFinalizer) prepare(ctx context.Context, packed []byte, auth RotationAuthorization) (*PreparedRotation, error) {
	_, newVaultScript, _, err := f.newVaultAddress(auth.Signers, auth.Threshold)
	if err != nil {
		return nil, candidateError("btc rotation prepare", err)
	}
	tx, err := deserializeUnsignedCanonical(packed)
	if err != nil {
		return nil, candidateError("btc rotation prepare", err)
	}
	if err := validateFixedTxFields(tx); err != nil {
		return nil, candidateError("btc rotation prepare", err)
	}
	if len(tx.TxOut) != 2 {
		return nil, candidateError("btc rotation prepare", fmt.Errorf("expected 2 outputs (new vault + OP_RETURN), got %d", len(tx.TxOut)))
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, newVaultScript) {
		return nil, candidateError("btc rotation prepare", fmt.Errorf("output 0 not paid to the new vault"))
	}
	wantMarker, err := txscript.NullDataScript(auth.OperationID[:])
	if err != nil {
		return nil, fmt.Errorf("btc rotation validate: opID marker script: %w", err)
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, wantMarker) {
		return nil, candidateError("btc rotation prepare", fmt.Errorf("final output is not OP_RETURN(opID)"))
	}
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}

	// Completeness: a rotation sweep must spend every currently-owned UTXO. The
	// prepared snapshot below proves every present input is a confirmed vault
	// UTXO; this set comparison catches a sweep that silently omits one.
	unspent, err := cur.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), cur.watchAddresses())
	if err != nil {
		return nil, fmt.Errorf("btc rotation validate: list vault utxos: %w", err)
	}
	owned, err := cur.toUTXOs(unspent)
	if err != nil {
		return nil, err
	}
	want := make(map[string]struct{}, len(owned))
	for _, u := range owned {
		want[fmt.Sprintf("%s:%d", u.TxID.String(), u.Vout)] = struct{}{}
	}
	got := make(map[string]struct{}, len(tx.TxIn))
	for _, in := range tx.TxIn {
		got[fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)] = struct{}{}
	}
	if len(want) != len(got) {
		return nil, candidateError("btc rotation prepare", fmt.Errorf("spends %d inputs, expected all %d owned utxos", len(got), len(want)))
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			return nil, candidateError("btc rotation prepare", fmt.Errorf("omits owned utxo %s", k))
		}
	}
	authHash, err := hashRotationAuthorization(auth)
	if err != nil {
		return nil, candidateError("btc rotation prepare", err)
	}
	prepared, err := cur.prepareTransactionFromCanonical(ctx, packed, preparedOperationRotation, authHash)
	if err != nil {
		return nil, fmt.Errorf("btc rotation validate: %w", err)
	}
	return &PreparedRotation{preparedTransaction: *prepared}, nil
}

// ValidatePrepared revalidates the exact rotation authorization and old-vault
// snapshot without reading VaultStore or the UTXO set.
func (f *RotationFinalizer) ValidatePrepared(prepared *PreparedRotation, auth RotationAuthorization) error {
	txState := rotationTransaction(prepared)
	hash, err := hashRotationAuthorization(auth)
	if err != nil {
		return err
	}
	if err := validateAuthorizationHash(txState, preparedOperationRotation, hash); err != nil {
		return err
	}
	verifier, err := f.preparedVerifier(txState)
	if err != nil {
		return err
	}
	tx, _, err := validatePreparedLocal(txState)
	if err != nil {
		return err
	}
	_, target, _, err := f.newVaultAddress(auth.Signers, auth.Threshold)
	if err != nil {
		return err
	}
	if len(tx.TxOut) != 2 || !bytes.Equal(tx.TxOut[0].PkScript, target) {
		return fmt.Errorf("btc rotation prepared: wrong target vault")
	}
	marker, err := txscript.NullDataScript(auth.OperationID[:])
	if err != nil {
		return err
	}
	if tx.TxOut[1].Value != 0 || !bytes.Equal(tx.TxOut[1].PkScript, marker) {
		return fmt.Errorf("btc rotation prepared: wrong operation marker")
	}
	_ = verifier
	return nil
}

func (f *RotationFinalizer) preparedVerifier(prepared *preparedTransaction) (*WithdrawalFinalizer, error) {
	base := &WithdrawalFinalizer{net: f.net, rpc: f.rpc, signer: f.signer, assets: f.assets}
	return base.offlineVerifier(prepared)
}

// SignPrepared signs the prepared sweep without reading its inputs from RPC.
func (f *RotationFinalizer) SignPrepared(ctx context.Context, prepared *PreparedRotation, auth RotationAuthorization) ([]byte, error) {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return nil, err
	}
	cur, err := f.preparedVerifier(rotationTransaction(prepared))
	if err != nil {
		return nil, err
	}
	return cur.signPrepared(ctx, rotationTransaction(prepared))
}

// VerifySharePrepared cryptographically validates one prepared sweep share.
func (f *RotationFinalizer) VerifySharePrepared(prepared *PreparedRotation, auth RotationAuthorization, share []byte) error {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return err
	}
	cur, err := f.preparedVerifier(rotationTransaction(prepared))
	if err != nil {
		return err
	}
	return cur.verifySharePrepared(rotationTransaction(prepared), share)
}

// FinalizePrepared filters invalid and duplicate shares and deterministically
// assembles the prepared sweep without reading its inputs from RPC.
func (f *RotationFinalizer) FinalizePrepared(prepared *PreparedRotation, auth RotationAuthorization, shares [][]byte) ([]byte, string, error) {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return nil, "", err
	}
	cur, err := f.preparedVerifier(rotationTransaction(prepared))
	if err != nil {
		return nil, "", err
	}
	return cur.finalizePrepared(rotationTransaction(prepared), shares)
}

// VerifyFinalizedPrepared verifies the finalized sweep and every input witness.
func (f *RotationFinalizer) VerifyFinalizedPrepared(prepared *PreparedRotation, auth RotationAuthorization, raw []byte) error {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return err
	}
	cur, err := f.preparedVerifier(rotationTransaction(prepared))
	if err != nil {
		return err
	}
	return cur.verifyFinalizedPrepared(rotationTransaction(prepared), raw)
}

// BroadcastPrepared cryptographically verifies the finalized sweep immediately
// before broadcasting it.
func (f *RotationFinalizer) BroadcastPrepared(ctx context.Context, prepared *PreparedRotation, auth RotationAuthorization, raw []byte) (string, error) {
	if err := f.VerifyFinalizedPrepared(prepared, auth, raw); err != nil {
		return "", err
	}
	base := &WithdrawalFinalizer{rpc: f.rpc}
	return base.sendVerifiedPrepared(ctx, raw, prepared.expectedTxID)
}

// Sign retains the core interface method. It captures and fully validates the
// live previous outputs before signing through the prepared implementation.
func (f *RotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return nil, err
	}
	legacyHash := sha256.Sum256(append([]byte("clearnet-sdk/btc/legacy-rotation/v1"), packed...))
	prepared, err := cur.prepareTransactionFromCanonical(ctx, packed, preparedOperationRotation, legacyHash)
	if err != nil {
		return nil, fmt.Errorf("btc rotation sign: %w", err)
	}
	return cur.signPrepared(ctx, prepared)
}

// Submit retains the core interface method. It recaptures and validates every
// live input, cryptographically verifies shares and finalized witnesses, then
// broadcasts through the prepared implementation.
func (f *RotationFinalizer) Submit(ctx context.Context, packed []byte, shares [][]byte) (string, error) {
	cur, err := f.currentVault(ctx)
	if err != nil {
		return "", err
	}
	legacyHash := sha256.Sum256(append([]byte("clearnet-sdk/btc/legacy-rotation/v1"), packed...))
	prepared, err := cur.prepareTransactionFromCanonical(ctx, packed, preparedOperationRotation, legacyHash)
	if err != nil {
		return "", fmt.Errorf("btc rotation submit: %w", err)
	}
	raw, _, err := cur.finalizePrepared(prepared, shares)
	if err != nil {
		return "", err
	}
	return cur.broadcastPrepared(ctx, prepared, raw)
}

// VerifyRotation reports whether the sweep landed — the new vault holds at least
// one confirmed UTXO — and, when so, pivots the store to the new vault. Binary;
// the sweep txID is not recovered here, so an empty txID is returned with
// done=true. Note: a vault with nothing to sweep cannot be observed as rotated.
func (f *RotationFinalizer) VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) (string, bool, error) {
	newVault, _, newPubkeys, err := f.newVaultAddress(newSigners, newThreshold)
	if err != nil {
		return "", false, err
	}
	unspent, err := f.rpc.ListUnspent(ctx, int(f.cfg.ConfirmationDepth), []string{newVault.EncodeAddress()})
	if err != nil {
		return "", false, fmt.Errorf("btc rotation verify: list new vault utxos: %w", err)
	}
	if len(unspent) == 0 {
		return "", false, nil
	}
	if err := f.store.Pivot(ctx, newPubkeys, newThreshold); err != nil {
		return "", false, fmt.Errorf("btc rotation verify: pivot: %w", err)
	}
	return "", true, nil
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

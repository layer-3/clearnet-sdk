// Package btc implements the Bitcoin custody vault adapter. The vault is a
// native SegWit P2WSH (BIP 141) address whose redeem script is an m-of-n
// OP_CHECKMULTISIG over the providers' secp256k1 public keys — no smart
// contract. See custody docs/btc_spec.md.
//
// This file holds the chain primitives that are a pure function of their
// inputs: redeem-script construction, vault address derivation, the
// deterministic unsigned-transaction builder, BIP-143 sighash computation, and
// witness assembly. They carry no daemon state and no network access, so every
// provider that observes the same authorized withdrawal and the same UTXO set
// builds a byte-identical transaction — the precondition for non-interactive
// multisig signing.
package btc

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// dustThresholdSats is the minimum value (in satoshis) a P2WSH change output
// may carry; below this the output is omitted and the remainder goes to fee.
const dustThresholdSats = 330

// RedeemScript builds the m-of-n OP_CHECKMULTISIG redeem script over the given
// compressed secp256k1 public keys. Keys are sorted lexicographically (BIP 67)
// before assembly so the script — and therefore the vault address — is
// independent of the order in which keys were supplied.
func RedeemScript(threshold int, pubkeys [][]byte) ([]byte, error) {
	sorted, err := sortedPubkeys(threshold, pubkeys)
	if err != nil {
		return nil, err
	}
	b := txscript.NewScriptBuilder()
	b.AddInt64(int64(threshold)) // OP_m
	for _, pk := range sorted {
		b.AddData(pk)
	}
	b.AddInt64(int64(len(sorted))) // OP_n
	b.AddOp(txscript.OP_CHECKMULTISIG)
	return b.Script()
}

// TaggedRedeemScript prefixes the m-of-n redeem script with `<tag> OP_DROP`,
// yielding a distinct witness script — and therefore a distinct P2WSH address —
// per tag while leaving the signing semantics identical: OP_DROP discards the
// tag before OP_CHECKMULTISIG, so the same keys sign the same way with no key
// derivation. This is how per-account deposit addresses are derived
// (tag = AccountTag(accountURI)); every deposit address is full m-of-n custody.
func TaggedRedeemScript(tag []byte, threshold int, pubkeys [][]byte) ([]byte, error) {
	if len(tag) == 0 || len(tag) > 64 {
		return nil, fmt.Errorf("btc: tag must be 1..64 bytes, got %d", len(tag))
	}
	sorted, err := sortedPubkeys(threshold, pubkeys)
	if err != nil {
		return nil, err
	}
	b := txscript.NewScriptBuilder()
	b.AddData(tag)
	b.AddOp(txscript.OP_DROP)
	b.AddInt64(int64(threshold)) // OP_m
	for _, pk := range sorted {
		b.AddData(pk)
	}
	b.AddInt64(int64(len(sorted))) // OP_n
	b.AddOp(txscript.OP_CHECKMULTISIG)
	return b.Script()
}

// AccountTag derives the per-account script tag from a clearnet account URI:
// the 32-byte SHA256 of the URI.
func AccountTag(accountURI string) []byte { return sha256Sum([]byte(accountURI)) }

// DepositAddress derives the per-account deposit P2WSH address and its witness
// script. Watch-only: a pure function of accountURI plus the base pubkeys, no
// private keys involved.
func DepositAddress(accountURI string, threshold int, pubkeys [][]byte, net *chaincfg.Params) (btcutil.Address, []byte, error) {
	redeem, err := TaggedRedeemScript(AccountTag(accountURI), threshold, pubkeys)
	if err != nil {
		return nil, nil, err
	}
	addr, err := VaultAddress(redeem, net)
	if err != nil {
		return nil, nil, err
	}
	return addr, redeem, nil
}

// sortedPubkeys validates the key set and returns BIP-67 (lexicographically)
// sorted copies of the compressed pubkeys.
func sortedPubkeys(threshold int, pubkeys [][]byte) ([][]byte, error) {
	if threshold < 1 || threshold > len(pubkeys) {
		return nil, fmt.Errorf("btc: threshold %d out of range for %d keys", threshold, len(pubkeys))
	}
	if len(pubkeys) > 15 {
		return nil, fmt.Errorf("btc: OP_CHECKMULTISIG supports at most 15 keys, got %d", len(pubkeys))
	}
	sorted := make([][]byte, len(pubkeys))
	for i, pk := range pubkeys {
		if len(pk) != 33 {
			return nil, fmt.Errorf("btc: pubkey %d is %d bytes, want 33 (compressed)", i, len(pk))
		}
		cp := make([]byte, 33)
		copy(cp, pk)
		sorted[i] = cp
	}
	sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i], sorted[j]) < 0 })
	return sorted, nil
}

// VaultAddress derives the P2WSH bech32 address for a redeem script: the
// witness program is SHA256(redeemScript).
func VaultAddress(redeemScript []byte, net *chaincfg.Params) (btcutil.Address, error) {
	return btcutil.NewAddressWitnessScriptHash(sha256Sum(redeemScript), net)
}

// PkScript returns the scriptPubKey for an address.
func PkScript(addr btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(addr)
}

// UTXO is a spendable vault output.
type UTXO struct {
	TxID   chainhash.Hash
	Vout   uint32
	Amount int64 // satoshis
}

// BuildUnsignedTx constructs the canonical unsigned withdrawal transaction.
// Inputs are the selected vault UTXOs (sorted BIP-69) so construction is
// order-independent. Outputs are emitted in a fixed canonical order: recipient,
// then change back to the vault (omitted if below dust), then a zero-value
// OP_RETURN carrying the 32-byte clearnet WithdrawalID.
func BuildUnsignedTx(
	utxos []UTXO,
	recipient btcutil.Address,
	amount int64,
	vault btcutil.Address,
	withdrawalID [32]byte,
	feeSats int64,
) (*wire.MsgTx, error) {
	if len(utxos) == 0 {
		return nil, fmt.Errorf("btc: no UTXOs selected")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("btc: non-positive amount %d", amount)
	}
	if feeSats < 0 {
		return nil, fmt.Errorf("btc: negative fee %d", feeSats)
	}

	ordered := make([]UTXO, len(utxos))
	copy(ordered, utxos)
	sort.Slice(ordered, func(i, j int) bool {
		if c := bytes.Compare(ordered[i].TxID[:], ordered[j].TxID[:]); c != 0 {
			return c < 0
		}
		return ordered[i].Vout < ordered[j].Vout
	})

	var inTotal int64
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, u := range ordered {
		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&u.TxID, u.Vout), nil, nil))
		inTotal += u.Amount
	}

	recipientScript, err := txscript.PayToAddrScript(recipient)
	if err != nil {
		return nil, fmt.Errorf("btc: recipient script: %w", err)
	}
	vaultScript, err := txscript.PayToAddrScript(vault)
	if err != nil {
		return nil, fmt.Errorf("btc: vault script: %w", err)
	}

	change := inTotal - amount - feeSats
	if change < 0 {
		return nil, fmt.Errorf("btc: inputs %d below amount %d + fee %d", inTotal, amount, feeSats)
	}

	tx.AddTxOut(wire.NewTxOut(amount, recipientScript))
	if change >= dustThresholdSats {
		tx.AddTxOut(wire.NewTxOut(change, vaultScript))
	}

	opReturn, err := txscript.NullDataScript(withdrawalID[:])
	if err != nil {
		return nil, fmt.Errorf("btc: OP_RETURN script: %w", err)
	}
	tx.AddTxOut(wire.NewTxOut(0, opReturn))

	return tx, nil
}

// SighashAll computes the BIP-143 SIGHASH_ALL digest for one input.
func SighashAll(
	tx *wire.MsgTx,
	inputIdx int,
	redeemScript []byte,
	amount int64,
	prevFetcher txscript.PrevOutputFetcher,
) ([]byte, error) {
	sigHashes := txscript.NewTxSigHashes(tx, prevFetcher)
	return txscript.CalcWitnessSigHash(redeemScript, sigHashes, txscript.SigHashAll, tx, inputIdx, amount)
}

// AssembleWitness builds the witness stack for a P2WSH multisig input:
//
//	[ <empty>, sig_1, ..., sig_m, redeemScript ]
//
// The leading empty element is the OP_CHECKMULTISIG off-by-one workaround. Each
// signature is DER-encoded with the SIGHASH_ALL type byte already appended, and
// the slice MUST already be ordered to match the position of the corresponding
// public key in the redeem script.
func AssembleWitness(redeemScript []byte, orderedSigs [][]byte) wire.TxWitness {
	w := make(wire.TxWitness, 0, len(orderedSigs)+2)
	w = append(w, nil) // CHECKMULTISIG dummy
	w = append(w, orderedSigs...)
	w = append(w, redeemScript)
	return w
}

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

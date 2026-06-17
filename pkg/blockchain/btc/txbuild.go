package btc

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/wire"
)

// validateFixedTxFields asserts the fixed fields the BIP-143 SIGHASH_ALL digest
// commits to, matching what the canonical builders produce: version
// wire.TxVersion, locktime 0, and final (non-RBF) input sequences. The sighash
// already binds these, so a Byzantine canonicalizer cannot make followers sign
// something inconsistent — but without the checks it could induce co-signing of
// a non-final or RBF-signalling tx and waste a signing round (ISS-006(b),
// griefing only). Shared by the withdrawal and rotation validators.
func validateFixedTxFields(tx *wire.MsgTx) error {
	if tx.Version != wire.TxVersion {
		return fmt.Errorf("unexpected tx version %d", tx.Version)
	}
	if tx.LockTime != 0 {
		return fmt.Errorf("non-zero locktime %d", tx.LockTime)
	}
	for i, in := range tx.TxIn {
		if in.Sequence != wire.MaxTxInSequenceNum {
			return fmt.Errorf("input %d non-final sequence %d", i, in.Sequence)
		}
	}
	return nil
}

// Witness/size constants for a P2WSH m-of-n input, used for fee estimation.
const (
	// p2wshInputVBytes is the vsize contribution of one signed P2WSH input
	// (witness bytes already discounted 4x), sized conservatively.
	p2wshInputVBytes = 120
	// txOverheadVBytes covers version, locktime, segwit marker/flag, counters.
	txOverheadVBytes = 11
	// outputVBytes is a conservative upper bound on one output's vsize.
	outputVBytes = 43
)

// EstimateFeeSats returns the fee for a transaction with numInputs P2WSH inputs
// and numOutputs outputs at the given rate.
func EstimateFeeSats(numInputs, numOutputs int, satPerVByte int64) int64 {
	vsize := int64(txOverheadVBytes) +
		int64(numInputs)*p2wshInputVBytes +
		int64(numOutputs)*outputVBytes
	return vsize * satPerVByte
}

// SelectUTXOs deterministically chooses inputs to cover amount plus the fee the
// resulting transaction will pay. UTXOs are sorted by (amount desc, txid, vout)
// and accumulated greedily until they cover amount + fee, where the fee grows
// with each added input. numFixedOutputs is the count of always-present outputs
// (recipient + OP_RETURN = 2); a change output is assumed for fee sizing.
func SelectUTXOs(available []UTXO, amount int64, satPerVByte int64, numFixedOutputs int) (selected []UTXO, feeSats int64, err error) {
	if amount <= 0 {
		return nil, 0, fmt.Errorf("btc: non-positive amount %d", amount)
	}
	pool := make([]UTXO, len(available))
	copy(pool, available)
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].Amount != pool[j].Amount {
			return pool[i].Amount > pool[j].Amount // largest first
		}
		if c := compareHash(pool[i].TxID[:], pool[j].TxID[:]); c != 0 {
			return c < 0
		}
		return pool[i].Vout < pool[j].Vout
	})

	var total int64
	for i, u := range pool {
		total += u.Amount
		n := i + 1
		fee := EstimateFeeSats(n, numFixedOutputs+1, satPerVByte)
		if total >= amount+fee {
			return pool[:n], fee, nil
		}
	}
	return nil, 0, fmt.Errorf("btc: insufficient vault balance: have %d, need %d + fee at %d sat/vB",
		total, amount, satPerVByte)
}

func compareHash(a, b []byte) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

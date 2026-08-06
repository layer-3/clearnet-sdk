package btc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
)

// These defaults apply only when the legacy custody Config leaves the
// corresponding depositor setting at zero. The public P2WPKHConfig remains
// explicit and does not apply defaults.
const (
	defaultDepositorMinConfirmations      uint64 = 1
	defaultDepositorFeeConfirmationTarget        = 6
	defaultDepositorFallbackFeeRate       int64  = 5
	defaultDepositorFeeCapSatPerVByte     int64  = 1_000
	defaultDepositorDustThresholdSats     int64  = 330
	defaultDepositorMaxInputs                    = 100
)

type depositorRPCBackend struct {
	rpc             RPC
	fallbackFeeRate int64
}

var _ DepositorBackend = (*depositorRPCBackend)(nil)

func (b *depositorRPCBackend) ListUnspent(ctx context.Context, address string, minConfirmations uint64) ([]UnspentOutput, error) {
	if minConfirmations > uint64(math.MaxInt) {
		return nil, fmt.Errorf("minimum confirmations %d exceed legacy RPC integer range", minConfirmations)
	}
	unspent, err := b.rpc.ListUnspent(ctx, int(minConfirmations), []string{address})
	if err != nil {
		return nil, err
	}
	out := make([]UnspentOutput, len(unspent))
	for i, candidate := range unspent {
		if candidate.Confirmations < 0 {
			return nil, fmt.Errorf("UTXO %s:%d has negative confirmations %d", candidate.TxID, candidate.Vout, candidate.Confirmations)
		}
		script, err := hex.DecodeString(candidate.ScriptPubKey)
		if err != nil {
			return nil, fmt.Errorf("decode UTXO %s:%d scriptPubKey: %w", candidate.TxID, candidate.Vout, err)
		}
		out[i] = UnspentOutput{
			TxID:          candidate.TxID,
			Vout:          candidate.Vout,
			AmountSats:    candidate.AmountSats,
			ScriptPubKey:  script,
			Confirmations: uint64(candidate.Confirmations),
		}
	}
	return out, nil
}

func (b *depositorRPCBackend) FeeRateSatPerVByte(ctx context.Context, confirmationTarget int) (int64, error) {
	return b.rpc.EstimateSmartFeeSatPerVByte(ctx, confirmationTarget, b.fallbackFeeRate)
}

func (b *depositorRPCBackend) Broadcast(ctx context.Context, rawTx []byte) (string, error) {
	tx, err := deserializeTx(rawTx)
	if err != nil {
		return "", fmt.Errorf("decode transaction before broadcast: %w", err)
	}
	txID := tx.TxHash().String()
	returnedID, err := b.rpc.SendRawTransaction(ctx, hex.EncodeToString(rawTx))
	if err != nil {
		if broadcastAlreadyAccepted(ctx, b.rpc, txID, err) {
			return txID, nil
		}
		return "", err
	}
	return returnedID, nil
}

func (b *depositorRPCBackend) GetTransactionConfirmations(ctx context.Context, txID string) (uint64, bool, error) {
	raw, err := b.rpc.GetRawTransaction(ctx, txID)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -5 { // RPC_INVALID_ADDRESS_OR_KEY: unknown tx
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("btc: getrawtransaction: %w", err)
	}
	if raw == nil {
		return 0, false, nil
	}
	if raw.Confirmations < 0 {
		// Preserve the legacy behavior for a known conflicted transaction: it
		// remains observed but has no active-chain confirmations.
		return 0, true, nil
	}
	return uint64(raw.Confirmations), true, nil
}

func depositorP2WPKHConfig(cfg Config) P2WPKHConfig {
	minConfirmations := cfg.ConfirmationDepth
	if minConfirmations == 0 {
		minConfirmations = defaultDepositorMinConfirmations
	}
	feeTarget := cfg.FeeConfTarget
	if feeTarget == 0 {
		feeTarget = defaultDepositorFeeConfirmationTarget
	}
	fallbackRate := cfg.FallbackFeeRate
	if fallbackRate == 0 {
		fallbackRate = defaultDepositorFallbackFeeRate
	}
	feeCap := cfg.FeeCapSatPerVByte
	if feeCap == 0 {
		feeCap = defaultDepositorFeeCapSatPerVByte
	}
	return P2WPKHConfig{
		MinConfirmations:           minConfirmations,
		FeeConfirmationTarget:      feeTarget,
		FallbackFeeRateSatPerVByte: fallbackRate,
		FeeCapSatPerVByte:          feeCap,
		DustThresholdSats:          defaultDepositorDustThresholdSats,
		MaxInputs:                  defaultDepositorMaxInputs,
	}
}

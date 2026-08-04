package btc

import (
	"context"
	"encoding/hex"
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

var _ P2WPKHBackend = (*depositorRPCBackend)(nil)

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
	return b.rpc.SendRawTransaction(ctx, hex.EncodeToString(rawTx))
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

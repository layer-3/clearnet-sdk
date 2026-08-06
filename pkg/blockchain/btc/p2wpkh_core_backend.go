package btc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// CoreUTXOMode selects how a CoreP2WPKHBackend discovers funding outputs.
// Wallet mode is fast but only sees outputs tracked by the configured Core
// wallet. Scan mode finds outputs for an arbitrary address without importing a
// key or descriptor, at the cost of a synchronous global UTXO-set scan.
type CoreUTXOMode uint8

const (
	CoreWalletUTXOs CoreUTXOMode = iota
	CoreScanTxOutSetUTXOs
)

// CoreP2WPKHBackendConfig configures transport behavior only. Funding policy
// such as confirmation depth, fallback and maximum fee rates, dust, and input
// limits belongs to P2WPKHConfig.
type CoreP2WPKHBackendConfig struct {
	UTXOMode                CoreUTXOMode
	FixedFeeRateSatPerVByte int64
	// RequestTimeout bounds wallet lookup, fee, broadcast, and transaction
	// observation calls when positive. Zero relies on the caller context and
	// Client's HTTP policy.
	RequestTimeout time.Duration
	// ScanTimeout independently bounds synchronous scantxoutset calls when
	// positive. Mainnet scans commonly need a larger budget than ordinary RPCs.
	ScanTimeout time.Duration
}

// CoreP2WPKHBackend supplies reusable Bitcoin Core chain access for a
// P2WPKHSender or Depositor.
type CoreP2WPKHBackend struct {
	client         *Client
	rpc            RPC
	utxoMode       CoreUTXOMode
	fixedFeeRate   int64
	legacyFallback int64
	requestTimeout time.Duration
	scanTimeout    time.Duration
}

var _ DepositorBackend = (*CoreP2WPKHBackend)(nil)

// NewCoreP2WPKHBackend constructs a Bitcoin Core depositor backend. The
// client's wallet name is used only in CoreWalletUTXOs mode.
func NewCoreP2WPKHBackend(client *Client, cfg CoreP2WPKHBackendConfig) (*CoreP2WPKHBackend, error) {
	if client == nil {
		return nil, fmt.Errorf("btc: Core P2WPKH client is required")
	}
	switch cfg.UTXOMode {
	case CoreWalletUTXOs, CoreScanTxOutSetUTXOs:
	default:
		return nil, fmt.Errorf("btc: unsupported Core P2WPKH UTXO mode %d", cfg.UTXOMode)
	}
	if cfg.FixedFeeRateSatPerVByte < 0 {
		return nil, fmt.Errorf("btc: Core P2WPKH fixed fee rate must not be negative")
	}
	if cfg.RequestTimeout < 0 || cfg.ScanTimeout < 0 {
		return nil, fmt.Errorf("btc: Core P2WPKH timeouts must not be negative")
	}
	return &CoreP2WPKHBackend{
		client:         client,
		rpc:            client,
		utxoMode:       cfg.UTXOMode,
		fixedFeeRate:   cfg.FixedFeeRateSatPerVByte,
		requestTimeout: cfg.RequestTimeout,
		scanTimeout:    cfg.ScanTimeout,
	}, nil
}

// newLegacyCoreP2WPKHBackend preserves NewDepositor's historical support for
// arbitrary RPC implementations and wallet-scoped listunspent. New reusable
// callers should use NewCoreP2WPKHBackend instead.
func newLegacyCoreP2WPKHBackend(rpc RPC, fallbackFeeRate int64) *CoreP2WPKHBackend {
	return &CoreP2WPKHBackend{
		rpc:            rpc,
		utxoMode:       CoreWalletUTXOs,
		legacyFallback: fallbackFeeRate,
	}
}

func (b *CoreP2WPKHBackend) ListUnspent(ctx context.Context, address string, minConfirmations uint64) ([]UnspentOutput, error) {
	if b.utxoMode == CoreWalletUTXOs {
		requestCtx, cancel := optionalBackendTimeout(ctx, b.requestTimeout)
		defer cancel()
		return listWalletP2WPKHUnspent(requestCtx, b.rpc, address, minConfirmations)
	}
	scanCtx, cancel := optionalBackendTimeout(ctx, b.scanTimeout)
	defer cancel()
	scan, err := b.client.ScanTxOutSet(scanCtx, []string{"addr(" + address + ")"})
	if err != nil {
		return nil, fmt.Errorf("btc: scan P2WPKH UTXOs: %w", err)
	}
	if scan == nil || !scan.Success {
		return nil, fmt.Errorf("btc: scantxoutset did not complete successfully")
	}
	out := make([]UnspentOutput, 0, len(scan.Unspents))
	for _, candidate := range scan.Unspents {
		if candidate.AmountSats <= 0 {
			return nil, fmt.Errorf("btc: scantxoutset UTXO %s:%d has invalid amount %d", candidate.TxID, candidate.Vout, candidate.AmountSats)
		}
		if candidate.Confirmations < minConfirmations {
			continue
		}
		out = append(out, UnspentOutput{
			TxID:          candidate.TxID,
			Vout:          candidate.Vout,
			AmountSats:    candidate.AmountSats,
			ScriptPubKey:  append([]byte(nil), candidate.ScriptPubKey...),
			Confirmations: candidate.Confirmations,
		})
	}
	return out, nil
}

func (b *CoreP2WPKHBackend) FeeRateSatPerVByte(ctx context.Context, confirmationTarget int) (int64, error) {
	if b.fixedFeeRate > 0 {
		return b.fixedFeeRate, nil
	}
	requestCtx, cancel := optionalBackendTimeout(ctx, b.requestTimeout)
	defer cancel()
	if b.legacyFallback > 0 {
		return b.rpc.EstimateSmartFeeSatPerVByte(requestCtx, confirmationTarget, b.legacyFallback)
	}
	rate, available, err := b.client.estimateSmartFeeSatPerVByte(requestCtx, confirmationTarget)
	if err != nil {
		return 0, err
	}
	if !available {
		return 0, ErrP2WPKHFeeEstimateUnavailable
	}
	return rate, nil
}

func (b *CoreP2WPKHBackend) Broadcast(ctx context.Context, rawTx []byte) (string, error) {
	requestCtx, cancel := optionalBackendTimeout(ctx, b.requestTimeout)
	defer cancel()
	return broadcastP2WPKHWithRPC(requestCtx, b.rpc, rawTx)
}

func (b *CoreP2WPKHBackend) GetTransactionConfirmations(ctx context.Context, txID string) (uint64, bool, error) {
	requestCtx, cancel := optionalBackendTimeout(ctx, b.requestTimeout)
	defer cancel()
	return transactionConfirmationsWithRPC(requestCtx, b.rpc, txID)
}

func listWalletP2WPKHUnspent(ctx context.Context, rpc interface {
	ListUnspent(context.Context, int, []string) ([]Unspent, error)
}, address string, minConfirmations uint64) ([]UnspentOutput, error) {
	if minConfirmations > uint64(math.MaxInt) {
		return nil, fmt.Errorf("minimum confirmations %d exceed legacy RPC integer range", minConfirmations)
	}
	unspent, err := rpc.ListUnspent(ctx, int(minConfirmations), []string{address})
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

func broadcastP2WPKHWithRPC(ctx context.Context, rpc RPC, rawTx []byte) (string, error) {
	tx, err := deserializeTx(rawTx)
	if err != nil {
		return "", fmt.Errorf("decode transaction before broadcast: %w", err)
	}
	txID := tx.TxHash().String()
	returnedID, err := rpc.SendRawTransaction(ctx, hex.EncodeToString(rawTx))
	if err != nil {
		if broadcastAlreadyAccepted(ctx, rpc, txID, err) {
			return txID, nil
		}
		return "", err
	}
	return returnedID, nil
}

func transactionConfirmationsWithRPC(ctx context.Context, rpc interface {
	GetRawTransaction(context.Context, string) (*RawTx, error)
}, txID string) (uint64, bool, error) {
	if err := validateHashString(txID); err != nil {
		return 0, false, fmt.Errorf("btc: invalid transaction ID %q: %w", txID, err)
	}
	raw, err := rpc.GetRawTransaction(ctx, txID)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -5 {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("btc: getrawtransaction: %w", err)
	}
	if raw == nil {
		return 0, false, nil
	}
	if raw.TxID != "" {
		if err := validateHashString(raw.TxID); err != nil {
			return 0, false, fmt.Errorf("btc: getrawtransaction returned invalid transaction ID %q: %w", raw.TxID, err)
		}
		if !strings.EqualFold(raw.TxID, txID) {
			return 0, false, fmt.Errorf("btc: getrawtransaction transaction ID mismatch: requested %s, returned %s", txID, raw.TxID)
		}
	}
	if raw.Confirmations < 0 {
		return 0, true, nil
	}
	return uint64(raw.Confirmations), true, nil
}

func optionalBackendTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

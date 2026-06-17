package core

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// AssetTransfer represents a single asset-amount pair in a multi-asset operation (§5.3.5).
type AssetTransfer struct {
	Asset  AssetID
	Amount decimal.Decimal
}

// CanonicalAssetTransfers validates, copies, sorts, and de-duplicates a
// TransferOp asset list.
func CanonicalAssetTransfers(in []AssetTransfer) ([]AssetTransfer, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("transfer assets required")
	}
	out := make([]AssetTransfer, len(in))
	for i, asset := range in {
		if asset.Asset == "" {
			return nil, fmt.Errorf("transfer asset %d missing Asset", i)
		}
		if asset.Amount.Sign() <= 0 {
			return nil, fmt.Errorf("transfer asset %s Amount must be > 0", asset.Asset)
		}
		out[i] = asset
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Asset < out[j].Asset })
	for i := 1; i < len(out); i++ {
		if out[i-1].Asset == out[i].Asset {
			return nil, fmt.Errorf("duplicate transfer asset %s", out[i].Asset)
		}
	}
	return out, nil
}

// TransferOp is the payload for a transfer block entry (§5.3.5).
// Issued by the sender's cluster after debiting the sender.
// Consumed by the recipient's cluster to credit the recipient.
type TransferOp struct {
	TxID   string          // Original transaction ID
	To     string          // Recipient address
	Assets []AssetTransfer // Assets transferred, sorted by AssetID (§7.2)
}

// NewSingleAssetTransferOp creates a TransferOp with a single asset (common case).
func NewSingleAssetTransferOp(txID, to string, asset AssetID, amount decimal.Decimal) *TransferOp {
	return &TransferOp{
		TxID:   txID,
		To:     to,
		Assets: []AssetTransfer{{Asset: asset, Amount: amount}},
	}
}

// ---------------------------------------------------------------------------
// Swap TransferOp TxID convention (§7.1)
//
// liquidity.md §8.3: liquidity operations are expressed as ordinary multi-asset
// TransferOps to pool URIs and require no synthetic prefix; the pool cluster
// dispatches by asset shape (2 assets ⇒ AddLiquidity, 1 LP-share asset ⇒
// RemoveLiquidity). Only swap retains a TxID prefix because the ordinary
// Transfer typed-data carries no AssetOut / MinAmountOut field — the prefix
// re-attaches that user-authorized data to the user→pool handoff op.
// ---------------------------------------------------------------------------

const (
	swapPrefix = "swap:"
)

// MakeSwapTxID creates the TransferOp TxID used for the user->pool swap
// handoff. TransferOp has no AssetOut field, so the pool cluster recovers the
// user-authorized output asset and slippage floor from this stable prefix after
// validating the sealed user block.
func MakeSwapTxID(baseTxID string, assetOut AssetID, minAmountOut *big.Int) string {
	min := "0"
	if minAmountOut != nil && minAmountOut.Sign() > 0 {
		min = minAmountOut.String()
	}
	return swapPrefix + string(assetOut) + ":" + min + ":" + baseTxID
}

// ParseSwapTxID extracts the fields encoded by MakeSwapTxID.
func ParseSwapTxID(txID string) (baseTxID string, assetOut AssetID, minAmountOut *big.Int, ok bool) {
	if !strings.HasPrefix(txID, swapPrefix) {
		return "", "", nil, false
	}
	parts := strings.SplitN(strings.TrimPrefix(txID, swapPrefix), ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return "", "", nil, false
	}
	min, parsed := new(big.Int).SetString(parts[1], 10)
	if !parsed || min.Sign() < 0 {
		return "", "", nil, false
	}
	return parts[2], AssetID(parts[0]), min, true
}

// IsSwapTxID returns true if the TxID represents a swap handoff TransferOp.
func IsSwapTxID(txID string) bool {
	return strings.HasPrefix(txID, swapPrefix)
}

// SwapOp is the payload for a swap execution block entry (§7.1).
// Issued by the pool cluster after executing the AMM swap.
// Consumed by the user's cluster to credit the output asset.
type SwapOp struct {
	TxID      string          // Original transaction ID
	AssetIn   AssetID         // Input asset (what the user paid)
	AssetOut  AssetID         // Output asset (what the user receives)
	AmountIn  decimal.Decimal // Amount of input asset consumed
	AmountOut decimal.Decimal // Amount of output asset produced (after fee)
	PoolID    string          // Pool that executed the swap
	Fee       decimal.Decimal // Fee deducted (in output asset)
	FeeRate   uint64          // Dynamic fee rate applied (bps) — §5.3
	PriceEMA  decimal.Decimal // Post-swap EMA price (1e18 fixed-point, data_layer.md §4.2)
	SpotPrice decimal.Decimal // Post-swap spot price (1e18 fixed-point, data_layer.md §4.2)
}

// RepegOp is the payload for a PriceScale repegging block entry (SS8.1.2).
// Issued by the pool cluster after a profit-gated PriceScale adjustment.
type RepegOp struct {
	PoolID        string   // Target pool
	OldPriceScale *big.Int // PriceScale before the repeg
	NewPriceScale *big.Int // PriceScale after the repeg
	OldVirtPrice  *big.Int // VirtualPrice before the repeg
	NewVirtPrice  *big.Int // VirtualPrice after the repeg
	Epoch         uint64   // Epoch number when the repeg occurred
	PriceEMA      *big.Int // Post-repeg PriceEMA (data_layer.md §4.2)
}

// WithdrawalOp is the payload for a withdrawal block entry (SS7.3.1).
// Issued by the user's cluster after locking funds for L1 withdrawal.
// Delivered to the custody layer after the clearing-side challenge window.
//
// Removed fields (now in block header or derived at vault level):
//   - WithdrawalID: derived at vault level as keccak256(accountId, blockHash, entryIndex, chainId, recipient, asset, amount, nonce)
//   - Nonce: in BlockEntry.Nonce
//   - K: in Block.K
//   - SignedAt: in Block.SealedAt
//   - SignerNodeIDs: recoverable from Block.Attestation.Bitmask + cluster lookup
type WithdrawalOp struct {
	Asset         AssetID         // Protocol-level asset name (e.g. "ETH") — needed for ledger finalization
	L1Asset       string          // On-chain asset address (e.g., "0xA0b8...USDC")
	Amount        decimal.Decimal // Withdrawal amount in token units
	ChainID       uint64          // Target L1 chain
	Recipient     string          // L1 recipient address (chain-native format)
	UserSignature []byte          // User's ECDSA authorization of the withdrawal
}

// SessionCloseOp is the payload for a session close block entry (service_sessions.md §3.3).
// Emitted during cooperative or unilateral session settlement.
type SessionCloseOp struct {
	SessionID     [32]byte        // keccak256(userAccountID || serviceAccountID || nonce)
	Version       uint64          // Latest co-signed version
	UserAmount    decimal.Decimal // Final user allocation
	ServiceAmount decimal.Decimal // Final service allocation
	Cooperative   bool            // True if both parties signed
}

// SessionChallengeOp is the payload for a session dispute challenge (service_sessions.md §5.1).
type SessionChallengeOp struct {
	SessionID       [32]byte // Session being challenged
	PreviousVersion uint64   // Version being replaced
	NewVersion      uint64   // Challenger's version (must be higher)
	NewDeadline     int64    // Unix timestamp when challenge window expires
}

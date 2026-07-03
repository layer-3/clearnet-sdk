package core

import "time"

// Withdrawal-authorization time bounds.
//
// Every withdrawal digest signed by the custody quorum embeds a `deadline`
// (unix seconds) past which the authorization is void — on EVM/Solana it is a
// consensus-enforced execution check, on XRPL it caps `LastLedgerSequence`, and
// clearnet uses it to decide when a reverted/unbroadcast withdrawal may be
// safely re-credited. Because the deadline is a *digest input*, these values
// are compile-time constants: they cannot ride the config registry (ADR-017)
// without opening a digest-divergence window across a rolling restart. Changing
// them is therefore a coordinated binary upgrade, not a runtime config change.
//
// The values are provisional pre-mainnet. They are defined ONCE here in the SDK
// (clearnet imports the SDK) so custody and clearnet cannot drift.
const (
	// MaxBlockAge bounds how stale a finalized block may be and still be
	// signed/submitted (freshness cutoff on the signing path). It also anchors
	// the withdrawal validity window: a block older than MaxBlockAge is never
	// re-signed, so no fresh authorization can be minted past that point.
	MaxBlockAge = 7 * 24 * time.Hour

	// ExecutionMargin is the slack added on top of MaxBlockAge to cover the time
	// a fresh authorization needs to be signed, submitted, and confirmed on L1.
	// It bounds the per-attempt liveness budget on XRPL (see LedgerBudget).
	ExecutionMargin = 2 * time.Hour

	// WithdrawalValidityWindow is the total lifetime of a withdrawal
	// authorization measured from the source block's SealedAt. Any signed
	// digest is dead once SealedAt + WithdrawalValidityWindow has passed.
	WithdrawalValidityWindow = MaxBlockAge + ExecutionMargin
)

// WithdrawalDeadline returns the unix-second time bound for a withdrawal whose
// source block sealed at sealedAt (also unix seconds). This is the single value
// threaded into every chain's withdrawal digest and enforced on-chain.
func WithdrawalDeadline(sealedAt int64) int64 {
	return sealedAt + int64(WithdrawalValidityWindow/time.Second)
}

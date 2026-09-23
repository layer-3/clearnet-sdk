package core

import (
	"fmt"
	"math"
	"time"
)

// Withdrawal-authorization time policy. ChallengeDuration is intentionally
// supplied by custody configuration: it is not an SDK or on-chain constant.
const (
	FinalizationAdmissionHorizon = 24 * time.Hour
	WithdrawalExecutionWindow    = time.Hour
	ClockSkewTolerance           = 5 * time.Minute
)

// WithdrawalTimeBounds carries the authenticated finalization time and its
// deterministic post-finalization execution ceiling. FinalizedAt is included
// in the clearnet BLS attestation; ValidUntil is always FinalizedAt plus
// WithdrawalExecutionWindow.
type WithdrawalTimeBounds struct {
	FinalizedAt int64
	ValidUntil  int64
}

// NewWithdrawalTimeBounds validates a newly received finalized withdrawal.
// sealedAt, finalizedAt, and localNow are Unix seconds.
func NewWithdrawalTimeBounds(sealedAt, finalizedAt, localNow int64, challengeDuration time.Duration) (WithdrawalTimeBounds, error) {
	if sealedAt < 0 {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: sealed_at must be non-negative")
	}
	if challengeDuration <= 0 || challengeDuration >= FinalizationAdmissionHorizon || challengeDuration%time.Second != 0 {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: challenge duration must be whole seconds in (0, %s)", FinalizationAdmissionHorizon)
	}
	if finalizedAt < sealedAt {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: finalized_at %d before sealed_at %d", finalizedAt, sealedAt)
	}
	delay := finalizedAt - sealedAt
	challengeSeconds := int64(challengeDuration / time.Second)
	if delay < challengeSeconds {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: finalization delay %ds below challenge duration %ds", delay, challengeSeconds)
	}
	if delay > int64(FinalizationAdmissionHorizon/time.Second) {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: finalization delay %ds exceeds admission horizon", delay)
	}
	if localNow > math.MaxInt64-int64(ClockSkewTolerance/time.Second) || finalizedAt > localNow+int64(ClockSkewTolerance/time.Second) {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: finalized_at %d exceeds local clock tolerance", finalizedAt)
	}
	if localNow > finalizedAt && localNow-finalizedAt > int64(FinalizationAdmissionHorizon/time.Second) {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: finalized_at %d is older than admission horizon", finalizedAt)
	}
	if finalizedAt > math.MaxInt64-int64(WithdrawalExecutionWindow/time.Second) {
		return WithdrawalTimeBounds{}, fmt.Errorf("withdrawal time bounds: valid_until overflow")
	}
	bounds := WithdrawalTimeBounds{
		FinalizedAt: finalizedAt,
		ValidUntil:  finalizedAt + int64(WithdrawalExecutionWindow/time.Second),
	}
	if err := bounds.Validate(); err != nil {
		return WithdrawalTimeBounds{}, err
	}
	return bounds, nil
}

// Validate checks the chain-independent relationship without consulting a
// clock or mutable configuration.
func (b WithdrawalTimeBounds) Validate() error {
	if b.FinalizedAt < 0 {
		return fmt.Errorf("withdrawal time bounds: finalized_at must be non-negative")
	}
	if b.FinalizedAt > math.MaxInt64-int64(WithdrawalExecutionWindow/time.Second) {
		return fmt.Errorf("withdrawal time bounds: valid_until overflow")
	}
	want := b.FinalizedAt + int64(WithdrawalExecutionWindow/time.Second)
	if b.ValidUntil != want {
		return fmt.Errorf("withdrawal time bounds: valid_until %d != finalized_at + %s (%d)", b.ValidUntil, WithdrawalExecutionWindow, want)
	}
	return nil
}

// RequireLive rejects starting a new expiring-chain ceremony after the exact
// inclusive authorization boundary has passed.
func (b WithdrawalTimeBounds) RequireLive(localNow int64) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if localNow > b.ValidUntil {
		return fmt.Errorf("withdrawal time bounds: authorization expired at %d", b.ValidUntil)
	}
	return nil
}

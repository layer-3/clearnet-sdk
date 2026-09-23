package core

import (
	"math"
	"testing"
	"time"
)

func TestNewWithdrawalTimeBounds(t *testing.T) {
	const sealedAt = int64(1_000_000)
	challenge := 30 * time.Minute

	tests := []struct {
		name        string
		sealedAt    int64
		finalizedAt int64
		now         int64
		challenge   time.Duration
		wantErr     bool
	}{
		{name: "challenge one second early", sealedAt: sealedAt, finalizedAt: sealedAt + 1799, now: sealedAt + 1799, challenge: challenge, wantErr: true},
		{name: "challenge exact", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: sealedAt + 1800, challenge: challenge},
		{name: "challenge one second late", sealedAt: sealedAt, finalizedAt: sealedAt + 1801, now: sealedAt + 1801, challenge: challenge},
		{name: "admission exact", sealedAt: sealedAt, finalizedAt: sealedAt + 86400, now: sealedAt + 86400, challenge: challenge},
		{name: "admission one second late", sealedAt: sealedAt, finalizedAt: sealedAt + 86401, now: sealedAt + 86401, challenge: challenge, wantErr: true},
		{name: "future skew exact", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: sealedAt + 1500, challenge: challenge},
		{name: "future skew one second late", sealedAt: sealedAt, finalizedAt: sealedAt + 1801, now: sealedAt + 1500, challenge: challenge, wantErr: true},
		{name: "absolute age exact", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: sealedAt + 1800 + 86400, challenge: challenge},
		{name: "absolute age one second late", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: sealedAt + 1800 + 86401, challenge: challenge, wantErr: true},
		{name: "negative sealed", sealedAt: -1, finalizedAt: 1800, now: 1800, challenge: challenge, wantErr: true},
		{name: "finalized before sealed", sealedAt: sealedAt, finalizedAt: sealedAt - 1, now: sealedAt, challenge: challenge, wantErr: true},
		{name: "zero challenge", sealedAt: sealedAt, finalizedAt: sealedAt, now: sealedAt, challenge: 0, wantErr: true},
		{name: "challenge one second below horizon", sealedAt: sealedAt, finalizedAt: sealedAt + 86399, now: sealedAt + 86399, challenge: 24*time.Hour - time.Second},
		{name: "challenge at horizon", sealedAt: sealedAt, finalizedAt: sealedAt + 86400, now: sealedAt + 86400, challenge: FinalizationAdmissionHorizon, wantErr: true},
		{name: "challenge above horizon", sealedAt: sealedAt, finalizedAt: sealedAt + 86400, now: sealedAt + 86400, challenge: 24*time.Hour + time.Second, wantErr: true},
		{name: "fractional challenge", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: sealedAt + 1800, challenge: challenge + time.Nanosecond, wantErr: true},
		{name: "clock addition overflow", sealedAt: sealedAt, finalizedAt: sealedAt + 1800, now: math.MaxInt64 - 1, challenge: challenge, wantErr: true},
		{name: "valid until overflow", sealedAt: math.MaxInt64 - 4000, finalizedAt: math.MaxInt64 - 3000, now: math.MaxInt64 - 3000, challenge: challenge, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWithdrawalTimeBounds(tt.sealedAt, tt.finalizedAt, tt.now, tt.challenge)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewWithdrawalTimeBounds() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got.ValidUntil != tt.finalizedAt+3600 {
				t.Fatalf("ValidUntil = %d, want %d", got.ValidUntil, tt.finalizedAt+3600)
			}
		})
	}
}

func TestWithdrawalTimeBoundsValidateAndRequireLive(t *testing.T) {
	bounds := WithdrawalTimeBounds{FinalizedAt: 1000, ValidUntil: 4600}
	if err := bounds.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := bounds.RequireLive(4600); err != nil {
		t.Fatalf("exact boundary rejected: %v", err)
	}
	if err := bounds.RequireLive(4601); err == nil {
		t.Fatal("one second past boundary accepted")
	}
	for _, invalid := range []WithdrawalTimeBounds{
		{FinalizedAt: -1, ValidUntil: 3599},
		{FinalizedAt: 1000, ValidUntil: 4599},
		{FinalizedAt: 1000, ValidUntil: 4601},
		{FinalizedAt: math.MaxInt64, ValidUntil: math.MaxInt64},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("Validate accepted %+v", invalid)
		}
	}
}

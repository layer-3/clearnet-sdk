package blockchain

import (
	"math"
	"testing"
)

func TestValidateMajorityThreshold(t *testing.T) {
	tests := []struct {
		name        string
		threshold   int
		signerCount int
		wantErr     bool
	}{
		{name: "one of one", threshold: 1, signerCount: 1},
		{name: "zero of one", threshold: 0, signerCount: 1, wantErr: true},
		{name: "two of three", threshold: 2, signerCount: 3},
		{name: "three of five", threshold: 3, signerCount: 5},
		{name: "four of six", threshold: 4, signerCount: 6},
		{name: "four of seven", threshold: 4, signerCount: 7},
		{name: "all of even set", threshold: 4, signerCount: 4},
		{name: "all signers", threshold: 7, signerCount: 7},
		{name: "zero threshold", threshold: 0, signerCount: 7, wantErr: true},
		{name: "negative threshold", threshold: -1, signerCount: 7, wantErr: true},
		{name: "empty signer set", threshold: 1, signerCount: 0, wantErr: true},
		{name: "one of three", threshold: 1, signerCount: 3, wantErr: true},
		{name: "half of four", threshold: 2, signerCount: 4, wantErr: true},
		{name: "half of even set", threshold: 3, signerCount: 6, wantErr: true},
		{name: "minority of odd set", threshold: 3, signerCount: 7, wantErr: true},
		{name: "above signer count", threshold: 8, signerCount: 7, wantErr: true},
		{name: "max int majority", threshold: math.MaxInt, signerCount: math.MaxInt},
		{name: "max int half", threshold: math.MaxInt / 2, signerCount: math.MaxInt, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMajorityThreshold(tt.threshold, tt.signerCount)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMajorityThreshold(%d, %d) error = %v, wantErr %v", tt.threshold, tt.signerCount, err, tt.wantErr)
			}
		})
	}
}

package core

import "testing"

func TestBlockSigningClusterSizePreProductionValue(t *testing.T) {
	if BlockSigningClusterSize != 1 {
		t.Fatalf("BlockSigningClusterSize = %d, want pre-production value 1", BlockSigningClusterSize)
	}
	if BlockSigningClusterSize == 0 || BlockSigningClusterSize > MaxClusterSize {
		t.Fatalf("BlockSigningClusterSize = %d, want 1..%d", BlockSigningClusterSize, MaxClusterSize)
	}
}

package evm

import (
	"context"
	"strings"
	"testing"
)

func TestAssetResolverValidateAssetAddress(t *testing.T) {
	r := &AssetResolver{}
	validToken := "0x" + strings.Repeat("ab", 20)
	zeroHex := "0x" + strings.Repeat("00", 20)

	for _, asset := range []string{nativeAssetAddress, validToken} {
		if err := r.ValidateAssetAddress(context.Background(), asset); err != nil {
			t.Fatalf("ValidateAssetAddress(%q): %v", asset, err)
		}
	}
	for _, asset := range []string{zeroHex, "ETH", ""} {
		if err := r.ValidateAssetAddress(context.Background(), asset); err == nil {
			t.Fatalf("ValidateAssetAddress(%q) accepted invalid asset", asset)
		}
	}
}

func TestNewAssetResolverPanicsWithoutClient(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewAssetResolver did not panic")
		}
	}()
	NewAssetResolver(nil, AssetResolverConfig{})
}

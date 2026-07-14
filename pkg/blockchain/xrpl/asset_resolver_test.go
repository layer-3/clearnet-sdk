package xrpl

import (
	"context"
	"testing"
)

const validIssuer = "rN7n7otQDd6FczFgLdSqtcsAUxDkw6fzRH"

func TestAssetResolverValidateAssetAddress(t *testing.T) {
	r := NewAssetResolver(AssetResolverConfig{})
	for _, asset := range []string{
		nativeAssetAddress,
		"USD." + validIssuer,
		"0123456789ABCDEF0123456789ABCDEF01234567." + validIssuer,
	} {
		if err := r.ValidateAssetAddress(context.Background(), asset); err != nil {
			t.Fatalf("ValidateAssetAddress(%q): %v", asset, err)
		}
	}
	for _, asset := range []string{
		"XRP." + validIssuer,
		"US." + validIssuer,
		"USDT." + validIssuer,
		"USD:" + validIssuer,
		"USD.rBad",
	} {
		if err := r.ValidateAssetAddress(context.Background(), asset); err == nil {
			t.Fatalf("ValidateAssetAddress(%q) accepted invalid asset", asset)
		}
	}
}

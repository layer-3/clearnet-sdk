package xrpl

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

const nativeAssetAddress = "0"

var (
	standardCurrencyPattern = regexp.MustCompile(`^[A-Za-z0-9]{3}$`)
	hexCurrencyPattern      = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
)

type AssetResolverConfig struct {
	IssuedDecimals map[string]uint8
}

type AssetResolver struct {
	issuedDecimals map[string]uint8
}

var _ blockchain.AssetResolver = (*AssetResolver)(nil)

func NewAssetResolver(cfg AssetResolverConfig) *AssetResolver {
	decimals := make(map[string]uint8, len(cfg.IssuedDecimals))
	for asset, value := range cfg.IssuedDecimals {
		decimals[asset] = value
	}
	return &AssetResolver{issuedDecimals: decimals}
}

func (r *AssetResolver) ValidateAssetAddress(_ context.Context, assetAddress string) error {
	if assetAddress == nativeAssetAddress {
		return nil
	}
	if _, _, err := parseIssuedAssetAddress(assetAddress); err != nil {
		return err
	}
	return nil
}

func (r *AssetResolver) AssetDecimals(ctx context.Context, assetAddress string) (uint8, error) {
	if err := r.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return 0, err
	}
	if assetAddress == nativeAssetAddress {
		return 6, nil
	}
	decimals, ok := r.issuedDecimals[assetAddress]
	if !ok {
		return 0, fmt.Errorf("xrpl: decimals unknown for issued asset %q", assetAddress)
	}
	return decimals, nil
}

func parseIssuedAssetAddress(assetAddress string) (currency, issuer string, err error) {
	if strings.TrimSpace(assetAddress) != assetAddress || assetAddress == "" {
		return "", "", fmt.Errorf("xrpl: invalid asset address %q", assetAddress)
	}
	if strings.Contains(assetAddress, ":") {
		return "", "", fmt.Errorf("xrpl: invalid asset address %q: expected CUR.rIssuer", assetAddress)
	}
	parts := strings.Split(assetAddress, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("xrpl: invalid asset address %q: expected CUR.rIssuer", assetAddress)
	}
	if strings.ContainsAny(parts[0], " \t\r\n.") {
		return "", "", fmt.Errorf("xrpl: invalid currency %q", parts[0])
	}
	if strings.EqualFold(parts[0], "XRP") || (!standardCurrencyPattern.MatchString(parts[0]) && !hexCurrencyPattern.MatchString(parts[0])) {
		return "", "", fmt.Errorf("xrpl: invalid currency %q: expected 3-character code or 20-byte hex code", parts[0])
	}
	if !addresscodec.IsValidClassicAddress(parts[1]) {
		return "", "", fmt.Errorf("xrpl: issuer %q is not a valid classic address", parts[1])
	}
	return parts[0], parts[1], nil
}

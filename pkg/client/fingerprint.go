package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var assetSymbolPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,47}$`)

// NetworkFingerprint binds the signing domain and asset catalogue, including
// conversion precision and backing assets. It matches the TypeScript client.
// Asset ordering and address case do not affect the fingerprint.
func NetworkFingerprint(info NetworkInfo) (string, error) {
	assets := append([]AssetInfo(nil), info.Assets...)
	seen := make(map[string]bool, len(assets))
	for i := range assets {
		asset := &assets[i]
		if !assetSymbolPattern.MatchString(asset.Symbol) || seen[asset.Symbol] || asset.Decimals > 77 {
			return "", fmt.Errorf("invalid gateway asset catalogue")
		}
		seen[asset.Symbol] = true
		asset.Address = strings.ToLower(asset.Address)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Symbol < assets[j].Symbol })
	identity := struct {
		NetworkID string      `json:"networkId"`
		ChainID   uint64      `json:"chainId"`
		Router    string      `json:"router"`
		Assets    []AssetInfo `json:"assets"`
	}{info.NetworkID, info.ChainID, strings.ToLower(info.VirtualRouter), assets}
	raw, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

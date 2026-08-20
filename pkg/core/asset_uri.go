package core

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const DefaultIssuer = "0x0000000000000000000000000000000000000000"

// AssetURI identifies an issuer-defined asset in protocol payloads.
//
// Core treats the asset id as opaque issuer-owned data. Chain-specific
// structure such as family, chain id, and token address belongs outside core.
type AssetURI string

type AssetURIParts struct {
	Network string
	Issuer  string
	AssetID string
}

// NewAssetURI builds a canonical asset URI:
//
//	yellow://ynet/asset/<issuer>/<asset-id>
//
// If issuer is empty, DefaultIssuer is used. The issuer must be a valid EVM
// address; it is normalized to its canonical 0x-prefixed hex form. The
// issuer-owned asset id is preserved as supplied.
func NewAssetURI(issuer, assetID string) (AssetURI, error) {
	if issuer == "" {
		issuer = DefaultIssuer
	}
	if !common.IsHexAddress(issuer) {
		return "", fmt.Errorf("asset URI issuer must be an EVM address: %q", issuer)
	}
	issuer = strings.ToLower(common.HexToAddress(issuer).Hex())
	uri := AssetURI(URIScheme + "://" + DefaultNetwork + "/asset/" + issuer + "/" + assetID)
	if err := ValidateAssetURI(uri); err != nil {
		return "", err
	}
	return uri, nil
}

// ParseAssetURI splits an asset URI into network, issuer, and issuer-owned
// asset id components.
func ParseAssetURI(uri AssetURI) (AssetURIParts, error) {
	if err := ValidateAssetURI(uri); err != nil {
		return AssetURIParts{}, err
	}
	network, _, path, err := ParseURI(string(uri))
	if err != nil {
		return AssetURIParts{}, err
	}
	parts := strings.SplitN(path, "/", 2)
	return AssetURIParts{
		Network: network,
		Issuer:  parts[0],
		AssetID: parts[1],
	}, nil
}

// ValidateAssetURI checks only core-level asset URI shape. It does not inspect
// issuer-specific asset id structure.
func ValidateAssetURI(uri AssetURI) error {
	raw := string(uri)
	if !IsURI(raw) {
		return fmt.Errorf("asset URI missing %s:// scheme: %q", URIScheme, raw)
	}
	network, kind, path, err := ParseURI(raw)
	if err != nil {
		return fmt.Errorf("asset URI parse: %w", err)
	}
	if network != DefaultNetwork {
		return fmt.Errorf("asset URI network %q unsupported (expected %q): %q", network, DefaultNetwork, raw)
	}
	if kind != "asset" {
		return fmt.Errorf("asset URI kind %q unsupported (expected asset): %q", kind, raw)
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("asset URI path must be issuer/asset-id: %q", raw)
	}
	if !isValidAssetIssuer(parts[0]) {
		return fmt.Errorf("asset URI issuer must be an EVM address: %q", parts[0])
	}
	if containsSpace(parts[1]) {
		return fmt.Errorf("asset URI asset id must not contain spaces: %q", parts[1])
	}
	return nil
}

func isValidAssetIssuer(s string) bool {
	return common.IsHexAddress(s)
}

// IssuerIDFromAssetURI returns the ConfigRegistry issuer id encoded in uri.
func IssuerIDFromAssetURI(uri AssetURI) (common.Address, error) {
	parts, err := ParseAssetURI(uri)
	if err != nil {
		return common.Address{}, err
	}
	if !common.IsHexAddress(parts.Issuer) {
		return common.Address{}, fmt.Errorf("asset URI issuer must be an EVM address: %q", parts.Issuer)
	}
	return common.HexToAddress(parts.Issuer), nil
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			return true
		}
	}
	return false
}

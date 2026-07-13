package blockchain

import (
	"fmt"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

func AssetIDFromURI(uri core.AssetURI) (AssetID, error) {
	parts, err := core.ParseAssetURI(uri)
	if err != nil {
		return AssetID{}, err
	}
	return ParseAssetID(parts.AssetID)
}

func AssetAddressForFamily(uri core.AssetURI, family ChainFamily) (string, error) {
	id, err := AssetIDFromURI(uri)
	if err != nil {
		return "", err
	}
	if id.Family != family {
		return "", fmt.Errorf("asset URI family %q does not match %q", id.Family, family)
	}
	return id.AssetAddress, nil
}

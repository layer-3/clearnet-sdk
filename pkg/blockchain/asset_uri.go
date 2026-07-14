package blockchain

import "github.com/layer-3/clearnet-sdk/pkg/core"

func AssetIDFromURI(uri core.AssetURI) (AssetID, error) {
	parts, err := core.ParseAssetURI(uri)
	if err != nil {
		return AssetID{}, err
	}
	return ParseAssetID(parts.AssetID)
}

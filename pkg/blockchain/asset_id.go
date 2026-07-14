package blockchain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

type ChainFamily string

const (
	ChainFamilyEVM  ChainFamily = "evm"
	ChainFamilySOL  ChainFamily = "sol"
	ChainFamilyXRPL ChainFamily = "xrpl"
	ChainFamilyBTC  ChainFamily = "btc"
)

type AssetID struct {
	Family       ChainFamily
	ChainID      uint64
	AssetAddress string
}

type AssetResolver interface {
	ValidateAssetAddress(ctx context.Context, assetAddress string) error
	AssetDecimals(ctx context.Context, assetAddress string) (uint8, error)
}

var ErrInvalidAssetDecimals = errors.New("asset amount cannot be represented in base units")

func ParseAssetID(s string) (AssetID, error) {
	parts := strings.SplitN(s, "/", 3)
	if len(parts) != 3 {
		return AssetID{}, fmt.Errorf("blockchain asset id %q must be family/chain-id/asset-address", s)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return AssetID{}, fmt.Errorf("blockchain asset id %q has empty component", s)
	}
	family := ChainFamily(strings.ToLower(parts[0]))
	switch family {
	case ChainFamilyEVM, ChainFamilySOL, ChainFamilyXRPL, ChainFamilyBTC:
	default:
		return AssetID{}, fmt.Errorf("unsupported blockchain asset family %q", parts[0])
	}
	chainID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return AssetID{}, fmt.Errorf("blockchain asset chain id %q: %w", parts[1], err)
	}
	if strings.ContainsAny(parts[2], " \t\r\n") {
		return AssetID{}, fmt.Errorf("blockchain asset address must not contain whitespace: %q", parts[2])
	}
	return AssetID{Family: family, ChainID: chainID, AssetAddress: parts[2]}, nil
}

func (id AssetID) String() string {
	return fmt.Sprintf("%s/%d/%s", id.Family, id.ChainID, id.AssetAddress)
}

func DecimalToBaseUnits(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	scaled := amount.Shift(int32(decimals))
	if !scaled.IsInteger() {
		return nil, fmt.Errorf("%w: amount %s cannot be represented with %d decimals", ErrInvalidAssetDecimals, amount.String(), decimals)
	}
	return scaled.BigInt(), nil
}

func BaseUnitsToDecimal(amount *big.Int, decimals uint8) decimal.Decimal {
	if amount == nil {
		amount = new(big.Int)
	}
	return decimal.NewFromBigInt(amount, -int32(decimals))
}

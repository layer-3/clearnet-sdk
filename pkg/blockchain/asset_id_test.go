package blockchain

import (
	"errors"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

func TestParseAssetID_NativeMarkerRequired(t *testing.T) {
	id, err := ParseAssetID("btc/0/0")
	if err != nil {
		t.Fatalf("ParseAssetID: %v", err)
	}
	if id.Family != ChainFamilyBTC || id.ChainID != 0 || id.AssetAddress != "0" {
		t.Fatalf("ParseAssetID = %+v", id)
	}

	if _, err := ParseAssetID("btc/0/"); err == nil {
		t.Fatal("empty asset address accepted")
	}
}

func TestDecimalBaseUnitConversions(t *testing.T) {
	base, err := DecimalToBaseUnits(decimal.NewFromBigInt(big.NewInt(123456789), -6), 6)
	if err != nil {
		t.Fatalf("DecimalToBaseUnits: %v", err)
	}
	if base.String() != "123456789" {
		t.Fatalf("base units = %s, want 123456789", base.String())
	}
	if got := BaseUnitsToDecimal(base, 6); !got.Equal(decimal.NewFromBigInt(big.NewInt(123456789), -6)) {
		t.Fatalf("BaseUnitsToDecimal = %s", got.String())
	}
	if _, err := DecimalToBaseUnits(decimal.NewFromBigInt(big.NewInt(1), -7), 6); err == nil {
		t.Fatal("fractional base unit amount accepted")
	} else if !errors.Is(err, ErrInvalidAssetDecimals) {
		t.Fatalf("DecimalToBaseUnits error = %v, want ErrInvalidAssetDecimals", err)
	}
}

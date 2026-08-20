package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var testIssuerAddress = common.HexToAddress("0x0000000000000000000000000000000000001234")

func TestNewAssetURI(t *testing.T) {
	got, err := NewAssetURI("", "evm/31337/0x0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("NewAssetURI: %v", err)
	}
	want := AssetURI("yellow://ynet/asset/" + DefaultIssuer + "/evm/31337/0x0000000000000000000000000000000000000000")
	if got != want {
		t.Fatalf("NewAssetURI = %q, want %q", got, want)
	}

	got, err = NewAssetURI(testIssuerAddress.Hex(), "CHAIN/Asset:ID")
	if err != nil {
		t.Fatalf("NewAssetURI mixed asset id: %v", err)
	}
	want = AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/CHAIN/Asset:ID")
	if got != want {
		t.Fatalf("NewAssetURI = %q, want %q", got, want)
	}
}

func TestParseAssetURI(t *testing.T) {
	parts, err := ParseAssetURI(AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/evm/31337/0xabc"))
	if err != nil {
		t.Fatalf("ParseAssetURI: %v", err)
	}
	if parts.Network != DefaultNetwork {
		t.Fatalf("Network = %q, want %q", parts.Network, DefaultNetwork)
	}
	if parts.Issuer != testIssuerAddress.Hex() {
		t.Fatalf("Issuer = %q, want %q", parts.Issuer, testIssuerAddress.Hex())
	}
	if parts.AssetID != "evm/31337/0xabc" {
		t.Fatalf("AssetID = %q", parts.AssetID)
	}
}

func TestValidateAssetURI(t *testing.T) {
	valid := []AssetURI{
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/evm/31337/0xabc"),
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/opaque:asset.id/with/slashes"),
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/UPPERCASE-ASSET-ID"),
	}
	for _, uri := range valid {
		if err := ValidateAssetURI(uri); err != nil {
			t.Fatalf("ValidateAssetURI(%q): %v", uri, err)
		}
	}

	invalid := []AssetURI{
		"",
		"yellow://ynet/user/0xabc",
		AssetURI("yellow://other/asset/" + testIssuerAddress.Hex() + "/evm/1/0xabc"),
		"yellow://ynet/asset//evm/1/0xabc",
		"yellow://ynet/asset/Custody/evm/1/0xabc",
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex()),
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/"),
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/asset id"),
		AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/asset\tid"),
	}
	for _, uri := range invalid {
		if err := ValidateAssetURI(uri); err == nil {
			t.Fatalf("ValidateAssetURI(%q) succeeded, want error", uri)
		}
	}
}

func TestValidateURIRejectsAssetURI(t *testing.T) {
	if err := ValidateURI(string(AssetURI("yellow://ynet/asset/" + testIssuerAddress.Hex() + "/evm/31337/0xabc"))); err == nil {
		t.Fatal("ValidateURI accepted asset URI as account URI")
	}
}

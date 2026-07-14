package core

import "testing"

func TestNewAssetURI(t *testing.T) {
	got, err := NewAssetURI("", "evm/31337/0x0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("NewAssetURI: %v", err)
	}
	want := AssetURI("yellow://ynet/asset/custody/evm/31337/0x0000000000000000000000000000000000000000")
	if got != want {
		t.Fatalf("NewAssetURI = %q, want %q", got, want)
	}

	got, err = NewAssetURI("Custody-Test", "CHAIN/Asset:ID")
	if err != nil {
		t.Fatalf("NewAssetURI mixed asset id: %v", err)
	}
	want = AssetURI("yellow://ynet/asset/custody-test/CHAIN/Asset:ID")
	if got != want {
		t.Fatalf("NewAssetURI = %q, want %q", got, want)
	}
}

func TestParseAssetURI(t *testing.T) {
	parts, err := ParseAssetURI("yellow://ynet/asset/custody/evm/31337/0xabc")
	if err != nil {
		t.Fatalf("ParseAssetURI: %v", err)
	}
	if parts.Network != DefaultNetwork {
		t.Fatalf("Network = %q, want %q", parts.Network, DefaultNetwork)
	}
	if parts.Issuer != DefaultIssuer {
		t.Fatalf("Issuer = %q, want %q", parts.Issuer, DefaultIssuer)
	}
	if parts.AssetID != "evm/31337/0xabc" {
		t.Fatalf("AssetID = %q", parts.AssetID)
	}
}

func TestValidateAssetURI(t *testing.T) {
	valid := []AssetURI{
		"yellow://ynet/asset/custody/evm/31337/0xabc",
		"yellow://ynet/asset/issuer-1/opaque:asset.id/with/slashes",
		"yellow://ynet/asset/custody/UPPERCASE-ASSET-ID",
	}
	for _, uri := range valid {
		if err := ValidateAssetURI(uri); err != nil {
			t.Fatalf("ValidateAssetURI(%q): %v", uri, err)
		}
	}

	invalid := []AssetURI{
		"",
		"yellow://ynet/user/0xabc",
		"yellow://other/asset/custody/evm/1/0xabc",
		"yellow://ynet/asset//evm/1/0xabc",
		"yellow://ynet/asset/Custody/evm/1/0xabc",
		"yellow://ynet/asset/custody",
		"yellow://ynet/asset/custody/",
		"yellow://ynet/asset/custody/asset id",
		"yellow://ynet/asset/custody/asset\tid",
	}
	for _, uri := range invalid {
		if err := ValidateAssetURI(uri); err == nil {
			t.Fatalf("ValidateAssetURI(%q) succeeded, want error", uri)
		}
	}
}

func TestValidateURIRejectsAssetURI(t *testing.T) {
	if err := ValidateURI("yellow://ynet/asset/custody/evm/31337/0xabc"); err == nil {
		t.Fatal("ValidateURI accepted asset URI as account URI")
	}
}

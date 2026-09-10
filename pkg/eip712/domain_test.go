package eip712

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func TestComputeDomainSeparatorForMatchesTypedData(t *testing.T) {
	for _, name := range []string{Name, "YellowCustody", "YellowConfigRegistry", ""} {
		for _, version := range []string{"1", "2"} {
			for _, chain := range []*big.Int{big.NewInt(0), big.NewInt(31337), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))} {
				td := apitypes.TypedData{Types: apitypes.Types{"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}}}, Domain: apitypes.TypedDataDomain{Name: name, Version: version, ChainId: (*math.HexOrDecimal256)(chain), VerifyingContract: RouterAddr.Hex()}}
				// Domain.Map omits empty strings; supply all declared fields explicitly.
				expected, err := td.HashStruct("EIP712Domain", apitypes.TypedDataMessage{"name": name, "version": version, "chainId": chain, "verifyingContract": RouterAddr.Hex()})
				if err != nil {
					t.Fatal(err)
				}
				got := ComputeDomainSeparatorFor(name, version, chain, RouterAddr)
				if common.Hash(got) != common.BytesToHash(expected) {
					t.Fatal("domain mismatch", name, version, chain)
				}
			}
		}
	}
	if ComputeDomainSeparator(big.NewInt(31337)) != ComputeDomainSeparatorFor(Name, Version, big.NewInt(31337), RouterAddr) {
		t.Fatal("existing domain changed")
	}
}

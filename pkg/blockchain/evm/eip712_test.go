package evm

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"math/big"
	"os"
	"strings"
	"testing"
)

type typedVector struct {
	TypedData apitypes.TypedData `json:"typedData"`
	Domain    string             `json:"domainSeparator"`
	Struct    string             `json:"structHash"`
	Digest    string             `json:"digest"`
	Signature string             `json:"signature"`
	Signer    string             `json:"signer"`
}

func readTypedVectors(t *testing.T) []typedVector {
	t.Helper()
	b, err := os.ReadFile("testdata/eip712.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []typedVector
	if err = json.Unmarshal(b, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}
func vectorDigest(td apitypes.TypedData) common.Hash {
	m := td.Message
	addr := func(k string) common.Address { return common.HexToAddress(m[k].(string)) }
	num := func(k string) *big.Int {
		n, ok := new(big.Int).SetString(m[k].(string), 0)
		if !ok {
			panic(k)
		}
		return n
	}
	hash := func(k string) common.Hash { return common.HexToHash(m[k].(string)) }
	keys := func(k string) []common.Address {
		var out []common.Address
		for _, v := range m[k].([]interface{}) {
			out = append(out, common.HexToAddress(v.(string)))
		}
		return out
	}
	chain := (*big.Int)(td.Domain.ChainId).Uint64()
	contract := common.HexToAddress(td.Domain.VerifyingContract)
	switch td.PrimaryType {
	case "Execute":
		return ComputeWithdrawalDigest(chain, contract, addr("to"), addr("asset"), num("amount"), hash("withdrawalId"), num("deadline"))
	case "UpdateSigners":
		return ComputeRotationDigest(chain, contract, keys("newSigners"), num("newThreshold"), num("signerNonce"))
	case "RegisterIssuer":
		return ComputeConfigRegistryRegistrationDigest(chain, contract, keys("issuerKeys"), num("threshold"))
	case "SetConfig":
		return ComputeConfigRegistrySetConfigDigest(chain, contract, addr("issuerId"), hash("key"), hash("checksum"), num("expectedNonce"))
	case "SetConfigWithData":
		return ComputeConfigRegistrySetConfigWithDataDigest(chain, contract, addr("issuerId"), hash("key"), hexutil.MustDecode(m["data"].(string)), num("expectedNonce"))
	case "UpdateIssuerSettings":
		return ComputeConfigRegistryUpdateIssuerSettingsDigest(chain, contract, addr("issuerId"), keys("newIssuerKeys"), num("newThreshold"), num("expectedNonce"))
	default:
		panic(td.PrimaryType)
	}
}
func assertReference(t *testing.T, td apitypes.TypedData) common.Hash {
	t.Helper()
	ref, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		t.Fatal(err)
	}
	got := vectorDigest(td)
	if got != common.BytesToHash(ref) {
		t.Fatalf("%s: implementation %s != independent encoder %x", td.PrimaryType, got, ref)
	}
	return got
}
func TestEIP712CanonicalVectors(t *testing.T) {
	for _, v := range readTypedVectors(t) {
		t.Run(v.TypedData.PrimaryType, func(t *testing.T) {
			td := v.TypedData
			got := assertReference(t, td)
			if got.Hex() != v.Digest {
				t.Fatal("golden digest drift")
			}
			domain := domainSeparator(td.Domain.Name, (*big.Int)(td.Domain.ChainId).Uint64(), common.HexToAddress(td.Domain.VerifyingContract))
			if domain.Hex() != v.Domain {
				t.Fatal("domain drift")
			}
			structure, err := td.HashStruct(td.PrimaryType, td.Message)
			if err != nil {
				t.Fatal(err)
			}
			if hexutil.Encode(structure) != v.Struct {
				t.Fatal("struct drift")
			}
			pub, err := crypto.SigToPub(got[:], hexutil.MustDecode(v.Signature))
			if err != nil {
				t.Fatal(err)
			}
			if crypto.PubkeyToAddress(*pub).Hex() != v.Signer {
				t.Fatal("signature drift")
			}
		})
	}
}
func TestEIP712EveryFieldAndArrayEncoding(t *testing.T) {
	for _, v := range readTypedVectors(t) {
		t.Run(v.TypedData.PrimaryType, func(t *testing.T) {
			original := assertReference(t, v.TypedData)
			for _, field := range v.TypedData.Types[v.TypedData.PrimaryType] {
				b, _ := json.Marshal(v.TypedData)
				var td apitypes.TypedData
				_ = json.Unmarshal(b, &td)
				var variants []interface{}
				switch field.Type {
				case "uint256":
					n, _ := new(big.Int).SetString(td.Message[field.Name].(string), 0)
					variants = []interface{}{new(big.Int).Add(n, big.NewInt(1)).String(), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()}
				case "address":
					variants = []interface{}{common.HexToAddress("0x999").Hex()}
				case "bytes32":
					variants = []interface{}{common.HexToHash("0x999").Hex()}
				case "bytes":
					variants = []interface{}{"0x", "0x00", "0x01020300"}
				case "address[]":
					a := td.Message[field.Name].([]interface{})
					variants = []interface{}{[]interface{}{}, a[:2], []interface{}{a[1], a[0], a[2]}, append(append([]interface{}{}, a...), common.HexToAddress("0x4").Hex())}
				}
				for i, value := range variants {
					td.Message[field.Name] = value
					t.Run(field.Name+string(rune('a'+i)), func(t *testing.T) {
						if assertReference(t, td) == original {
							t.Fatal("field not bound")
						}
					})
				}
			}
		})
	}
}

// Public helpers keep their typed API and identify bad caller values precisely.
func TestEIP712InvalidUint256NamesField(t *testing.T) {
	address := common.HexToAddress("0x1")
	keys := []common.Address{address}
	good := big.NewInt(1)
	cases := []struct {
		field  string
		digest func(*big.Int) common.Hash
	}{
		{"amount", func(n *big.Int) common.Hash {
			return ComputeWithdrawalDigest(1, address, address, address, n, [32]byte{}, good)
		}},
		{"deadline", func(n *big.Int) common.Hash {
			return ComputeWithdrawalDigest(1, address, address, address, good, [32]byte{}, n)
		}},
		{"newThreshold", func(n *big.Int) common.Hash { return ComputeRotationDigest(1, address, keys, n, good) }},
		{"signerNonce", func(n *big.Int) common.Hash { return ComputeRotationDigest(1, address, keys, good, n) }},
		{"threshold", func(n *big.Int) common.Hash { return ComputeConfigRegistryRegistrationDigest(1, address, keys, n) }},
		{"expectedNonce", func(n *big.Int) common.Hash {
			return ComputeConfigRegistrySetConfigDigest(1, address, address, [32]byte{}, [32]byte{}, n)
		}},
		{"expectedNonce", func(n *big.Int) common.Hash {
			return ComputeConfigRegistrySetConfigWithDataDigest(1, address, address, [32]byte{}, nil, n)
		}},
		{"newThreshold", func(n *big.Int) common.Hash {
			return ComputeConfigRegistryUpdateIssuerSettingsDigest(1, address, address, keys, n, good)
		}},
		{"expectedNonce", func(n *big.Int) common.Hash {
			return ComputeConfigRegistryUpdateIssuerSettingsDigest(1, address, address, keys, good, n)
		}},
	}
	for _, tc := range cases {
		for _, invalid := range []*big.Int{nil, big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
			t.Run(tc.field, func(t *testing.T) {
				defer func() {
					r := recover()
					if r == nil || !strings.Contains(fmt.Sprint(r), "uint256 "+tc.field+":") {
						t.Fatalf("panic = %v; want field %s", r, tc.field)
					}
				}()
				tc.digest(invalid)
			})
		}
	}
}

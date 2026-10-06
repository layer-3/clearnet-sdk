package evm

import (
	"context"
	"errors"
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/layer-3/clearnet-sdk/pkg/bls"
)

// Only eth_chainId is exposed: invalid proof inputs must fail without any
// approval, gas estimation, or broadcast RPCs.
type registrationRPC struct{}

func (registrationRPC) ChainId() hexutil.Big { return hexutil.Big(*big.NewInt(31337)) }

func TestRegistryLockRejectsMalformedProofBeforeApproval(t *testing.T) {
	srv := rpc.NewServer()
	if err := srv.RegisterName("eth", registrationRPC{}); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(srv)
	defer httpServer.Close()
	client, err := ethclient.Dial(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewRegistryAdapter(context.Background(), client, common.HexToAddress("0x1234"), common.HexToAddress("0x5678"), key)
	if err != nil {
		t.Fatal(err)
	}
	kp := bls.KeyPairFromSeed([]byte("registration"))
	for _, proof := range [][2]*big.Int{{}, {new(big.Int), new(big.Int)}, {big.NewInt(-1), big.NewInt(1)}} {
		if _, err := adapter.Lock(context.Background(), bls.G1ToCoords(kp.PublicG1), bls.G2ToCoords(kp.PublicG2), proof, nil); !errors.Is(err, ErrInvalidRegistryProof) {
			t.Fatalf("proof was not rejected locally: %v", err)
		}
	}
}

func TestRegistryProofValidation(t *testing.T) {
	kp := bls.KeyPairFromSeed([]byte("registration"))
	other := bls.KeyPairFromSeed([]byte("other registration"))
	chain := big.NewInt(31337)
	registry, operator := common.HexToAddress("0x1234"), common.HexToAddress("0x5678")
	g1, g2 := bls.G1ToCoords(kp.PublicG1), bls.G2ToCoords(kp.PublicG2)
	digest, err := bls.RegistryProofOfPossessionDigest(chain, registry, operator, g1, g2)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := bls.Sign(&kp.Secret, digest)
	if err != nil {
		t.Fatal(err)
	}
	proof := bls.G1ToCoords(sig)
	if err := validateRegistryProof(chain, registry, operator, g1, g2, proof); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		chain              *big.Int
		registry, operator common.Address
		g1                 [2]*big.Int
		g2                 [4]*big.Int
	}{
		{big.NewInt(31338), registry, operator, g1, g2},
		{chain, common.HexToAddress("0xabcd"), operator, g1, g2},
		{chain, registry, common.HexToAddress("0xabcd"), g1, g2},
		{chain, registry, operator, bls.G1ToCoords(other.PublicG1), g2},
		{chain, registry, operator, g1, bls.G2ToCoords(other.PublicG2)},
	} {
		if err := validateRegistryProof(tc.chain, tc.registry, tc.operator, tc.g1, tc.g2, proof); !errors.Is(err, ErrInvalidRegistryProof) {
			t.Fatalf("invalid proof binding accepted: %v", err)
		}
	}
	// Even a correctly signed digest cannot bind an inconsistent G1 key.
	g1 = bls.G1ToCoords(other.PublicG1)
	digest, _ = bls.RegistryProofOfPossessionDigest(chain, registry, operator, g1, g2)
	sig, _ = bls.Sign(&kp.Secret, digest)
	if err := validateRegistryProof(chain, registry, operator, g1, g2, bls.G1ToCoords(sig)); !errors.Is(err, ErrInvalidRegistryProof) {
		t.Fatalf("inconsistent key accepted: %v", err)
	}
}

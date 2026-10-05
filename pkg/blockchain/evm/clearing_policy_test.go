package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type policyRPC struct {
	registry, config            common.Address
	code                        []byte
	head                        uint64
	getter                      []byte
	getterErr, headErr, codeErr error
	owner                       []byte
	epoch                       []byte
	checksum                    []byte
	callErr                     error
	blocks                      []uint64
}

func (r *policyRPC) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	return &types.Header{Number: new(big.Int).SetUint64(r.head)}, r.headErr
}
func (r *policyRPC) CodeAt(_ context.Context, _ common.Address, block *big.Int) ([]byte, error) {
	r.blocks = append(r.blocks, block.Uint64())
	return r.code, r.codeErr
}
func (r *policyRPC) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	r.blocks = append(r.blocks, block.Uint64())
	if *msg.To == r.registry {
		return r.getter, r.getterErr
	}
	if *msg.To != r.config {
		return nil, errors.New("wrong config address")
	}
	if r.callErr != nil {
		return nil, r.callErr
	}
	switch common.Bytes2Hex(msg.Data[:4]) {
	case common.Bytes2Hex(crypto.Keccak256([]byte("owner()"))[:4]):
		return r.owner, nil
	case common.Bytes2Hex(crypto.Keccak256([]byte("configEpoch(bytes32)"))[:4]):
		if common.BytesToHash(msg.Data[4:]) != ClearingFinalityPolicyKey {
			return nil, errors.New("wrong key")
		}
		return r.epoch, nil
	case common.Bytes2Hex(crypto.Keccak256([]byte("latestConfigChecksum(bytes32)"))[:4]):
		return r.checksum, nil
	}
	return nil, errors.New("unexpected call")
}

type policyRevert struct{ message string }

func (e policyRevert) Error() string {
	if e.message != "" {
		return e.message
	}
	return "execution reverted"
}
func (policyRevert) ErrorData() interface{} { return "0x" }

func newPolicyRPC(k uint64) *policyRPC {
	r := &policyRPC{registry: common.HexToAddress("0x1234"), config: common.HexToAddress("0x5678"), head: 100}
	r.code = append([]byte{0x63}, crypto.Keccak256([]byte("CONFIG()"))[:4]...)
	r.getter = common.LeftPadBytes(r.config.Bytes(), 32)
	r.owner = common.LeftPadBytes(r.registry.Bytes(), 32)
	r.epoch = common.LeftPadBytes([]byte{1}, 32)
	r.checksum = ClearingFinalityPolicyChecksum(k).Bytes()
	return r
}

func TestRegistryClearingPolicyConfirmedAndIndependent(t *testing.T) {
	for _, k := range []uint64{1, 5, 256} {
		r := newPolicyRPC(k)
		p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 1)
		if err != nil || p.Legacy || p.SigningClusterSize != k || p.ConfigAddress != r.config || p.ConfirmedBlock != 93 {
			t.Fatalf("policy=%+v err=%v", p, err)
		}
		for _, block := range r.blocks {
			if block != 93 {
				t.Fatalf("unconfirmed read at %d", block)
			}
		}
		// Re-read after restart: no YAML default or event replay is needed.
		again, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 5)
		if err != nil || again != p {
			t.Fatalf("restart policy=%+v err=%v", again, err)
		}
	}
}

func TestRegistryClearingPolicyLegacyCompatibility(t *testing.T) {
	for _, k := range []uint64{1, 5} {
		for _, reverted := range []bool{false, true} {
			r := newPolicyRPC(k)
			r.code, r.getter = []byte{0x60, 0x00}, nil
			if reverted {
				r.getterErr = policyRevert{}
			}
			p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, k)
			if err != nil || !p.Legacy || p.SigningClusterSize != k || p.ConfigAddress != (common.Address{}) {
				t.Fatalf("policy=%+v err=%v", p, err)
			}
		}
	}
}

func TestRegistryClearingPolicyFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*policyRPC)
	}{
		{"rpc loss", func(r *policyRPC) { r.headErr = errors.New("offline") }},
		{"code RPC loss", func(r *policyRPC) { r.codeErr = errors.New("offline") }},
		{"undeployed", func(r *policyRPC) { r.code = nil }},
		{"no confirmed state", func(r *policyRPC) { r.head = 1 }},
		{"legacy RPC loss", func(r *policyRPC) { r.code = []byte{0}; r.getterErr = errors.New("offline"); r.getter = nil }},
		{"RPC error with empty data is not a revert", func(r *policyRPC) {
			r.code = []byte{0}
			r.getterErr = policyRevert{message: "RPC unavailable"}
			r.getter = nil
		}},
		{"broken advertised getter", func(r *policyRPC) { r.getterErr = policyRevert{} }},
		{"missing advertised getter result", func(r *policyRPC) { r.getter = nil }},
		{"zero config", func(r *policyRPC) { r.getter = make([]byte, 32) }},
		{"malformed config", func(r *policyRPC) { r.getter[0] = 1 }},
		{"missing owner", func(r *policyRPC) { r.owner = nil }},
		{"wrong owner", func(r *policyRPC) { r.owner = r.getter }},
		{"missing epoch", func(r *policyRPC) { r.epoch = make([]byte, 32) }},
		{"changed epoch", func(r *policyRPC) { r.epoch[31] = 2 }},
		{"invalid checksum", func(r *policyRPC) { r.checksum[0] ^= 1 }},
		{"zero K", func(r *policyRPC) { r.checksum = ClearingFinalityPolicyChecksum(0).Bytes() }},
		{"oversized K", func(r *policyRPC) { r.checksum = ClearingFinalityPolicyChecksum(257).Bytes() }},
		{"Config RPC loss", func(r *policyRPC) { r.callErr = errors.New("offline") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newPolicyRPC(5)
			tc.mutate(r)
			if p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 1); err == nil {
				t.Fatalf("accepted invalid policy: %+v", p)
			}
		})
	}
}

func TestClearingPolicyChecksumMatchesSolidityABI(t *testing.T) {
	typ, err := abi.NewType("uint64", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []uint64{1, 5, 256} {
		payload, err := (abi.Arguments{{Type: typ}}).Pack(k)
		if err != nil {
			t.Fatal(err)
		}
		if ClearingFinalityPolicyChecksum(k) != crypto.Keccak256Hash(payload) {
			t.Fatal("Solidity checksum mismatch")
		}
	}
}

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
	blocks                      []common.Hash
	headerOverride              *types.Header
	nilHeader                   bool
	reorg                       bool
	headerReads                 int
}

func (r *policyRPC) HeaderByNumber(_ context.Context, number *big.Int) (*types.Header, error) {
	r.headerReads++
	if r.nilHeader || r.headerOverride != nil {
		return r.headerOverride, r.headErr
	}
	if number == nil {
		number = new(big.Int).SetUint64(r.head)
	}
	h := &types.Header{Number: new(big.Int).Set(number)}
	if r.reorg && r.headerReads > 2 {
		h.Extra = []byte{1}
	}
	return h, r.headErr
}
func (r *policyRPC) CodeAtHash(_ context.Context, _ common.Address, block common.Hash) ([]byte, error) {
	r.blocks = append(r.blocks, block)
	return r.code, r.codeErr
}
func (r *policyRPC) CallContractAtHash(_ context.Context, msg ethereum.CallMsg, block common.Hash) ([]byte, error) {
	r.blocks = append(r.blocks, block)
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
		if common.BytesToHash(msg.Data[4:]) != RegistryFinalityPolicyKey {
			return nil, errors.New("wrong key")
		}
		return r.epoch, nil
	case common.Bytes2Hex(crypto.Keccak256([]byte("latestConfigChecksum(bytes32)"))[:4]):
		return r.checksum, nil
	}
	return nil, errors.New("unexpected call")
}

type policyRevert struct{ message, data string }

func (e policyRevert) Error() string {
	if e.message != "" {
		return e.message
	}
	return "execution reverted"
}
func (e policyRevert) ErrorData() interface{} {
	if e.data != "" {
		return e.data
	}
	return "0x"
}

func newPolicyRPC(k uint64) *policyRPC {
	r := &policyRPC{registry: common.HexToAddress("0x1234"), config: common.HexToAddress("0x5678"), head: 100}
	r.code = append([]byte{0x63}, crypto.Keccak256([]byte("CONFIG()"))[:4]...)
	r.getter = common.LeftPadBytes(r.config.Bytes(), 32)
	r.owner = common.LeftPadBytes(r.registry.Bytes(), 32)
	r.epoch = common.LeftPadBytes([]byte{1}, 32)
	r.checksum = RegistryFinalityPolicyChecksum(k).Bytes()
	return r
}

func TestRegistryFinalityPolicyConfirmedAndIndependent(t *testing.T) {
	for _, k := range []uint64{1, 5, 256} {
		r := newPolicyRPC(k)
		p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 1)
		if err != nil || p.Legacy || p.SigningClusterSize != k || p.ConfigAddress != r.config || p.ConfirmedBlock != 93 {
			t.Fatalf("policy=%+v err=%v", p, err)
		}
		for _, block := range r.blocks {
			if block != (&types.Header{Number: big.NewInt(93)}).Hash() {
				t.Fatalf("mixed fork read at %s", block)
			}
		}
		// Re-read after restart: no YAML default or event replay is needed.
		again, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 5)
		if err != nil || again != p {
			t.Fatalf("restart policy=%+v err=%v", again, err)
		}
	}
}

func TestRegistryFinalityPolicyLegacyCompatibility(t *testing.T) {
	for _, k := range []uint64{1, 5} {
		for _, pushData := range []byte{0, 0xf4} {
			r := newPolicyRPC(k)
			r.code, r.getter = []byte{0x60, pushData, 0x60, 0, 0xfd}, nil
			r.getterErr = policyRevert{}
			p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, k)
			if err != nil || !p.Legacy || p.SigningClusterSize != k || p.ConfigAddress != (common.Address{}) {
				t.Fatalf("policy=%+v err=%v", p, err)
			}
		}
	}
}

func TestRegistryFinalityPolicyFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*policyRPC)
	}{
		{"nil header", func(r *policyRPC) { r.nilHeader = true }},
		{"nil header number", func(r *policyRPC) { r.headerOverride = &types.Header{} }},
		{"oversized header number", func(r *policyRPC) { r.headerOverride = &types.Header{Number: new(big.Int).Lsh(big.NewInt(1), 65)} }},
		{"negative header number", func(r *policyRPC) { r.headerOverride = &types.Header{Number: big.NewInt(-1)} }},
		{"anchor reorg", func(r *policyRPC) { r.reorg = true }},
		{"empty success", func(r *policyRPC) { r.code = []byte{0}; r.getter = nil }},
		{"proxy without implementation", func(r *policyRPC) { r.code = []byte{0xf4}; r.getter = nil }},
		{"reverting proxy implementation", func(r *policyRPC) { r.code = []byte{0xf4}; r.getter = nil; r.getterErr = policyRevert{} }},
		{"callcode proxy", func(r *policyRPC) { r.code = []byte{0xf2}; r.getter = nil; r.getterErr = policyRevert{} }},
		{"nonempty revert", func(r *policyRPC) { r.code = []byte{0}; r.getter = nil; r.getterErr = policyRevert{data: "0x1234"} }},
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
		{"zero K", func(r *policyRPC) { r.checksum = RegistryFinalityPolicyChecksum(0).Bytes() }},
		{"oversized K", func(r *policyRPC) { r.checksum = RegistryFinalityPolicyChecksum(257).Bytes() }},
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

func TestRegistryFinalityPolicyInputs(t *testing.T) {
	for _, tc := range []struct {
		registry common.Address
		k        uint64
	}{
		{common.Address{}, 1}, {common.HexToAddress("0x1234"), 0}, {common.HexToAddress("0x1234"), 257},
	} {
		r := newPolicyRPC(5)
		if _, err := ReadRegistryFinalityPolicy(context.Background(), r, tc.registry, 7, tc.k); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	for _, confirmations := range []uint64{0, 100} {
		r := newPolicyRPC(5)
		if p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, confirmations, 1); err != nil || p.ConfirmedBlock != 100-confirmations {
			t.Fatalf("valid boundary policy=%+v error=%v", p, err)
		}
	}
}

func TestRegistryFinalityPolicyAdvertisedThroughProxy(t *testing.T) {
	r := newPolicyRPC(5)
	r.code = []byte{0xf4}
	if p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7, 1); err != nil || p.Legacy || p.SigningClusterSize != 5 {
		t.Fatalf("valid advertised policy=%+v error=%v", p, err)
	}
}

func TestRegistryFinalityPolicyChecksumMatchesSolidityABI(t *testing.T) {
	typ, err := abi.NewType("uint64", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []uint64{1, 5, 256} {
		payload, err := (abi.Arguments{{Type: typ}}).Pack(k)
		if err != nil {
			t.Fatal(err)
		}
		if RegistryFinalityPolicyChecksum(k) != crypto.Keccak256Hash(payload) {
			t.Fatal("Solidity checksum mismatch")
		}
	}
}

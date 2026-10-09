package evm

import (
	"context"
	"errors"
	"math/big"
	"strings"
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

func newPolicyRPC(k uint64) *policyRPC {
	r := &policyRPC{registry: common.HexToAddress("0x1234"), config: common.HexToAddress("0x5678"), head: 100}
	r.code = []byte{0} // Only deployment presence matters; no selector inspection.
	r.getter = common.LeftPadBytes(r.config.Bytes(), 32)
	r.owner = common.LeftPadBytes(r.registry.Bytes(), 32)
	r.epoch = common.LeftPadBytes([]byte{1}, 32)
	r.checksum = RegistryFinalityPolicyChecksum(k).Bytes()
	return r
}

func TestRegistryFinalityPolicyConfirmedAndIndependent(t *testing.T) {
	for _, k := range []uint64{1, 2, 5, 7, 64, 256} {
		r := newPolicyRPC(k)
		p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7)
		if err != nil || p.SigningClusterSize != k || p.ConfigAddress != r.config || p.ConfirmedBlock != 93 {
			t.Fatalf("policy=%+v err=%v", p, err)
		}
		for _, block := range r.blocks {
			if block != (&types.Header{Number: big.NewInt(93)}).Hash() {
				t.Fatalf("mixed fork read at %s", block)
			}
		}
		// Re-read after restart: no YAML default or event replay is needed.
		again, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7)
		if err != nil || again != p {
			t.Fatalf("restart policy=%+v err=%v", again, err)
		}
	}
}

func TestRegistryFinalityPolicyRejectsMissingConfig(t *testing.T) {
	r := newPolicyRPC(1)
	r.getter = nil
	r.getterErr = errors.New("execution reverted")
	p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7)
	if !errors.Is(err, r.getterErr) || !strings.Contains(err.Error(), "Registry must implement IClearnetRegistryConfig") || p.SigningClusterSize != 0 || p.ConfigAddress != (common.Address{}) {
		t.Fatalf("missing CONFIG() policy=%+v err=%v", p, err)
	}
}

func TestRegistryFinalityPolicyFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*policyRPC)
		want   string
	}{
		{"nil header", func(r *policyRPC) { r.nilHeader = true }, "confirmed anchor state unavailable"},
		{"nil header number", func(r *policyRPC) { r.headerOverride = &types.Header{} }, "confirmed anchor state unavailable"},
		{"oversized header number", func(r *policyRPC) { r.headerOverride = &types.Header{Number: new(big.Int).Lsh(big.NewInt(1), 65)} }, "confirmed anchor state unavailable"},
		{"negative header number", func(r *policyRPC) { r.headerOverride = &types.Header{Number: big.NewInt(-1)} }, "confirmed anchor state unavailable"},
		{"anchor reorg", func(r *policyRPC) { r.reorg = true }, "anchor changed during policy read"},
		{"empty success", func(r *policyRPC) { r.getter = nil }, "CONFIG: invalid address encoding"},
		{"head RPC loss", func(r *policyRPC) { r.headErr = errors.New("offline") }, "anchor head: offline"},
		{"code RPC loss", func(r *policyRPC) { r.codeErr = errors.New("offline") }, "registry code: offline"},
		{"undeployed", func(r *policyRPC) { r.code = nil }, "registry not deployed at confirmed block"},
		{"no confirmed state", func(r *policyRPC) { r.head = 1 }, "confirmed anchor state unavailable"},
		{"CONFIG() call fails", func(r *policyRPC) { r.getterErr = errors.New("offline") }, "Registry must implement IClearnetRegistryConfig): offline"},
		{"zero config", func(r *policyRPC) { r.getter = make([]byte, 32) }, "CONFIG: invalid address encoding"},
		{"malformed config", func(r *policyRPC) { r.getter[0] = 1 }, "CONFIG: invalid address encoding"},
		{"missing owner", func(r *policyRPC) { r.owner = nil }, "malformed owner"},
		{"wrong owner", func(r *policyRPC) { r.owner = r.getter }, "Config is not owned by Registry"},
		{"missing epoch", func(r *policyRPC) { r.epoch = make([]byte, 32) }, "expected one frozen finality epoch"},
		{"changed epoch", func(r *policyRPC) { r.epoch[31] = 2 }, "expected one frozen finality epoch"},
		{"invalid checksum", func(r *policyRPC) { r.checksum[0] ^= 1 }, "unknown finality checksum"},
		{"zero K", func(r *policyRPC) { r.checksum = RegistryFinalityPolicyChecksum(0).Bytes() }, "unknown finality checksum"},
		{"oversized K", func(r *policyRPC) { r.checksum = RegistryFinalityPolicyChecksum(257).Bytes() }, "unknown finality checksum"},
		{"Config RPC loss", func(r *policyRPC) { r.callErr = errors.New("offline") }, "owner: offline"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newPolicyRPC(5)
			tc.mutate(r)
			if p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, 7); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("policy=%+v err=%v, want %q", p, err, tc.want)
			}
		})
	}
}

func TestRegistryFinalityPolicyInputs(t *testing.T) {
	r := newPolicyRPC(5)
	if _, err := ReadRegistryFinalityPolicy(context.Background(), r, common.Address{}, 7); err == nil {
		t.Fatal("zero registry address accepted")
	}
	for _, confirmations := range []uint64{0, 100} {
		r := newPolicyRPC(5)
		if p, err := ReadRegistryFinalityPolicy(context.Background(), r, r.registry, confirmations); err != nil || p.ConfirmedBlock != 100-confirmations {
			t.Fatalf("valid boundary policy=%+v error=%v", p, err)
		}
	}
}

func TestRegistryFinalityPolicyChecksumMatchesSolidityABI(t *testing.T) {
	typ, err := abi.NewType("uint64", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []uint64{1, 2, 5, 7, 64, 256} {
		payload, err := (abi.Arguments{{Type: typ}}).Pack(k)
		if err != nil {
			t.Fatal(err)
		}
		if RegistryFinalityPolicyChecksum(k) != crypto.Keccak256Hash(payload) {
			t.Fatal("Solidity checksum mismatch")
		}
	}
}

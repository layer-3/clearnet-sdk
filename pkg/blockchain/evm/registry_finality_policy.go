package evm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// RegistryFinalityPolicyKey identifies the Registry's fixed signing-cluster policy.
// Its v1 payload is abi.encode(uint64(K)).
var RegistryFinalityPolicyKey = crypto.Keccak256Hash([]byte("CLEARNET_FINALITY_POLICY_V1"))

type RegistryPolicyReader interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	CodeAtHash(context.Context, common.Address, common.Hash) ([]byte, error)
	CallContractAtHash(context.Context, ethereum.CallMsg, common.Hash) ([]byte, error)
}

type RegistryFinalityPolicy struct {
	SigningClusterSize uint64
	ConfigAddress      common.Address
	ConfirmedBlock     uint64
	ConfirmedHash      common.Hash
}

func RegistryFinalityPolicyChecksum(k uint64) common.Hash {
	var payload [32]byte
	binary.BigEndian.PutUint64(payload[24:], k)
	return crypto.Keccak256Hash(payload[:])
}

// ReadRegistryFinalityPolicy reads the signing cluster size at a confirmed block.
// Registry policy is required for every signing cluster, including K=1.
// `evm.ReadRegistryFinalityPolicy(ctx, reader, registry, confirmations)` requires
// `CONFIG()` to return a nonzero Registry-owned Config with finality epoch 1 and
// a known K=1..256 checksum. Reads are pinned to one confirmed block hash and
// rechecked for an anchor reorg; missing or invalid policy fails closed.
// The Registry must prevent later policy writes.
// This reader does not prove immutability: use a non-upgradeable Registry with a
// frozen CONFIG address and finality row. Live policy changes are unsupported.
// Read at startup with a bounded context, not per withdrawal. State reads
// require RPC endpoints that support EIP-1898 hash parameters.
// confirmations=0 explicitly selects unconfirmed state for local development;
// production callers must use their chain's confirmation policy.
func ReadRegistryFinalityPolicy(ctx context.Context, r RegistryPolicyReader, registry common.Address, confirmations uint64) (RegistryFinalityPolicy, error) {
	var policy RegistryFinalityPolicy
	if registry == (common.Address{}) {
		return policy, errors.New("clearing policy: invalid registry")
	}
	head, err := r.HeaderByNumber(ctx, nil)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: anchor head: %w", err)
	}
	if head == nil || head.Number == nil || !head.Number.IsUint64() || head.Number.Uint64() < confirmations {
		return policy, errors.New("clearing policy: confirmed anchor state unavailable")
	}
	policy.ConfirmedBlock = head.Number.Uint64() - confirmations
	block := new(big.Int).SetUint64(policy.ConfirmedBlock)
	anchor, err := r.HeaderByNumber(ctx, block)
	if err != nil || anchor == nil || anchor.Number == nil || anchor.Number.Cmp(block) != 0 {
		return policy, fmt.Errorf("clearing policy: anchor header unavailable: %v", err)
	}
	policy.ConfirmedHash = anchor.Hash()
	code, err := r.CodeAtHash(ctx, registry, policy.ConfirmedHash)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: registry code: %w", err)
	}
	if len(code) == 0 {
		return policy, errors.New("clearing policy: registry not deployed at confirmed block")
	}
	registryABI, err := ClearnetRegistryConfigMetaData.GetAbi()
	if err != nil {
		return policy, err
	}
	selector := registryABI.Methods["CONFIG"].ID
	result, err := r.CallContractAtHash(ctx, ethereum.CallMsg{To: &registry, Data: selector}, policy.ConfirmedHash)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: Registry CONFIG() call failed (Registry must implement IClearnetRegistryConfig): %w", err)
	}
	config, err := policyAddress(result)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: CONFIG: %w", err)
	}
	policy.ConfigAddress = config
	configABI, err := ConfigMetaData.GetAbi()
	if err != nil {
		return policy, err
	}
	call := func(method string, args ...interface{}) ([]byte, error) {
		data, err := configABI.Pack(method, args...)
		if err != nil {
			return nil, err
		}
		out, err := r.CallContractAtHash(ctx, ethereum.CallMsg{To: &config, Data: data}, policy.ConfirmedHash)
		if err != nil {
			return nil, fmt.Errorf("clearing policy: %s: %w", method, err)
		}
		if len(out) != 32 {
			return nil, fmt.Errorf("clearing policy: malformed %s", method)
		}
		return out, nil
	}
	ownerBytes, err := call("owner")
	if err != nil {
		return policy, err
	}
	owner, err := policyAddress(ownerBytes)
	if err != nil || owner != registry {
		return policy, errors.New("clearing policy: Config is not owned by Registry")
	}
	epoch, err := call("configEpoch", RegistryFinalityPolicyKey)
	if err != nil {
		return policy, err
	}
	if new(big.Int).SetBytes(epoch).Cmp(big.NewInt(1)) != 0 {
		return policy, errors.New("clearing policy: expected one frozen finality epoch")
	}
	checksum, err := call("latestConfigChecksum", RegistryFinalityPolicyKey)
	if err != nil {
		return policy, err
	}
	// The v1 domain is only 1..256. Resolve its checksum directly, avoiding
	// off-chain payload distribution and an event-history watcher for a frozen
	// 32-byte setting. Unknown schemas/checksums are rejected, never defaulted.
	for k := uint64(1); k <= core.MaxClusterSize; k++ {
		if RegistryFinalityPolicyChecksum(k) == common.BytesToHash(checksum) {
			policy.SigningClusterSize = k
			return policy, checkPolicyAnchor(ctx, r, block, policy.ConfirmedHash)
		}
	}
	return policy, errors.New("clearing policy: unknown finality checksum")
}

func checkPolicyAnchor(ctx context.Context, r RegistryPolicyReader, block *big.Int, hash common.Hash) error {
	head, err := r.HeaderByNumber(ctx, block)
	if err != nil {
		return fmt.Errorf("clearing policy: recheck anchor: %w", err)
	}
	if head == nil || head.Hash() != hash {
		return errors.New("clearing policy: anchor changed during policy read")
	}
	return nil
}

func policyAddress(word []byte) (common.Address, error) {
	if len(word) != 32 || !bytes.Equal(word[:12], make([]byte, 12)) || common.BytesToAddress(word[12:]) == (common.Address{}) {
		return common.Address{}, errors.New("invalid address encoding")
	}
	return common.BytesToAddress(word[12:]), nil
}

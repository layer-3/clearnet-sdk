package evm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// ClearingFinalityPolicyKey identifies the Registry's fixed signing-cluster policy.
// Its v1 payload is abi.encode(uint64(K)).
var ClearingFinalityPolicyKey = crypto.Keccak256Hash([]byte("CLEARNET_FINALITY_POLICY_V1"))

type ClearingPolicyReader interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

type ClearingFinalityPolicy struct {
	SigningClusterSize uint64
	ConfigAddress      common.Address
	ConfirmedBlock     uint64
	Legacy             bool
}

func ClearingFinalityPolicyChecksum(k uint64) common.Hash {
	var payload [32]byte
	binary.BigEndian.PutUint64(payload[24:], k)
	return crypto.Keccak256Hash(payload[:])
}

// ReadRegistryFinalityPolicy reads the signing cluster size at a confirmed block.
// When CONFIG() exists, its Config must be Registry-owned with epoch 1 and a
// known v1 checksum. The Registry must prevent later policy writes.
// legacyK applies only to direct Solidity Registries without CONFIG(); missing
// contracts, RPC failures, and invalid advertised policy return errors.
// Proxies cannot use legacy fallback because their bytecode hides the getter.
func ReadRegistryFinalityPolicy(ctx context.Context, r ClearingPolicyReader, registry common.Address, confirmations, legacyK uint64) (ClearingFinalityPolicy, error) {
	var policy ClearingFinalityPolicy
	if registry == (common.Address{}) || legacyK == 0 || legacyK > core.MaxClusterSize {
		return policy, errors.New("clearing policy: invalid registry or legacy quorum")
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
	code, err := r.CodeAt(ctx, registry, block)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: registry code: %w", err)
	}
	if len(code) == 0 {
		return policy, errors.New("clearing policy: registry not deployed at confirmed block")
	}
	selector := crypto.Keccak256([]byte("CONFIG()"))[:4]
	result, err := r.CallContract(ctx, ethereum.CallMsg{To: &registry, Data: selector}, block)
	// Unknown selectors on old Solidity registries revert without data. Require
	// both that evidence and the selector's absence; do not mask a failing getter.
	if !bytes.Contains(code, selector) && ((err == nil && len(result) == 0) || emptyRevert(err)) {
		policy.SigningClusterSize, policy.Legacy = legacyK, true
		return policy, nil
	}
	if err != nil {
		return policy, fmt.Errorf("clearing policy: CONFIG: %w", err)
	}
	config, err := policyAddress(result)
	if err != nil {
		return policy, fmt.Errorf("clearing policy: CONFIG: %w", err)
	}
	policy.ConfigAddress = config
	call := func(signature string, args ...[]byte) ([]byte, error) {
		data := append([]byte{}, crypto.Keccak256([]byte(signature))[:4]...)
		for _, arg := range args {
			data = append(data, arg...)
		}
		out, err := r.CallContract(ctx, ethereum.CallMsg{To: &config, Data: data}, block)
		if err != nil {
			return nil, fmt.Errorf("clearing policy: %s: %w", signature, err)
		}
		if len(out) != 32 {
			return nil, fmt.Errorf("clearing policy: malformed %s", signature)
		}
		return out, nil
	}
	ownerBytes, err := call("owner()")
	if err != nil {
		return policy, err
	}
	owner, err := policyAddress(ownerBytes)
	if err != nil || owner != registry {
		return policy, errors.New("clearing policy: Config is not owned by Registry")
	}
	epoch, err := call("configEpoch(bytes32)", ClearingFinalityPolicyKey[:])
	if err != nil {
		return policy, err
	}
	if new(big.Int).SetBytes(epoch).Cmp(big.NewInt(1)) != 0 {
		return policy, errors.New("clearing policy: expected one frozen finality epoch")
	}
	checksum, err := call("latestConfigChecksum(bytes32)", ClearingFinalityPolicyKey[:])
	if err != nil {
		return policy, err
	}
	// The v1 domain is only 1..256. Resolve its checksum directly, avoiding
	// off-chain payload distribution and an event-history watcher for a frozen
	// 32-byte setting. Unknown schemas/checksums are rejected, never defaulted.
	for k := uint64(1); k <= core.MaxClusterSize; k++ {
		if ClearingFinalityPolicyChecksum(k) == common.BytesToHash(checksum) {
			policy.SigningClusterSize = k
			return policy, nil
		}
	}
	return policy, errors.New("clearing policy: unknown finality checksum")
}

// ReadRegistryClearingPolicy is the original name used by coordinated callers.
// Deprecated: use ReadRegistryFinalityPolicy.
func ReadRegistryClearingPolicy(ctx context.Context, r ClearingPolicyReader, registry common.Address, confirmations, legacyK uint64) (ClearingFinalityPolicy, error) {
	return ReadRegistryFinalityPolicy(ctx, r, registry, confirmations, legacyK)
}

func policyAddress(word []byte) (common.Address, error) {
	if len(word) != 32 || !bytes.Equal(word[:12], make([]byte, 12)) || common.BytesToAddress(word[12:]) == (common.Address{}) {
		return common.Address{}, errors.New("invalid address encoding")
	}
	return common.BytesToAddress(word[12:]), nil
}

func emptyRevert(err error) bool {
	var dataErr rpc.DataError
	return errors.As(err, &dataErr) && dataErr.ErrorData() == "0x" && strings.HasPrefix(err.Error(), "execution reverted")
}

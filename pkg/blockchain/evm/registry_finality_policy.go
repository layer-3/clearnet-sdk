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
	Legacy             bool
}

func RegistryFinalityPolicyChecksum(k uint64) common.Hash {
	var payload [32]byte
	binary.BigEndian.PutUint64(payload[24:], k)
	return crypto.Keccak256Hash(payload[:])
}

// ReadRegistryFinalityPolicy reads the signing cluster size at a confirmed block.
// When CONFIG() exists, its Config must be Registry-owned with epoch 1 and a
// known v1 checksum. The Registry must prevent later policy writes.
// This reader does not prove immutability: use a non-upgradeable Registry with a
// frozen CONFIG address and finality row. Live policy changes are unsupported.
// Read at startup with a bounded context, not per withdrawal. All state reads
// use one block hash; RPC endpoints must support EIP-1898 hash parameters.
// confirmations=0 explicitly selects unconfirmed state for local development;
// production callers must use their chain's confirmation policy.
// legacyK applies only to direct Solidity Registries without CONFIG(); missing
// contracts, RPC failures, and invalid advertised policy return errors.
// Legacy fallback requires an empty execution revert, never empty success, and
// rejects DELEGATECALL/CALLCODE bytecode. It is not proxy implementation discovery.
// The caller must independently approve the direct legacy deployment and its K;
// this heuristic is not proof that arbitrary bytecode implements a Registry.
// Unknown RPC error shapes fail closed; empty reverts require geth-compatible
// rpc.DataError("0x") and an "execution reverted" message.
func ReadRegistryFinalityPolicy(ctx context.Context, r RegistryPolicyReader, registry common.Address, confirmations, legacyK uint64) (RegistryFinalityPolicy, error) {
	var policy RegistryFinalityPolicy
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
	// Unknown selectors on old Solidity registries revert without data. Require
	// both that evidence and the selector's absence; do not mask a failing getter.
	if !bytes.Contains(code, selector) && !hasDelegateCall(code) && emptyRevert(err) {
		policy.SigningClusterSize, policy.Legacy = legacyK, true
		return policy, checkPolicyAnchor(ctx, r, block, policy.ConfirmedHash)
	}
	if err != nil {
		return policy, fmt.Errorf("clearing policy: CONFIG: %w", err)
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

// Walk instructions, not raw bytes: PUSH data can contain any opcode value.
func hasDelegateCall(code []byte) bool {
	for pc := 0; pc < len(code); pc++ {
		op := code[pc]
		if op == 0xf4 || op == 0xf2 { // DELEGATECALL or CALLCODE
			return true
		}
		if op >= 0x60 && op <= 0x7f {
			pc += int(op - 0x5f)
		}
	}
	return false
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

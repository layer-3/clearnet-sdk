package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// configLookupWindow bounds the eth_getLogs range when resolving the tx hash of
// an already-applied config commit / operator rotation.
const configLookupWindow = uint64(50_000)

// fetchLiveOperatorQuorum reads ConfigGovernor's current operator set and
// threshold. The operator quorum is what authorises both setConfig (config
// commit) and updateOperators (operator rotation), so both size and filter
// against it — the governor analogue of fetchLiveQuorum.
func fetchLiveOperatorQuorum(ctx context.Context, gov *ConfigGovernor) ([]common.Address, int, error) {
	ops, err := gov.Operators(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read operators: %w", err)
	}
	thr, err := gov.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(ops)) {
		return nil, 0, fmt.Errorf("on-chain operator threshold %s out of range for %d operators", thr, len(ops))
	}
	return ops, int(thr.Int64()), nil
}

// ConfigCommitFinalizer commits a content-addressed config checksum to the
// Config registry via ConfigGovernor.setConfig, authorised by the operator
// quorum. It is the config-commit analogue of RotationFinalizer: the off-chain
// ceremony collects operator-key signatures over the commit digest, and the
// HRW-elected submitter calls Submit. It owns the node's operator signer and
// the governor address + chain id supplied at construction.
type ConfigCommitFinalizer struct {
	client     *ethclient.Client
	governor   *ConfigGovernor
	registry   *Config
	govAddr    common.Address
	chainID    uint64
	signer     sign.Signer
	signerAddr common.Address
	fees       FeeConfig
}

// NewConfigCommitFinalizer binds the ConfigGovernor at govAddr, resolves the
// Config registry it writes through, and reads the chain id from client. signer
// is this node's operator-key identity.
func NewConfigCommitFinalizer(ctx context.Context, client *ethclient.Client, govAddr common.Address, signer sign.Signer, fees FeeConfig) (*ConfigCommitFinalizer, error) {
	gov, err := NewConfigGovernor(govAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config governor: %w", err)
	}
	regAddr, err := gov.Config(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("read governor registry: %w", err)
	}
	registry, err := NewConfig(regAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config registry: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	return &ConfigCommitFinalizer{
		client:     client,
		governor:   gov,
		registry:   registry,
		govAddr:    govAddr,
		chainID:    chainID.Uint64(),
		signer:     signer,
		signerAddr: addr,
		fees:       fees,
	}, nil
}

// SignerAddress is the operator address this finalizer signs as.
func (f *ConfigCommitFinalizer) SignerAddress() common.Address { return f.signerAddr }

// evmCommitPacked is the canonical config-commit payload: the registry key, the
// content checksum, and the expectedEpoch the digest is bound to.
type evmCommitPacked struct {
	Key           string `json:"key"`           // 0x-prefixed 32-byte hex
	Checksum      string `json:"checksum"`      // 0x-prefixed 32-byte hex
	ExpectedEpoch uint64 `json:"expectedEpoch"` // registry configEpoch(key) at sign time
}

// Pack reads the live registry epoch for key and returns the canonical JSON for
// committing checksum at that epoch. The epoch is the commit's only replay
// token (Config.sol has no nonce), bound into the signed digest.
func (f *ConfigCommitFinalizer) Pack(ctx context.Context, key [32]byte, checksum [32]byte) ([]byte, error) {
	epoch, err := f.registry.ConfigEpoch(&bind.CallOpts{Context: ctx}, key)
	if err != nil {
		return nil, fmt.Errorf("read config epoch: %w", err)
	}
	p := evmCommitPacked{
		Key:           hexBytes32(key),
		Checksum:      hexBytes32(checksum),
		ExpectedEpoch: epoch,
	}
	return json.Marshal(p)
}

// Validate re-derives the commit target and asserts the packed payload matches,
// including a re-read of the live epoch to reject a packer that bound a stale
// expectedEpoch (the commit would revert on-chain otherwise).
func (f *ConfigCommitFinalizer) Validate(ctx context.Context, packed []byte, key [32]byte, checksum [32]byte) error {
	var got evmCommitPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	if !strEqFold(got.Key, hexBytes32(key)) || !strEqFold(got.Checksum, hexBytes32(checksum)) {
		return fmt.Errorf("packed commit does not match request")
	}
	epoch, err := f.registry.ConfigEpoch(&bind.CallOpts{Context: ctx}, key)
	if err != nil {
		return fmt.Errorf("read config epoch: %w", err)
	}
	if got.ExpectedEpoch != epoch {
		return fmt.Errorf("packed expectedEpoch %d != live %d", got.ExpectedEpoch, epoch)
	}
	return nil
}

// Sign produces this node's 65-byte ECDSA signature (V ∈ {0,1}) over the commit
// digest derived from the packed bytes.
func (f *ConfigCommitFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.signer, digest[:], f.signerAddr)
}

// Submit merges the collected operator signatures against the live operator set
// and broadcasts setConfig. Idempotent: if this exact commit already landed it
// returns the prior tx hash without re-submitting.
func (f *ConfigCommitFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (core.TxRef, error) {
	var p evmCommitPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return core.TxRef{}, fmt.Errorf("decode packed: %w", err)
	}
	key, err := parseBytes32(p.Key)
	if err != nil {
		return core.TxRef{}, err
	}
	checksum, err := parseBytes32(p.Checksum)
	if err != nil {
		return core.TxRef{}, err
	}

	if txHash, done, err := f.VerifyCommit(ctx, key, checksum, p.ExpectedEpoch); err != nil {
		return core.TxRef{}, err
	} else if done {
		return core.TxRef{Hash: txHash, Raw: common.Hash(txHash).Hex()}, nil
	}

	digest := ComputeConfigCommitDigest(f.chainID, f.govAddr, key, checksum, p.ExpectedEpoch)
	liveOps, liveThreshold, err := fetchLiveOperatorQuorum(ctx, f.governor)
	if err != nil {
		return core.TxRef{}, err
	}
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, liveOps, liveThreshold)
	if err != nil {
		return core.TxRef{}, err
	}

	opts, _, err := signerTransactOpts(ctx, f.client, f.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	if err := applyFees(ctx, f.client, f.fees, opts); err != nil {
		return core.TxRef{}, err
	}
	if err := f.estimateGas(ctx, opts, key, checksum, p.ExpectedEpoch, sigs); err != nil {
		return core.TxRef{}, err
	}
	tx, err := f.governor.SetConfig(opts, key, checksum, p.ExpectedEpoch, sigs)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("setConfig: %w", err)
	}
	if err := waitMined(ctx, f.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

// VerifyCommit reports whether checksum was committed at expectedEpoch (i.e. the
// registry has advanced past expectedEpoch and the slot our commit would have
// pushed holds checksum). When set, it resolves the ConfigCommitted tx hash.
func (f *ConfigCommitFinalizer) VerifyCommit(ctx context.Context, key [32]byte, checksum [32]byte, expectedEpoch uint64) ([32]byte, bool, error) {
	epoch, err := f.registry.ConfigEpoch(&bind.CallOpts{Context: ctx}, key)
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read config epoch: %w", err)
	}
	if epoch <= expectedEpoch {
		return [32]byte{}, false, nil
	}
	// setConfig pushes at index == old length == expectedEpoch.
	at, err := f.registry.ConfigChecksumAt(&bind.CallOpts{Context: ctx}, key, new(big.Int).SetUint64(expectedEpoch))
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read checksum at epoch: %w", err)
	}
	if at != checksum {
		return [32]byte{}, false, nil
	}
	return f.lookupCommitTxHash(ctx, key, checksum), true, nil
}

func (f *ConfigCommitFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p evmCommitPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("decode packed: %w", err)
	}
	key, err := parseBytes32(p.Key)
	if err != nil {
		return [32]byte{}, err
	}
	checksum, err := parseBytes32(p.Checksum)
	if err != nil {
		return [32]byte{}, err
	}
	return ComputeConfigCommitDigest(f.chainID, f.govAddr, key, checksum, p.ExpectedEpoch), nil
}

func (f *ConfigCommitFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, key [32]byte, checksum [32]byte, expectedEpoch uint64, sigs [][]byte) error {
	parsed, err := ConfigGovernorMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	data, err := parsed.Pack("setConfig", key, checksum, expectedEpoch, sigs)
	if err != nil {
		return fmt.Errorf("pack setConfig calldata: %w", err)
	}
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.signerAddr,
		To:        &f.govAddr,
		Data:      data,
		GasTipCap: opts.GasTipCap,
		GasFeeCap: opts.GasFeeCap,
		GasPrice:  opts.GasPrice,
	})
	if err != nil {
		return fmt.Errorf("estimate gas: %w", err)
	}
	opts.GasLimit = uint64(float64(est) * f.fees.gasLimitMultiplier())
	return nil
}

func (f *ConfigCommitFinalizer) lookupCommitTxHash(ctx context.Context, key [32]byte, checksum [32]byte) [32]byte {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		return [32]byte{}
	}
	var from uint64
	if head > configLookupWindow {
		from = head - configLookupWindow
	}
	it, err := f.governor.FilterConfigCommitted(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, [][32]byte{key})
	if err != nil {
		return [32]byte{}
	}
	defer it.Close()
	var last [32]byte
	for it.Next() {
		if it.Event.Checksum == checksum {
			last = it.Event.Raw.TxHash // keep the most recent match
		}
	}
	return last
}

// OperatorRotationFinalizer rotates ConfigGovernor's own operator set via
// updateOperators, authorised by the current operator quorum. It is the operator
// analogue of RotationFinalizer (which rotates the vault signer set) and is the
// anchor-chain step of an operator handoff (ADR-017 rotation step 3).
type OperatorRotationFinalizer struct {
	client     *ethclient.Client
	governor   *ConfigGovernor
	govAddr    common.Address
	chainID    uint64
	signer     sign.Signer
	signerAddr common.Address
	fees       FeeConfig
}

// NewOperatorRotationFinalizer binds the ConfigGovernor at govAddr and reads the
// chain id from client. signer is this node's operator-key identity.
func NewOperatorRotationFinalizer(ctx context.Context, client *ethclient.Client, govAddr common.Address, signer sign.Signer, fees FeeConfig) (*OperatorRotationFinalizer, error) {
	gov, err := NewConfigGovernor(govAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config governor: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	return &OperatorRotationFinalizer{
		client:     client,
		governor:   gov,
		govAddr:    govAddr,
		chainID:    chainID.Uint64(),
		signer:     signer,
		signerAddr: addr,
		fees:       fees,
	}, nil
}

// evmOpRotPacked is the canonical operator-rotation payload: the new operator
// set (ascending) + threshold, and the operatorNonce the digest is bound to.
type evmOpRotPacked struct {
	NewOperators []string `json:"newOperators"` // ascending hex addresses
	NewThreshold int      `json:"newThreshold"`
	OperatorNonce string  `json:"operatorNonce"` // decimal
}

// Pack reads the live operatorNonce and returns the canonical JSON for rotating
// to newOperators / newThreshold (operators sorted ascending, as the contract
// requires).
func (f *OperatorRotationFinalizer) Pack(ctx context.Context, newOperators []string, newThreshold int) ([]byte, error) {
	addrs, err := parseSignerAddresses(newOperators)
	if err != nil {
		return nil, err
	}
	if newThreshold <= 0 || newThreshold > len(addrs) {
		return nil, fmt.Errorf("evm: operator threshold %d out of range for %d operators", newThreshold, len(addrs))
	}
	nonce, err := f.governor.OperatorNonce(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("read operator nonce: %w", err)
	}
	p := evmOpRotPacked{NewOperators: addrsToHex(addrs), NewThreshold: newThreshold, OperatorNonce: nonce.String()}
	return json.Marshal(p)
}

// Validate re-derives the rotation target and asserts the packed payload
// matches, including a re-read of the live nonce.
func (f *OperatorRotationFinalizer) Validate(ctx context.Context, packed []byte, newOperators []string, newThreshold int) error {
	var got evmOpRotPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(newOperators)
	if err != nil {
		return err
	}
	if got.NewThreshold != newThreshold || !equalStrings(got.NewOperators, addrsToHex(addrs)) {
		return fmt.Errorf("packed operator rotation does not match request")
	}
	nonce, err := f.governor.OperatorNonce(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("read operator nonce: %w", err)
	}
	if got.OperatorNonce != nonce.String() {
		return fmt.Errorf("packed operator nonce %s != live %s", got.OperatorNonce, nonce)
	}
	return nil
}

// Sign produces this node's 65-byte ECDSA signature over the operator-rotation
// digest derived from the packed bytes.
func (f *OperatorRotationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.signer, digest[:], f.signerAddr)
}

// Submit merges the collected signatures against the current operator set and
// broadcasts updateOperators. Idempotent: if the operator set already equals the
// target it returns the prior tx hash.
func (f *OperatorRotationFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (core.TxRef, error) {
	var p evmOpRotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return core.TxRef{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(p.NewOperators)
	if err != nil {
		return core.TxRef{}, err
	}
	if txHash, done, err := f.VerifyRotation(ctx, p.NewOperators, p.NewThreshold); err != nil {
		return core.TxRef{}, err
	} else if done {
		return core.TxRef{Hash: txHash, Raw: common.Hash(txHash).Hex()}, nil
	}

	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return core.TxRef{}, err
	}
	liveOps, liveThreshold, err := fetchLiveOperatorQuorum(ctx, f.governor)
	if err != nil {
		return core.TxRef{}, err
	}
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, liveOps, liveThreshold)
	if err != nil {
		return core.TxRef{}, err
	}
	nonce, ok := new(big.Int).SetString(p.OperatorNonce, 10)
	if !ok {
		return core.TxRef{}, fmt.Errorf("bad operator nonce %q", p.OperatorNonce)
	}

	opts, _, err := signerTransactOpts(ctx, f.client, f.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	if err := applyFees(ctx, f.client, f.fees, opts); err != nil {
		return core.TxRef{}, err
	}
	newThreshold := big.NewInt(int64(p.NewThreshold))
	if err := f.estimateGas(ctx, opts, addrs, newThreshold, nonce, sigs); err != nil {
		return core.TxRef{}, err
	}
	tx, err := f.governor.UpdateOperators(opts, addrs, newThreshold, nonce, sigs)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("updateOperators: %w", err)
	}
	if err := waitMined(ctx, f.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

// VerifyRotation reports whether the on-chain operator set now equals
// newOperators with the given threshold. When set, it resolves the
// OperatorsUpdated event's tx hash within the lookback window.
func (f *OperatorRotationFinalizer) VerifyRotation(ctx context.Context, newOperators []string, newThreshold int) ([32]byte, bool, error) {
	addrs, err := parseSignerAddresses(newOperators)
	if err != nil {
		return [32]byte{}, false, err
	}
	live, err := f.governor.Operators(&bind.CallOpts{Context: ctx})
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read operators: %w", err)
	}
	thr, err := f.governor.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || int(thr.Int64()) != newThreshold || !addrSetEqual(live, addrs) {
		return [32]byte{}, false, nil
	}
	return f.lookupRotationTxHash(ctx, addrs), true, nil
}

func (f *OperatorRotationFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p evmOpRotPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, err := parseSignerAddresses(p.NewOperators)
	if err != nil {
		return [32]byte{}, err
	}
	nonce, ok := new(big.Int).SetString(p.OperatorNonce, 10)
	if !ok {
		return [32]byte{}, fmt.Errorf("bad operator nonce %q", p.OperatorNonce)
	}
	return ComputeOperatorRotationDigest(f.chainID, f.govAddr, addrs, big.NewInt(int64(p.NewThreshold)), nonce), nil
}

func (f *OperatorRotationFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, newOperators []common.Address, newThreshold, operatorNonce *big.Int, sigs [][]byte) error {
	parsed, err := ConfigGovernorMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	data, err := parsed.Pack("updateOperators", newOperators, newThreshold, operatorNonce, sigs)
	if err != nil {
		return fmt.Errorf("pack updateOperators calldata: %w", err)
	}
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.signerAddr,
		To:        &f.govAddr,
		Data:      data,
		GasTipCap: opts.GasTipCap,
		GasFeeCap: opts.GasFeeCap,
		GasPrice:  opts.GasPrice,
	})
	if err != nil {
		return fmt.Errorf("estimate gas: %w", err)
	}
	opts.GasLimit = uint64(float64(est) * f.fees.gasLimitMultiplier())
	return nil
}

func (f *OperatorRotationFinalizer) lookupRotationTxHash(ctx context.Context, addrs []common.Address) [32]byte {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		return [32]byte{}
	}
	var from uint64
	if head > configLookupWindow {
		from = head - configLookupWindow
	}
	it, err := f.governor.FilterOperatorsUpdated(&bind.FilterOpts{Context: ctx, Start: from, End: &head})
	if err != nil {
		return [32]byte{}
	}
	defer it.Close()
	var last [32]byte
	for it.Next() {
		if addrSetEqual(it.Event.NewOperators, addrs) {
			last = it.Event.Raw.TxHash
		}
	}
	return last
}

// --- shared helpers ---

func hexBytes32(b [32]byte) string {
	return "0x" + common.Bytes2Hex(b[:])
}

// parseBytes32 decodes a 0x-prefixed 32-byte hex string.
func parseBytes32(s string) ([32]byte, error) {
	var out [32]byte
	h := s
	if len(h) >= 2 && (h[0:2] == "0x" || h[0:2] == "0X") {
		h = h[2:]
	}
	if len(h) != 64 {
		return out, fmt.Errorf("evm: %q is not a 32-byte hex string", s)
	}
	raw := common.FromHex("0x" + h)
	if len(raw) != 32 {
		return out, fmt.Errorf("evm: %q is not valid hex", s)
	}
	copy(out[:], raw)
	return out, nil
}

func strEqFold(a, b string) bool { return bytes.EqualFold([]byte(a), []byte(b)) }

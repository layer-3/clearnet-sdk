package evm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/log"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

const (
	defaultConfigLookupWindow = uint64(50_000)

	configRegistryOpRegisterIssuer       = "registerIssuer"
	configRegistryOpSetConfig            = "setConfig"
	configRegistryOpSetConfigWithData    = "setConfigWithData"
	configRegistryOpUpdateIssuerSettings = "updateIssuerSettings"
)

// ConfigRegistryIssuerRegistrationResult is returned by issuer registration.
// IssuerID is always populated, including the idempotent already-registered
// path where the original IssuerRegistered tx is outside the lookup window.
type ConfigRegistryIssuerRegistrationResult struct {
	TxID     string
	IssuerID common.Address
}

// ConfigRegistryIssuerRegistrationFinalizer registers a new ConfigRegistry
// issuer via the same Pack / Validate / Sign / Submit ceremony shape used by
// other EVM finalizers. The signature set is self-referential: a quorum of the
// claimed issuer keys signs over the claimed issuer key set and threshold.
type ConfigRegistryIssuerRegistrationFinalizer struct {
	client         *ethclient.Client
	registry       *ConfigRegistry
	registryAddr   common.Address
	chainID        uint64
	authorizer     sign.Signer
	authorizerAddr common.Address
	transactor     Transactor
	fees           FeeConfig
	logger         log.Logger
	lookupWindow   uint64
}

func NewConfigRegistryIssuerRegistrationFinalizer(ctx context.Context, client *ethclient.Client, registryAddr common.Address, authorizer sign.Signer, payer Transactor, fees FeeConfig) (*ConfigRegistryIssuerRegistrationFinalizer, error) {
	if payer == nil {
		return nil, fmt.Errorf("evm: transactor is required")
	}
	registry, err := NewConfigRegistry(registryAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config registry: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(authorizer)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryIssuerRegistrationFinalizer{
		client:         client,
		registry:       registry,
		registryAddr:   registryAddr,
		chainID:        chainID.Uint64(),
		authorizer:     authorizer,
		authorizerAddr: addr,
		transactor:     payer,
		fees:           fees,
		logger:         log.NewNoopLogger(),
		lookupWindow:   defaultConfigLookupWindow,
	}, nil
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) SetLogger(l log.Logger) {
	if l == nil {
		l = log.NewNoopLogger()
	}
	f.logger = l
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) SetLookupWindow(blocks uint64) {
	if blocks == 0 {
		blocks = defaultConfigLookupWindow
	}
	f.lookupWindow = blocks
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) SignerAddress() common.Address {
	return f.authorizerAddr
}

type configRegistryIssuerRegistrationPacked struct {
	Op         string   `json:"op"`
	IssuerKeys []string `json:"issuerKeys"`
	Threshold  int      `json:"threshold"`
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) Pack(_ context.Context, issuerKeys []string, threshold int) ([]byte, error) {
	addrs, err := parseIssuerAddresses(issuerKeys)
	if err != nil {
		return nil, err
	}
	if err := validateThreshold(threshold, len(addrs), "issuer"); err != nil {
		return nil, err
	}
	return json.Marshal(configRegistryIssuerRegistrationPacked{
		Op:         configRegistryOpRegisterIssuer,
		IssuerKeys: addrsToHex(addrs),
		Threshold:  threshold,
	})
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) Validate(_ context.Context, packed []byte, issuerKeys []string, threshold int) error {
	var got configRegistryIssuerRegistrationPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	wantBytes, err := f.Pack(context.Background(), issuerKeys, threshold)
	if err != nil {
		return err
	}
	var want configRegistryIssuerRegistrationPacked
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		return fmt.Errorf("decode wanted packed: %w", err)
	}
	if got.Op != want.Op || got.Threshold != want.Threshold || !equalStrings(got.IssuerKeys, want.IssuerKeys) {
		return fmt.Errorf("packed issuer registration does not match request")
	}
	return nil
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.authorizer, digest[:], f.authorizerAddr)
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (ConfigRegistryIssuerRegistrationResult, error) {
	var p configRegistryIssuerRegistrationPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return ConfigRegistryIssuerRegistrationResult{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, threshold, err := parseRegistrationPacked(p)
	if err != nil {
		return ConfigRegistryIssuerRegistrationResult{}, err
	}
	issuerID, err := f.registry.ComputeIssuerId(&bind.CallOpts{Context: ctx}, addrs, threshold)
	if err != nil {
		return ConfigRegistryIssuerRegistrationResult{}, fmt.Errorf("compute issuer id: %w", err)
	}
	result := ConfigRegistryIssuerRegistrationResult{IssuerID: issuerID}

	registered, err := f.registry.IsRegistered(&bind.CallOpts{Context: ctx}, issuerID)
	if err != nil {
		return result, fmt.Errorf("read issuer registration: %w", err)
	}
	if registered {
		result.TxID = f.lookupIssuerRegisteredTxID(ctx, issuerID)
		return result, nil
	}

	digest := ComputeConfigRegistryRegistrationDigest(f.chainID, f.registryAddr, addrs, threshold)
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, addrs, int(threshold.Int64()))
	if err != nil {
		return result, err
	}

	tx, err := f.transactor.Transact(ctx, f.fees, func(opts *bind.TransactOpts) (*gethtypes.Transaction, error) {
		if err := f.estimateGas(ctx, opts, addrs, threshold, sigs); err != nil {
			return nil, err
		}
		tx, err := f.registry.RegisterIssuer(opts, addrs, threshold, sigs)
		if err != nil {
			return nil, fmt.Errorf("registerIssuer: %w", err)
		}
		return tx, nil
	})
	if err != nil {
		return result, err
	}
	result.TxID = tx.Hash().Hex()
	return result, nil
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	var p configRegistryIssuerRegistrationPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return [32]byte{}, fmt.Errorf("decode packed: %w", err)
	}
	addrs, threshold, err := parseRegistrationPacked(p)
	if err != nil {
		return [32]byte{}, err
	}
	return ComputeConfigRegistryRegistrationDigest(f.chainID, f.registryAddr, addrs, threshold), nil
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, issuerKeys []common.Address, threshold *big.Int, sigs [][]byte) error {
	abi, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	data, err := abi.Pack("registerIssuer", issuerKeys, threshold, sigs)
	if err != nil {
		return fmt.Errorf("pack registerIssuer calldata: %w", err)
	}
	return f.estimateCallGas(ctx, opts, data)
}

func (f *ConfigRegistryIssuerRegistrationFinalizer) estimateCallGas(ctx context.Context, opts *bind.TransactOpts, data []byte) error {
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.transactor.From(),
		To:        &f.registryAddr,
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

func (f *ConfigRegistryIssuerRegistrationFinalizer) lookupIssuerRegisteredTxID(ctx context.Context, issuerID common.Address) string {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		f.logger.Warn("issuer registration txID lookup: block number failed, returning empty txID",
			"issuerId", issuerID.Hex(), "error", err)
		return ""
	}
	var from uint64
	if head > f.lookupWindow {
		from = head - f.lookupWindow
	}
	it, err := f.registry.FilterIssuerRegistered(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, []common.Address{issuerID})
	if err != nil {
		f.logger.Warn("issuer registration txID lookup: filter failed, returning empty txID",
			"issuerId", issuerID.Hex(), "error", err)
		return ""
	}
	defer it.Close()
	var last string
	for it.Next() {
		last = it.Event.Raw.TxHash.Hex()
	}
	if last == "" {
		f.logger.Warn("issuer registration txID lookup: no matching IssuerRegistered in window, returning empty txID",
			"issuerId", issuerID.Hex(), "window", f.lookupWindow)
	}
	return last
}

// ConfigRegistryCommitFinalizer commits checksum-only and payload-carrying
// config writes for one issuer. It binds signatures to ConfigRegistry's
// per-issuer nonce, not to the per-key Config epoch.
type ConfigRegistryCommitFinalizer struct {
	client         *ethclient.Client
	registry       *ConfigRegistry
	config         *IConfig
	registryAddr   common.Address
	issuerID       common.Address
	chainID        uint64
	authorizer     sign.Signer
	authorizerAddr common.Address
	transactor     Transactor
	fees           FeeConfig
	logger         log.Logger
	lookupWindow   uint64
}

func NewConfigRegistryCommitFinalizer(ctx context.Context, client *ethclient.Client, registryAddr common.Address, issuerID common.Address, authorizer sign.Signer, payer Transactor, fees FeeConfig) (*ConfigRegistryCommitFinalizer, error) {
	if payer == nil {
		return nil, fmt.Errorf("evm: transactor is required")
	}
	registry, err := NewConfigRegistry(registryAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config registry: %w", err)
	}
	config, err := NewIConfig(issuerID, client)
	if err != nil {
		return nil, fmt.Errorf("load issuer config: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(authorizer)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryCommitFinalizer{
		client:         client,
		registry:       registry,
		config:         config,
		registryAddr:   registryAddr,
		issuerID:       issuerID,
		chainID:        chainID.Uint64(),
		authorizer:     authorizer,
		authorizerAddr: addr,
		transactor:     payer,
		fees:           fees,
		logger:         log.NewNoopLogger(),
		lookupWindow:   defaultConfigLookupWindow,
	}, nil
}

func (f *ConfigRegistryCommitFinalizer) SetLogger(l log.Logger) {
	if l == nil {
		l = log.NewNoopLogger()
	}
	f.logger = l
}

func (f *ConfigRegistryCommitFinalizer) SetLookupWindow(blocks uint64) {
	if blocks == 0 {
		blocks = defaultConfigLookupWindow
	}
	f.lookupWindow = blocks
}

func (f *ConfigRegistryCommitFinalizer) SignerAddress() common.Address { return f.authorizerAddr }

type configRegistryCommitPacked struct {
	Op            string `json:"op"`
	IssuerID      string `json:"issuerId"`
	Key           string `json:"key"`
	Checksum      string `json:"checksum,omitempty"`
	Data          string `json:"data,omitempty"`
	ExpectedNonce string `json:"expectedNonce"`
}

func (f *ConfigRegistryCommitFinalizer) Pack(ctx context.Context, key [32]byte, checksum [32]byte) ([]byte, error) {
	nonce, err := f.registry.Nonce(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return nil, fmt.Errorf("read issuer nonce: %w", err)
	}
	return json.Marshal(configRegistryCommitPacked{
		Op:            configRegistryOpSetConfig,
		IssuerID:      f.issuerID.Hex(),
		Key:           hexBytes32(key),
		Checksum:      hexBytes32(checksum),
		ExpectedNonce: nonce.String(),
	})
}

func (f *ConfigRegistryCommitFinalizer) PackWithData(ctx context.Context, key [32]byte, data []byte) ([]byte, error) {
	nonce, err := f.registry.Nonce(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return nil, fmt.Errorf("read issuer nonce: %w", err)
	}
	return json.Marshal(configRegistryCommitPacked{
		Op:            configRegistryOpSetConfigWithData,
		IssuerID:      f.issuerID.Hex(),
		Key:           hexBytes32(key),
		Data:          "0x" + common.Bytes2Hex(data),
		ExpectedNonce: nonce.String(),
	})
}

func (f *ConfigRegistryCommitFinalizer) Validate(ctx context.Context, packed []byte, key [32]byte, checksum [32]byte) error {
	var got configRegistryCommitPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	if got.Op != configRegistryOpSetConfig || !strings.EqualFold(got.IssuerID, f.issuerID.Hex()) ||
		!strings.EqualFold(got.Key, hexBytes32(key)) || !strings.EqualFold(got.Checksum, hexBytes32(checksum)) {
		return fmt.Errorf("packed config commit does not match request")
	}
	return f.validatePackedNonce(ctx, got.ExpectedNonce)
}

func (f *ConfigRegistryCommitFinalizer) ValidateWithData(ctx context.Context, packed []byte, key [32]byte, data []byte) error {
	var got configRegistryCommitPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	if got.Op != configRegistryOpSetConfigWithData || !strings.EqualFold(got.IssuerID, f.issuerID.Hex()) ||
		!strings.EqualFold(got.Key, hexBytes32(key)) || !strings.EqualFold(got.Data, "0x"+common.Bytes2Hex(data)) {
		return fmt.Errorf("packed config-with-data commit does not match request")
	}
	return f.validatePackedNonce(ctx, got.ExpectedNonce)
}

func (f *ConfigRegistryCommitFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.authorizer, digest[:], f.authorizerAddr)
}

func (f *ConfigRegistryCommitFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (string, error) {
	p, key, checksum, data, nonce, err := f.parsePacked(packed)
	if err != nil {
		return "", err
	}
	if txID, done, err := f.verifyCommit(ctx, key, checksum, p.Op); err != nil {
		return "", err
	} else if done {
		return txID, nil
	}

	digest, err := f.digestFromParsed(p, key, checksum, data, nonce)
	if err != nil {
		return "", err
	}
	liveKeys, liveThreshold, err := fetchLiveIssuerQuorum(ctx, f.registry, f.issuerID)
	if err != nil {
		return "", err
	}
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, liveKeys, liveThreshold)
	if err != nil {
		return "", err
	}

	tx, err := f.transactor.Transact(ctx, f.fees, func(opts *bind.TransactOpts) (*gethtypes.Transaction, error) {
		switch p.Op {
		case configRegistryOpSetConfig:
			if err := f.estimateGas(ctx, opts, key, checksum, nil, nonce, sigs); err != nil {
				return nil, err
			}
			tx, err := f.registry.SetConfig(opts, f.issuerID, key, checksum, nonce, sigs)
			if err != nil {
				return nil, fmt.Errorf("setConfig: %w", err)
			}
			return tx, nil
		case configRegistryOpSetConfigWithData:
			if err := f.estimateGas(ctx, opts, key, [32]byte{}, data, nonce, sigs); err != nil {
				return nil, err
			}
			tx, err := f.registry.SetConfigWithData(opts, f.issuerID, key, data, nonce, sigs)
			if err != nil {
				return nil, fmt.Errorf("setConfigWithData: %w", err)
			}
			return tx, nil
		default:
			return nil, fmt.Errorf("unsupported config registry op %q", p.Op)
		}
	})
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (f *ConfigRegistryCommitFinalizer) VerifyCommit(ctx context.Context, key [32]byte, checksum [32]byte) (string, bool, error) {
	return f.verifyCommit(ctx, key, checksum, configRegistryOpSetConfig)
}

func (f *ConfigRegistryCommitFinalizer) VerifyCommitWithData(ctx context.Context, key [32]byte, data []byte) (string, bool, error) {
	return f.verifyCommit(ctx, key, crypto.Keccak256Hash(data), configRegistryOpSetConfigWithData)
}

func (f *ConfigRegistryCommitFinalizer) verifyCommit(ctx context.Context, key [32]byte, checksum [32]byte, op string) (string, bool, error) {
	epoch, err := f.config.ConfigEpoch(&bind.CallOpts{Context: ctx}, key)
	if err != nil {
		return "", false, fmt.Errorf("read config epoch: %w", err)
	}
	if epoch == 0 {
		return "", false, nil
	}
	latest, err := f.config.LatestConfigChecksum(&bind.CallOpts{Context: ctx}, key)
	if err != nil {
		return "", false, fmt.Errorf("read latest config checksum: %w", err)
	}
	if latest != checksum {
		return "", false, nil
	}
	switch op {
	case configRegistryOpSetConfig:
		return f.lookupCommitTxID(ctx, key, checksum, op), true, nil
	case configRegistryOpSetConfigWithData:
		txID := f.lookupCommitTxID(ctx, key, checksum, op)
		if txID == "" {
			return "", false, nil
		}
		return txID, true, nil
	default:
		return "", false, fmt.Errorf("unsupported config registry op %q", op)
	}
}

func (f *ConfigRegistryCommitFinalizer) validatePackedNonce(ctx context.Context, got string) error {
	nonce, err := f.registry.Nonce(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return fmt.Errorf("read issuer nonce: %w", err)
	}
	if got != nonce.String() {
		return fmt.Errorf("packed expectedNonce %s != live %s", got, nonce)
	}
	return nil
}

func (f *ConfigRegistryCommitFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	p, key, checksum, data, nonce, err := f.parsePacked(packed)
	if err != nil {
		return [32]byte{}, err
	}
	return f.digestFromParsed(p, key, checksum, data, nonce)
}

func (f *ConfigRegistryCommitFinalizer) digestFromParsed(p configRegistryCommitPacked, key [32]byte, checksum [32]byte, data []byte, nonce *big.Int) ([32]byte, error) {
	switch p.Op {
	case configRegistryOpSetConfig:
		return ComputeConfigRegistrySetConfigDigest(f.chainID, f.registryAddr, f.issuerID, key, checksum, nonce), nil
	case configRegistryOpSetConfigWithData:
		return ComputeConfigRegistrySetConfigWithDataDigest(f.chainID, f.registryAddr, f.issuerID, key, data, nonce), nil
	default:
		return [32]byte{}, fmt.Errorf("unsupported config registry op %q", p.Op)
	}
}

func (f *ConfigRegistryCommitFinalizer) parsePacked(packed []byte) (configRegistryCommitPacked, [32]byte, [32]byte, []byte, *big.Int, error) {
	var p configRegistryCommitPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return p, [32]byte{}, [32]byte{}, nil, nil, fmt.Errorf("decode packed: %w", err)
	}
	if !strings.EqualFold(p.IssuerID, f.issuerID.Hex()) {
		return p, [32]byte{}, [32]byte{}, nil, nil, fmt.Errorf("packed issuerId %s != bound issuer %s", p.IssuerID, f.issuerID.Hex())
	}
	key, err := parseBytes32(p.Key)
	if err != nil {
		return p, [32]byte{}, [32]byte{}, nil, nil, err
	}
	nonce, ok := new(big.Int).SetString(p.ExpectedNonce, 10)
	if !ok {
		return p, [32]byte{}, [32]byte{}, nil, nil, fmt.Errorf("bad expectedNonce %q", p.ExpectedNonce)
	}
	switch p.Op {
	case configRegistryOpSetConfig:
		checksum, err := parseBytes32(p.Checksum)
		if err != nil {
			return p, [32]byte{}, [32]byte{}, nil, nil, err
		}
		return p, key, checksum, nil, nonce, nil
	case configRegistryOpSetConfigWithData:
		data, err := parseHexBytes(p.Data)
		if err != nil {
			return p, [32]byte{}, [32]byte{}, nil, nil, err
		}
		return p, key, crypto.Keccak256Hash(data), data, nonce, nil
	default:
		return p, [32]byte{}, [32]byte{}, nil, nil, fmt.Errorf("unsupported config registry op %q", p.Op)
	}
}

func (f *ConfigRegistryCommitFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, key [32]byte, checksum [32]byte, data []byte, nonce *big.Int, sigs [][]byte) error {
	abi, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	var calldata []byte
	if data == nil {
		calldata, err = abi.Pack("setConfig", f.issuerID, key, checksum, nonce, sigs)
	} else {
		calldata, err = abi.Pack("setConfigWithData", f.issuerID, key, data, nonce, sigs)
	}
	if err != nil {
		return fmt.Errorf("pack config registry calldata: %w", err)
	}
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.transactor.From(),
		To:        &f.registryAddr,
		Data:      calldata,
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

func (f *ConfigRegistryCommitFinalizer) lookupCommitTxID(ctx context.Context, key [32]byte, checksum [32]byte, op string) string {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		f.logger.Warn("config commit txID lookup: block number failed, returning empty txID",
			"issuerId", f.issuerID.Hex(), "key", hexBytes32(key), "error", err)
		return ""
	}
	var from uint64
	if head > f.lookupWindow {
		from = head - f.lookupWindow
	}

	switch op {
	case configRegistryOpSetConfig:
		it, err := f.registry.FilterConfigCommitted(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, []common.Address{f.issuerID}, [][32]byte{key})
		if err != nil {
			f.logger.Warn("config commit txID lookup: filter failed, returning empty txID",
				"issuerId", f.issuerID.Hex(), "key", hexBytes32(key), "error", err)
			return ""
		}
		defer it.Close()
		var last string
		for it.Next() {
			if it.Event.Checksum == checksum {
				last = it.Event.Raw.TxHash.Hex()
			}
		}
		return f.warnEmptyCommitTxID(last, key, checksum)
	case configRegistryOpSetConfigWithData:
		it, err := f.registry.FilterConfigWithDataCommitted(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, []common.Address{f.issuerID}, [][32]byte{key})
		if err != nil {
			f.logger.Warn("config-with-data commit txID lookup: filter failed, returning empty txID",
				"issuerId", f.issuerID.Hex(), "key", hexBytes32(key), "error", err)
			return ""
		}
		defer it.Close()
		var last string
		for it.Next() {
			if it.Event.Checksum == checksum {
				last = it.Event.Raw.TxHash.Hex()
			}
		}
		return f.warnEmptyCommitTxID(last, key, checksum)
	default:
		return ""
	}
}

func (f *ConfigRegistryCommitFinalizer) warnEmptyCommitTxID(txID string, key [32]byte, checksum [32]byte) string {
	if txID == "" {
		f.logger.Warn("config commit txID lookup: no matching event in window, returning empty txID",
			"issuerId", f.issuerID.Hex(), "key", hexBytes32(key), "checksum", hexBytes32(checksum), "window", f.lookupWindow)
	}
	return txID
}

// ConfigRegistryIssuerSettingsUpdateFinalizer rotates one issuer's registry key
// set via updateIssuerSettings, authorized by the current issuer quorum.
type ConfigRegistryIssuerSettingsUpdateFinalizer struct {
	client         *ethclient.Client
	registry       *ConfigRegistry
	registryAddr   common.Address
	issuerID       common.Address
	chainID        uint64
	authorizer     sign.Signer
	authorizerAddr common.Address
	transactor     Transactor
	fees           FeeConfig
	logger         log.Logger
	lookupWindow   uint64
}

func NewConfigRegistryIssuerSettingsUpdateFinalizer(ctx context.Context, client *ethclient.Client, registryAddr common.Address, issuerID common.Address, authorizer sign.Signer, payer Transactor, fees FeeConfig) (*ConfigRegistryIssuerSettingsUpdateFinalizer, error) {
	if payer == nil {
		return nil, fmt.Errorf("evm: transactor is required")
	}
	registry, err := NewConfigRegistry(registryAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load config registry: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(authorizer)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryIssuerSettingsUpdateFinalizer{
		client:         client,
		registry:       registry,
		registryAddr:   registryAddr,
		issuerID:       issuerID,
		chainID:        chainID.Uint64(),
		authorizer:     authorizer,
		authorizerAddr: addr,
		transactor:     payer,
		fees:           fees,
		logger:         log.NewNoopLogger(),
		lookupWindow:   defaultConfigLookupWindow,
	}, nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) SetLogger(l log.Logger) {
	if l == nil {
		l = log.NewNoopLogger()
	}
	f.logger = l
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) SetLookupWindow(blocks uint64) {
	if blocks == 0 {
		blocks = defaultConfigLookupWindow
	}
	f.lookupWindow = blocks
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) SignerAddress() common.Address {
	return f.authorizerAddr
}

type configRegistryIssuerSettingsUpdatePacked struct {
	Op            string   `json:"op"`
	IssuerID      string   `json:"issuerId"`
	NewIssuerKeys []string `json:"newIssuerKeys"`
	NewThreshold  int      `json:"newThreshold"`
	ExpectedNonce string   `json:"expectedNonce"`
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) Pack(ctx context.Context, newIssuerKeys []string, newThreshold int) ([]byte, error) {
	addrs, err := parseIssuerAddresses(newIssuerKeys)
	if err != nil {
		return nil, err
	}
	if err := validateThreshold(newThreshold, len(addrs), "issuer"); err != nil {
		return nil, err
	}
	nonce, err := f.registry.Nonce(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return nil, fmt.Errorf("read issuer nonce: %w", err)
	}
	return json.Marshal(configRegistryIssuerSettingsUpdatePacked{
		Op:            configRegistryOpUpdateIssuerSettings,
		IssuerID:      f.issuerID.Hex(),
		NewIssuerKeys: addrsToHex(addrs),
		NewThreshold:  newThreshold,
		ExpectedNonce: nonce.String(),
	})
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) Validate(ctx context.Context, packed []byte, newIssuerKeys []string, newThreshold int) error {
	var got configRegistryIssuerSettingsUpdatePacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	wantBytes, err := f.Pack(ctx, newIssuerKeys, newThreshold)
	if err != nil {
		return err
	}
	var want configRegistryIssuerSettingsUpdatePacked
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		return fmt.Errorf("decode wanted packed: %w", err)
	}
	if got.Op != want.Op || !strings.EqualFold(got.IssuerID, want.IssuerID) ||
		got.NewThreshold != want.NewThreshold || got.ExpectedNonce != want.ExpectedNonce ||
		!equalStrings(got.NewIssuerKeys, want.NewIssuerKeys) {
		return fmt.Errorf("packed issuer settings update does not match request")
	}
	return nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	digest, err := f.digestFromPacked(packed)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.authorizer, digest[:], f.authorizerAddr)
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) Submit(ctx context.Context, packed []byte, signatures [][]byte) (string, error) {
	p, addrs, threshold, nonce, err := f.parsePacked(packed)
	if err != nil {
		return "", err
	}
	if txID, done, err := f.VerifyUpdate(ctx, p.NewIssuerKeys, p.NewThreshold); err != nil {
		return "", err
	} else if done {
		return txID, nil
	}

	digest := ComputeConfigRegistryUpdateIssuerSettingsDigest(f.chainID, f.registryAddr, f.issuerID, addrs, threshold, nonce)
	liveKeys, liveThreshold, err := fetchLiveIssuerQuorum(ctx, f.registry, f.issuerID)
	if err != nil {
		return "", err
	}
	sigs, err := mergeQuorumSigs(common.Hash(digest), signatures, liveKeys, liveThreshold)
	if err != nil {
		return "", err
	}

	tx, err := f.transactor.Transact(ctx, f.fees, func(opts *bind.TransactOpts) (*gethtypes.Transaction, error) {
		if err := f.estimateGas(ctx, opts, addrs, threshold, nonce, sigs); err != nil {
			return nil, err
		}
		tx, err := f.registry.UpdateIssuerSettings(opts, f.issuerID, addrs, threshold, nonce, sigs)
		if err != nil {
			return nil, fmt.Errorf("updateIssuerSettings: %w", err)
		}
		return tx, nil
	})
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) VerifyUpdate(ctx context.Context, newIssuerKeys []string, newThreshold int) (string, bool, error) {
	addrs, err := parseIssuerAddresses(newIssuerKeys)
	if err != nil {
		return "", false, err
	}
	live, err := f.registry.IssuerKeys(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return "", false, fmt.Errorf("read issuer keys: %w", err)
	}
	thr, err := f.registry.Threshold(&bind.CallOpts{Context: ctx}, f.issuerID)
	if err != nil {
		return "", false, fmt.Errorf("read issuer threshold: %w", err)
	}
	if !thr.IsInt64() || int(thr.Int64()) != newThreshold || !addrSetEqual(live, addrs) {
		return "", false, nil
	}
	return f.lookupIssuerSettingsUpdatedTxID(ctx, addrs), true, nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) digestFromPacked(packed []byte) ([32]byte, error) {
	_, addrs, threshold, nonce, err := f.parsePacked(packed)
	if err != nil {
		return [32]byte{}, err
	}
	return ComputeConfigRegistryUpdateIssuerSettingsDigest(f.chainID, f.registryAddr, f.issuerID, addrs, threshold, nonce), nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) parsePacked(packed []byte) (configRegistryIssuerSettingsUpdatePacked, []common.Address, *big.Int, *big.Int, error) {
	var p configRegistryIssuerSettingsUpdatePacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return p, nil, nil, nil, fmt.Errorf("decode packed: %w", err)
	}
	if p.Op != configRegistryOpUpdateIssuerSettings {
		return p, nil, nil, nil, fmt.Errorf("unsupported config registry op %q", p.Op)
	}
	if !strings.EqualFold(p.IssuerID, f.issuerID.Hex()) {
		return p, nil, nil, nil, fmt.Errorf("packed issuerId %s != bound issuer %s", p.IssuerID, f.issuerID.Hex())
	}
	addrs, err := parseIssuerAddresses(p.NewIssuerKeys)
	if err != nil {
		return p, nil, nil, nil, err
	}
	if err := validateThreshold(p.NewThreshold, len(addrs), "issuer"); err != nil {
		return p, nil, nil, nil, err
	}
	nonce, ok := new(big.Int).SetString(p.ExpectedNonce, 10)
	if !ok {
		return p, nil, nil, nil, fmt.Errorf("bad expectedNonce %q", p.ExpectedNonce)
	}
	return p, addrs, big.NewInt(int64(p.NewThreshold)), nonce, nil
}

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, newIssuerKeys []common.Address, newThreshold *big.Int, nonce *big.Int, sigs [][]byte) error {
	abi, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	data, err := abi.Pack("updateIssuerSettings", f.issuerID, newIssuerKeys, newThreshold, nonce, sigs)
	if err != nil {
		return fmt.Errorf("pack updateIssuerSettings calldata: %w", err)
	}
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.transactor.From(),
		To:        &f.registryAddr,
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

func (f *ConfigRegistryIssuerSettingsUpdateFinalizer) lookupIssuerSettingsUpdatedTxID(ctx context.Context, addrs []common.Address) string {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		f.logger.Warn("issuer settings update txID lookup: block number failed, returning empty txID",
			"issuerId", f.issuerID.Hex(), "error", err)
		return ""
	}
	var from uint64
	if head > f.lookupWindow {
		from = head - f.lookupWindow
	}
	it, err := f.registry.FilterIssuerSettingsUpdated(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, []common.Address{f.issuerID})
	if err != nil {
		f.logger.Warn("issuer settings update txID lookup: filter failed, returning empty txID",
			"issuerId", f.issuerID.Hex(), "error", err)
		return ""
	}
	defer it.Close()
	var last string
	for it.Next() {
		if addrSetEqual(it.Event.NewIssuerKeys, addrs) {
			last = it.Event.Raw.TxHash.Hex()
		}
	}
	if last == "" {
		f.logger.Warn("issuer settings update txID lookup: no matching IssuerSettingsUpdated in window, returning empty txID",
			"issuerId", f.issuerID.Hex(), "window", f.lookupWindow)
	}
	return last
}

func fetchLiveIssuerQuorum(ctx context.Context, registry *ConfigRegistry, issuerID common.Address) ([]common.Address, int, error) {
	keys, err := registry.IssuerKeys(&bind.CallOpts{Context: ctx}, issuerID)
	if err != nil {
		return nil, 0, fmt.Errorf("read issuer keys: %w", err)
	}
	thr, err := registry.Threshold(&bind.CallOpts{Context: ctx}, issuerID)
	if err != nil {
		return nil, 0, fmt.Errorf("read issuer threshold: %w", err)
	}
	if !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(keys)) {
		return nil, 0, fmt.Errorf("on-chain issuer threshold %s out of range for %d keys", thr, len(keys))
	}
	return keys, int(thr.Int64()), nil
}

func parseRegistrationPacked(p configRegistryIssuerRegistrationPacked) ([]common.Address, *big.Int, error) {
	if p.Op != configRegistryOpRegisterIssuer {
		return nil, nil, fmt.Errorf("unsupported config registry op %q", p.Op)
	}
	addrs, err := parseIssuerAddresses(p.IssuerKeys)
	if err != nil {
		return nil, nil, err
	}
	if err := validateThreshold(p.Threshold, len(addrs), "issuer"); err != nil {
		return nil, nil, err
	}
	return addrs, big.NewInt(int64(p.Threshold)), nil
}

func parseIssuerAddresses(in []string) ([]common.Address, error) {
	addrs, err := parseSignerAddresses(in)
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if a == (common.Address{}) {
			return nil, fmt.Errorf("evm: zero issuer key")
		}
	}
	return addrs, nil
}

func validateThreshold(threshold, count int, label string) error {
	if threshold <= 0 || threshold > count {
		return fmt.Errorf("evm: %s threshold %d out of range for %d keys", label, threshold, count)
	}
	return nil
}

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
	raw, err := hex.DecodeString(h)
	if err != nil {
		return out, fmt.Errorf("evm: %q is not valid hex", s)
	}
	copy(out[:], raw)
	return out, nil
}

func parseHexBytes(s string) ([]byte, error) {
	h := s
	if len(h) >= 2 && (h[0:2] == "0x" || h[0:2] == "0X") {
		h = h[2:]
	}
	if len(h)%2 != 0 {
		return nil, fmt.Errorf("evm: %q is not even-length hex", s)
	}
	raw, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("evm: %q is not valid hex", s)
	}
	return raw, nil
}

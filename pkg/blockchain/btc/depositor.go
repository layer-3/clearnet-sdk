package btc

import (
	"context"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// These defaults apply only when the legacy custody Config leaves the
// corresponding depositor setting at zero. The public P2WPKHConfig remains
// explicit and does not apply defaults.
const (
	defaultDepositorMinConfirmations      uint64 = 1
	defaultDepositorFeeConfirmationTarget        = 6
	defaultDepositorFallbackFeeRate       int64  = 5
	defaultDepositorFeeCapSatPerVByte     int64  = 1_000
	defaultDepositorDustThresholdSats     int64  = 330
	defaultDepositorMaxInputs                    = 100
)

// Depositor funds a per-account deposit address from the depositor's own
// P2WPKH wallet (the key the supplied sign.Signer holds). It implements
// core.VaultDepositor. The deposit address is derived from the vault's pubkeys
// + threshold (the same address the withdrawal finalizer can later spend).
type Depositor struct {
	net          *chaincfg.Params
	backend      DepositorBackend
	sender       *P2WPKHSender
	vaultPubkeys [][]byte
	threshold    int
	assets       blockchain.AssetResolver
}

var _ core.VaultDepositor = (*Depositor)(nil)

// DepositorBackend supplies the chain access needed to submit and observe BTC
// deposits without prescribing Bitcoin Core, Esplora, wallet ownership, or a
// transport. GetTransactionConfirmations reports whether txID is known in the
// mempool or active chain and, when known, its current confirmation count. The
// Depositor owns the core.VaultDepositor status and minimum-depth semantics.
type DepositorBackend interface {
	P2WPKHBackend
	GetTransactionConfirmations(ctx context.Context, txID string) (confirmations uint64, known bool, err error)
}

// NewDepositor builds the BTC depositor. signer is the depositor's secp256k1
// key; vaultPubkeys + threshold define the vault whose per-account deposit
// addresses funds are sent to. For backward compatibility, zero legacy Config
// values are normalized for the funding sender to one confirmation, a six-block
// fee target, a 5 sat/vB fallback, and a 1000 sat/vB fee cap. Deposits also use
// a fixed 330-satoshi change threshold and a 100-input limit; those policies do
// not borrow the P2WSH-only MaxInputsPerWithdrawal setting.
func NewDepositor(net *chaincfg.Params, rpc RPC, signer sign.Signer, vaultPubkeys [][]byte, threshold int, cfg Config, assets blockchain.AssetResolver) (*Depositor, error) {
	if assets == nil {
		return nil, fmt.Errorf("btc: asset resolver is required")
	}
	if nilInterface(rpc) {
		return nil, fmt.Errorf("btc: depositor RPC is required")
	}
	senderCfg := depositorP2WPKHConfig(cfg)
	backend := newLegacyCoreP2WPKHBackend(rpc, senderCfg.FallbackFeeRateSatPerVByte)
	return NewDepositorWithBackend(net, backend, signer, vaultPubkeys, threshold, senderCfg, assets)
}

func depositorP2WPKHConfig(cfg Config) P2WPKHConfig {
	minConfirmations := cfg.ConfirmationDepth
	if minConfirmations == 0 {
		minConfirmations = defaultDepositorMinConfirmations
	}
	feeTarget := cfg.FeeConfTarget
	if feeTarget == 0 {
		feeTarget = defaultDepositorFeeConfirmationTarget
	}
	fallbackRate := cfg.FallbackFeeRate
	if fallbackRate == 0 {
		fallbackRate = defaultDepositorFallbackFeeRate
	}
	feeCap := cfg.FeeCapSatPerVByte
	if feeCap == 0 {
		feeCap = defaultDepositorFeeCapSatPerVByte
	}
	return P2WPKHConfig{
		MinConfirmations:           minConfirmations,
		FeeConfirmationTarget:      feeTarget,
		FallbackFeeRateSatPerVByte: fallbackRate,
		FeeCapSatPerVByte:          feeCap,
		DustThresholdSats:          defaultDepositorDustThresholdSats,
		MaxInputs:                  defaultDepositorMaxInputs,
	}
}

// NewDepositorWithBackend builds a BTC depositor using transport-independent
// chain access. Unlike the legacy RPC constructor, cfg is explicit: callers
// must provide a fully populated P2WPKHConfig accepted by NewP2WPKHSender.
// backend owns funding transaction submission and transport-neutral transaction
// observation; Depositor applies deposit verification semantics so the returned
// value fully implements core.VaultDepositor.
func NewDepositorWithBackend(net *chaincfg.Params, backend DepositorBackend, signer sign.Signer, vaultPubkeys [][]byte, threshold int, cfg P2WPKHConfig, assets blockchain.AssetResolver) (*Depositor, error) {
	if assets == nil {
		return nil, fmt.Errorf("btc: asset resolver is required")
	}
	if nilInterface(backend) {
		return nil, fmt.Errorf("btc: depositor backend is required")
	}
	sender, err := NewP2WPKHSender(net, backend, signer, cfg)
	if err != nil {
		return nil, fmt.Errorf("btc: create depositor sender: %w", err)
	}
	return &Depositor{
		net:          net,
		backend:      backend,
		sender:       sender,
		vaultPubkeys: vaultPubkeys,
		threshold:    threshold,
		assets:       assets,
	}, nil
}

// DepositorAddress returns the depositor's own P2WPKH funding address.
func (d *Depositor) DepositorAddress() string { return d.sender.Address() }

func normalizeDepositAssetAddress(assetAddress string) string {
	if strings.TrimSpace(assetAddress) == "" {
		return nativeAssetAddress
	}
	return assetAddress
}

// SubmitDeposit sends amount from the depositor's wallet to the per-account
// deposit address for dest.Account. assetAddress must be "" for native BTC.
// Builds, signs (P2WPKH), and broadcasts the funding tx. A
// non-zero dest.Ref is rejected: the account is encoded in the deposit address
// and a plain BTC send has no side-data channel for a sub-account (ADR-015 has
// no BTC reference).
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination) (string, error) {
	assetAddress = normalizeDepositAssetAddress(assetAddress)
	if err := d.assets.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return "", err
	}
	if dest.Ref != ([32]byte{}) {
		return "", fmt.Errorf("btc: deposit reference not supported")
	}
	if amount.Sign() <= 0 {
		return "", fmt.Errorf("btc: amount %s not positive", amount.String())
	}
	decimals, err := d.assets.AssetDecimals(ctx, assetAddress)
	if err != nil {
		return "", err
	}
	baseUnits, err := blockchain.DecimalToBaseUnits(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("btc: amount: %w", err)
	}
	if !baseUnits.IsInt64() || baseUnits.Int64() <= 0 {
		return "", fmt.Errorf("btc: amount %s not a positive int64 satoshi value", amount.String())
	}
	sats := baseUnits.Int64()

	depositAddr, _, err := DepositAddress(dest.Account, d.threshold, d.vaultPubkeys, d.net)
	if err != nil {
		return "", fmt.Errorf("btc: derive deposit address: %w", err)
	}
	return d.sender.Send(ctx, depositAddr.EncodeAddress(), sats)
}

// VerifyDeposit reports the backend's on-chain status for the deposit txID.
// The legacy RPC backend requires the node to resolve the tx (txindex=1, or the
// tx unspent / in the mempool). A tx it has never seen — or one reorged out and
// dropped — reads as DepositAbsent; a mempool tx (0 confs) is DepositPending
// until it is mined with at least max(1, minConf) confirmations (a deposit is
// only Confirmed once on chain, consistent with the other chains).
func (d *Depositor) VerifyDeposit(ctx context.Context, txID string, minConf uint64) (core.DepositStatus, error) {
	confirmations, known, err := d.backend.GetTransactionConfirmations(ctx, txID)
	if err != nil {
		return core.DepositAbsent, err
	}
	if !known {
		return core.DepositAbsent, nil
	}
	required := minConf
	if required == 0 {
		required = 1
	}
	if confirmations >= required {
		return core.DepositConfirmed, nil
	}
	return core.DepositPending, nil
}

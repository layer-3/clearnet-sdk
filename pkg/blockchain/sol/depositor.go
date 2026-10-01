package sol

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor moves funds into the custody vault on Solana, signed by the
// depositor's own ed25519 key. It implements core.VaultDepositor. Native SOL
// and SPL tokens are both supported. The deposit credits the 20-byte clearnet
// account encoded in `account` (hex).
type Depositor struct {
	client       *rpc.Client
	programID    solana.PublicKey
	vaultPDA     solana.PublicKey
	eventAuth    solana.PublicKey
	signer       sign.Signer
	depositorPub solana.PublicKey
	commitment   rpc.CommitmentType
	assets       blockchain.AssetResolver
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor builds the Solana depositor over the JSON-RPC at rpcURL. signer
// is the depositor's ed25519 key (it pays + funds). commitment is the level the
// deposit tx's blockhash + preflight use; empty → CommitmentFinalized (see the
// NOTE on the withdrawal finalizer's Config.Commitment for the test tradeoff).
func NewDepositor(rpcURL string, programID solana.PublicKey, signer sign.Signer, commitment rpc.CommitmentType, assets blockchain.AssetResolver) (*Depositor, error) {
	if assets == nil {
		return nil, fmt.Errorf("sol: asset resolver is required")
	}
	pub, err := solanaPub(signer)
	if err != nil {
		return nil, err
	}
	if commitment == "" {
		commitment = rpc.CommitmentFinalized
	}
	return &Depositor{
		client:       rpc.New(rpcURL),
		programID:    programID,
		vaultPDA:     VaultPDA(programID),
		eventAuth:    eventAuthorityPDA(programID),
		signer:       signer,
		depositorPub: pub,
		commitment:   commitment,
		assets:       assets,
	}, nil
}

// DepositorAddress returns the depositor's Solana address.
func (d *Depositor) DepositorAddress() string { return d.depositorPub.String() }

func normalizeDepositAssetAddress(assetAddress string) string {
	if strings.TrimSpace(assetAddress) == "" {
		return nativeAssetAddress
	}
	return assetAddress
}

// SubmitDeposit transfers amount of assetAddress into the vault,
// crediting clearnet dest.Account (20-byte hex) with the optional ADR-015
// dest.Ref sub-account reference. assetAddress is "" for native or a base58
// mint.
//
// The transaction carries one deposit instruction, so DepositID is
// DepositID(signature, 0). TxHash is the base58 signature.
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination) (core.SubmitDepositResult, error) {
	acct, err := parseClearnetAccount(dest.Account)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	assetAddress = normalizeDepositAssetAddress(assetAddress)
	if err := d.assets.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return core.SubmitDepositResult{}, err
	}
	if amount.Sign() <= 0 {
		return core.SubmitDepositResult{}, fmt.Errorf("sol: amount %s not positive", amount.String())
	}
	decimals, err := d.assets.AssetDecimals(ctx, assetAddress)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	baseUnits, err := blockchain.DecimalToBaseUnits(amount, decimals)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("sol: amount: %w", err)
	}
	if !baseUnits.IsUint64() {
		return core.SubmitDepositResult{}, fmt.Errorf("sol: amount %s overflows uint64 base units", amount.String())
	}
	lamports := baseUnits.Uint64()
	mint, err := resolveMint(assetAddress)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}

	var ix solana.Instruction
	if mint.IsZero() {
		ix, err = custody.NewDepositSolInstruction(
			acct, dest.Ref, lamports,
			d.depositorPub, d.vaultPDA, solana.SystemProgramID, d.eventAuth, d.programID,
		)
	} else {
		depositorATA, _, e := solana.FindAssociatedTokenAddress(d.depositorPub, mint)
		if e != nil {
			return core.SubmitDepositResult{}, fmt.Errorf("sol: depositor ATA: %w", e)
		}
		vaultATA, _, e := solana.FindAssociatedTokenAddress(d.vaultPDA, mint)
		if e != nil {
			return core.SubmitDepositResult{}, fmt.Errorf("sol: vault ATA: %w", e)
		}
		ix, err = custody.NewDepositSplInstruction(
			acct, dest.Ref, lamports,
			d.depositorPub, mint, depositorATA, d.vaultPDA, vaultATA,
			solana.TokenProgramID, solana.SPLAssociatedTokenAccountProgramID, d.eventAuth, d.programID,
		)
	}
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("sol: build deposit ix: %w", err)
	}

	sig, err := signAndSend(ctx, d.client, []solana.Instruction{ix}, d.depositorPub, d.signer, d.commitment, solana.PublicKey{})
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	return core.SubmitDepositResult{TxHash: txID(sig), DepositID: DepositID(sig, 0)}, nil
}

// ChainDepositStatus reports the on-chain status of the deposit identified by
// txHash (the base58 signature) and depositId (DepositID(signature, index)).
// minConf maps onto Solana's commitment ladder, which has no numeric depth:
// minConf 0 accepts "confirmed" (~1-2 slots), minConf >= 1 requires
// "finalized". A failed tx, or one whose event at index is not a vault
// Deposited event, reads as DepositAbsent (it credited nothing). A confirmed
// transaction the RPC does not serve yet reads as DepositPending. Custody's
// crediting rules are not applied, so DepositConfirmed does not guarantee a
// credit.
func (d *Depositor) ChainDepositStatus(ctx context.Context, txHash, depositId string, minConf uint64) (core.DepositStatus, error) {
	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return core.DepositAbsent, fmt.Errorf("sol: bad signature %q: %w", txHash, err)
	}
	index, err := parseDepositID(sig, depositId)
	if err != nil {
		return core.DepositAbsent, err
	}
	out, err := d.client.GetSignatureStatuses(ctx, true, sig)
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return core.DepositAbsent, nil
		}
		return core.DepositAbsent, fmt.Errorf("sol: signature status: %w", err)
	}
	if len(out.Value) == 0 || out.Value[0] == nil {
		return core.DepositAbsent, nil
	}
	st := out.Value[0]
	if st.Err != nil {
		return core.DepositAbsent, nil
	}
	var status core.DepositStatus
	switch st.ConfirmationStatus {
	case rpc.ConfirmationStatusFinalized:
		status = core.DepositConfirmed
	case rpc.ConfirmationStatusConfirmed:
		status = core.DepositPending
		if minConf == 0 {
			status = core.DepositConfirmed
		}
	default:
		return core.DepositPending, nil
	}

	maxVersion := uint64(0)
	res, err := d.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return core.DepositPending, nil
		}
		return core.DepositAbsent, fmt.Errorf("sol: get transaction: %w", err)
	}
	if res == nil || res.Meta == nil || res.Transaction == nil {
		return core.DepositPending, nil
	}
	if res.Meta.Err != nil {
		return core.DepositAbsent, nil
	}
	tx, err := res.Transaction.GetTransaction()
	if err != nil {
		return core.DepositAbsent, fmt.Errorf("sol: decode transaction: %w", err)
	}
	if !depositEventAt(tx, res.Meta, d.programID, index) {
		return core.DepositAbsent, nil
	}
	return status, nil
}

// parseClearnetAccount decodes a 20-byte clearnet account address from a
// bare hex, an optional case-insensitive "0x" prefix, or a yellow://.../user/<hex>
// URI's last segment, and surrounding whitespace).
func parseClearnetAccount(account string) ([20]byte, error) {
	acct, err := core.ParseClearnetAccount(account)
	if err != nil {
		return [20]byte{}, fmt.Errorf("sol: %w", err)
	}
	return acct, nil
}

func txID(sig solana.Signature) string {
	return sig.String()
}

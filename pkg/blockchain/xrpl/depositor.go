package xrpl

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor sends a Payment from the depositor's account (the key the
// sign.Signer holds) to the vault, crediting a clearnet account via a
// `ynet-account` memo (a 20-byte account followed by a 32-byte ADR-015
// reference). It implements core.VaultDepositor. Native XRP and issued
// currencies ("CUR.rIssuer") are both supported.
type Depositor struct {
	client       *rpc.Client
	vaultAddress string
	signer       sign.Signer
	id           Identity
	assets       blockchain.AssetResolver
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor builds the XRPL depositor against the rippled JSON-RPC at rpcURL.
func NewDepositor(rpcURL, vaultAddress string, signer sign.Signer, assets blockchain.AssetResolver) (*Depositor, error) {
	if assets == nil {
		return nil, fmt.Errorf("xrpl: asset resolver is required")
	}
	client, err := newRPCClient(rpcURL)
	if err != nil {
		return nil, err
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &Depositor{client: client, vaultAddress: vaultAddress, signer: signer, id: id, assets: assets}, nil
}

// DepositorAddress returns the depositor's classic r-address.
func (d *Depositor) DepositorAddress() string { return d.id.ClassicAddress }

func normalizeDepositAssetAddress(assetAddress string) string {
	if strings.TrimSpace(assetAddress) == "" {
		return nativeAssetAddress
	}
	return assetAddress
}

// SubmitDeposit sends amount of assetAddress to the vault, crediting
// dest.Account via a `ynet-account` memo carrying the 20-byte account and the
// 32-byte ADR-015 dest.Ref. assetAddress is "" for native or "CUR.rIssuer" for
// an issued currency. TxHash is the upper-case transaction hash and DepositID
// is DepositID(TxHash).
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount decimal.Decimal, dest core.DepositDestination) (core.SubmitDepositResult, error) {
	memo, err := accountMemo(dest)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	assetAddress = normalizeDepositAssetAddress(assetAddress)
	if err := d.assets.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return core.SubmitDepositResult{}, err
	}
	if amount.Sign() <= 0 {
		return core.SubmitDepositResult{}, fmt.Errorf("xrpl: amount must be positive")
	}
	decimals, err := d.assets.AssetDecimals(ctx, assetAddress)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	baseUnits, err := blockchain.DecimalToBaseUnits(amount, decimals)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("xrpl: amount: %w", err)
	}
	xrplAmount, err := currencyAmountFromBaseUnits(assetAddress, baseUnits, decimals)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}

	payment := transaction.Payment{
		BaseTx: transaction.BaseTx{
			Account: types.Address(d.id.ClassicAddress),
			Memos:   []types.MemoWrapper{memo},
		},
		Destination: types.Address(d.vaultAddress),
		Amount:      xrplAmount,
	}
	flatTx := payment.Flatten()
	if err := ensureNetworkID(d.client); err != nil {
		return core.SubmitDepositResult{}, err
	}
	if err := d.client.Autofill(&flatTx); err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("xrpl: autofill: %w", err)
	}

	blob, err := signSingle(ctx, d.signer, d.id, flatTx)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	hash, err := computeTxHash(blob)
	if err != nil {
		return core.SubmitDepositResult{}, err
	}
	result, err := d.client.SubmitTxBlob(blob, false)
	if err != nil {
		return core.SubmitDepositResult{}, fmt.Errorf("xrpl: submit: %w", err)
	}
	switch result.EngineResult {
	case "tesSUCCESS", "terQUEUED":
		txHash := hashHex(hash)
		return core.SubmitDepositResult{TxHash: txHash, DepositID: DepositID(txHash)}, nil
	default:
		return core.SubmitDepositResult{}, fmt.Errorf("xrpl: deposit rejected: %s - %s", result.EngineResult, result.EngineResultMessage)
	}
}

// accountMemoType is the MemoType (as plain text, hex-encoded on the wire)
// that marks the ynet-account memo carrying the deposit destination.
const accountMemoType = "ynet-account"

// accountMemo builds the ynet-account memo: MemoData is the 20-byte clearnet
// account followed by the 32-byte ADR-015 reference (zero for no sub-account),
// hex-encoded; MemoType is "ynet-account", hex-encoded. The account is
// decoded from a bare hex, an optional case-insensitive "0x" prefix, or a
// yellow://.../user/<hex> URI's last segment, and surrounding whitespace).
func accountMemo(dest core.DepositDestination) (types.MemoWrapper, error) {
	account, err := core.ParseClearnetAccount(dest.Account)
	if err != nil {
		return types.MemoWrapper{}, fmt.Errorf("xrpl: %w", err)
	}
	data := append(account[:], dest.Ref[:]...)
	return types.MemoWrapper{Memo: types.Memo{
		MemoType: hex.EncodeToString([]byte(accountMemoType)),
		MemoData: hex.EncodeToString(data),
	}}, nil
}

// ChainDepositStatus reports the on-chain status of the deposit identified by
// txHash and depositId (DepositID(txHash)). XRPL finality is binary, so minConf
// is not a depth. The deposit is DepositAbsent for an unknown hash, a
// transaction that is not a Payment from another account to the vault carrying
// a ynet-account memo with a non-zero account, or a validated transaction whose
// result is not tesSUCCESS. Otherwise a validated tx is DepositConfirmed and an
// unvalidated one DepositPending. Custody's crediting rules (partial payments,
// asset support) are not applied, so DepositConfirmed does not guarantee a
// credit.
func (d *Depositor) ChainDepositStatus(_ context.Context, txHash, depositId string, _ uint64) (core.DepositStatus, error) {
	if depositId != DepositID(txHash) {
		return core.DepositAbsent, fmt.Errorf("xrpl: deposit ID %q is not the lower-cased transaction hash %s", depositId, txHash)
	}
	res, err := d.client.Request(&transactions.TxRequest{Transaction: txHash})
	if err != nil {
		if strings.Contains(err.Error(), "txnNotFound") {
			return core.DepositAbsent, nil
		}
		return core.DepositAbsent, fmt.Errorf("xrpl: tx lookup: %w", err)
	}
	var tx transactions.TxResponse
	if err := res.GetResult(&tx); err != nil {
		return core.DepositAbsent, fmt.Errorf("xrpl: decode tx: %w", err)
	}
	return depositStatus(&tx, d.vaultAddress), nil
}

func depositStatus(tx *transactions.TxResponse, vaultAddress string) core.DepositStatus {
	if !IsVaultDepositPayment(tx.TxJSON, vaultAddress) || !hasDepositMemo(tx.TxJSON["Memos"]) {
		return core.DepositAbsent
	}
	if !tx.Validated {
		return core.DepositPending
	}
	if tx.Meta.TransactionResult != "tesSUCCESS" {
		return core.DepositAbsent
	}
	return core.DepositConfirmed
}

// hasDepositMemo reports whether memosRaw (the decoded Memos field) carries a
// deposit memo naming a non-zero account.
func hasDepositMemo(memosRaw any) bool {
	account, _, ok := ParseDepositMemo(memosRaw)
	return ok && account != [20]byte{}
}

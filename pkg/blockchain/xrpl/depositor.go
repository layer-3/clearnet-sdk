package xrpl

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

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
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor builds the XRPL depositor against the rippled JSON-RPC at rpcURL.
func NewDepositor(rpcURL, vaultAddress string, signer sign.Signer) (*Depositor, error) {
	client, err := newRPCClient(rpcURL)
	if err != nil {
		return nil, err
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &Depositor{client: client, vaultAddress: vaultAddress, signer: signer, id: id}, nil
}

// DepositorAddress returns the depositor's classic r-address.
func (d *Depositor) DepositorAddress() string { return d.id.ClassicAddress }

// SubmitDeposit sends amount base units of assetAddress to the vault, crediting
// dest.Account via a `ynet-account` memo carrying the 20-byte account and the
// 32-byte ADR-015 dest.Ref. assetAddress is "" / "XRP" for native or
// "CUR.rIssuer" for an issued currency; issued-currency generic deposits are
// integer-valued by this interface.
func (d *Depositor) SubmitDeposit(ctx context.Context, assetAddress string, amount *big.Int, dest core.DepositDestination) (core.TxRef, error) {
	memo, err := accountMemo(dest)
	if err != nil {
		return core.TxRef{}, err
	}
	if amount == nil || amount.Sign() <= 0 {
		return core.TxRef{}, fmt.Errorf("xrpl: amount must be positive")
	}
	xrplAmount, err := currencyAmount(assetAddress, decimal.NewFromBigInt(amount, 0))
	if err != nil {
		return core.TxRef{}, err
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
		return core.TxRef{}, err
	}
	if err := d.client.Autofill(&flatTx); err != nil {
		return core.TxRef{}, fmt.Errorf("xrpl: autofill: %w", err)
	}

	blob, err := signSingle(ctx, d.signer, d.id, flatTx)
	if err != nil {
		return core.TxRef{}, err
	}
	hash, err := computeTxHash(blob)
	if err != nil {
		return core.TxRef{}, err
	}
	result, err := d.client.SubmitTxBlob(blob, false)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("xrpl: submit: %w", err)
	}
	switch result.EngineResult {
	case "tesSUCCESS", "terQUEUED":
		return core.TxRef{Hash: hash, Raw: hashHex(hash)}, nil
	default:
		return core.TxRef{}, fmt.Errorf("xrpl: deposit rejected: %s - %s", result.EngineResult, result.EngineResultMessage)
	}
}

// accountMemoType is the MemoType (as plain text, hex-encoded on the wire)
// that marks the ynet-account memo carrying the deposit destination.
const accountMemoType = "ynet-account"

// accountMemo builds the ynet-account memo: MemoData is the 20-byte clearnet
// account followed by the 32-byte ADR-015 reference (zero for no sub-account),
// hex-encoded; MemoType is "ynet-account", hex-encoded.
func accountMemo(dest core.DepositDestination) (types.MemoWrapper, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(dest.Account), "0x")
	account, err := hex.DecodeString(raw)
	if err != nil {
		return types.MemoWrapper{}, fmt.Errorf("xrpl: account not hex: %w", err)
	}
	if len(account) != 20 {
		return types.MemoWrapper{}, fmt.Errorf("xrpl: account must be 20 bytes, got %d", len(account))
	}
	data := append(account, dest.Ref[:]...)
	return types.MemoWrapper{Memo: types.Memo{
		MemoType: hex.EncodeToString([]byte(accountMemoType)),
		MemoData: hex.EncodeToString(data),
	}}, nil
}

// VerifyDeposit reports the on-chain status of the deposit tx in ref (matched by
// hash, ref.Raw). XRPL finality is binary — a validated transaction cannot be
// reorged — so minConf is not a depth here: a validated tx is DepositConfirmed,
// one found but not yet validated is DepositPending, and an unknown hash
// (never submitted, or dropped before validation) is DepositAbsent.
func (d *Depositor) VerifyDeposit(_ context.Context, ref core.TxRef, _ uint64) (core.DepositStatus, error) {
	res, err := d.client.Request(&transactions.TxRequest{Transaction: ref.Raw})
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
	if tx.Validated {
		return core.DepositConfirmed, nil
	}
	return core.DepositPending, nil
}

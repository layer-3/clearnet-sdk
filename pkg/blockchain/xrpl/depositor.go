package xrpl

import (
	"context"
	"fmt"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor sends a tagged Payment from the depositor's account (the key the
// sign.Signer holds) to the vault, crediting a clearnet account via the
// DestinationTag. It implements core.VaultDepositor. Native XRP and issued
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
	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("xrpl: create rpc config: %w", err)
	}
	id, err := DeriveIdentity(signer)
	if err != nil {
		return nil, err
	}
	return &Depositor{client: rpc.NewClient(cfg), vaultAddress: vaultAddress, signer: signer, id: id}, nil
}

// DepositorAddress returns the depositor's classic r-address.
func (d *Depositor) DepositorAddress() string { return d.id.ClassicAddress }

// SubmitDeposit sends `amount` of `asset` to the vault, crediting `account` via its
// DestinationTag. asset is "" / "XRP" for native or "CUR.rIssuer" for an issued
// currency; account must be of the form xrpl-<tag> (the tag the watcher credits).
func (d *Depositor) SubmitDeposit(ctx context.Context, asset string, amount decimal.Decimal, account string) (core.TxRef, error) {
	tag, err := parseDepositTag(account)
	if err != nil {
		return core.TxRef{}, err
	}
	xrplAmount, err := currencyAmount(asset, amount)
	if err != nil {
		return core.TxRef{}, err
	}

	payment := transaction.Payment{
		BaseTx:      transaction.BaseTx{Account: types.Address(d.id.ClassicAddress)},
		Destination: types.Address(d.vaultAddress),
		Amount:      xrplAmount,
	}
	flatTx := payment.Flatten()
	flatTx["DestinationTag"] = tag
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

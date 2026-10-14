package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func validOperationID(id string) bool { return eip712.ValidTransferOperationID(id) }

// PrepareTransfer resolves a live nonce and converts human decimal amounts to
// base units, but does not sign, submit, or choose a fee ceiling for the user.
func (c *Client) PrepareTransfer(ctx context.Context, signer sign.Signer, recipient string, assets []TransferAmount, maxFeeYELLOW string) (*TransferIntent, error) {
	if signer == nil {
		return nil, fmt.Errorf("transfer signer is required")
	}
	address, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	if _, err := eip712.TransferRecipientAddress(recipient); err != nil {
		return nil, err
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	if !info.HasCapability(CapabilityTransferStatus) {
		return nil, fmt.Errorf("gateway cannot resolve uncertain transfer submissions")
	}
	metadata := make(map[string]AssetInfo, len(info.Assets))
	for _, asset := range info.Assets {
		metadata[asset.Symbol] = asset
	}
	yellow, ok := metadata["YELLOW"]
	if !ok || maxFeeYELLOW == "" {
		return nil, fmt.Errorf("an explicit YELLOW-denominated fee ceiling is required")
	}
	fee, err := AmountInBaseUnits(maxFeeYELLOW, yellow.Decimals)
	if err != nil {
		return nil, fmt.Errorf("maxFee: %w", err)
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("at least one transfer asset is required")
	}
	legs := make([]TransferAsset, len(assets))
	seen := make(map[string]bool, len(assets))
	for i, input := range assets {
		asset, ok := metadata[input.Asset]
		if !ok || seen[input.Asset] {
			return nil, fmt.Errorf("unknown or duplicate transfer asset %q", input.Asset)
		}
		seen[input.Asset] = true
		units, err := AmountInBaseUnits(input.Amount, asset.Decimals)
		if err != nil || units.Sign() <= 0 {
			return nil, fmt.Errorf("invalid transfer amount for %s: %v", input.Asset, err)
		}
		legs[i] = TransferAsset{Asset: input.Asset, Amount: units.String()}
	}
	sort.Slice(legs, func(i, j int) bool { return legs[i].Asset < legs[j].Asset })
	balance, err := c.Balance(ctx, address.Hex())
	if err != nil {
		return nil, err
	}
	return &TransferIntent{
		NetworkID: info.NetworkID, ChainID: info.ChainID, From: address.Hex(), To: recipient,
		Assets: legs, MaxFee: fee.String(), Nonce: balance.Nonce,
	}, nil
}

func transferFields(intent TransferIntent) ([]eip712.TransferAsset, *big.Int, error) {
	if _, err := eip712.TransferRecipientAddress(intent.To); err != nil {
		return nil, nil, err
	}
	if !common.IsHexAddress(intent.From) || common.HexToAddress(intent.From) == (common.Address{}) {
		return nil, nil, fmt.Errorf("invalid transfer sender")
	}
	fee, err := parseBaseUnitInteger(intent.MaxFee)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid transfer maxFee: %w", err)
	}
	if len(intent.Assets) == 0 {
		return nil, nil, fmt.Errorf("transfer assets required")
	}
	assets := make([]eip712.TransferAsset, len(intent.Assets))
	for i, leg := range intent.Assets {
		if i > 0 && intent.Assets[i-1].Asset >= leg.Asset {
			return nil, nil, fmt.Errorf("transfer assets must be sorted and unique")
		}
		amount, err := parseBaseUnitInteger(leg.Amount)
		if err != nil || amount.Sign() <= 0 || leg.Asset == "" {
			return nil, nil, fmt.Errorf("invalid transfer asset %d amount", i)
		}
		assets[i] = eip712.TransferAsset{Asset: leg.Asset, Amount: amount}
	}
	return assets, fee, nil
}

func parseBaseUnitInteger(raw string) (*big.Int, error) {
	if raw == "" {
		return nil, fmt.Errorf("integer is required")
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return nil, fmt.Errorf("integer must contain only decimal digits")
		}
	}
	v, _ := new(big.Int).SetString(raw, 10)
	if v == nil || v.BitLen() > 256 {
		return nil, fmt.Errorf("integer exceeds uint256")
	}
	return v, nil
}

// SignTransfer binds the exact prepared nonce, recipient, asset legs, fee cap
// and expected network. The returned intent is an independent copy so later
// mutation of the caller's preparation cannot alter its signed payload.
func (c *Client) SignTransfer(ctx context.Context, signer sign.Signer, prepared *TransferIntent) (*SignedTransfer, error) {
	if signer == nil || prepared == nil {
		return nil, fmt.Errorf("signer and prepared transfer required")
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	if !info.HasCapability(CapabilityTransferStatus) || prepared.NetworkID != info.NetworkID || prepared.ChainID != info.ChainID {
		return nil, fmt.Errorf("prepared transfer network changed or lacks operation status")
	}
	from, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(prepared.From, from.Hex()) {
		return nil, fmt.Errorf("prepared transfer signer does not match the selected account")
	}
	intent := *prepared
	intent.Assets = append([]TransferAsset(nil), prepared.Assets...)
	assets, fee, err := transferFields(intent)
	if err != nil {
		return nil, err
	}
	sig, id, err := eip712.SignTransfer(ctx, signer, new(big.Int).SetUint64(intent.ChainID), intent.To, assets, fee, intent.Nonce)
	if err != nil {
		return nil, err
	}
	return &SignedTransfer{OperationID: id, Intent: intent, Signature: "0x" + hex.EncodeToString(sig)}, nil
}

// SubmitTransfer submits an already-signed request unchanged. There is no
// automatic POST retry: an HTTP timeout may happen after network inclusion.
func (c *Client) SubmitTransfer(ctx context.Context, signed *SignedTransfer) (*SubmitResult, error) {
	if signed == nil || !validOperationID(signed.OperationID) {
		return nil, fmt.Errorf("valid signed transfer and operation ID required")
	}
	id := signed.OperationID
	info, err := c.Info(ctx)
	if err != nil {
		return nil, &SubmissionError{OperationID: id, Err: err}
	}
	if !info.HasCapability(CapabilityTransferStatus) || signed.Intent.NetworkID != info.NetworkID || signed.Intent.ChainID != info.ChainID {
		return nil, &SubmissionError{OperationID: id, Err: fmt.Errorf("gateway network or transfer status capability changed")}
	}
	assets, fee, err := transferFields(signed.Intent)
	if err != nil {
		return nil, &SubmissionError{OperationID: id, Err: err}
	}
	if len(signed.Signature) != 132 || !strings.HasPrefix(signed.Signature, "0x") {
		return nil, &SubmissionError{OperationID: id, Err: fmt.Errorf("invalid transfer signature length")}
	}
	sig, err := hex.DecodeString(signed.Signature[2:])
	if err != nil {
		return nil, &SubmissionError{OperationID: id, Err: err}
	}
	to, _ := eip712.TransferRecipientAddress(signed.Intent.To)
	chainID := new(big.Int).SetUint64(info.ChainID)
	computed, err := eip712.TransferOperationID(chainID, common.HexToAddress(signed.Intent.From), signed.Intent.To, assets, fee, signed.Intent.Nonce)
	if err != nil || computed != id {
		return nil, &SubmissionError{OperationID: id, Err: fmt.Errorf("signed transfer operation ID does not match its fields")}
	}
	recovered, err := eip712.RecoverTransfer(chainID, to, assets, fee, signed.Intent.Nonce, sig)
	if err != nil || !strings.EqualFold(recovered.Hex(), signed.Intent.From) {
		return nil, &SubmissionError{OperationID: id, Err: fmt.Errorf("signed transfer does not match its sender or payload")}
	}
	body := struct {
		To        string          `json:"to"`
		Assets    []TransferAsset `json:"assets"`
		MaxFee    string          `json:"maxFee"`
		Nonce     uint64          `json:"nonce"`
		Signature string          `json:"signature"`
	}{signed.Intent.To, signed.Intent.Assets, signed.Intent.MaxFee, signed.Intent.Nonce, signed.Signature}
	var raw json.RawMessage
	status, err := c.request(ctx, http.MethodPost, "/v1/transfers", body, &raw)
	if err != nil {
		unknown := true
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			switch apiErr.HTTPStatus {
			case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusTooManyRequests:
				unknown = false
			}
		}
		return nil, &SubmissionError{Submitted: true, OperationID: id, OutcomeUnknown: unknown, Err: err}
	}
	if status == http.StatusAccepted {
		var pending OperationStatus
		if err := json.Unmarshal(raw, &pending); err != nil || pending.OperationID != id || pending.Status != "pending" {
			return nil, &SubmissionError{Submitted: true, OperationID: id, OutcomeUnknown: true, Err: fmt.Errorf("invalid pending transfer response")}
		}
		return &SubmitResult{OperationID: id, Status: "pending"}, nil
	}
	var receipt Receipt
	if err := json.Unmarshal(raw, &receipt); err != nil || receipt.OperationID != id || receipt.BlockHash == "" {
		return nil, &SubmissionError{Submitted: true, OperationID: id, OutcomeUnknown: true, Err: fmt.Errorf("invalid verified transfer receipt")}
	}
	return &SubmitResult{OperationID: id, Status: "sealed", Receipt: &receipt}, nil
}

// Package client is the public, user-facing Clearnet gateway client. It
// exposes typed REST reads and prepared EIP-712 transfers without importing
// node, validator, or operator tooling from the clearing implementation.
package client

import (
	"encoding/json"
	"fmt"
)

const (
	CapabilityTransferStatus  = "transfer-operation-status-v1"
	CapabilityRecentTransfers = "recent-transfers-v1"
)

type GatewayContracts struct {
	Custody  string `json:"custody"`
	Registry string `json:"registry"`
	Token    string `json:"token"`
	Faucet   string `json:"faucet"`
}

type AssetInfo struct {
	Symbol   string `json:"symbol"`
	ChainID  uint64 `json:"chainId"`
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
	Native   bool   `json:"native"`
}

type NetworkInfo struct {
	ChainID       uint64           `json:"chainId"`
	NetworkID     string           `json:"networkId"`
	RPCURL        string           `json:"rpcUrl"`
	L1ChainID     uint64           `json:"l1ChainId"`
	Confirmations uint64           `json:"confirmations"`
	Contracts     GatewayContracts `json:"contracts"`
	Assets        []AssetInfo      `json:"assets"`
	Capabilities  []string         `json:"capabilities"`
	VirtualRouter string           `json:"virtualRouter"`
	Version       string           `json:"version"`
}

func (i NetworkInfo) HasCapability(name string) bool {
	for _, capability := range i.Capabilities {
		if capability == name {
			return true
		}
	}
	return false
}

type BalanceEntry struct {
	Asset             string `json:"asset"`
	Symbol            string `json:"symbol"`
	L1Balance         string `json:"l1Balance,omitempty"`
	ClearnetBalance   string `json:"clearnetBalance"`
	PendingWithdrawal string `json:"pendingWithdrawal,omitempty"`
	FeeRevenue        string `json:"feeRevenue,omitempty"`
}

type Balance struct {
	Address     string                       `json:"address"`
	AccountType string                       `json:"accountType"`
	Nonce       uint64                       `json:"nonce"`
	Balances    []BalanceEntry               `json:"balances"`
	Codes       map[string]map[string]string `json:"codes"`
}

type TransferAsset struct {
	Asset  string `json:"asset"`
	Amount string `json:"amount"` // integer asset base units
}

// TransferAmount is an input in human-readable decimal asset units; the SDK
// converts it exactly using decimals discovered from the configured gateway.
type TransferAmount struct {
	Asset  string
	Amount string
}

type TransferIntent struct {
	NetworkID string          `json:"networkId"`
	ChainID   uint64          `json:"chainId"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Assets    []TransferAsset `json:"assets"`
	MaxFee    string          `json:"maxFee"` // integer YELLOW wei
	Nonce     uint64          `json:"nonce"`
}

// SignedTransfer is a spend authorization. Treat its signature and serialized
// form as sensitive; persist it safely before submitting if retry recovery is
// required. Never rebuild it at a fresh nonce after an unknown outcome.
type SignedTransfer struct {
	OperationID string         `json:"operationId"`
	Intent      TransferIntent `json:"intent"`
	Signature   string         `json:"signature"`
}

// Receipt is a verified seal. The operation ID is required: SubmitTransfer and
// operation reconciliation refuse a receipt not bound to the signed intent.
// Other gateways must supply it even though the production gateway's own JSON
// struct uses omitempty (its sealed response always fills the field).
type Receipt struct {
	BlockHash   string `json:"blockHash"`
	EntryIndex  uint64 `json:"entryIndex"`
	SealedAt    int64  `json:"sealedAt"`
	StateRoot   string `json:"stateRoot"`
	EntryHash   string `json:"entryHash,omitempty"`
	OperationID string `json:"operationId"`
}

type OperationStatus struct {
	OperationID string `json:"operationId"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	// PendingUntil is the original gateway admission window end, in Unix seconds.
	// Zero means the gateway has not supplied evidence that replay is safe.
	PendingUntil int64    `json:"pendingUntil,omitempty"`
	Receipt      *Receipt `json:"receipt,omitempty"`
}

type SubmitResult struct {
	OperationID string
	Status      string // sealed or pending; pending does not guarantee execution
	Receipt     *Receipt
}

type TransferActivity struct {
	OperationID string          `json:"operationId,omitempty"`
	BlockHash   string          `json:"blockHash"`
	EntryHash   string          `json:"entryHash"`
	EntryIndex  uint64          `json:"entryIndex"`
	SealedAt    int64           `json:"sealedAt"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	Direction   string          `json:"direction"`
	Delivery    string          `json:"delivery"` // inbound credit remains unverified
	Nonce       uint64          `json:"nonce"`
	Assets      []TransferAsset `json:"assets"`
}

type TransferPage struct {
	Items         []TransferActivity `json:"items"`
	NextCursor    *string            `json:"nextCursor,omitempty"`
	RetentionDays int                `json:"retentionDays"`
}

type TransferPageOptions struct {
	Limit  int
	Cursor string
}

// APIError is a definite gateway HTTP error only when OutcomeUnknown is false
// on its enclosing SubmissionError. Numeric codes match the REST error body.
type APIError struct {
	HTTPStatus int
	Code       int
	Message    string
	Field      string
	Data       json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gateway HTTP %d (code %d): %s", e.HTTPStatus, e.Code, e.Message)
}

// SubmissionError always carries the precomputed operation ID. An unknown
// outcome is NOT retryable merely because it arose from a network timeout.
type SubmissionError struct {
	// Submitted records whether the POST was attempted. False establishes that
	// this call failed during preflight; it says nothing about earlier submissions.
	Submitted      bool
	OperationID    string
	OutcomeUnknown bool
	Err            error
}

func (e *SubmissionError) Error() string {
	return fmt.Sprintf("transfer %s: %v", e.OperationID, e.Err)
}

func (e *SubmissionError) Unwrap() error { return e.Err }

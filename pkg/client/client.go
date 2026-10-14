package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

const maxResponseBytes = 2 << 20

type Option func(*Client)

// WithHTTPClient installs a caller-owned transport; nil keeps the default
// bounded transport. The SDK never writes private keys to this transport.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithInsecureHTTP permits an explicitly trusted remote plaintext endpoint.
// Loopback HTTP is allowed for local development without this option.
func WithInsecureHTTP() Option { return func(c *Client) { c.allowInsecure = true } }

type Client struct {
	baseURL       string
	expectedNet   string
	http          *http.Client
	allowInsecure bool
}

// New configures an HTTP gateway client scoped to one expected network.
// No I/O is performed until Info or another method is called.
func New(baseURL, expectedNetworkID string, options ...Option) (*Client, error) {
	if len(expectedNetworkID) != 4 || eip712.NetworkChainID(expectedNetworkID).Sign() == 0 {
		return nil, fmt.Errorf("expected four-byte Clearnet network ID is required")
	}
	c := &Client{expectedNet: expectedNetworkID, http: &http.Client{Timeout: 15 * time.Second}}
	for _, option := range options {
		if option != nil {
			option(c)
		}
	}
	u, err := url.Parse(baseURL)
	if err != nil || u == nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("gateway URL must be an absolute HTTP(S) URL without credentials or query")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("gateway URL must use HTTP(S)")
	}
	if u.Scheme == "http" && !c.allowInsecure && !isLoopbackHost(u.Hostname()) {
		return nil, fmt.Errorf("remote HTTP gateway requires explicit WithInsecureHTTP (HTTPS recommended)")
	}
	c.baseURL = strings.TrimRight(u.String(), "/")
	return c, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) request(ctx context.Context, method, path string, body any, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return resp.StatusCode, err
	}
	if len(raw) > maxResponseBytes {
		return resp.StatusCode, fmt.Errorf("gateway response exceeds %d bytes", maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var wire struct {
			Error struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Field   string          `json:"field"`
				Data    json.RawMessage `json:"data"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &wire) != nil || wire.Error.Message == "" {
			return resp.StatusCode, &APIError{HTTPStatus: resp.StatusCode, Message: "unrecognized gateway error"}
		}
		return resp.StatusCode, &APIError{
			HTTPStatus: resp.StatusCode, Code: wire.Error.Code,
			Message: wire.Error.Message, Field: wire.Error.Field, Data: wire.Error.Data,
		}
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode gateway response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// Info returns self-described network and asset metadata, rejecting a gateway
// whose EIP-712 domain or virtual router differs from the expected deployment.
func (c *Client) Info(ctx context.Context) (*NetworkInfo, error) {
	var info NetworkInfo
	if _, err := c.request(ctx, http.MethodGet, "/v1/gateway/info", nil, &info); err != nil {
		return nil, err
	}
	if info.NetworkID != c.expectedNet || info.ChainID != eip712.NetworkChainID(c.expectedNet).Uint64() {
		return nil, fmt.Errorf("gateway network identity mismatch: expected %s, got %s/%d", c.expectedNet, info.NetworkID, info.ChainID)
	}
	if !strings.EqualFold(info.VirtualRouter, eip712.RouterHex) {
		return nil, fmt.Errorf("gateway virtual router does not match the EIP-712 deployment")
	}
	seen := make(map[string]bool, len(info.Assets))
	for _, asset := range info.Assets {
		if asset.Symbol == "" || seen[asset.Symbol] {
			return nil, fmt.Errorf("gateway asset catalog has an empty or duplicated symbol")
		}
		seen[asset.Symbol] = true
	}
	sort.Slice(info.Assets, func(i, j int) bool { return info.Assets[i].Symbol < info.Assets[j].Symbol })
	return &info, nil
}

func normalizeUserAccount(account string) (string, error) {
	if core.IsURI(account) {
		if err := core.ValidateURI(account); err != nil {
			return "", err
		}
		_, kind, path, err := core.ParseURI(account)
		if err != nil || kind != "user" || !common.IsHexAddress(path) {
			return "", fmt.Errorf("account must be a user address or canonical user URI")
		}
		account = path
	}
	if !common.IsHexAddress(account) || common.HexToAddress(account) == (common.Address{}) {
		return "", fmt.Errorf("account must be a nonzero Ethereum address")
	}
	return common.HexToAddress(account).Hex(), nil
}

func (c *Client) Balance(ctx context.Context, account string) (*Balance, error) {
	address, err := normalizeUserAccount(account)
	if err != nil {
		return nil, err
	}
	if _, err := c.Info(ctx); err != nil {
		return nil, err
	}
	var balance Balance
	if _, err := c.request(ctx, http.MethodGet, "/v1/accounts/"+url.PathEscape(address)+"/balances", nil, &balance); err != nil {
		return nil, err
	}
	return &balance, nil
}

func (c *Client) Transfers(ctx context.Context, account string, options TransferPageOptions) (*TransferPage, error) {
	address, err := normalizeUserAccount(account)
	if err != nil {
		return nil, err
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	if !info.HasCapability(CapabilityRecentTransfers) {
		return nil, fmt.Errorf("gateway does not support recent account transfers")
	}
	if options.Limit < 0 || options.Limit > 100 {
		return nil, fmt.Errorf("transfer page limit must be between 1 and 100")
	}
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", fmt.Sprint(options.Limit))
	}
	if options.Cursor != "" {
		query.Set("cursor", options.Cursor)
	}
	path := "/v1/accounts/" + url.PathEscape(address) + "/transfers"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var page TransferPage
	if _, err := c.request(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	if page.Items == nil {
		page.Items = []TransferActivity{}
	}
	return &page, nil
}

// Operation checks one gateway's locally persisted intent against verified
// blocks. A different gateway or an incomplete index may report unknown;
// callers must not treat that as evidence a fresh nonce is safe.
func (c *Client) Operation(ctx context.Context, operationID string) (*OperationStatus, error) {
	if !validOperationID(operationID) {
		return nil, fmt.Errorf("operation ID must be a lowercase 0x-prefixed 32-byte hash")
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	if !info.HasCapability(CapabilityTransferStatus) {
		return nil, fmt.Errorf("gateway does not support transfer operation status")
	}
	var status OperationStatus
	if _, err := c.request(ctx, http.MethodGet, "/v1/operations/"+operationID, nil, &status); err != nil {
		return nil, err
	}
	if status.OperationID != operationID || status.Kind != "transfer" {
		return nil, fmt.Errorf("gateway operation response does not match the requested ID or kind")
	}
	switch status.Status {
	case "pending", "sealed", "rejected", "unknown":
	default:
		return nil, fmt.Errorf("gateway returned an unsupported operation status %q", status.Status)
	}
	return &status, nil
}

// AmountInBaseUnits converts a human decimal asset amount exactly, without
// floats or silent rounding. Zero is allowed for fee ceilings, not transfers.
func AmountInBaseUnits(amount string, decimals uint8) (*big.Int, error) {
	if amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	parts := strings.Split(amount, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return nil, fmt.Errorf("invalid decimal amount")
	}
	for _, part := range parts {
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return nil, fmt.Errorf("amount must contain only decimal digits")
			}
		}
	}
	if len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > int(decimals)) {
		return nil, fmt.Errorf("amount has too many fractional digits for asset")
	}
	whole, _ := new(big.Int).SetString(parts[0], 10)
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	result := new(big.Int).Mul(whole, base)
	if len(parts) == 2 {
		fraction, _ := new(big.Int).SetString(parts[1], 10)
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(int(decimals)-len(parts[1]))), nil)
		result.Add(result, fraction.Mul(fraction, scale))
	}
	if result.BitLen() > 256 {
		return nil, fmt.Errorf("amount exceeds uint256")
	}
	return result, nil
}

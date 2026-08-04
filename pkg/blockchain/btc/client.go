package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// maxRPCResponseBytes is a defensive memory bound: 8 MiB leaves room for occasional
// scantxoutset responses while preventing an untrusted or misconfigured endpoint
// from allocating without bound.
const maxRPCResponseBytes int64 = 8 << 20

// Client is a concrete Bitcoin Core JSON-RPC client implementing RPC.
// Wallet-scoped calls (ListUnspent) route to /wallet/<wallet>; the rest hit the
// configured node endpoint. Wallet provisioning, mining, and funding remain
// deliberately out of scope.
type Client struct {
	url    string
	wallet string
	user   string
	pass   string
	http   *http.Client
}

var _ RPC = (*Client)(nil)

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default *http.Client (30s timeout). The client
// is used unchanged, allowing callers to set operation-specific timeouts and
// transports. A nil client is ignored.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// NewClient builds a Bitcoin Core RPC client at url with Basic authentication.
// wallet is the wallet name used by wallet-scoped RPCs and may be empty for the
// default loaded wallet.
func NewClient(url, user, pass, wallet string, opts ...Option) *Client {
	c := &Client{
		url:    strings.TrimRight(url, "/"),
		wallet: wallet,
		user:   user,
		pass:   pass,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// RPCError is a typed Bitcoin Core JSON-RPC error response. Code lets callers
// classify protocol outcomes without parsing the rendered error text.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("bitcoind rpc error %d: %s", e.Code, e.Message)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

func (c *Client) post(ctx context.Context, endpoint, method string, params any, out any) error {
	body, err := json.Marshal(rpcRequest{JSONRPC: "1.0", ID: "sdk", Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("btc rpc %s: marshal request: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("btc rpc %s: construct request: %w", method, err)
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("btc rpc %s: transport: %w", method, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRPCResponseBytes+1))
	if err != nil {
		return fmt.Errorf("btc rpc %s: read response: %w", method, err)
	}
	if int64(len(responseBody)) > maxRPCResponseBytes {
		return fmt.Errorf("btc rpc %s: read response: body exceeds %d bytes", method, maxRPCResponseBytes)
	}

	var env rpcEnvelope
	decodeErr := json.Unmarshal(responseBody, &env)
	if decodeErr == nil && env.Error != nil {
		return fmt.Errorf("btc rpc %s: JSON-RPC response: %w", method, env.Error)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("btc rpc %s: HTTP status %s", method, resp.Status)
	}
	if decodeErr != nil {
		return fmt.Errorf("btc rpc %s: decode response envelope: %w", method, decodeErr)
	}
	if out != nil {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return fmt.Errorf("btc rpc %s: decode result: %w", method, err)
		}
	}
	return nil
}

func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	return c.post(ctx, c.url, method, params, out)
}

func (c *Client) walletCall(ctx context.Context, method string, params any, out any) error {
	endpoint := c.url + "/wallet/" + url.PathEscape(c.wallet)
	return c.post(ctx, endpoint, method, params, out)
}

// CoreChain is the chain identifier reported by Bitcoin Core's
// getblockchaininfo RPC. It intentionally uses Core's protocol vocabulary.
type CoreChain string

const (
	CoreChainMain     CoreChain = "main"
	CoreChainTest     CoreChain = "test"
	CoreChainTestnet4 CoreChain = "testnet4"
	CoreChainSignet   CoreChain = "signet"
	CoreChainRegtest  CoreChain = "regtest"
)

// GetBlockchainInfo calls getblockchaininfo and returns its validated Core
// chain identifier.
func (c *Client) GetBlockchainInfo(ctx context.Context) (CoreChain, error) {
	var result struct {
		Chain CoreChain `json:"chain"`
	}
	if err := c.call(ctx, "getblockchaininfo", []any{}, &result); err != nil {
		return "", err
	}
	switch result.Chain {
	case CoreChainMain, CoreChainTest, CoreChainTestnet4, CoreChainSignet, CoreChainRegtest:
		return result.Chain, nil
	default:
		return "", fmt.Errorf("btc rpc getblockchaininfo: unsupported chain identifier %q", result.Chain)
	}
}

// ScanTxOutSetResult is the typed result of a synchronous scantxoutset scan.
type ScanTxOutSetResult struct {
	Success         bool
	TxOuts          uint64
	Height          uint64
	BestBlock       string
	Unspents        []ScanTxOutSetUnspent
	TotalAmountSats int64
}

// ScanTxOutSetUnspent is one output returned by scantxoutset. ScriptPubKey is
// decoded from Core's validated hexadecimal representation. Height is zero
// when Core omits it; Confirmations is derived when both heights are usable.
type ScanTxOutSetUnspent struct {
	TxID          string
	Vout          uint32
	ScriptPubKey  []byte
	Descriptor    string
	AmountSats    int64
	Height        uint64
	Confirmations uint64
	Coinbase      bool
}

// ErrScanAlreadyInProgress classifies Core's competing-scan response. The
// returned error also wraps the original *RPCError.
var ErrScanAlreadyInProgress = errors.New("btc: scantxoutset scan already in progress")

type scanAlreadyInProgressError struct {
	rpc *RPCError
}

func (e *scanAlreadyInProgressError) Error() string {
	return fmt.Sprintf("%s: %v", ErrScanAlreadyInProgress, e.rpc)
}

func (e *scanAlreadyInProgressError) Unwrap() []error {
	return []error{ErrScanAlreadyInProgress, e.rpc}
}

// ScanTxOutSet synchronously scans the UTXO set for the supplied Core output
// descriptors, for example []string{"addr(bcrt1...)"}. It does not retry,
// abort another scan, or apply confirmation policy.
func (c *Client) ScanTxOutSet(ctx context.Context, descriptors []string) (*ScanTxOutSetResult, error) {
	if len(descriptors) == 0 {
		return nil, fmt.Errorf("btc rpc scantxoutset: at least one descriptor is required")
	}
	for i, descriptor := range descriptors {
		if strings.TrimSpace(descriptor) == "" {
			return nil, fmt.Errorf("btc rpc scantxoutset: descriptor %d is empty", i)
		}
	}

	var raw struct {
		Success     bool        `json:"success"`
		TxOuts      uint64      `json:"txouts"`
		Height      uint64      `json:"height"`
		BestBlock   string      `json:"bestblock"`
		TotalAmount json.Number `json:"total_amount"`
		Unspents    []struct {
			TxID         string      `json:"txid"`
			Vout         uint32      `json:"vout"`
			ScriptPubKey string      `json:"scriptPubKey"`
			Descriptor   string      `json:"desc"`
			Amount       json.Number `json:"amount"`
			Height       uint64      `json:"height"`
			Coinbase     bool        `json:"coinbase"`
		} `json:"unspents"`
	}
	if err := c.call(ctx, "scantxoutset", []any{"start", descriptors}, &raw); err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -8 && isScanAlreadyInProgressMessage(rpcErr.Message) {
			return nil, &scanAlreadyInProgressError{rpc: rpcErr}
		}
		return nil, err
	}

	total, err := btcJSONNumberToSats(raw.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("btc rpc scantxoutset: total_amount: %w", err)
	}
	if raw.BestBlock != "" {
		if err := validateHashString(raw.BestBlock); err != nil {
			return nil, fmt.Errorf("btc rpc scantxoutset: bestblock: %w", err)
		}
	}
	result := &ScanTxOutSetResult{
		Success:         raw.Success,
		TxOuts:          raw.TxOuts,
		Height:          raw.Height,
		BestBlock:       raw.BestBlock,
		TotalAmountSats: total,
		Unspents:        make([]ScanTxOutSetUnspent, len(raw.Unspents)),
	}
	for i, unspent := range raw.Unspents {
		if err := validateHashString(unspent.TxID); err != nil {
			return nil, fmt.Errorf("btc rpc scantxoutset: unspent %d txid: %w", i, err)
		}
		amount, err := btcJSONNumberToSats(unspent.Amount)
		if err != nil {
			return nil, fmt.Errorf("btc rpc scantxoutset: unspent %d amount: %w", i, err)
		}
		script, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return nil, fmt.Errorf("btc rpc scantxoutset: unspent %d scriptPubKey: %w", i, err)
		}
		var confirmations uint64
		if unspent.Height > 0 && raw.Height >= unspent.Height {
			confirmations = raw.Height - unspent.Height + 1
		}
		result.Unspents[i] = ScanTxOutSetUnspent{
			TxID:          unspent.TxID,
			Vout:          unspent.Vout,
			ScriptPubKey:  script,
			Descriptor:    unspent.Descriptor,
			AmountSats:    amount,
			Height:        unspent.Height,
			Confirmations: confirmations,
			Coinbase:      unspent.Coinbase,
		}
	}
	return result, nil
}

func isScanAlreadyInProgressMessage(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "scan") &&
		strings.Contains(message, "already") &&
		strings.Contains(message, "progress")
}

// ListUnspent returns the vault UTXOs at addrs with at least minConf confirmations.
func (c *Client) ListUnspent(ctx context.Context, minConf int, addrs []string) ([]Unspent, error) {
	var raw []struct {
		TxID          string      `json:"txid"`
		Vout          uint32      `json:"vout"`
		Amount        json.Number `json:"amount"`
		Confirmations int64       `json:"confirmations"`
		ScriptPubKey  string      `json:"scriptPubKey"`
	}
	if err := c.walletCall(ctx, "listunspent", []any{minConf, 9999999, addrs}, &raw); err != nil {
		return nil, err
	}
	out := make([]Unspent, len(raw))
	for i, u := range raw {
		amount, err := btcJSONNumberToSats(u.Amount)
		if err != nil {
			return nil, fmt.Errorf("btc rpc listunspent: result %d amount: %w", i, err)
		}
		out[i] = Unspent{TxID: u.TxID, Vout: u.Vout, AmountSats: amount, Confirmations: u.Confirmations, ScriptPubKey: u.ScriptPubKey}
	}
	return out, nil
}

// GetTxOut returns the unspent output txid:vout, or nil if it is spent/unknown.
func (c *Client) GetTxOut(ctx context.Context, txid string, vout uint32, includeMempool bool) (*TxOut, error) {
	var raw *struct {
		Confirmations int64       `json:"confirmations"`
		Value         json.Number `json:"value"`
		ScriptPubKey  struct {
			Hex string `json:"hex"`
		} `json:"scriptPubKey"`
	}
	if err := c.call(ctx, "gettxout", []any{txid, vout, includeMempool}, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	amount, err := btcJSONNumberToSats(raw.Value)
	if err != nil {
		return nil, fmt.Errorf("btc rpc gettxout: value: %w", err)
	}
	return &TxOut{AmountSats: amount, ScriptPubKey: raw.ScriptPubKey.Hex, Confirmations: raw.Confirmations}, nil
}

// SendRawTransaction broadcasts hexTx and returns its txid.
func (c *Client) SendRawTransaction(ctx context.Context, hexTx string) (string, error) {
	var txid string
	if err := c.call(ctx, "sendrawtransaction", []any{hexTx}, &txid); err != nil {
		return "", err
	}
	if err := validateHashString(txid); err != nil {
		return "", fmt.Errorf("btc rpc sendrawtransaction: decode result: txid: %w", err)
	}
	return txid, nil
}

// EstimateSmartFeeSatPerVByte returns Core's estimate rounded upward to an
// integer sat/vB. A valid response with no estimate uses fallbackRate;
// transport, HTTP, JSON-RPC, and malformed-result failures are returned.
func (c *Client) EstimateSmartFeeSatPerVByte(ctx context.Context, confTarget int, fallbackRate int64) (int64, error) {
	var raw struct {
		FeeRate *json.Number `json:"feerate"`
		Errors  []string     `json:"errors"`
	}
	if err := c.call(ctx, "estimatesmartfee", []any{confTarget}, &raw); err != nil {
		return 0, err
	}
	if raw.FeeRate == nil {
		for _, message := range raw.Errors {
			if strings.TrimSpace(message) != "" {
				if fallbackRate <= 0 {
					return 0, fmt.Errorf("btc rpc estimatesmartfee: unavailable estimate has non-positive fallback rate %d", fallbackRate)
				}
				return fallbackRate, nil
			}
		}
		return 0, fmt.Errorf("btc rpc estimatesmartfee: decode result: missing feerate without an unavailable-estimate error")
	}
	rate, err := feeRateBTCPerKVByteToSatPerVByte(*raw.FeeRate)
	if err != nil {
		return 0, fmt.Errorf("btc rpc estimatesmartfee: feerate: %w", err)
	}
	return rate, nil
}

// GetBlockCount returns the height of the most-work fully-validated chain.
func (c *Client) GetBlockCount(ctx context.Context) (int64, error) {
	var height *int64
	if err := c.call(ctx, "getblockcount", []any{}, &height); err != nil {
		return 0, err
	}
	if height == nil || *height < 0 {
		return 0, fmt.Errorf("btc rpc getblockcount: decode result: invalid height")
	}
	return *height, nil
}

// GetBlockHash returns the block hash at the given height.
func (c *Client) GetBlockHash(ctx context.Context, height int64) (string, error) {
	var h string
	if err := c.call(ctx, "getblockhash", []any{height}, &h); err != nil {
		return "", err
	}
	if err := validateHashString(h); err != nil {
		return "", fmt.Errorf("btc rpc getblockhash: decode result: block hash: %w", err)
	}
	return h, nil
}

// GetBlockTxids returns the txids in the block (verbosity 1).
func (c *Client) GetBlockTxids(ctx context.Context, blockHash string) ([]string, error) {
	var raw struct {
		Tx []string `json:"tx"`
	}
	if err := c.call(ctx, "getblock", []any{blockHash, 1}, &raw); err != nil {
		return nil, err
	}
	return raw.Tx, nil
}

// GetRawTransaction returns the decoded transaction (verbose=true). Requires
// txindex=1, or for the transaction to be unspent/in the mempool.
func (c *Client) GetRawTransaction(ctx context.Context, txid string) (*RawTx, error) {
	var raw *struct {
		TxID          string `json:"txid"`
		Confirmations int64  `json:"confirmations"`
		Vout          []struct {
			Value        json.Number `json:"value"`
			ScriptPubKey struct {
				Hex string `json:"hex"`
			} `json:"scriptPubKey"`
		} `json:"vout"`
	}
	if err := c.call(ctx, "getrawtransaction", []any{txid, true}, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	out := &RawTx{TxID: raw.TxID, Confirmations: raw.Confirmations, Vouts: make([]RawVout, len(raw.Vout))}
	for i, vout := range raw.Vout {
		amount, err := btcJSONNumberToSats(vout.Value)
		if err != nil {
			return nil, fmt.Errorf("btc rpc getrawtransaction: vout %d value: %w", i, err)
		}
		out.Vouts[i] = RawVout{ValueSats: amount, ScriptPubKeyHex: vout.ScriptPubKey.Hex}
	}
	return out, nil
}

func btcJSONNumberToSats(number json.Number) (int64, error) {
	return exactDecimalToInt64(number, 8, false)
}

func feeRateBTCPerKVByteToSatPerVByte(number json.Number) (int64, error) {
	return exactDecimalToInt64(number, 5, true)
}

func exactDecimalToInt64(number json.Number, shift int32, roundUp bool) (int64, error) {
	text := number.String()
	if text == "" {
		return 0, fmt.Errorf("missing decimal value")
	}
	value, err := decimal.NewFromString(text)
	if err != nil {
		return 0, fmt.Errorf("invalid decimal %q: %w", text, err)
	}
	if value.IsNegative() {
		return 0, fmt.Errorf("decimal %q is negative", text)
	}
	// Bound work performed by IsInteger/BigInt for hostile scientific notation.
	scaledExponent := int64(value.Exponent()) + int64(shift)
	if scaledExponent < -256 || scaledExponent > 64 {
		return 0, fmt.Errorf("decimal %q is outside the int64 range", text)
	}
	scaled := value.Shift(shift)
	if roundUp {
		scaled = scaled.Ceil()
	} else if !scaled.IsInteger() {
		return 0, fmt.Errorf("decimal %q has sub-satoshi precision", text)
	}
	integer := scaled.BigInt()
	if !integer.IsInt64() {
		return 0, fmt.Errorf("decimal %q is outside the int64 range", text)
	}
	result := integer.Int64()
	if roundUp && result <= 0 {
		return 0, fmt.Errorf("decimal %q produces a non-positive fee rate", text)
	}
	return result, nil
}

func validateHashString(hash string) error {
	if len(hash) != chainhash.MaxHashStringSize {
		return fmt.Errorf("expected %d hexadecimal characters, got %d", chainhash.MaxHashStringSize, len(hash))
	}
	if _, err := chainhash.NewHashFromStr(hash); err != nil {
		return err
	}
	return nil
}

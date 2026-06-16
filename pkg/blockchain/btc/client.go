package btc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is a concrete bitcoind JSON-RPC client implementing RPC. Wallet-scoped
// calls (ListUnspent) route to /wallet/<wallet>; the rest hit the node root.
// It carries only the read + broadcast surface the adapters need — wallet
// provisioning (createwallet, mining, funding) is deliberately out of scope.
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

// WithHTTPClient overrides the default *http.Client (30s timeout).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient builds a bitcoind RPC client at url with basic-auth user/pass.
// wallet is the wallet name wallet-scoped RPCs route to (may be empty if the
// node has a single default wallet loaded).
func NewClient(url, user, pass, wallet string, opts ...Option) *Client {
	c := &Client{
		url:    url,
		wallet: wallet,
		user:   user,
		pass:   pass,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RPCError is a typed bitcoind JSON-RPC error response. The Code lets callers
// branch on outcome (e.g. already-in-chain) without string matching.
type RPCError struct {
	Code    int
	Message string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("bitcoind rpc error %d: %s", e.Code, e.Message)
}

func (c *Client) post(ctx context.Context, endpoint, method string, params []any, out any) error {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "1.0", "id": "sdk", "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("btc rpc %s: decode response: %w", method, err)
	}
	if env.Error != nil {
		return &RPCError{Code: env.Error.Code, Message: env.Error.Message}
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

func (c *Client) call(ctx context.Context, method string, params []any, out any) error {
	return c.post(ctx, c.url, method, params, out)
}

func (c *Client) walletCall(ctx context.Context, method string, params []any, out any) error {
	return c.post(ctx, c.url+"/wallet/"+c.wallet, method, params, out)
}

// ListUnspent returns the vault UTXOs at addrs with at least minConf confirmations.
func (c *Client) ListUnspent(ctx context.Context, minConf int, addrs []string) ([]Unspent, error) {
	var raw []struct {
		TxID          string  `json:"txid"`
		Vout          uint32  `json:"vout"`
		Amount        float64 `json:"amount"`
		Confirmations int64   `json:"confirmations"`
		ScriptPubKey  string  `json:"scriptPubKey"`
	}
	if err := c.walletCall(ctx, "listunspent", []any{minConf, 9999999, addrs}, &raw); err != nil {
		return nil, err
	}
	out := make([]Unspent, len(raw))
	for i, u := range raw {
		out[i] = Unspent{TxID: u.TxID, Vout: u.Vout, AmountSats: btcToSats(u.Amount), Confirmations: u.Confirmations, ScriptPubKey: u.ScriptPubKey}
	}
	return out, nil
}

// GetTxOut returns the unspent output txid:vout, or nil if it is spent/unknown.
func (c *Client) GetTxOut(ctx context.Context, txid string, vout uint32, includeMempool bool) (*TxOut, error) {
	var raw *struct {
		Confirmations int64   `json:"confirmations"`
		Value         float64 `json:"value"`
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
	return &TxOut{AmountSats: btcToSats(raw.Value), ScriptPubKey: raw.ScriptPubKey.Hex, Confirmations: raw.Confirmations}, nil
}

// SendRawTransaction broadcasts hexTx and returns its txid.
func (c *Client) SendRawTransaction(ctx context.Context, hexTx string) (string, error) {
	var txid string
	return txid, c.call(ctx, "sendrawtransaction", []any{hexTx}, &txid)
}

// EstimateSmartFeeSatPerVByte returns the node's fee estimate for confTarget
// blocks, falling back to fallbackRate when the node cannot estimate (e.g. on
// regtest).
func (c *Client) EstimateSmartFeeSatPerVByte(ctx context.Context, confTarget int, fallbackRate int64) (int64, error) {
	var raw struct {
		FeeRate float64 `json:"feerate"`
	}
	if err := c.call(ctx, "estimatesmartfee", []any{confTarget}, &raw); err != nil || raw.FeeRate <= 0 {
		return fallbackRate, nil
	}
	rate := int64(raw.FeeRate*1e8/1000 + 0.5)
	if rate < 1 {
		rate = fallbackRate
	}
	return rate, nil
}

// GetBlockCount returns the height of the most-work fully-validated chain.
func (c *Client) GetBlockCount(ctx context.Context) (int64, error) {
	var n int64
	return n, c.call(ctx, "getblockcount", []any{}, &n)
}

// GetBlockHash returns the block hash at the given height.
func (c *Client) GetBlockHash(ctx context.Context, height int64) (string, error) {
	var h string
	return h, c.call(ctx, "getblockhash", []any{height}, &h)
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

// GetRawTransaction returns the decoded transaction (verbose=true). Requires the
// node to find the tx: txindex=1, or the tx unspent/in mempool.
func (c *Client) GetRawTransaction(ctx context.Context, txid string) (*RawTx, error) {
	var raw *struct {
		TxID          string `json:"txid"`
		Confirmations int64  `json:"confirmations"`
		Vout          []struct {
			Value        float64 `json:"value"`
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
	for i, vo := range raw.Vout {
		out.Vouts[i] = RawVout{ValueSats: btcToSats(vo.Value), ScriptPubKeyHex: vo.ScriptPubKey.Hex}
	}
	return out, nil
}

// btcToSats converts a BTC float amount to integer satoshis.
func btcToSats(v float64) int64 { return int64(v*1e8 + 0.5) }

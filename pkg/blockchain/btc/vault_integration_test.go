//go:build integration

package btc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// BTC full deposit + withdrawal flow against a real bitcoind (the devnet
// regtest by default). Self-provisioning: each run generates a fresh vault
// (m-of-n P2WSH) + depositor, funds the depositor from a mined node wallet,
// deposits, then runs the quorum withdrawal — so re-running is a clean run with
// no shared state (only the node wallet's coinbase persists).
//
// Build-tagged `integration`. Defaults target `make devnet`; override with
// BTC_RPC_URL / BTC_RPC_USER / BTC_RPC_PASS.

const (
	defaultBTCRPC  = "http://127.0.0.1:18443"
	btcWallet      = "sdk"
	btcSignerCount = 3
	btcThreshold   = 2
)

func TestIntegrationBTC_DepositAndWithdraw(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	node := &bitcoindRPC{
		url:    btcEnv("BTC_RPC_URL", defaultBTCRPC),
		wallet: btcWallet,
		user:   btcEnv("BTC_RPC_USER", "sdk"),
		pass:   btcEnv("BTC_RPC_PASS", "sdk"),
		http:   &http.Client{Timeout: 30 * time.Second},
	}
	net := &chaincfg.RegressionNetParams

	// ── Setup: wallet + mined funds ───────────────────────────────────────────
	node.ensureWallet(ctx, t)
	miner := node.getNewAddress(ctx, t)
	node.generateToAddress(ctx, t, 101, miner) // coinbase maturity

	// Fresh keys this run → fresh vault, depositor, deposit address.
	signers := make([]sign.Signer, btcSignerCount)
	pubkeys := make([][]byte, btcSignerCount)
	for i := range signers {
		signers[i] = genSecpSigner(t)
		pubkeys[i] = signers[i].PublicKey()
	}
	depositorSigner := genSecpSigner(t)
	const account = "yellow://ynet/user/btc-itest"
	cfg := Config{ConfirmationDepth: 1, FeeConfTarget: 6, FallbackFeeRate: 5, FeeCapSatPerVByte: 10_000}

	depositor, err := NewDepositor(net, node, depositorSigner, pubkeys, btcThreshold, cfg)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	depositAddr, _, err := DepositAddress(account, btcThreshold, pubkeys, net)
	if err != nil {
		t.Fatalf("DepositAddress: %v", err)
	}

	// Watch the depositor + deposit addresses so listunspent/gettxout see them,
	// then fund the depositor from the node wallet.
	node.importAddress(ctx, t, depositor.DepositorAddress())
	node.importAddress(ctx, t, depositAddr.EncodeAddress())
	node.sendToAddress(ctx, t, depositor.DepositorAddress(), 1.0) // 1 BTC
	node.generateToAddress(ctx, t, 1, miner)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	depRef, err := depositor.Deposit(ctx, "BTC", decimal.NewFromInt(20_000_000), account) // 0.2 BTC
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	node.generateToAddress(ctx, t, 1, miner) // confirm the deposit UTXO
	t.Logf("deposit tx %s -> %s", depRef.Raw, depositAddr.EncodeAddress())

	// ── Withdrawal flow (quorum in-process) ───────────────────────────────────
	finalizers := make([]*WithdrawalFinalizer, btcSignerCount)
	for i, s := range signers {
		f, err := NewWithdrawalFinalizer(net, node, s, pubkeys, btcThreshold, cfg)
		if err != nil {
			t.Fatalf("NewWithdrawalFinalizer %d: %v", i, err)
		}
		if err := f.RegisterDepositAccounts(account); err != nil {
			t.Fatalf("register deposit account: %v", err)
		}
		finalizers[i] = f
	}

	var wid [32]byte
	wid[0], wid[31] = 0xB7, 0xC0
	op := &core.WithdrawalOp{Recipient: miner, Amount: decimal.NewFromInt(10_000_000)} // 0.1 BTC to the miner addr

	packed, err := finalizers[0].Pack(ctx, op, wid)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	shares := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, wid); err != nil {
			t.Fatalf("Validate[%d]: %v", i, err)
		}
		s, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("Sign[%d]: %v", i, err)
		}
		shares = append(shares, s)
	}
	merged, err := finalizers[0].Merge(ctx, packed, shares)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	ref, err := finalizers[0].Submit(ctx, merged)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	node.generateToAddress(ctx, t, 1, miner) // confirm the withdrawal
	t.Logf("withdrawal tx %s", ref.Raw)

	if _, executed, err := finalizers[0].VerifyExecution(ctx, wid); err != nil {
		t.Fatalf("VerifyExecution: %v", err)
	} else if !executed {
		t.Fatal("withdrawal not reported executed")
	}
}

func genSecpSigner(t *testing.T) sign.Signer {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return sign.NewKeySignerFromECDSA(k)
}

func btcEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── minimal bitcoind JSON-RPC client (implements btc.RPC + test setup) ────────

type bitcoindRPC struct {
	url    string // node endpoint
	wallet string // wallet name (wallet RPCs route to /wallet/<name>)
	user   string
	pass   string
	http   *http.Client
}

var _ RPC = (*bitcoindRPC)(nil)

func (c *bitcoindRPC) post(ctx context.Context, endpoint, method string, params []any, out any) error {
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
		return err
	}
	if env.Error != nil {
		return fmt.Errorf("rpc %s: %d %s", method, env.Error.Code, env.Error.Message)
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

func (c *bitcoindRPC) call(ctx context.Context, method string, params []any, out any) error {
	return c.post(ctx, c.url, method, params, out)
}

func (c *bitcoindRPC) walletCall(ctx context.Context, method string, params []any, out any) error {
	return c.post(ctx, c.url+"/wallet/"+c.wallet, method, params, out)
}

// --- btc.RPC interface ---

func (c *bitcoindRPC) ListUnspent(ctx context.Context, minConf int, addrs []string) ([]Unspent, error) {
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

func (c *bitcoindRPC) GetTxOut(ctx context.Context, txid string, vout uint32, includeMempool bool) (*TxOut, error) {
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

func (c *bitcoindRPC) SendRawTransaction(ctx context.Context, hexTx string) (string, error) {
	var txid string
	return txid, c.call(ctx, "sendrawtransaction", []any{hexTx}, &txid)
}

func (c *bitcoindRPC) EstimateSmartFeeSatPerVByte(ctx context.Context, confTarget int, fallbackRate int64) (int64, error) {
	var raw struct {
		FeeRate float64 `json:"feerate"`
	}
	if err := c.call(ctx, "estimatesmartfee", []any{confTarget}, &raw); err != nil || raw.FeeRate <= 0 {
		return fallbackRate, nil // regtest has no fee estimate
	}
	rate := int64(raw.FeeRate*1e8/1000 + 0.5)
	if rate < 1 {
		rate = fallbackRate
	}
	return rate, nil
}

func (c *bitcoindRPC) GetBlockCount(ctx context.Context) (int64, error) {
	var n int64
	return n, c.call(ctx, "getblockcount", []any{}, &n)
}

func (c *bitcoindRPC) GetBlockHash(ctx context.Context, height int64) (string, error) {
	var h string
	return h, c.call(ctx, "getblockhash", []any{height}, &h)
}

func (c *bitcoindRPC) GetBlockTxids(ctx context.Context, blockHash string) ([]string, error) {
	var raw struct {
		Tx []string `json:"tx"`
	}
	if err := c.call(ctx, "getblock", []any{blockHash, 1}, &raw); err != nil {
		return nil, err
	}
	return raw.Tx, nil
}

func (c *bitcoindRPC) GetRawTransaction(ctx context.Context, txid string) (*RawTx, error) {
	var raw struct {
		TxID string `json:"txid"`
		Vout []struct {
			Value        float64 `json:"value"`
			ScriptPubKey struct {
				Hex string `json:"hex"`
			} `json:"scriptPubKey"`
		} `json:"vout"`
	}
	// verbose=true; blockhash omitted (txindex=1 on the devnet node).
	if err := c.call(ctx, "getrawtransaction", []any{txid, true}, &raw); err != nil {
		return nil, err
	}
	out := &RawTx{TxID: raw.TxID, Vouts: make([]RawVout, len(raw.Vout))}
	for i, vo := range raw.Vout {
		out.Vouts[i] = RawVout{ValueSats: btcToSats(vo.Value), ScriptPubKeyHex: vo.ScriptPubKey.Hex}
	}
	return out, nil
}

// --- test-only setup helpers ---

func (c *bitcoindRPC) ensureWallet(ctx context.Context, t *testing.T) {
	t.Helper()
	// Legacy wallet (descriptors=false) so importaddress watch-only works.
	err := c.call(ctx, "createwallet", []any{c.wallet, false, false, "", false, false}, nil)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") &&
		!strings.Contains(strings.ToLower(err.Error()), "already loaded") {
		// loadwallet covers the "exists on disk but not loaded" case.
		if lerr := c.call(ctx, "loadwallet", []any{c.wallet}, nil); lerr != nil &&
			!strings.Contains(strings.ToLower(lerr.Error()), "already loaded") {
			t.Fatalf("createwallet/loadwallet: %v / %v", err, lerr)
		}
	}
}

func (c *bitcoindRPC) getNewAddress(ctx context.Context, t *testing.T) string {
	t.Helper()
	var addr string
	if err := c.walletCall(ctx, "getnewaddress", []any{"", "bech32"}, &addr); err != nil {
		t.Fatalf("getnewaddress: %v", err)
	}
	return addr
}

func (c *bitcoindRPC) generateToAddress(ctx context.Context, t *testing.T, n int, addr string) {
	t.Helper()
	if err := c.call(ctx, "generatetoaddress", []any{n, addr}, nil); err != nil {
		t.Fatalf("generatetoaddress: %v", err)
	}
}

func (c *bitcoindRPC) importAddress(ctx context.Context, t *testing.T, addr string) {
	t.Helper()
	// rescan=false: every address we import is funded only afterwards.
	if err := c.walletCall(ctx, "importaddress", []any{addr, "", false}, nil); err != nil {
		t.Fatalf("importaddress %s: %v", addr, err)
	}
}

func (c *bitcoindRPC) sendToAddress(ctx context.Context, t *testing.T, addr string, btc float64) {
	t.Helper()
	var txid string
	if err := c.walletCall(ctx, "sendtoaddress", []any{addr, btc}, &txid); err != nil {
		t.Fatalf("sendtoaddress: %v", err)
	}
}

func btcToSats(v float64) int64 { return int64(v*1e8 + 0.5) }

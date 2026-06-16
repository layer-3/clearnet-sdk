//go:build integration

package btc

import (
	"context"
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

	node := &bitcoindRPC{Client: NewClient(
		btcEnv("BTC_RPC_URL", defaultBTCRPC),
		btcEnv("BTC_RPC_USER", "sdk"),
		btcEnv("BTC_RPC_PASS", "sdk"),
		btcWallet,
	)}
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
	depRef, err := depositor.SubmitDeposit(ctx, "BTC", decimal.NewFromInt(20_000_000), account) // 0.2 BTC
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
	ref, err := finalizers[0].Submit(ctx, packed, shares)
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

// ── test harness: a *Client plus regtest provisioning helpers ─────────────────

// bitcoindRPC embeds the SDK's concrete *Client (which provides the RPC
// surface) and adds regtest-only setup calls (wallet, mining, funding) that the
// shipped client deliberately omits. The setup helpers reach the embedded
// client's unexported call/walletCall directly (same package).
type bitcoindRPC struct {
	*Client
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

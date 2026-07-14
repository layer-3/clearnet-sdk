//go:build integration

package btc

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"sync"
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

	assets := NewAssetResolver()
	depositor, err := NewDepositor(net, node, depositorSigner, pubkeys, btcThreshold, cfg, assets)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	depositAddr, _, err := DepositAddress(account, btcThreshold, pubkeys, net)
	if err != nil {
		t.Fatalf("DepositAddress: %v", err)
	}

	// Base vault address — the withdrawal pays change here, and the rotation
	// sweep later spends it; watch it up front (rescan=false) so the change UTXO
	// is tracked from creation.
	baseRedeem, err := RedeemScript(btcThreshold, pubkeys)
	if err != nil {
		t.Fatalf("base redeem: %v", err)
	}
	baseVault, err := VaultAddress(baseRedeem, net)
	if err != nil {
		t.Fatalf("base vault: %v", err)
	}

	// Watch the depositor + deposit + base-vault addresses so listunspent/gettxout
	// see them, then fund the depositor from the node wallet.
	node.importAddress(ctx, t, depositor.DepositorAddress())
	node.importAddress(ctx, t, depositAddr.EncodeAddress())
	node.importAddress(ctx, t, baseVault.EncodeAddress())
	node.sendToAddress(ctx, t, depositor.DepositorAddress(), 1.0) // 1 BTC
	node.generateToAddress(ctx, t, 1, miner)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	depRef, err := depositor.SubmitDeposit(ctx, "", decimal.NewFromBigInt(big.NewInt(20_000_000), -8), core.DepositDestination{Account: account}) // 0.2 BTC
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	node.generateToAddress(ctx, t, 1, miner) // confirm the deposit UTXO
	t.Logf("deposit tx %s -> %s", depRef.Raw, depositAddr.EncodeAddress())

	// ── Withdrawal flow (quorum in-process) ───────────────────────────────────
	finalizers := make([]*WithdrawalFinalizer, btcSignerCount)
	for i, s := range signers {
		f, err := NewWithdrawalFinalizer(net, node, s, pubkeys, btcThreshold, cfg, assets)
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
	op := &core.WithdrawalOp{Recipient: miner, AssetURI: "yellow://ynet/asset/custody/btc/0/0", Amount: decimal.NewFromBigInt(big.NewInt(10_000_000), -8)} // 0.1 BTC to the miner addr

	// deadline is accepted but ignored on BTC (no consensus expiry); a
	// far-future value keeps parity with the other chains' happy-path tests.
	deadline := time.Now().Add(24 * time.Hour).Unix()
	packed, err := finalizers[0].Pack(ctx, op, wid, deadline)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	shares := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, wid, deadline); err != nil {
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

	// ── Rotation flow (sweep the vault into a new vault; current signers sign) ─
	newSigners := make([]sign.Signer, btcSignerCount)
	newPubkeys := make([][]byte, btcSignerCount)
	newPubHex := make([]string, btcSignerCount)
	for i := range newSigners {
		newSigners[i] = genSecpSigner(t)
		newPubkeys[i] = newSigners[i].PublicKey()
		newPubHex[i] = hex.EncodeToString(newPubkeys[i])
	}
	// Watch the new vault so listunspent sees the swept output.
	newRedeem, err := RedeemScript(btcThreshold, newPubkeys)
	if err != nil {
		t.Fatalf("new redeem: %v", err)
	}
	newVaultAddr, err := VaultAddress(newRedeem, net)
	if err != nil {
		t.Fatalf("new vault addr: %v", err)
	}
	node.importAddress(ctx, t, newVaultAddr.EncodeAddress())

	store := &memVaultStore{pubkeys: pubkeys, threshold: btcThreshold}
	rotators := make([]*RotationFinalizer, btcSignerCount)
	for i, s := range signers {
		r, err := NewRotationFinalizer(net, node, s, store, cfg, assets, account)
		if err != nil {
			t.Fatalf("NewRotationFinalizer %d: %v", i, err)
		}
		rotators[i] = r
	}

	var rotID [32]byte
	rotID[0], rotID[31] = 0xB7, 0x7A
	rPacked, err := rotators[0].Pack(ctx, rotID, newPubHex, btcThreshold)
	if err != nil {
		t.Fatalf("rotation Pack: %v", err)
	}
	rShares := make([][]byte, 0, len(rotators))
	for i, r := range rotators {
		if err := r.Validate(ctx, rotID, rPacked, newPubHex, btcThreshold); err != nil {
			t.Fatalf("rotation Validate[%d]: %v", i, err)
		}
		s, err := r.Sign(ctx, rPacked)
		if err != nil {
			t.Fatalf("rotation Sign[%d]: %v", i, err)
		}
		rShares = append(rShares, s)
	}
	rRef, err := rotators[0].Submit(ctx, rPacked, rShares)
	if err != nil {
		t.Fatalf("rotation Submit: %v", err)
	}
	node.generateToAddress(ctx, t, 1, miner) // confirm the sweep
	t.Logf("rotation sweep tx %s", rRef.Raw)

	if _, done, err := rotators[0].VerifyRotation(ctx, newPubHex, btcThreshold); err != nil {
		t.Fatalf("VerifyRotation: %v", err)
	} else if !done {
		t.Fatal("rotation not reported done")
	}
	// VerifyRotation pivots the store on success.
	if cur, _, _ := store.Current(ctx); len(cur) != btcSignerCount {
		t.Fatalf("store not pivoted: %d pubkeys", len(cur))
	}
}

// memVaultStore is an in-memory btc.VaultStore for the rotation test.
type memVaultStore struct {
	mu        sync.Mutex
	pubkeys   [][]byte
	threshold int
}

func (s *memVaultStore) Current(context.Context) ([][]byte, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pubkeys, s.threshold, nil
}

func (s *memVaultStore) Pivot(_ context.Context, pubkeys [][]byte, threshold int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pubkeys, s.threshold = pubkeys, threshold
	return nil
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

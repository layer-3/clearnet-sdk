//go:build integration

package xrpl

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	xrplkeypairs "github.com/Peersyst/xrpl-go/keypairs"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// XRPL full deposit + withdrawal flow against a standalone rippled (the devnet
// by default). Self-provisioning: each run funds a fresh vault account from the
// genesis master, configures its SignerList over fresh signer keys, creates a
// Ticket, then runs deposit + the quorum withdrawal. Standalone rippled does
// not auto-close ledgers, so the harness calls `ledger_accept` after each
// submit. Re-running is a clean run (fresh accounts); only the genesis master
// persists.
//
// Build-tagged `integration`. Default node http://127.0.0.1:5005; override via
// XRPL_RPC_URL.
//
// NOTE: this is the least-validated integration test — standalone provisioning
// (ledger_accept cadence, SignerListSet/TicketCreate encoding) may need
// iteration against a live node.

const (
	defaultXRPLRPC  = "http://127.0.0.1:5005"
	genesisSeed     = "snoPBrXtMeMyMHUVTgbuqAfg1SUTb" // rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh, ~100B XRP
	xrplSignerCount = 3
	xrplQuorum      = 2
	depositTag      = 42
)

func TestIntegrationXRPL_DepositAndWithdraw(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	url := xrplEnv("XRPL_RPC_URL", defaultXRPLRPC)
	cfg, err := rpc.NewClientConfig(url)
	if err != nil {
		t.Fatalf("rpc config: %v", err)
	}
	h := &xrplHarness{url: url, client: rpc.NewClient(cfg), http: &http.Client{Timeout: 30 * time.Second}}

	master := masterSigner(t)
	masterID := mustIdentity(t, master)

	// Fresh accounts this run.
	vault := genEd25519(t)
	vaultID := mustIdentity(t, vault)
	depositor := genEd25519(t)
	depID := mustIdentity(t, depositor)
	recipient := genEd25519(t)
	recID := mustIdentity(t, recipient)
	signers := make([]sign.Signer, xrplSignerCount)
	signerAddrs := make([]string, xrplSignerCount)
	for i := range signers {
		signers[i] = genEd25519(t)
		signerAddrs[i] = mustIdentity(t, signers[i]).ClassicAddress
	}

	// ── Setup ─────────────────────────────────────────────────────────────────
	h.fund(ctx, t, master, masterID, vaultID.ClassicAddress, "1000000000") // 1000 XRP
	h.fund(ctx, t, master, masterID, depID.ClassicAddress, "1000000000")   // 1000 XRP
	h.signerListSet(ctx, t, vault, vaultID, signerAddrs, xrplQuorum)
	ticketSeq := h.ticketCreate(ctx, t, vault, vaultID)
	t.Logf("vault %s signer-list set (quorum %d), ticket %d", vaultID.ClassicAddress, xrplQuorum, ticketSeq)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	dep, err := NewDepositor(url, vaultID.ClassicAddress, depositor)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	depRef, err := dep.SubmitDeposit(ctx, "XRP", decimal.NewFromInt(100_000_000), fmt.Sprintf("xrpl-%d", depositTag)) // 100 XRP
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	h.ledgerAccept(ctx, t)
	t.Logf("deposit tx %s (from %s)", depRef.Raw, dep.DepositorAddress())

	// ── Withdrawal flow (quorum in-process) ───────────────────────────────────
	finalizers := make([]*WithdrawalFinalizer, len(signers))
	for i, s := range signers {
		f, err := NewWithdrawalFinalizer(url, vaultID.ClassicAddress, xrplQuorum, s, fixedTicket(ticketSeq))
		if err != nil {
			t.Fatalf("NewWithdrawalFinalizer %d: %v", i, err)
		}
		finalizers[i] = f
	}

	var wid [32]byte
	wid[0], wid[31] = 0x12, 0x34
	op := &core.WithdrawalOp{Recipient: recID.ClassicAddress, L1Asset: "XRP", Amount: decimal.NewFromInt(50_000_000)} // 50 XRP

	packed, err := finalizers[0].Pack(ctx, op, wid)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	blobs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, wid); err != nil {
			t.Fatalf("Validate[%d]: %v", i, err)
		}
		b, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("Sign[%d]: %v", i, err)
		}
		blobs = append(blobs, b)
	}
	ref, err := finalizers[0].Submit(ctx, packed, blobs)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	h.ledgerAccept(ctx, t)
	t.Logf("withdrawal tx %s", ref.Raw)

	if _, executed, err := finalizers[0].VerifyExecution(ctx, wid); err != nil {
		t.Fatalf("VerifyExecution: %v", err)
	} else if !executed {
		t.Fatal("withdrawal not reported executed")
	}

	// ── Rotation flow (current SignerList members authorize the new list) ─────
	newSigners := make([]sign.Signer, xrplSignerCount)
	newAddrs := make([]string, xrplSignerCount)
	for i := range newSigners {
		newSigners[i] = genEd25519(t) // SignerList members need not be funded accounts
		newAddrs[i] = mustIdentity(t, newSigners[i]).ClassicAddress
	}

	rotators := make([]*RotationFinalizer, len(signers))
	for i, s := range signers {
		r, err := NewRotationFinalizer(url, vaultID.ClassicAddress, xrplQuorum, s)
		if err != nil {
			t.Fatalf("NewRotationFinalizer %d: %v", i, err)
		}
		rotators[i] = r
	}

	rPacked, err := rotators[0].Pack(ctx, newAddrs, xrplQuorum)
	if err != nil {
		t.Fatalf("rotation Pack: %v", err)
	}
	rBlobs := make([][]byte, 0, len(rotators))
	for i, r := range rotators {
		if err := r.Validate(ctx, rPacked, newAddrs, xrplQuorum); err != nil {
			t.Fatalf("rotation Validate[%d]: %v", i, err)
		}
		b, err := r.Sign(ctx, rPacked)
		if err != nil {
			t.Fatalf("rotation Sign[%d]: %v", i, err)
		}
		rBlobs = append(rBlobs, b)
	}
	rRef, err := rotators[0].Submit(ctx, rPacked, rBlobs)
	if err != nil {
		t.Fatalf("rotation Submit: %v", err)
	}
	h.ledgerAccept(ctx, t)
	t.Logf("rotation tx %s", rRef.Raw)

	if _, done, err := rotators[0].VerifyRotation(ctx, newAddrs, xrplQuorum); err != nil {
		t.Fatalf("VerifyRotation: %v", err)
	} else if !done {
		t.Fatal("rotation not reported done")
	}
}

type fixedTicket uint32

func (f fixedTicket) TicketFor(context.Context, [32]byte) (uint32, error) { return uint32(f), nil }

// ── harness ───────────────────────────────────────────────────────────────────

type xrplHarness struct {
	url    string
	client *rpc.Client
	http   *http.Client
}

// submit autofills, single-signs with the given account, submits, and accepts a
// ledger so the tx validates before the next call reads account state.
func (h *xrplHarness) submit(ctx context.Context, t *testing.T, s sign.Signer, id Identity, flatTx transaction.FlatTransaction) {
	t.Helper()
	if err := h.client.Autofill(&flatTx); err != nil {
		t.Fatalf("autofill: %v", err)
	}
	blob, err := signSingle(ctx, s, id, flatTx)
	if err != nil {
		t.Fatalf("sign setup tx: %v", err)
	}
	res, err := h.client.SubmitTxBlob(blob, false)
	if err != nil {
		t.Fatalf("submit setup tx: %v", err)
	}
	if !strings.HasPrefix(res.EngineResult, "tes") && !strings.HasPrefix(res.EngineResult, "ter") {
		t.Fatalf("setup tx rejected: %s - %s", res.EngineResult, res.EngineResultMessage)
	}
	h.ledgerAccept(ctx, t)
}

func (h *xrplHarness) fund(ctx context.Context, t *testing.T, s sign.Signer, id Identity, dest, drops string) {
	t.Helper()
	h.submit(ctx, t, s, id, transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         id.ClassicAddress,
		"Destination":     dest,
		"Amount":          drops,
	})
}

func (h *xrplHarness) signerListSet(ctx context.Context, t *testing.T, s sign.Signer, id Identity, signerAddrs []string, quorum int) {
	t.Helper()
	entries := make([]any, len(signerAddrs))
	for i, a := range signerAddrs {
		entries[i] = map[string]any{"SignerEntry": map[string]any{"Account": a, "SignerWeight": 1}}
	}
	h.submit(ctx, t, s, id, transaction.FlatTransaction{
		"TransactionType": "SignerListSet",
		"Account":         id.ClassicAddress,
		"SignerQuorum":    quorum,
		"SignerEntries":   entries,
	})
}

// ticketCreate creates one Ticket on the account and returns its sequence.
func (h *xrplHarness) ticketCreate(ctx context.Context, t *testing.T, s sign.Signer, id Identity) uint32 {
	t.Helper()
	h.submit(ctx, t, s, id, transaction.FlatTransaction{
		"TransactionType": "TicketCreate",
		"Account":         id.ClassicAddress,
		"TicketCount":     1,
	})
	// Read the created Ticket's sequence from account_objects.
	var resp struct {
		Result struct {
			AccountObjects []struct {
				LedgerEntryType string `json:"LedgerEntryType"`
				TicketSequence  uint32 `json:"TicketSequence"`
			} `json:"account_objects"`
		} `json:"result"`
	}
	h.rawRPC(ctx, t, "account_objects", map[string]any{"account": id.ClassicAddress, "type": "ticket"}, &resp)
	for _, o := range resp.Result.AccountObjects {
		if o.LedgerEntryType == "Ticket" {
			return o.TicketSequence
		}
	}
	t.Fatal("no Ticket object found after TicketCreate")
	return 0
}

func (h *xrplHarness) ledgerAccept(ctx context.Context, t *testing.T) {
	t.Helper()
	h.rawRPC(ctx, t, "ledger_accept", nil, nil)
}

// rawRPC posts a rippled JSON-RPC method (params wrapped in the single-element
// array rippled expects) and optionally unmarshals the full envelope into out.
func (h *xrplHarness) rawRPC(ctx context.Context, t *testing.T, method string, params map[string]any, out any) {
	t.Helper()
	p := []any{}
	if params != nil {
		p = append(p, params)
	}
	body, _ := json.Marshal(map[string]any{"method": method, "params": p})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("rawRPC %s: %v", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.http.Do(req)
	if err != nil {
		t.Fatalf("rawRPC %s: %v", method, err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("rawRPC %s decode: %v", method, err)
		}
	}
}

// ── key helpers ───────────────────────────────────────────────────────────────

func genEd25519(t *testing.T) sign.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	ks, err := sign.NewKeySignerFromEd25519(priv)
	if err != nil {
		t.Fatalf("ed25519 signer: %v", err)
	}
	return ks
}

// masterSigner derives the genesis (master) secp256k1 key from its family seed.
func masterSigner(t *testing.T) sign.Signer {
	t.Helper()
	privHex, _, err := xrplkeypairs.DeriveKeypair(genesisSeed, false)
	if err != nil {
		t.Fatalf("derive genesis keypair: %v", err)
	}
	// secp256k1 derivation hex carries a "00" prefix over the 32-byte scalar.
	raw, err := hex.DecodeString(strings.TrimPrefix(strings.ToUpper(privHex), "00"))
	if err != nil || len(raw) != 32 {
		t.Fatalf("decode genesis scalar (len=%d): %v", len(raw), err)
	}
	k, err := crypto.ToECDSA(raw)
	if err != nil {
		t.Fatalf("genesis scalar to ECDSA: %v", err)
	}
	return sign.NewKeySignerFromECDSA(k)
}

func mustIdentity(t *testing.T, s sign.Signer) Identity {
	t.Helper()
	id, err := DeriveIdentity(s)
	if err != nil {
		t.Fatalf("derive identity: %v", err)
	}
	return id
}

func xrplEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

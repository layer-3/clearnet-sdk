//go:build integration

package sol

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Solana full deposit + withdrawal flow against the devnet validator (which
// preloads the custody program upgradeable at its fixed id, upgrade authority =
// devnet/sol-upgrade-authority.json). Self-provisioning: airdrop-funds the
// authority + depositor, Initializes the Config once (idempotent), deposits
// native SOL, then runs the quorum withdrawal with a fresh withdrawalID.
//
// Unlike EVM, the Config PDA is a singleton, so the signer set is FIXED across
// runs (derived from fixed seeds) and only the withdrawalID is fresh per run —
// re-runs stay clean without restarting the validator. Build-tagged
// `integration`; defaults target `make devnet` (override via SOL_RPC_URL).

const (
	solChainID     = 1002
	solSignerCount = 3
	solThreshold   = 2
)

func TestIntegrationSOL_DepositAndWithdraw(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	rpcURL := solEnv("SOL_RPC_URL", "http://127.0.0.1:8899")
	client := rpc.New(rpcURL)
	programID := custody.ProgramID

	// Upgrade authority (vendored) doubles as Initialize payer + withdrawal fee payer.
	authority := loadAuthority(t)
	authorityPub, err := solanaPub(authority)
	if err != nil {
		t.Fatalf("authority pub: %v", err)
	}

	// Fixed signer set (Config is a singleton — same keys every run).
	signers := make([]sign.Signer, solSignerCount)
	for i := range signers {
		signers[i] = fixedEd25519(t, fmt.Sprintf("clearnet-sdk/sol-itest/signer/%d", i))
	}
	signerPubs := make([]solana.PublicKey, solSignerCount)
	for i, s := range signers {
		p, _ := solanaPub(s)
		signerPubs[i] = p
	}
	// Program requires the on-chain signer set strictly ascending by raw bytes.
	sort.Slice(signerPubs, func(i, j int) bool {
		a, b := signerPubs[i], signerPubs[j]
		return string(a[:]) < string(b[:])
	})

	depositor := fixedEd25519(t, "clearnet-sdk/sol-itest/depositor")
	depositorPub, _ := solanaPub(depositor)

	// Fund authority + depositor.
	airdrop(ctx, t, client, authorityPub, 5*solana.LAMPORTS_PER_SOL)
	airdrop(ctx, t, client, depositorPub, 5*solana.LAMPORTS_PER_SOL)

	// Initialize the Config once (idempotent — skip if it already exists).
	if _, err := fetchConfig(ctx, client, programID, rpc.CommitmentConfirmed); err != nil {
		programData, _, e := solana.FindProgramAddress([][]byte{programID[:]}, solana.BPFLoaderUpgradeableProgramID)
		if e != nil {
			t.Fatalf("program-data PDA: %v", e)
		}
		ix, e := custody.NewInitializeInstruction(
			signerPubs, uint8(solThreshold), uint64(solChainID),
			ConfigPDA(programID), authorityPub, programID, programData, solana.SystemProgramID,
		)
		if e != nil {
			t.Fatalf("build initialize: %v", e)
		}
		if _, e := signAndSend(ctx, client, []solana.Instruction{ix}, authorityPub, authority, rpc.CommitmentConfirmed); e != nil {
			t.Fatalf("initialize: %v", e)
		}
		waitConfig(ctx, t, client, programID)
		t.Logf("initialized Config (signers=%d threshold=%d)", solSignerCount, solThreshold)
	} else {
		t.Logf("Config already initialized; reusing")
	}

	// ── Deposit flow ──────────────────────────────────────────────────────────
	dep, err := NewDepositor(rpcURL, programID, depositor, rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	const account = "00000000000000000000000000000000000000a1" // 20-byte clearnet addr
	depRef, err := dep.Deposit(ctx, "SOL", decimal.NewFromInt(100_000_000), account)
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	t.Logf("deposit tx %s (from %s)", depRef.Raw, dep.DepositorAddress())
	// The depositor fire-and-forwards; wait until the vault PDA actually holds
	// the funds before withdrawing.
	waitBalance(ctx, t, client, VaultPDA(programID), 100_000_000)

	// ── Withdrawal flow (quorum in-process) ───────────────────────────────────
	finalizers := make([]*WithdrawalFinalizer, solSignerCount)
	for i, s := range signers {
		f, e := NewWithdrawalFinalizer(rpcURL, programID, s, authority, Config{ChainID: solChainID, Commitment: rpc.CommitmentConfirmed})
		if e != nil {
			t.Fatalf("NewWithdrawalFinalizer %d: %v", i, e)
		}
		finalizers[i] = f
	}

	var wid [32]byte
	if _, err := rand.Read(wid[:]); err != nil {
		t.Fatalf("rand wid: %v", err)
	}
	recipient := fixedEd25519(t, "clearnet-sdk/sol-itest/recipient/"+hex.EncodeToString(wid[:4]))
	recipientPub, _ := solanaPub(recipient)
	op := &core.WithdrawalOp{Recipient: recipientPub.String(), L1Asset: "SOL", Amount: decimal.NewFromInt(40_000_000)}

	packed, err := finalizers[0].Pack(ctx, op, wid)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	shares := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, wid); err != nil {
			t.Fatalf("Validate[%d]: %v", i, err)
		}
		s, e := f.Sign(ctx, packed)
		if e != nil {
			t.Fatalf("Sign[%d]: %v", i, e)
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
	t.Logf("withdrawal tx %s", ref.Raw)

	if _, executed, err := finalizers[0].VerifyExecution(ctx, wid); err != nil {
		t.Fatalf("VerifyExecution: %v", err)
	} else if !executed {
		t.Fatal("withdrawal not reported executed")
	}
}

// --- helpers ---

func solEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadAuthority reads the vendored upgrade-authority keypair (a 64-byte Solana
// keypair JSON) as an ed25519 signer.
func loadAuthority(t *testing.T) sign.Signer {
	t.Helper()
	// repo-root devnet/sol-upgrade-authority.json, from pkg/blockchain/sol.
	path := filepath.Join("..", "..", "..", "devnet", "sol-upgrade-authority.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authority keypair: %v", err)
	}
	var b []byte
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("parse authority keypair: %v", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		t.Fatalf("authority keypair is %d bytes, want %d", len(b), ed25519.PrivateKeySize)
	}
	ks, err := sign.NewKeySignerFromEd25519(ed25519.PrivateKey(b))
	if err != nil {
		t.Fatalf("authority signer: %v", err)
	}
	return ks
}

func fixedEd25519(t *testing.T, seedStr string) sign.Signer {
	t.Helper()
	seed := sha256.Sum256([]byte(seedStr))
	ks, err := sign.NewKeySignerFromEd25519(ed25519.NewKeyFromSeed(seed[:]))
	if err != nil {
		t.Fatalf("ed25519 from seed: %v", err)
	}
	return ks
}

func airdrop(ctx context.Context, t *testing.T, client *rpc.Client, pub solana.PublicKey, lamports uint64) {
	t.Helper()
	if _, err := client.RequestAirdrop(ctx, pub, lamports, rpc.CommitmentConfirmed); err != nil {
		t.Fatalf("airdrop %s: %v", pub, err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		bal, err := client.GetBalance(ctx, pub, rpc.CommitmentConfirmed)
		if err == nil && bal != nil && bal.Value > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("airdrop to %s not credited in time", pub)
		}
		time.Sleep(time.Second)
	}
}

func waitBalance(ctx context.Context, t *testing.T, client *rpc.Client, pub solana.PublicKey, min uint64) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		bal, err := client.GetBalance(ctx, pub, rpc.CommitmentConfirmed)
		if err == nil && bal != nil && bal.Value >= min {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("balance of %s did not reach %d in time", pub, min)
		}
		time.Sleep(time.Second)
	}
}

func waitConfig(ctx context.Context, t *testing.T, client *rpc.Client, programID solana.PublicKey) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := fetchConfig(ctx, client, programID, rpc.CommitmentConfirmed); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Config not visible after initialize")
		}
		time.Sleep(time.Second)
	}
}

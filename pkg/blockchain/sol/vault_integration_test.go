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
	"strconv"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	addresslookuptable "github.com/gagliardetto/solana-go/programs/address-lookup-table"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
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
		if _, e := signAndSend(ctx, client, []solana.Instruction{ix}, authorityPub, authority, rpc.CommitmentConfirmed, solana.PublicKey{}); e != nil {
			t.Fatalf("initialize: %v", e)
		}
		waitConfig(ctx, t, client, programID)
		t.Logf("initialized Config (signers=%d threshold=%d)", solSignerCount, solThreshold)
	} else {
		t.Logf("Config already initialized; reusing")
	}

	// Self-heal: a prior run that died mid-rotation may have left the singleton
	// Config on the rotated set; restore the fixed set before deposit/withdraw.
	rotCfg := Config{ChainID: solChainID, Commitment: rpc.CommitmentConfirmed}
	rotatedSigners := solRotatedSigners(t)
	ensureFixedSigners(ctx, t, rpcURL, programID, authority, rotCfg, signerPubs, rotatedSigners)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	dep, err := NewDepositor(rpcURL, programID, depositor, rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	const account = "00000000000000000000000000000000000000a1" // 20-byte clearnet addr
	depRef, err := dep.SubmitDeposit(ctx, "SOL", decimal.NewFromInt(100_000_000), core.DepositDestination{Account: account})
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
	ref, err := finalizers[0].Submit(ctx, packed, shares)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	t.Logf("withdrawal tx %s", ref.Raw)

	if _, executed, err := finalizers[0].VerifyExecution(ctx, wid); err != nil {
		t.Fatalf("VerifyExecution: %v", err)
	} else if !executed {
		t.Fatal("withdrawal not reported executed")
	}

	// ── SPL withdrawal flow (over a v0 transaction + ALT) ───────────────────────
	// Create a 0-decimal mint, fund the vault's ATA, build an Address Lookup Table
	// over the static execute accounts, then withdraw under the same quorum with
	// the ALT configured. Exercises the SPL execute path (recipient-ATA creation +
	// token remaining-accounts) and the v0/ALT submit path together.
	authorityKey := loadAuthorityKey(t)
	mint := setupSPLMintFundVault(ctx, t, client, authorityPub, authorityKey, programID, 100)
	// Build the ALT from the SDK's invariant account set — the same knowledge
	// buildExecuteIx uses, so the table can't drift from the instruction layout.
	alt := createActiveALT(ctx, t, client, authorityPub, authorityKey, VaultLookupAddresses(programID, mint))

	splCfg := Config{ChainID: solChainID, Commitment: rpc.CommitmentConfirmed, AddressLookupTable: alt}
	splFinalizers := make([]*WithdrawalFinalizer, solSignerCount)
	for i, s := range signers {
		f, e := NewWithdrawalFinalizer(rpcURL, programID, s, authority, splCfg)
		if e != nil {
			t.Fatalf("NewWithdrawalFinalizer (spl) %d: %v", i, e)
		}
		splFinalizers[i] = f
	}

	var splWid [32]byte
	if _, err := rand.Read(splWid[:]); err != nil {
		t.Fatalf("rand spl wid: %v", err)
	}
	splRecipient := fixedEd25519(t, "clearnet-sdk/sol-itest/spl-recipient/"+hex.EncodeToString(splWid[:4]))
	splRecipientPub, _ := solanaPub(splRecipient)
	splOp := &core.WithdrawalOp{Recipient: splRecipientPub.String(), L1Asset: mint.String(), Amount: decimal.NewFromInt(40)}

	splPacked, err := splFinalizers[0].Pack(ctx, splOp, splWid)
	if err != nil {
		t.Fatalf("SPL Pack: %v", err)
	}
	splShares := make([][]byte, 0, len(splFinalizers))
	for i, f := range splFinalizers {
		if err := f.Validate(ctx, splPacked, splOp, splWid); err != nil {
			t.Fatalf("SPL Validate[%d]: %v", i, err)
		}
		s, e := f.Sign(ctx, splPacked)
		if e != nil {
			t.Fatalf("SPL Sign[%d]: %v", i, e)
		}
		splShares = append(splShares, s)
	}
	if _, err := splFinalizers[0].Submit(ctx, splPacked, splShares); err != nil {
		t.Fatalf("SPL Submit: %v", err)
	}
	recipientATA, _, err := solana.FindAssociatedTokenAddress(splRecipientPub, mint)
	if err != nil {
		t.Fatalf("recipient ATA: %v", err)
	}
	waitTokenBalance(ctx, t, client, recipientATA, 40)
	t.Logf("SPL withdrawal credited 40 tokens to %s", recipientATA)

	// ── Rotation flow ─────────────────────────────────────────────────────────
	// Config is a singleton, so rotate to the (deterministic) rotated set and
	// restore the original — both directions exercised. A run that dies between
	// the two rotations is healed by ensureFixedSigners on the next run.
	newPubs := pubsBase58(rotatedSigners)
	origPubs := pubsBase58(signers)
	runRotation(ctx, t, rpcURL, programID, authority, rotCfg, signers, newPubs, solThreshold)
	t.Logf("rotated to fresh signer set")
	runRotation(ctx, t, rpcURL, programID, authority, rotCfg, rotatedSigners, origPubs, solThreshold)
	t.Logf("restored original signer set")
}

// runRotation drives one rotation: `current` are the signers that authorize it,
// `target` the new on-chain set (base58). Fails the test on any step.
func runRotation(ctx context.Context, t *testing.T, rpcURL string, programID solana.PublicKey, feePayer sign.Signer, cfg Config, current []sign.Signer, target []string, threshold int) {
	t.Helper()
	rotators := make([]*RotationFinalizer, len(current))
	for i, s := range current {
		r, err := NewRotationFinalizer(rpcURL, programID, s, feePayer, cfg)
		if err != nil {
			t.Fatalf("NewRotationFinalizer %d: %v", i, err)
		}
		rotators[i] = r
	}
	var rotID [32]byte
	rotID[0], rotID[31] = 0x50, 0x7A
	packed, err := rotators[0].Pack(ctx, rotID, target, threshold)
	if err != nil {
		t.Fatalf("rotation Pack: %v", err)
	}
	shares := make([][]byte, 0, len(rotators))
	for i, r := range rotators {
		if err := r.Validate(ctx, rotID, packed, target, threshold); err != nil {
			t.Fatalf("rotation Validate[%d]: %v", i, err)
		}
		s, err := r.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("rotation Sign[%d]: %v", i, err)
		}
		shares = append(shares, s)
	}
	if _, err := rotators[0].Submit(ctx, packed, shares); err != nil {
		t.Fatalf("rotation Submit: %v", err)
	}
	if _, done, err := rotators[0].VerifyRotation(ctx, target, threshold); err != nil {
		t.Fatalf("VerifyRotation: %v", err)
	} else if !done {
		t.Fatal("rotation not reported done")
	}
}

// solRotatedSigners returns the deterministic "rotated" signer set the rotation
// flow rotates to (distinct from the fixed deposit/withdraw set).
func solRotatedSigners(t *testing.T) []sign.Signer {
	t.Helper()
	out := make([]sign.Signer, solSignerCount)
	for i := range out {
		out[i] = fixedEd25519(t, fmt.Sprintf("clearnet-sdk/sol-itest/rotated/%d", i))
	}
	return out
}

// pubsBase58 maps signers to their base58 pubkeys.
func pubsBase58(signers []sign.Signer) []string {
	out := make([]string, len(signers))
	for i, s := range signers {
		p, _ := solanaPub(s)
		out[i] = p.String()
	}
	return out
}

// ensureFixedSigners restores the singleton Config to the fixed signer set if a
// prior run left it on the rotated set. Both sets are deterministic, so recovery
// is a rotation from the rotated set (whose keys the test holds) back to fixed.
func ensureFixedSigners(ctx context.Context, t *testing.T, rpcURL string, programID solana.PublicKey, feePayer sign.Signer, cfg Config, fixedPubs []solana.PublicKey, rotated []sign.Signer) {
	t.Helper()
	conf, err := fetchConfig(ctx, rpc.New(rpcURL), programID, cfg.Commitment)
	if err != nil {
		return // not initialized; Initialize already set the fixed set
	}
	want := make([]string, len(fixedPubs))
	for i := range fixedPubs {
		want[i] = fixedPubs[i].String()
	}
	if sameSignerSet(conf.Signers, want) {
		return
	}
	if sameSignerSet(conf.Signers, pubsBase58(rotated)) {
		runRotation(ctx, t, rpcURL, programID, feePayer, cfg, rotated, want, solThreshold)
		t.Logf("self-healed Config back to the fixed signer set")
		return
	}
	t.Fatal("Config in an unrecognized signer state; manual reset needed")
}

// sameSignerSet reports whether the on-chain signer set equals want (as a set).
func sameSignerSet(got []solana.PublicKey, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	set := make(map[string]struct{}, len(want))
	for _, w := range want {
		set[w] = struct{}{}
	}
	for _, g := range got {
		if _, ok := set[g.String()]; !ok {
			return false
		}
	}
	return true
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

// loadAuthorityKey reads the vendored upgrade-authority keypair as a raw Solana
// private key (for signing token-setup txs that need more than the fee payer).
func loadAuthorityKey(t *testing.T) solana.PrivateKey {
	t.Helper()
	path := filepath.Join("..", "..", "..", "devnet", "sol-upgrade-authority.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authority keypair: %v", err)
	}
	var b []byte
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("parse authority keypair: %v", err)
	}
	return solana.PrivateKey(b)
}

// setupSPLMintFundVault creates a fresh 0-decimal mint (authority = mint
// authority), creates the vault's ATA, and mints `amount` tokens into it — so a
// subsequent SPL withdrawal has a funded vault to draw from. Returns the mint.
func setupSPLMintFundVault(ctx context.Context, t *testing.T, client *rpc.Client, authorityPub solana.PublicKey, authorityKey solana.PrivateKey, programID solana.PublicKey, amount uint64) solana.PublicKey {
	t.Helper()
	mintKP, err := solana.NewRandomPrivateKey()
	if err != nil {
		t.Fatalf("mint keypair: %v", err)
	}
	mintPub := mintKP.PublicKey()

	rent, err := client.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE, rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatalf("rent exemption: %v", err)
	}
	vaultATA, _, err := solana.FindAssociatedTokenAddress(VaultPDA(programID), mintPub)
	if err != nil {
		t.Fatalf("vault ATA: %v", err)
	}

	ixs := []solana.Instruction{
		system.NewCreateAccountInstruction(rent, token.MINT_SIZE, solana.TokenProgramID, authorityPub, mintPub).Build(),
		token.NewInitializeMint2Instruction(0, authorityPub, authorityPub, mintPub).Build(),
		associatedtokenaccount.NewCreateInstruction(authorityPub, VaultPDA(programID), mintPub).Build(),
		token.NewMintToInstruction(amount, mintPub, vaultATA, authorityPub, nil).Build(),
	}
	keys := map[solana.PublicKey]solana.PrivateKey{authorityPub: authorityKey, mintPub: mintKP}
	sendMultiSigned(ctx, t, client, ixs, authorityPub, keys)
	waitTokenBalance(ctx, t, client, vaultATA, amount)
	return mintPub
}

// createActiveALT creates an Address Lookup Table, extends it with addresses,
// and waits until it is populated and old enough to use in a v0 transaction.
func createActiveALT(ctx context.Context, t *testing.T, client *rpc.Client, authorityPub solana.PublicKey, authorityKey solana.PrivateKey, addresses []solana.PublicKey) solana.PublicKey {
	t.Helper()
	// CreateLookupTable requires recent_slot to be a slot already in the
	// SlotHashes sysvar — i.e. strictly in the past. The finalized slot always
	// lags the execution slot, so it is safely present; the current (confirmed)
	// slot can equal the execution slot and is rejected ("not a recent slot").
	slot, err := client.GetSlot(ctx, rpc.CommitmentFinalized)
	if err != nil {
		t.Fatalf("get slot: %v", err)
	}
	createIx, altAddr, err := addresslookuptable.NewCreateLookupTableInstruction(authorityPub, authorityPub, slot)
	if err != nil {
		t.Fatalf("create ALT ix: %v", err)
	}
	extendIx := addresslookuptable.NewExtendLookupTableInstruction(altAddr, authorityPub, authorityPub, addresses).Build()
	sendMultiSigned(ctx, t, client, []solana.Instruction{createIx.Build(), extendIx}, authorityPub, map[solana.PublicKey]solana.PrivateKey{authorityPub: authorityKey})

	// Wait until the table is readable with all addresses, then let it age a slot
	// (a lookup table cannot be used in the same slot it was extended).
	deadline := time.Now().Add(30 * time.Second)
	for {
		state, err := addresslookuptable.GetAddressLookupTable(ctx, client, altAddr)
		if err == nil && state != nil && len(state.Addresses) >= len(addresses) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("ALT %s not populated in time", altAddr)
		}
		time.Sleep(time.Second)
	}
	time.Sleep(2 * time.Second)
	return altAddr
}

// sendMultiSigned builds, signs (with every key the message needs), and sends a
// transaction, then waits for the vault-ATA balance to reflect it via the
// caller's own wait.
func sendMultiSigned(ctx context.Context, t *testing.T, client *rpc.Client, ixs []solana.Instruction, payer solana.PublicKey, keys map[solana.PublicKey]solana.PrivateKey) {
	t.Helper()
	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatalf("blockhash: %v", err)
	}
	tx, err := solana.NewTransaction(ixs, bh.Value.Blockhash, solana.TransactionPayer(payer))
	if err != nil {
		t.Fatalf("build tx: %v", err)
	}
	if _, err := tx.Sign(func(pub solana.PublicKey) *solana.PrivateKey {
		if k, ok := keys[pub]; ok {
			return &k
		}
		return nil
	}); err != nil {
		t.Fatalf("sign tx: %v", err)
	}
	if _, err := client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{PreflightCommitment: rpc.CommitmentConfirmed}); err != nil {
		t.Fatalf("send tx: %v", err)
	}
}

// waitTokenBalance blocks until the SPL token account holds at least min.
func waitTokenBalance(ctx context.Context, t *testing.T, client *rpc.Client, account solana.PublicKey, min uint64) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		res, err := client.GetTokenAccountBalance(ctx, account, rpc.CommitmentConfirmed)
		if err == nil && res != nil && res.Value != nil {
			if v, perr := strconv.ParseUint(res.Value.Amount, 10, 64); perr == nil && v >= min {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("token account %s did not reach %d in time", account, min)
		}
		time.Sleep(time.Second)
	}
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

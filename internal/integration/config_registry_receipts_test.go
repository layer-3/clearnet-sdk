//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/evm"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/receipt"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

const (
	defaultAnvilRPC      = "http://127.0.0.1:8545"
	defaultAnvilDeployer = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

func TestIntegration_ConfigRegistryReceipts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := ethclient.Dial(envOr("EVM_RPC_URL", defaultAnvilRPC))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	deployerKey, err := crypto.HexToECDSA(envOr("EVM_DEPLOYER_KEY", defaultAnvilDeployer))
	if err != nil {
		t.Fatalf("parse deployer key: %v", err)
	}
	deployer := sign.NewKeySignerFromECDSA(deployerKey)
	deployOpts := keyedTransactOpts(ctx, t, client, deployerKey)
	registryAddr, deployTx, registry, err := evm.DeployConfigRegistry(deployOpts, client)
	if err != nil {
		t.Fatalf("deploy ConfigRegistry: %v", err)
	}
	waitMined(ctx, t, client, deployTx)

	issuer1Keys := makeIssuerKeys(t, 7)
	issuer2Keys := makeIssuerKeys(t, 4)
	issuer1 := registerIssuer(ctx, t, client, registryAddr, deployer, issuer1Keys, 5)
	issuer2 := registerIssuer(ctx, t, client, registryAddr, deployer, issuer2Keys, 3)
	verifyConfigCommitIdempotency(ctx, t, client, registryAddr, registry, deployer, issuer1)

	store := newConfigEventStore()
	forwarder, err := evm.NewConfigRegistryEventForwarder(
		[]common.Address{issuer1.id, issuer2.id},
		[][32]byte{receipt.ConfigRegistrySignersKey},
		store,
	)
	if err != nil {
		t.Fatalf("forwarder: %v", err)
	}
	watcher, err := evm.NewConfigRegistryWatcher(client, registryAddr, registry, 0, forwarder)
	if err != nil {
		t.Fatalf("watcher: %v", err)
	}
	watcher.SetCursorSource(store)
	watcher.SetInitialLookback(100)
	watcher.SetPollInterval(100 * time.Millisecond)
	if err := watcher.Backfill(ctx); err != nil {
		t.Fatalf("watcher backfill: %v", err)
	}
	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()
	go func() {
		if err := watcher.Watch(watchCtx); err != nil {
			t.Logf("watcher stopped: %v", err)
		}
	}()

	writeSignerPayload(ctx, t, client, registryAddr, deployer, issuer1, issuer1.addrs, issuer1.threshold)
	writeSignerPayload(ctx, t, client, registryAddr, deployer, issuer2, issuer2.addrs, issuer2.threshold)
	if ev := waitForSignerEvent(ctx, t, store, registryAddr, issuer1.id, issuer1.threshold); ev.Epoch != 1 {
		t.Fatalf("issuer1 initial signer epoch = %d, want 1", ev.Epoch)
	}
	if ev := waitForSignerEvent(ctx, t, store, registryAddr, issuer2.id, issuer2.threshold); ev.Epoch != 1 {
		t.Fatalf("issuer2 initial signer epoch = %d, want 1", ev.Epoch)
	}

	src, err := receipt.NewRegistrySignerSource(registryAddr, store)
	if err != nil {
		t.Fatalf("registry signer source: %v", err)
	}
	resolver := withdrawalIssuerMap{}
	verifier := receipt.NewReceiptVerifier(src, resolver)

	verifyIssuerReceipts(ctx, t, verifier, resolver, issuer1, issuer2)
	verifyIssuerReceipts(ctx, t, verifier, resolver, issuer2, issuer1)

	verifyFilteredKeyAdvancesCursor(ctx, t, client, registryAddr, deployer, store, issuer1)
	verifyMalformedPayloadRecovery(ctx, t, client, registryAddr, deployer, store, src, issuer1)
	verifySignerPayloadOverwrite(ctx, t, client, registryAddr, deployer, store, verifier, resolver, issuer1, issuer2)
	stopWatch()
	verifyWatcherResumeFromCursor(ctx, t, client, registryAddr, registry, deployer, store, issuer2)
}

type issuerFixture struct {
	id        common.Address
	keys      []*ecdsa.PrivateKey
	signers   []sign.Signer
	addrs     []common.Address
	threshold int
}

type issuerKeys struct {
	keys    []*ecdsa.PrivateKey
	signers []sign.Signer
	addrs   []common.Address
}

func registerIssuer(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, keys issuerKeys, threshold int) issuerFixture {
	t.Helper()
	keysHex := addrsToHex(keys.addrs)
	finalizers := makeRegistrationFinalizers(ctx, t, client, registry, payer, keys.signers)
	packed, err := finalizers[0].Pack(ctx, keysHex, threshold)
	if err != nil {
		t.Fatalf("register pack: %v", err)
	}
	sigs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, keysHex, threshold); err != nil {
			t.Fatalf("register validate[%d]: %v", i, err)
		}
		sig, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("register sign[%d]: %v", i, err)
		}
		sigs = append(sigs, sig)
	}
	result, err := finalizers[0].Submit(ctx, packed, sigs)
	if err != nil {
		t.Fatalf("register submit: %v", err)
	}
	return issuerFixture{id: result.IssuerID, keys: keys.keys, signers: keys.signers, addrs: keys.addrs, threshold: threshold}
}

func writeSignerPayload(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, issuer issuerFixture, signers []common.Address, threshold int) {
	t.Helper()
	payload, err := receipt.MarshalReceiptSignerPayload(core.ReceiptSignerSet{Signers: signers, Threshold: threshold})
	if err != nil {
		t.Fatalf("marshal signer payload: %v", err)
	}
	writeConfigData(ctx, t, client, registry, payer, issuer, receipt.ConfigRegistrySignersKey, payload)
}

func writeConfigData(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, issuer issuerFixture, key [32]byte, data []byte) string {
	t.Helper()
	finalizers := makeCommitFinalizers(ctx, t, client, registry, payer, issuer)
	packed, err := finalizers[0].PackWithData(ctx, key, data)
	if err != nil {
		t.Fatalf("commit pack: %v", err)
	}
	sigs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.ValidateWithData(ctx, packed, key, data); err != nil {
			t.Fatalf("commit validate[%d]: %v", i, err)
		}
		sig, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("commit sign[%d]: %v", i, err)
		}
		sigs = append(sigs, sig)
	}
	txID, err := finalizers[0].Submit(ctx, packed, sigs)
	if err != nil {
		t.Fatalf("commit submit: %v", err)
	}
	return txID
}

func writeConfigChecksum(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, issuer issuerFixture, key [32]byte, checksum [32]byte) string {
	t.Helper()
	finalizers := makeCommitFinalizers(ctx, t, client, registry, payer, issuer)
	packed, err := finalizers[0].Pack(ctx, key, checksum)
	if err != nil {
		t.Fatalf("checksum commit pack: %v", err)
	}
	sigs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, key, checksum); err != nil {
			t.Fatalf("checksum commit validate[%d]: %v", i, err)
		}
		sig, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("checksum commit sign[%d]: %v", i, err)
		}
		sigs = append(sigs, sig)
	}
	txID, err := finalizers[0].Submit(ctx, packed, sigs)
	if err != nil {
		t.Fatalf("checksum commit submit: %v", err)
	}
	return txID
}

func verifyConfigCommitIdempotency(ctx context.Context, t *testing.T, client *ethclient.Client, registryAddr common.Address, registry *evm.ConfigRegistry, payer sign.Signer, issuer issuerFixture) {
	t.Helper()
	var repairKey [32]byte
	repairKey[31] = 0xD1
	repairData := []byte("payload initially missing")
	repairChecksum := crypto.Keccak256Hash(repairData)

	checksumTx := writeConfigChecksum(ctx, t, client, registryAddr, payer, issuer, repairKey, repairChecksum)
	if got := countConfigWithDataCommitted(ctx, t, client, registry, issuer.id, repairKey, repairChecksum); got != 0 {
		t.Fatalf("checksum-only commit emitted ConfigWithDataCommitted: got %d", got)
	}
	verifier := makeCommitFinalizers(ctx, t, client, registryAddr, payer, issuer)[0]
	if txID, done, err := verifier.VerifyCommit(ctx, repairKey, repairChecksum); err != nil {
		t.Fatalf("VerifyCommit after checksum-only write: %v", err)
	} else if !done || txID != checksumTx {
		t.Fatalf("VerifyCommit after checksum-only write = txID %q done %v, want txID %q done true", txID, done, checksumTx)
	}
	if txID, done, err := verifier.VerifyCommitWithData(ctx, repairKey, repairData); err != nil {
		t.Fatalf("VerifyCommitWithData after checksum-only write: %v", err)
	} else if done || txID != "" {
		t.Fatalf("VerifyCommitWithData after checksum-only write = txID %q done %v, want empty txID done false", txID, done)
	}
	repairTx := writeConfigData(ctx, t, client, registryAddr, payer, issuer, repairKey, repairData)
	if repairTx == "" || repairTx == checksumTx {
		t.Fatalf("setConfigWithData repair did not submit a new tx: checksum=%q repair=%q", checksumTx, repairTx)
	}
	if got := countConfigWithDataCommitted(ctx, t, client, registry, issuer.id, repairKey, repairChecksum); got != 1 {
		t.Fatalf("ConfigWithDataCommitted repair count = %d, want 1", got)
	}
	if txID, done, err := verifier.VerifyCommitWithData(ctx, repairKey, repairData); err != nil {
		t.Fatalf("VerifyCommitWithData after repair write: %v", err)
	} else if !done || txID != repairTx {
		t.Fatalf("VerifyCommitWithData after repair write = txID %q done %v, want txID %q done true", txID, done, repairTx)
	}

	nonceBeforeSkip := readIssuerNonce(ctx, t, registry, issuer.id)
	skipTx := writeConfigData(ctx, t, client, registryAddr, payer, issuer, repairKey, repairData)
	if skipTx != repairTx {
		t.Fatalf("setConfigWithData idempotent tx = %q, want original repair tx %q", skipTx, repairTx)
	}
	if nonceAfterSkip := readIssuerNonce(ctx, t, registry, issuer.id); nonceAfterSkip.Cmp(nonceBeforeSkip) != 0 {
		t.Fatalf("setConfigWithData idempotent skip changed nonce: before=%s after=%s", nonceBeforeSkip, nonceAfterSkip)
	}
	if got := countConfigWithDataCommitted(ctx, t, client, registry, issuer.id, repairKey, repairChecksum); got != 1 {
		t.Fatalf("ConfigWithDataCommitted idempotent count = %d, want 1", got)
	}

	var checksumKey [32]byte
	checksumKey[31] = 0xD2
	checksumOnly := crypto.Keccak256Hash([]byte("checksum-only idempotency"))
	firstChecksumTx := writeConfigChecksum(ctx, t, client, registryAddr, payer, issuer, checksumKey, checksumOnly)
	if got := countConfigCommitted(ctx, t, client, registry, issuer.id, checksumKey, checksumOnly); got != 1 {
		t.Fatalf("ConfigCommitted count after first checksum write = %d, want 1", got)
	}
	if txID, done, err := verifier.VerifyCommit(ctx, checksumKey, checksumOnly); err != nil {
		t.Fatalf("VerifyCommit after first checksum write: %v", err)
	} else if !done || txID != firstChecksumTx {
		t.Fatalf("VerifyCommit after first checksum write = txID %q done %v, want txID %q done true", txID, done, firstChecksumTx)
	}
	nonceBeforeChecksumSkip := readIssuerNonce(ctx, t, registry, issuer.id)
	secondChecksumTx := writeConfigChecksum(ctx, t, client, registryAddr, payer, issuer, checksumKey, checksumOnly)
	if secondChecksumTx != firstChecksumTx {
		t.Fatalf("setConfig idempotent tx = %q, want original tx %q", secondChecksumTx, firstChecksumTx)
	}
	if nonceAfterChecksumSkip := readIssuerNonce(ctx, t, registry, issuer.id); nonceAfterChecksumSkip.Cmp(nonceBeforeChecksumSkip) != 0 {
		t.Fatalf("setConfig idempotent skip changed nonce: before=%s after=%s", nonceBeforeChecksumSkip, nonceAfterChecksumSkip)
	}
	if got := countConfigCommitted(ctx, t, client, registry, issuer.id, checksumKey, checksumOnly); got != 1 {
		t.Fatalf("ConfigCommitted count after idempotent checksum write = %d, want 1", got)
	}
}

func verifyIssuerReceipts(ctx context.Context, t *testing.T, verifier *receipt.ReceiptVerifier, resolver withdrawalIssuerMap, issuer issuerFixture, other issuerFixture) {
	t.Helper()
	mint := &core.MintReceipt{
		TxID:     "mint/" + issuer.id.Hex(),
		Account:  "yellow://ynet/user/0xabc",
		AssetURI: core.AssetURI("yellow://ynet/asset/" + issuer.id.Hex() + "/evm/31337/0"),
		Amount:   decimal.NewFromInt(1),
	}
	signMint(t, mint, issuer.keys[:issuer.threshold]...)
	if err := verifier.VerifyMintReceipt(ctx, mint); err != nil {
		t.Fatalf("valid mint for issuer %s: %v", issuer.id.Hex(), err)
	}

	wrongMint := cloneMint(mint)
	signMint(t, wrongMint, other.keys[:other.threshold]...)
	if err := verifier.VerifyMintReceipt(ctx, wrongMint); err == nil {
		t.Fatalf("wrong-issuer mint signatures verified for issuer %s", issuer.id.Hex())
	}

	lowMint := cloneMint(mint)
	signMint(t, lowMint, issuer.keys[:issuer.threshold-1]...)
	if err := verifier.VerifyMintReceipt(ctx, lowMint); err == nil {
		t.Fatalf("below-threshold mint verified for issuer %s", issuer.id.Hex())
	}

	dupMint := cloneMint(mint)
	signMint(t, dupMint, issuer.keys[0], issuer.keys[0], issuer.keys[0], issuer.keys[0], issuer.keys[0])
	if err := verifier.VerifyMintReceipt(ctx, dupMint); err == nil {
		t.Fatalf("duplicate mint signatures verified for issuer %s", issuer.id.Hex())
	}

	burn := &core.BurnReceipt{
		WithdrawalID:  crypto.Keccak256Hash([]byte("burn/" + issuer.id.Hex())),
		BlockEntryRef: core.BlockEntryRef{BlockHash: crypto.Keccak256Hash([]byte("block/" + issuer.id.Hex())), EntryIndex: 1},
		TxID:          "burn-tx/" + issuer.id.Hex(),
		Status:        core.WithdrawalExecuted,
	}
	resolver[burn.WithdrawalID] = issuer.id
	signBurn(t, burn, issuer.keys[:issuer.threshold]...)
	if err := verifier.VerifyBurnReceipt(ctx, burn); err != nil {
		t.Fatalf("valid burn for issuer %s: %v", issuer.id.Hex(), err)
	}

	wrongBurn := cloneBurn(burn)
	signBurn(t, wrongBurn, other.keys[:other.threshold]...)
	if err := verifier.VerifyBurnReceipt(ctx, wrongBurn); err == nil {
		t.Fatalf("wrong-issuer burn signatures verified for issuer %s", issuer.id.Hex())
	}

	lowBurn := cloneBurn(burn)
	signBurn(t, lowBurn, issuer.keys[:issuer.threshold-1]...)
	if err := verifier.VerifyBurnReceipt(ctx, lowBurn); err == nil {
		t.Fatalf("below-threshold burn verified for issuer %s", issuer.id.Hex())
	}

	dupBurn := cloneBurn(burn)
	signBurn(t, dupBurn, issuer.keys[0], issuer.keys[0], issuer.keys[0], issuer.keys[0], issuer.keys[0])
	if err := verifier.VerifyBurnReceipt(ctx, dupBurn); err == nil {
		t.Fatalf("duplicate burn signatures verified for issuer %s", issuer.id.Hex())
	}
}

func verifyFilteredKeyAdvancesCursor(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, store *configEventStore, issuer issuerFixture) {
	t.Helper()
	before := store.eventCount()
	var otherKey [32]byte
	otherKey[31] = 0xA7
	txID := writeConfigData(ctx, t, client, registry, payer, issuer, otherKey, []byte("not receipt signers"))
	waitForCursorTx(ctx, t, store, registry, common.HexToHash(txID))
	if _, ok, err := store.LatestConfigRegistryEvent(ctx, registry, issuer.id, otherKey); err != nil {
		t.Fatalf("latest other key event: %v", err)
	} else if ok {
		t.Fatal("filtered non-signer key was stored as a signer event")
	}
	if got := store.eventCount(); got != before {
		t.Fatalf("filtered key changed event count: got %d want %d", got, before)
	}
}

func verifyMalformedPayloadRecovery(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, store *configEventStore, src *receipt.RegistrySignerSource, issuer issuerFixture) {
	t.Helper()
	writeConfigData(ctx, t, client, registry, payer, issuer, receipt.ConfigRegistrySignersKey, []byte("not-json"))
	if ev := waitForRawSignerEvent(ctx, t, store, registry, issuer.id, []byte("not-json")); ev.Epoch != 2 {
		t.Fatalf("malformed signer payload epoch = %d, want 2", ev.Epoch)
	}
	if _, err := src.LoadReceiptSigners(ctx, issuer.id); err == nil {
		t.Fatal("malformed signer payload loaded successfully")
	}
	writeSignerPayload(ctx, t, client, registry, payer, issuer, issuer.addrs, issuer.threshold)
	if ev := waitForSignerEvent(ctx, t, store, registry, issuer.id, issuer.threshold); ev.Epoch != 3 {
		t.Fatalf("recovered signer payload epoch = %d, want 3", ev.Epoch)
	}
	if _, err := src.LoadReceiptSigners(ctx, issuer.id); err != nil {
		t.Fatalf("valid signer payload did not recover source: %v", err)
	}
}

func verifySignerPayloadOverwrite(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, store *configEventStore, verifier *receipt.ReceiptVerifier, resolver withdrawalIssuerMap, issuer issuerFixture, other issuerFixture) {
	t.Helper()
	oldMint := &core.MintReceipt{
		TxID:     "pre-rotation/" + issuer.id.Hex(),
		Account:  "yellow://ynet/user/0xabc",
		AssetURI: core.AssetURI("yellow://ynet/asset/" + issuer.id.Hex() + "/evm/31337/0"),
		Amount:   decimal.NewFromInt(1),
	}
	signMint(t, oldMint, issuer.keys[:issuer.threshold]...)
	if err := verifier.VerifyMintReceipt(ctx, oldMint); err != nil {
		t.Fatalf("old signer set should verify before overwrite: %v", err)
	}

	next := makeIssuerKeys(t, 7)
	nextThreshold := 4
	writeSignerPayload(ctx, t, client, registry, payer, issuer, next.addrs, nextThreshold)
	if ev := waitForSignerEvent(ctx, t, store, registry, issuer.id, nextThreshold); ev.Epoch != 4 {
		t.Fatalf("overwritten signer payload epoch = %d, want 4", ev.Epoch)
	}

	if err := verifier.VerifyMintReceipt(ctx, oldMint); err == nil {
		t.Fatal("old signer set still verified after KEY_SIGNERS overwrite")
	}
	newMint := cloneMint(oldMint)
	newMint.TxID = "post-rotation/" + issuer.id.Hex()
	signMint(t, newMint, next.keys[:nextThreshold]...)
	if err := verifier.VerifyMintReceipt(ctx, newMint); err != nil {
		t.Fatalf("new signer set did not verify after overwrite: %v", err)
	}

	verifyIssuerReceipts(ctx, t, verifier, resolver, other, issuerFixture{keys: next.keys, threshold: nextThreshold})
}

func verifyWatcherResumeFromCursor(ctx context.Context, t *testing.T, client *ethclient.Client, registryAddr common.Address, registry *evm.ConfigRegistry, payer sign.Signer, store *configEventStore, issuer issuerFixture) {
	t.Helper()
	cursorBefore, ok, err := store.LatestConfigRegistryCursor(ctx, registryAddr)
	if err != nil {
		t.Fatalf("cursor before restart: %v", err)
	}
	if !ok {
		t.Fatal("expected cursor before watcher restart")
	}

	forwarder, err := evm.NewConfigRegistryEventForwarder([]common.Address{issuer.id}, [][32]byte{receipt.ConfigRegistrySignersKey}, store)
	if err != nil {
		t.Fatalf("resume forwarder: %v", err)
	}
	watcher, err := evm.NewConfigRegistryWatcher(client, registryAddr, registry, 0, forwarder)
	if err != nil {
		t.Fatalf("resume watcher: %v", err)
	}
	watcher.SetCursorSource(store)
	watcher.SetInitialLookback(100)
	watcher.SetPollInterval(100 * time.Millisecond)
	if err := watcher.Backfill(ctx); err != nil {
		t.Fatalf("resume backfill: %v", err)
	}
	if cur, ok := watcher.Cursor(); !ok || cur != cursorBefore {
		t.Fatalf("resume cursor = %+v ok=%v, want %+v", cur, ok, cursorBefore)
	}
	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()
	go func() {
		if err := watcher.Watch(watchCtx); err != nil {
			t.Logf("resumed watcher stopped: %v", err)
		}
	}()

	before := store.eventWriteCount(registryAddr, issuer.id, receipt.ConfigRegistrySignersKey)
	next := makeIssuerKeys(t, len(issuer.addrs))
	writeSignerPayload(ctx, t, client, registryAddr, payer, issuer, next.addrs, issuer.threshold)
	waitForSignerEventWrites(ctx, t, store, registryAddr, issuer.id, before+1)
	if ev := waitForSignerEvent(ctx, t, store, registryAddr, issuer.id, issuer.threshold); ev.Epoch != 2 {
		t.Fatalf("post-resume signer payload epoch = %d, want 2", ev.Epoch)
	}
	if got := store.eventWriteCount(registryAddr, issuer.id, receipt.ConfigRegistrySignersKey); got != before+1 {
		t.Fatalf("resume watcher replayed old signer events: got writes %d want %d", got, before+1)
	}
}

type configEventStore struct {
	mu          sync.RWMutex
	events      map[configEventKey]core.ConfigRegistryEvent
	eventWrites map[configEventKey]int
	cursor      map[common.Address]core.ConfigRegistryCursor
}

type configEventKey struct {
	registry common.Address
	issuer   common.Address
	key      [32]byte
}

func newConfigEventStore() *configEventStore {
	return &configEventStore{
		events:      make(map[configEventKey]core.ConfigRegistryEvent),
		eventWrites: make(map[configEventKey]int),
		cursor:      make(map[common.Address]core.ConfigRegistryCursor),
	}
}

func (s *configEventStore) StoreConfigRegistryEvent(_ context.Context, ev core.ConfigRegistryEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := configEventKey{registry: ev.Registry, issuer: ev.IssuerID, key: ev.Key}
	s.events[key] = ev
	s.eventWrites[key]++
	s.cursor[ev.Registry] = core.ConfigRegistryCursor{Registry: ev.Registry, BlockNumber: ev.BlockNumber, LogIndex: ev.LogIndex, TxHash: ev.TxHash}
	return nil
}

func (s *configEventStore) StoreConfigRegistryCursor(_ context.Context, cursor core.ConfigRegistryCursor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursor[cursor.Registry] = cursor
	return nil
}

func (s *configEventStore) LatestConfigRegistryCursor(_ context.Context, registry common.Address) (core.ConfigRegistryCursor, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cur, ok := s.cursor[registry]
	return cur, ok, nil
}

func (s *configEventStore) LatestConfigRegistryEvent(_ context.Context, registry common.Address, issuerID common.Address, key [32]byte) (core.ConfigRegistryEvent, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ev, ok := s.events[configEventKey{registry: registry, issuer: issuerID, key: key}]
	return ev, ok, nil
}

func (s *configEventStore) eventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

func (s *configEventStore) eventWriteCount(registry common.Address, issuerID common.Address, key [32]byte) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eventWrites[configEventKey{registry: registry, issuer: issuerID, key: key}]
}

type withdrawalIssuerMap map[[32]byte]common.Address

func (m withdrawalIssuerMap) IssuerIDByWithdrawalID(_ context.Context, withdrawalID [32]byte) (common.Address, error) {
	issuer, ok := m[withdrawalID]
	if !ok {
		return common.Address{}, fmt.Errorf("unknown withdrawal %x", withdrawalID)
	}
	return issuer, nil
}

func makeRegistrationFinalizers(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, signers []sign.Signer) []*evm.ConfigRegistryIssuerRegistrationFinalizer {
	t.Helper()
	txr := makeTransactor(ctx, t, client, payer)
	out := make([]*evm.ConfigRegistryIssuerRegistrationFinalizer, len(signers))
	for i, s := range signers {
		f, err := evm.NewConfigRegistryIssuerRegistrationFinalizer(ctx, client, registry, s, txr, evm.FeeConfig{})
		if err != nil {
			t.Fatalf("registration finalizer[%d]: %v", i, err)
		}
		out[i] = f
	}
	return out
}

func makeCommitFinalizers(ctx context.Context, t *testing.T, client *ethclient.Client, registry common.Address, payer sign.Signer, issuer issuerFixture) []*evm.ConfigRegistryCommitFinalizer {
	t.Helper()
	txr := makeTransactor(ctx, t, client, payer)
	out := make([]*evm.ConfigRegistryCommitFinalizer, len(issuer.signers))
	for i, s := range issuer.signers {
		f, err := evm.NewConfigRegistryCommitFinalizer(ctx, client, registry, issuer.id, s, txr, evm.FeeConfig{})
		if err != nil {
			t.Fatalf("commit finalizer[%d]: %v", i, err)
		}
		out[i] = f
	}
	return out
}

func makeTransactor(ctx context.Context, t *testing.T, client *ethclient.Client, payer sign.Signer) evm.Transactor {
	t.Helper()
	txr, err := evm.NewSignerTransactor(ctx, client, payer)
	if err != nil {
		t.Fatalf("transactor: %v", err)
	}
	return txr
}

func waitForSignerEvent(ctx context.Context, t *testing.T, store *configEventStore, registry common.Address, issuer common.Address, threshold int) core.ConfigRegistryEvent {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		ev, ok, err := store.LatestConfigRegistryEvent(ctx, registry, issuer, receipt.ConfigRegistrySignersKey)
		if err != nil {
			t.Fatalf("latest event: %v", err)
		}
		if ok {
			set, err := receipt.ParseSignerPayload(ev.Data)
			if err != nil {
				t.Fatalf("parse signer event payload: %v", err)
			}
			if len(set.Signers) > 0 && set.Threshold == threshold {
				return ev
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for signer event for issuer %s", issuer.Hex())
		case <-ticker.C:
		}
	}
}

func waitForRawSignerEvent(ctx context.Context, t *testing.T, store *configEventStore, registry common.Address, issuer common.Address, data []byte) core.ConfigRegistryEvent {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		ev, ok, err := store.LatestConfigRegistryEvent(ctx, registry, issuer, receipt.ConfigRegistrySignersKey)
		if err != nil {
			t.Fatalf("latest event: %v", err)
		}
		if ok && bytes.Equal(ev.Data, data) {
			return ev
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for raw signer event for issuer %s", issuer.Hex())
		case <-ticker.C:
		}
	}
}

func waitForCursorTx(ctx context.Context, t *testing.T, store *configEventStore, registry common.Address, tx common.Hash) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		cur, ok, err := store.LatestConfigRegistryCursor(ctx, registry)
		if err != nil {
			t.Fatalf("latest cursor: %v", err)
		}
		if ok && cur.TxHash == tx {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for cursor tx %s", tx.Hex())
		case <-ticker.C:
		}
	}
}

func waitForSignerEventWrites(ctx context.Context, t *testing.T, store *configEventStore, registry common.Address, issuer common.Address, want int) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if got := store.eventWriteCount(registry, issuer, receipt.ConfigRegistrySignersKey); got >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for signer event writes for issuer %s", issuer.Hex())
		case <-ticker.C:
		}
	}
}

func countConfigCommitted(ctx context.Context, t *testing.T, client *ethclient.Client, registry *evm.ConfigRegistry, issuer common.Address, key [32]byte, checksum [32]byte) int {
	t.Helper()
	head, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("block number: %v", err)
	}
	it, err := registry.FilterConfigCommitted(&bind.FilterOpts{Context: ctx, Start: 0, End: &head}, []common.Address{issuer}, [][32]byte{key})
	if err != nil {
		t.Fatalf("filter ConfigCommitted: %v", err)
	}
	defer it.Close()
	var count int
	for it.Next() {
		if it.Event.Checksum == checksum {
			count++
		}
	}
	if err := it.Error(); err != nil {
		t.Fatalf("iterate ConfigCommitted: %v", err)
	}
	return count
}

func countConfigWithDataCommitted(ctx context.Context, t *testing.T, client *ethclient.Client, registry *evm.ConfigRegistry, issuer common.Address, key [32]byte, checksum [32]byte) int {
	t.Helper()
	head, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("block number: %v", err)
	}
	it, err := registry.FilterConfigWithDataCommitted(&bind.FilterOpts{Context: ctx, Start: 0, End: &head}, []common.Address{issuer}, [][32]byte{key})
	if err != nil {
		t.Fatalf("filter ConfigWithDataCommitted: %v", err)
	}
	defer it.Close()
	var count int
	for it.Next() {
		if it.Event.Checksum == checksum {
			count++
		}
	}
	if err := it.Error(); err != nil {
		t.Fatalf("iterate ConfigWithDataCommitted: %v", err)
	}
	return count
}

func readIssuerNonce(ctx context.Context, t *testing.T, registry *evm.ConfigRegistry, issuer common.Address) *big.Int {
	t.Helper()
	nonce, err := registry.Nonce(&bind.CallOpts{Context: ctx}, issuer)
	if err != nil {
		t.Fatalf("read issuer nonce: %v", err)
	}
	return new(big.Int).Set(nonce)
}

func makeIssuerKeys(t *testing.T, n int) issuerKeys {
	t.Helper()
	type entry struct {
		key  *ecdsa.PrivateKey
		addr common.Address
	}
	entries := make([]entry, n)
	for i := range entries {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("generate signer key: %v", err)
		}
		entries[i] = entry{key: k, addr: crypto.PubkeyToAddress(k.PublicKey)}
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].addr[:], entries[j].addr[:]) < 0
	})
	keys := make([]*ecdsa.PrivateKey, n)
	signers := make([]sign.Signer, n)
	addrs := make([]common.Address, n)
	for i, e := range entries {
		keys[i] = e.key
		signers[i] = sign.NewKeySignerFromECDSA(e.key)
		addrs[i] = e.addr
	}
	return issuerKeys{keys: keys, signers: signers, addrs: addrs}
}

func signerAddresses(t *testing.T, signers []sign.Signer) []common.Address {
	t.Helper()
	out := make([]common.Address, len(signers))
	for i, s := range signers {
		addr, err := sign.EthAddress(s)
		if err != nil {
			t.Fatalf("signer address[%d]: %v", i, err)
		}
		out[i] = addr
	}
	return out
}

func addrsToHex(addrs []common.Address) []string {
	out := make([]string, len(addrs))
	for i, addr := range addrs {
		out[i] = addr.Hex()
	}
	return out
}

func thresholdFor(issuer issuerFixture) int {
	switch len(issuer.keys) {
	case 7:
		return 5
	case 4:
		return 3
	default:
		panic("unexpected issuer fixture")
	}
}

func signMint(t *testing.T, r *core.MintReceipt, keys ...*ecdsa.PrivateKey) {
	t.Helper()
	digest := receipt.MintReceiptDigest(r)
	r.Signatures = signDigest(t, digest, keys...)
}

func signBurn(t *testing.T, r *core.BurnReceipt, keys ...*ecdsa.PrivateKey) {
	t.Helper()
	digest := receipt.BurnReceiptDigest(r)
	r.Signatures = signDigest(t, digest, keys...)
}

func signDigest(t *testing.T, digest []byte, keys ...*ecdsa.PrivateKey) [][]byte {
	t.Helper()
	out := make([][]byte, len(keys))
	for i, key := range keys {
		sig, err := crypto.Sign(digest, key)
		if err != nil {
			t.Fatalf("sign digest[%d]: %v", i, err)
		}
		out[i] = sig
	}
	return out
}

func cloneMint(r *core.MintReceipt) *core.MintReceipt {
	cp := *r
	cp.Signatures = nil
	return &cp
}

func cloneBurn(r *core.BurnReceipt) *core.BurnReceipt {
	cp := *r
	cp.Signatures = nil
	return &cp
}

func keyedTransactOpts(ctx context.Context, t *testing.T, client *ethclient.Client, key *ecdsa.PrivateKey) *bind.TransactOpts {
	t.Helper()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chain id: %v", err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		t.Fatalf("transactor opts: %v", err)
	}
	opts.Context = ctx
	return opts
}

func waitMined(ctx context.Context, t *testing.T, client *ethclient.Client, tx *gethtypes.Transaction) {
	t.Helper()
	r, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		t.Fatalf("wait mined %s: %v", tx.Hash().Hex(), err)
	}
	if r.Status == 0 {
		t.Fatalf("transaction reverted %s", tx.Hash().Hex())
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

//go:build integration

package evm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// EVM full deposit + withdrawal flow against a real chain (the devnet anvil by
// default). Self-bootstrapping: deploys a fresh Custody vault whose signer set
// is N freshly-generated keys, then exercises the SDK depositor + the quorum
// withdrawal finalizer end-to-end. Build-tagged `integration`; run with:
//
//	go test -tags integration ./pkg/blockchain/evm/ -run TestIntegrationEVM -v
//
// Env (defaults target `make devnet`):
//   EVM_RPC_URL      — default http://127.0.0.1:8545
//   EVM_DEPLOYER_KEY — hex privkey, funded; default anvil account 0

const (
	defaultAnvilRPC        = "http://127.0.0.1:8545"
	defaultAnvilDeployer   = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	integrationSignerCount = 3
	integrationThreshold   = 2
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestIntegrationEVM_DepositAndWithdraw(t *testing.T) {
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

	// N vault signers, each a fresh key funded by the deployer for gas.
	signers := make([]sign.Signer, integrationSignerCount)
	signerAddrs := make([]common.Address, integrationSignerCount)
	for i := range signers {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen signer key: %v", err)
		}
		signers[i] = sign.NewKeySignerFromECDSA(k)
		signerAddrs[i] = crypto.PubkeyToAddress(k.PublicKey)
		fundETH(ctx, t, client, deployerKey, signerAddrs[i], big.NewInt(1e18)) // 1 ETH for gas
	}
	// Custody's constructor requires initialSigners sorted ascending.
	sort.Slice(signerAddrs, func(i, j int) bool {
		return bytes.Compare(signerAddrs[i][:], signerAddrs[j][:]) < 0
	})

	// Deploy a fresh Custody vault over the signer set.
	depOpts, _, err := signerTransactOpts(ctx, client, deployer)
	if err != nil {
		t.Fatalf("deploy opts: %v", err)
	}
	custodyAddr, deployTx, _, err := DeployCustody(depOpts, client, signerAddrs, big.NewInt(integrationThreshold))
	if err != nil {
		t.Fatalf("deploy custody: %v", err)
	}
	if err := waitMined(ctx, client, deployTx); err != nil {
		t.Fatalf("deploy wait: %v", err)
	}
	deployedCustody, err := NewCustody(custodyAddr, client)
	if err != nil {
		t.Fatalf("bind deployed custody: %v", err)
	}
	window, err := deployedCustody.WITHDRAWALEXECUTIONWINDOW(nil)
	if err != nil {
		t.Fatalf("read deployed execution window: %v", err)
	}
	wantWindow := big.NewInt(int64(core.WithdrawalExecutionWindow / time.Second))
	if window.Cmp(wantWindow) != 0 {
		t.Fatalf("deployed execution window = %s, SDK = %s", window, wantWindow)
	}
	t.Logf("deployed Custody at %s (signers=%d threshold=%d)", custodyAddr.Hex(), integrationSignerCount, integrationThreshold)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	assets := NewAssetResolver(client, AssetResolverConfig{})
	depositor, err := NewDepositor(client, custodyAddr, deployer, assets)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	account := crypto.PubkeyToAddress(deployerKey.PublicKey)
	depositAmt := decimal.NewFromBigInt(big.NewInt(1_000_000_000_000), -18) // 1e12 wei
	depRef, err := depositor.SubmitDeposit(ctx, "", depositAmt, core.DepositDestination{Account: account.Hex()})
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chain id: %v", err)
	}
	if want := DepositID(chainID, custodyAddr, account, big.NewInt(0)); depRef.DepositID != want {
		t.Fatalf("DepositID = %s, want %s", depRef.DepositID, want)
	}
	if st, err := depositor.ChainDepositStatus(ctx, depRef.TxHash, depRef.DepositID, 1); err != nil || st != core.DepositConfirmed {
		t.Fatalf("ChainDepositStatus = (%v, %v), want confirmed", st, err)
	}
	t.Logf("deposit tx %s id %s", depRef.TxHash, depRef.DepositID)

	// ── Withdrawal flow (the quorum runs in-process) ──────────────────────────
	finalizers := make([]*WithdrawalFinalizer, len(signers))
	for i, s := range signers {
		payer, err := NewSignerTransactor(ctx, client, s)
		if err != nil {
			t.Fatalf("NewSignerTransactor %d: %v", i, err)
		}
		f, err := NewWithdrawalFinalizer(ctx, client, custodyAddr, s, payer, FeeConfig{}, assets)
		if err != nil {
			t.Fatalf("NewWithdrawalFinalizer %d: %v", i, err)
		}
		finalizers[i] = f
	}

	var withdrawalID [32]byte
	withdrawalID[0], withdrawalID[31] = 0x11, 0x22
	op := &core.WithdrawalOp{
		Recipient: signerAddrs[0].Hex(),
		AssetURI:  "yellow://ynet/asset/0x0000000000000000000000000000000000001234/evm/31337/0",
		Amount:    decimal.NewFromBigInt(big.NewInt(400_000_000_000), -18), // < deposited
	}

	// 1. Pack (any node — here the first) with realistic admission bounds.
	finalizedAt := time.Now().Unix()
	bounds, err := core.NewWithdrawalTimeBounds(finalizedAt-60, finalizedAt, finalizedAt, time.Minute)
	if err != nil {
		t.Fatalf("NewWithdrawalTimeBounds: %v", err)
	}
	packed, err := finalizers[0].Pack(ctx, op, withdrawalID, bounds)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	// 2. Every node validates then signs.
	sigs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, withdrawalID, bounds); err != nil {
			t.Fatalf("Validate[%d]: %v", i, err)
		}
		s, err := f.Sign(ctx, packed)
		if err != nil {
			t.Fatalf("Sign[%d]: %v", i, err)
		}
		sigs = append(sigs, s)
	}
	// 3. Submit (a submitter node merges the quorum and broadcasts).
	wRef, err := finalizers[0].Submit(ctx, packed, sigs)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	t.Logf("withdrawal tx %s", wRef)

	// 4. Verify execution.
	_, executed, err := finalizers[0].VerifyExecution(ctx, withdrawalID)
	if err != nil {
		t.Fatalf("VerifyExecution: %v", err)
	}
	if !executed {
		t.Fatal("withdrawal not reported executed")
	}

	// ── Rotation flow (the current quorum authorizes the new signer set) ──────
	newAddrs := make([]string, integrationSignerCount)
	for i := range newAddrs {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen new signer key: %v", err)
		}
		newAddrs[i] = crypto.PubkeyToAddress(k.PublicKey).Hex()
	}

	rotators := make([]*RotationFinalizer, len(signers))
	for i, s := range signers {
		payer, err := NewSignerTransactor(ctx, client, s)
		if err != nil {
			t.Fatalf("NewSignerTransactor %d: %v", i, err)
		}
		r, err := NewRotationFinalizer(ctx, client, custodyAddr, s, payer, FeeConfig{})
		if err != nil {
			t.Fatalf("NewRotationFinalizer %d: %v", i, err)
		}
		rotators[i] = r
	}

	var rotID [32]byte
	rotID[0], rotID[31] = 0xE0, 0x7A
	rPacked, err := rotators[0].Pack(ctx, rotID, newAddrs, integrationThreshold)
	if err != nil {
		t.Fatalf("rotation Pack: %v", err)
	}
	rSigs := make([][]byte, 0, len(rotators))
	for i, r := range rotators {
		if err := r.Validate(ctx, rotID, rPacked, newAddrs, integrationThreshold); err != nil {
			t.Fatalf("rotation Validate[%d]: %v", i, err)
		}
		s, err := r.Sign(ctx, rPacked)
		if err != nil {
			t.Fatalf("rotation Sign[%d]: %v", i, err)
		}
		rSigs = append(rSigs, s)
	}
	rRef, err := rotators[0].Submit(ctx, rPacked, rSigs)
	if err != nil {
		t.Fatalf("rotation Submit: %v", err)
	}
	t.Logf("rotation tx %s", rRef)

	if _, done, err := rotators[0].VerifyRotation(ctx, newAddrs, integrationThreshold); err != nil {
		t.Fatalf("VerifyRotation: %v", err)
	} else if !done {
		t.Fatal("rotation not reported done")
	}
}

// The deposit nonce flow against a real chain: consecutive nonces, a
// non-default key, the pending-to-confirmed transition, a wrong ID, a stale
// nonce, and an ERC-20 retry that sends no second approve.
func TestIntegrationEVM_DepositNonceFlow(t *testing.T) {
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
	depositorAddr := crypto.PubkeyToAddress(deployerKey.PublicKey)

	signerAddrs := make([]common.Address, integrationSignerCount)
	for i := range signerAddrs {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("gen signer key: %v", err)
		}
		signerAddrs[i] = crypto.PubkeyToAddress(k.PublicKey)
	}
	sort.Slice(signerAddrs, func(i, j int) bool {
		return bytes.Compare(signerAddrs[i][:], signerAddrs[j][:]) < 0
	})
	opts, _, err := signerTransactOpts(ctx, client, deployer)
	if err != nil {
		t.Fatalf("deploy opts: %v", err)
	}
	custodyAddr, deployTx, custody, err := DeployCustody(opts, client, signerAddrs, big.NewInt(integrationThreshold))
	if err != nil {
		t.Fatalf("deploy custody: %v", err)
	}
	if err := waitMined(ctx, client, deployTx); err != nil {
		t.Fatalf("deploy wait: %v", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chain id: %v", err)
	}
	depositor, err := NewDepositor(client, custodyAddr, deployer, NewAssetResolver(client, AssetResolverConfig{}))
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	dest := core.DepositDestination{Account: depositorAddr.Hex()}
	amount := decimal.NewFromBigInt(big.NewInt(1_000_000_000_000), -18)

	// Consecutive nonces n and n+1, IDs from (chainid, vault, depositor, nonce).
	var first core.SubmitDepositResult
	for seq := int64(0); seq < 2; seq++ {
		res, err := depositor.SubmitDeposit(ctx, "", amount, dest)
		if err != nil {
			t.Fatalf("deposit %d: %v", seq, err)
		}
		if want := DepositID(chainID, custodyAddr, depositorAddr, big.NewInt(seq)); res.DepositID != want {
			t.Fatalf("deposit %d: DepositID = %s, want %s", seq, res.DepositID, want)
		}
		if seq == 0 {
			first = res
		}
	}

	// Key 7 uses nonce 7<<64.
	keyed, err := depositor.SubmitDepositWithKey(ctx, "", amount, dest, big.NewInt(7))
	if err != nil {
		t.Fatalf("keyed deposit: %v", err)
	}
	if want := DepositID(chainID, custodyAddr, depositorAddr, ComposeNonce(big.NewInt(7), 0)); keyed.DepositID != want {
		t.Fatalf("keyed DepositID = %s, want %s", keyed.DepositID, want)
	}

	// Pending below the requested depth, confirmed once enough blocks exist.
	head, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("block number: %v", err)
	}
	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(keyed.TxHash))
	if err != nil {
		t.Fatalf("receipt: %v", err)
	}
	depth := head - receipt.BlockNumber.Uint64() + 1
	if st, err := depositor.ChainDepositStatus(ctx, keyed.TxHash, keyed.DepositID, depth+2); err != nil || st != core.DepositPending {
		t.Fatalf("below depth: (%v, %v), want pending", st, err)
	}
	for i := 0; i < 2; i++ {
		if err := client.Client().CallContext(ctx, nil, "evm_mine"); err != nil {
			t.Fatalf("evm_mine: %v", err)
		}
	}
	if st, err := depositor.ChainDepositStatus(ctx, keyed.TxHash, keyed.DepositID, depth+2); err != nil || st != core.DepositConfirmed {
		t.Fatalf("at depth: (%v, %v), want confirmed", st, err)
	}

	// A wrong ID for a real deposit transaction is absent.
	if st, err := depositor.ChainDepositStatus(ctx, first.TxHash, keyed.DepositID, 1); err != nil || st != core.DepositAbsent {
		t.Fatalf("wrong ID: (%v, %v), want absent", st, err)
	}

	// Stale nonce: with automine off, a direct deposit from the same account
	// takes nonce 2 in the pool; SubmitDeposit still reads 2 at `latest`, and
	// its transaction reverts (or fails estimation) once the first mines.
	setAutomine := func(on bool) {
		if err := client.Client().CallContext(ctx, nil, "evm_setAutomine", on); err != nil {
			t.Fatalf("evm_setAutomine(%v): %v", on, err)
		}
	}
	setAutomine(false)
	defer setAutomine(true)
	directOpts, _, err := signerTransactOpts(ctx, client, deployer)
	if err != nil {
		t.Fatalf("direct opts: %v", err)
	}
	directOpts.Value = big.NewInt(1_000_000_000_000)
	if _, err := custody.Deposit(directOpts, depositorAddr, common.Address{}, directOpts.Value, [32]byte{}, big.NewInt(2)); err != nil {
		t.Fatalf("direct deposit: %v", err)
	}
	staleErr := make(chan error, 1)
	go func() {
		_, err := depositor.SubmitDeposit(ctx, "", amount, dest)
		staleErr <- err
	}()
	mineUntil := time.After(30 * time.Second)
	for done := false; !done; {
		select {
		case err := <-staleErr:
			if !errors.Is(err, ErrStaleNonce) {
				t.Fatalf("stale deposit err = %v, want ErrStaleNonce", err)
			}
			done = true
		case <-time.After(500 * time.Millisecond):
			if err := client.Client().CallContext(ctx, nil, "evm_mine"); err != nil {
				t.Fatalf("evm_mine: %v", err)
			}
		case <-mineUntil:
			t.Fatal("stale deposit did not return")
		}
	}
	setAutomine(true)

	// ERC-20: an allowance left by an earlier approve means the retry sends
	// only the deposit transaction.
	tokenAddr, tokenTx, token, err := DeployMockERC20(mustOpts(ctx, t, client, deployer), client, "Test", "TST")
	if err != nil {
		t.Fatalf("deploy token: %v", err)
	}
	if err := waitMined(ctx, client, tokenTx); err != nil {
		t.Fatalf("token deploy wait: %v", err)
	}
	units := big.NewInt(1_000_000_000_000_000_000)
	mintTx, err := token.Mint(mustOpts(ctx, t, client, deployer), depositorAddr, units)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if err := waitMined(ctx, client, mintTx); err != nil {
		t.Fatalf("mint wait: %v", err)
	}
	approveTx, err := token.Approve(mustOpts(ctx, t, client, deployer), custodyAddr, units)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := waitMined(ctx, client, approveTx); err != nil {
		t.Fatalf("approve wait: %v", err)
	}
	before, err := client.PendingNonceAt(ctx, depositorAddr)
	if err != nil {
		t.Fatalf("account nonce: %v", err)
	}
	if _, err := depositor.SubmitDeposit(ctx, tokenAddr.Hex(), decimal.NewFromInt(1), dest); err != nil {
		t.Fatalf("ERC-20 deposit: %v", err)
	}
	after, err := client.PendingNonceAt(ctx, depositorAddr)
	if err != nil {
		t.Fatalf("account nonce: %v", err)
	}
	if after-before != 1 {
		t.Fatalf("ERC-20 deposit sent %d transactions, want 1 (no approve)", after-before)
	}
}

func mustOpts(ctx context.Context, t *testing.T, client *ethclient.Client, s sign.Signer) *bind.TransactOpts {
	t.Helper()
	opts, _, err := signerTransactOpts(ctx, client, s)
	if err != nil {
		t.Fatalf("transact opts: %v", err)
	}
	return opts
}

// fundETH sends value from key to addr via a raw anvil tx and waits for it.
func fundETH(ctx context.Context, t *testing.T, client *ethclient.Client, key *ecdsa.PrivateKey, to common.Address, value *big.Int) {
	t.Helper()
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("chain id: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("gas price: %v", err)
	}
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      21000,
		GasPrice: gasPrice,
	})
	signed, err := gethtypes.SignTx(tx, gethtypes.LatestSignerForChainID(chainID), key)
	if err != nil {
		t.Fatalf("sign fund tx: %v", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("send fund tx: %v", err)
	}
	if err := waitMined(ctx, client, signed); err != nil {
		t.Fatalf("fund wait: %v", err)
	}
}

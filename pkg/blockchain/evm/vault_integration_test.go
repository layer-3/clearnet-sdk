//go:build integration

package evm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"
	"sort"
	"testing"
	"time"

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
	t.Logf("deployed Custody at %s (signers=%d threshold=%d)", custodyAddr.Hex(), integrationSignerCount, integrationThreshold)

	// ── Deposit flow ──────────────────────────────────────────────────────────
	depositor, err := NewDepositor(client, custodyAddr, deployer)
	if err != nil {
		t.Fatalf("NewDepositor: %v", err)
	}
	account := crypto.PubkeyToAddress(deployerKey.PublicKey)
	const zeroAsset = "0x0000000000000000000000000000000000000000" // native ETH
	depositAmt := decimal.NewFromInt(1_000_000_000_000)            // 1e12 wei
	depRef, err := depositor.SubmitDeposit(ctx, zeroAsset, depositAmt, account.Hex())
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	t.Logf("deposit tx %s", depRef.Raw)

	// ── Withdrawal flow (the quorum runs in-process) ──────────────────────────
	finalizers := make([]*WithdrawalFinalizer, len(signers))
	for i, s := range signers {
		f, err := NewWithdrawalFinalizer(ctx, client, custodyAddr, s, FeeConfig{})
		if err != nil {
			t.Fatalf("NewWithdrawalFinalizer %d: %v", i, err)
		}
		finalizers[i] = f
	}

	var withdrawalID [32]byte
	withdrawalID[0], withdrawalID[31] = 0x11, 0x22
	op := &core.WithdrawalOp{
		Recipient: signerAddrs[0].Hex(),
		L1Asset:   zeroAsset,
		Amount:    decimal.NewFromInt(400_000_000_000), // < deposited
	}

	// 1. Pack (any node — here the first).
	packed, err := finalizers[0].Pack(ctx, op, withdrawalID)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	// 2. Every node validates then signs.
	sigs := make([][]byte, 0, len(finalizers))
	for i, f := range finalizers {
		if err := f.Validate(ctx, packed, op, withdrawalID); err != nil {
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
	t.Logf("withdrawal tx %s", wRef.Raw)

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
		r, err := NewRotationFinalizer(ctx, client, custodyAddr, s, FeeConfig{})
		if err != nil {
			t.Fatalf("NewRotationFinalizer %d: %v", i, err)
		}
		rotators[i] = r
	}

	rPacked, err := rotators[0].Pack(ctx, newAddrs, integrationThreshold)
	if err != nil {
		t.Fatalf("rotation Pack: %v", err)
	}
	rSigs := make([][]byte, 0, len(rotators))
	for i, r := range rotators {
		if err := r.Validate(ctx, rPacked, newAddrs, integrationThreshold); err != nil {
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
	t.Logf("rotation tx %s", rRef.Raw)

	if _, done, err := rotators[0].VerifyRotation(ctx, newAddrs, integrationThreshold); err != nil {
		t.Fatalf("VerifyRotation: %v", err)
	} else if !done {
		t.Fatal("rotation not reported done")
	}
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

package evm

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// executedLookupWindow bounds the eth_getLogs range when resolving the tx hash
// of an already-executed withdrawal.
const executedLookupWindow = uint64(50_000)

// FeeConfig tunes the submit transaction's gas pricing. Zero values fall back
// to sensible defaults (no cap; 1.5x gas-limit margin).
type FeeConfig struct {
	TipGwei            float64 // EIP-1559 priority tip
	CapGwei            float64 // refuse to submit above this effective price (0 = no cap)
	GasLimitMultiplier float64 // safety margin over eth_estimateGas (0 => 1.5)
}

func (f FeeConfig) gasLimitMultiplier() float64 {
	if f.GasLimitMultiplier <= 0 {
		return 1.5
	}
	return f.GasLimitMultiplier
}

// WithdrawalFinalizer turns an authorized withdrawal into a Custody.execute
// call. It owns the node's signer (used both to sign the k-of-n digest and to
// submit the tx) and the vault address + chain id supplied at construction. It
// implements core.VaultWithdrawalFinalizer.
type WithdrawalFinalizer struct {
	client     *ethclient.Client
	custody    *Custody
	vaultAddr  common.Address
	chainID    uint64
	signer     sign.Signer
	signerAddr common.Address
	fees       FeeConfig
}

var _ core.VaultWithdrawalFinalizer = (*WithdrawalFinalizer)(nil)

// NewWithdrawalFinalizer binds the Custody vault at vaultAddr and reads the
// chain id from client. signer is this node's secp256k1 identity.
func NewWithdrawalFinalizer(ctx context.Context, client *ethclient.Client, vaultAddr common.Address, signer sign.Signer, fees FeeConfig) (*WithdrawalFinalizer, error) {
	custody, err := NewCustody(vaultAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load custody: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	addr, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalFinalizer{
		client:     client,
		custody:    custody,
		vaultAddr:  vaultAddr,
		chainID:    chainID.Uint64(),
		signer:     signer,
		signerAddr: addr,
		fees:       fees,
	}, nil
}

// evmPacked is the canonical withdrawal payload: enough to recompute the
// signing digest and to rebuild the execute() call.
type evmPacked struct {
	To           string `json:"to"`           // recipient address (hex)
	Asset        string `json:"asset"`        // asset address (hex); zero = ETH
	Amount       string `json:"amount"`       // base units (decimal string)
	WithdrawalID string `json:"withdrawalId"` // 32-byte hex
}

// evmMerged is evmPacked plus the ordered, contract-ready signatures.
type evmMerged struct {
	evmPacked
	Sigs []string `json:"sigs"` // 65-byte sigs, sorted by signer, V ∈ {27,28}, hex
}

// Pack returns the canonical JSON for the withdrawal. Pure — no chain access.
func (f *WithdrawalFinalizer) Pack(_ context.Context, op *core.WithdrawalOp, withdrawalID [32]byte) ([]byte, error) {
	return json.Marshal(packedFromOp(op, withdrawalID))
}

// Validate re-derives the canonical payload from the op and asserts the packed
// bytes match it exactly — the defense against a Byzantine packer.
func (f *WithdrawalFinalizer) Validate(_ context.Context, packed []byte, op *core.WithdrawalOp, withdrawalID [32]byte) error {
	var got evmPacked
	if err := json.Unmarshal(packed, &got); err != nil {
		return fmt.Errorf("decode packed: %w", err)
	}
	want := packedFromOp(op, withdrawalID)
	if got != want {
		return fmt.Errorf("packed withdrawal does not match op: got %+v want %+v", got, want)
	}
	return nil
}

// Sign produces this node's 65-byte ECDSA signature (V ∈ {0,1}) over the
// withdrawal digest derived from the packed bytes.
func (f *WithdrawalFinalizer) Sign(ctx context.Context, packed []byte) ([]byte, error) {
	var p evmPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return nil, fmt.Errorf("decode packed: %w", err)
	}
	digest, err := f.digest(p)
	if err != nil {
		return nil, err
	}
	return sign.SignEthDigest(ctx, f.signer, digest[:], f.signerAddr)
}

// Merge filters the collected signatures against the live on-chain signer set,
// trims to the live threshold, orders them by signer address (Custody.sol
// requires ascending, no duplicates), shifts V to {27,28}, and returns the
// merged artifact.
func (f *WithdrawalFinalizer) Merge(ctx context.Context, packed []byte, signatures [][]byte) ([]byte, error) {
	var p evmPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return nil, fmt.Errorf("decode packed: %w", err)
	}
	digest, err := f.digest(p)
	if err != nil {
		return nil, err
	}

	liveSigners, liveThreshold, err := f.liveQuorum(ctx)
	if err != nil {
		return nil, err
	}
	authorized := make(map[common.Address]struct{}, len(liveSigners))
	for _, a := range liveSigners {
		authorized[a] = struct{}{}
	}

	type sigAddr struct {
		sig  []byte
		addr common.Address
	}
	kept := make([]sigAddr, 0, len(signatures))
	seen := make(map[common.Address]struct{})
	for _, s := range signatures {
		if len(s) != 65 {
			return nil, fmt.Errorf("signature has wrong length %d", len(s))
		}
		pub, err := crypto.SigToPub(digest[:], s)
		if err != nil {
			return nil, fmt.Errorf("recover signer: %w", err)
		}
		addr := crypto.PubkeyToAddress(*pub)
		if _, ok := authorized[addr]; !ok {
			continue // not in the live signer set
		}
		if _, dup := seen[addr]; dup {
			continue
		}
		seen[addr] = struct{}{}
		kept = append(kept, sigAddr{sig: s, addr: addr})
	}
	if len(kept) < liveThreshold {
		return nil, fmt.Errorf("only %d of %d authorized signatures", len(kept), liveThreshold)
	}
	// Custody.sol's _verifySignatures stops at `threshold` and rejects extras.
	kept = kept[:liveThreshold]
	// Ascending uint160 order == bytes order over [20]byte.
	sort.Slice(kept, func(i, j int) bool { return bytes.Compare(kept[i].addr[:], kept[j].addr[:]) < 0 })

	merged := evmMerged{evmPacked: p, Sigs: make([]string, len(kept))}
	for i, k := range kept {
		cp := make([]byte, 65)
		copy(cp, k.sig)
		if cp[64] < 27 {
			cp[64] += 27 // shift V {0,1} -> {27,28} at the contract boundary
		}
		merged.Sigs[i] = hex.EncodeToString(cp)
	}
	return json.Marshal(merged)
}

// Submit broadcasts the merged artifact via Custody.execute and returns the tx
// reference. Idempotent: if the withdrawal is already executed it returns the
// prior tx hash without re-submitting.
func (f *WithdrawalFinalizer) Submit(ctx context.Context, merged []byte) (core.TxRef, error) {
	var m evmMerged
	if err := json.Unmarshal(merged, &m); err != nil {
		return core.TxRef{}, fmt.Errorf("decode merged: %w", err)
	}
	wid, err := decodeHex32(m.WithdrawalID)
	if err != nil {
		return core.TxRef{}, err
	}
	if txHash, executed, err := f.VerifyExecution(ctx, wid); err != nil {
		return core.TxRef{}, err
	} else if executed {
		return core.TxRef{Hash: txHash, Raw: common.Hash(txHash).Hex()}, nil
	}

	to := common.HexToAddress(m.To)
	asset := common.HexToAddress(m.Asset)
	amount, ok := new(big.Int).SetString(m.Amount, 10)
	if !ok {
		return core.TxRef{}, fmt.Errorf("bad amount %q", m.Amount)
	}
	sigs := make([][]byte, len(m.Sigs))
	for i, s := range m.Sigs {
		b, err := hex.DecodeString(s)
		if err != nil {
			return core.TxRef{}, fmt.Errorf("decode sig %d: %w", i, err)
		}
		sigs[i] = b
	}

	opts, _, err := signerTransactOpts(ctx, f.client, f.signer)
	if err != nil {
		return core.TxRef{}, err
	}
	if err := f.applyFees(ctx, opts); err != nil {
		return core.TxRef{}, err
	}
	if err := f.estimateGas(ctx, opts, to, asset, amount, wid, sigs); err != nil {
		return core.TxRef{}, err
	}
	tx, err := f.custody.Execute(opts, to, asset, amount, wid, sigs)
	if err != nil {
		return core.TxRef{}, fmt.Errorf("execute: %w", err)
	}
	// Block until mined so the returned ref corresponds to an executed
	// withdrawal (and a subsequent VerifyExecution observes it).
	if err := waitMined(ctx, f.client, tx); err != nil {
		return core.TxRef{}, err
	}
	return core.TxRef{Hash: tx.Hash(), Raw: tx.Hash().Hex()}, nil
}

// VerifyExecution reads Custody.executed(id) and, when set, looks up the
// Executed event's tx hash within the lookback window.
func (f *WithdrawalFinalizer) VerifyExecution(ctx context.Context, withdrawalID [32]byte) ([32]byte, bool, error) {
	executed, err := f.custody.Executed(&bind.CallOpts{Context: ctx}, withdrawalID)
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("check executed: %w", err)
	}
	if !executed {
		return [32]byte{}, false, nil
	}
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		return [32]byte{}, true, nil // executed; hash unknown
	}
	var from uint64
	if head > executedLookupWindow {
		from = head - executedLookupWindow
	}
	it, err := f.custody.FilterExecuted(&bind.FilterOpts{Context: ctx, Start: from, End: &head}, [][32]byte{withdrawalID}, nil)
	if err != nil {
		return [32]byte{}, true, nil
	}
	defer it.Close()
	if it.Next() {
		return it.Event.Raw.TxHash, true, nil
	}
	return [32]byte{}, true, nil
}

// --- helpers ---

func packedFromOp(op *core.WithdrawalOp, withdrawalID [32]byte) evmPacked {
	return evmPacked{
		To:           common.HexToAddress(op.Recipient).Hex(),
		Asset:        common.HexToAddress(op.L1Asset).Hex(),
		Amount:       op.Amount.BigInt().String(),
		WithdrawalID: hex.EncodeToString(withdrawalID[:]),
	}
}

// digest computes the Custody.execute signing digest:
// keccak256(abi.encode(chainId, vault, to, asset, amount, withdrawalId)).
func (f *WithdrawalFinalizer) digest(p evmPacked) (common.Hash, error) {
	amount, ok := new(big.Int).SetString(p.Amount, 10)
	if !ok {
		return common.Hash{}, fmt.Errorf("bad amount %q", p.Amount)
	}
	wid, err := decodeHex32(p.WithdrawalID)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(
		common.LeftPadBytes(new(big.Int).SetUint64(f.chainID).Bytes(), 32),
		common.LeftPadBytes(f.vaultAddr.Bytes(), 32),
		common.LeftPadBytes(common.HexToAddress(p.To).Bytes(), 32),
		common.LeftPadBytes(common.HexToAddress(p.Asset).Bytes(), 32),
		common.LeftPadBytes(amount.Bytes(), 32),
		wid[:],
	), nil
}

func (f *WithdrawalFinalizer) liveQuorum(ctx context.Context) ([]common.Address, int, error) {
	signers, err := f.custody.Signers(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read signers: %w", err)
	}
	thr, err := f.custody.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(signers)) {
		return nil, 0, fmt.Errorf("on-chain threshold %s out of range for %d signers", thr, len(signers))
	}
	return signers, int(thr.Int64()), nil
}

func (f *WithdrawalFinalizer) applyFees(ctx context.Context, opts *bind.TransactOpts) error {
	tip := gweiToWei(f.fees.TipGwei)
	cap := gweiToWei(f.fees.CapGwei)
	head, err := f.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("fee head: %w", err)
	}
	if head.BaseFee == nil {
		price, err := f.client.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("suggest gas price: %w", err)
		}
		if cap.Sign() > 0 && price.Cmp(cap) > 0 {
			return fmt.Errorf("gas price %s exceeds cap", price)
		}
		opts.GasPrice = price
		return nil
	}
	maxFee := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	if cap.Sign() > 0 && maxFee.Cmp(cap) > 0 {
		return fmt.Errorf("max fee %s exceeds cap", maxFee)
	}
	opts.GasTipCap = tip
	opts.GasFeeCap = maxFee
	return nil
}

func (f *WithdrawalFinalizer) estimateGas(ctx context.Context, opts *bind.TransactOpts, to, asset common.Address, amount *big.Int, withdrawalID [32]byte, sigs [][]byte) error {
	abi, err := CustodyMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("parse ABI: %w", err)
	}
	data, err := abi.Pack("execute", to, asset, amount, withdrawalID, sigs)
	if err != nil {
		return fmt.Errorf("pack execute calldata: %w", err)
	}
	est, err := f.client.EstimateGas(ctx, ethereum.CallMsg{
		From:      f.signerAddr,
		To:        &f.vaultAddr,
		Data:      data,
		GasTipCap: opts.GasTipCap,
		GasFeeCap: opts.GasFeeCap,
		GasPrice:  opts.GasPrice,
	})
	if err != nil {
		return fmt.Errorf("estimate gas: %w", err)
	}
	opts.GasLimit = uint64(float64(est) * f.fees.gasLimitMultiplier())
	return nil
}

func gweiToWei(g float64) *big.Int {
	if g <= 0 {
		return new(big.Int)
	}
	wei, _ := new(big.Float).Mul(big.NewFloat(g), big.NewFloat(1e9)).Int(nil)
	return wei
}

func decodeHex32(s string) ([32]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return [32]byte{}, fmt.Errorf("bad 32-byte hex %q (len=%d): %v", s, len(b), err)
	}
	var out [32]byte
	copy(out[:], b)
	return out, nil
}

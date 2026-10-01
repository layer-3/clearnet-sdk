package evm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// The tests in this file drive Depositor against fakeDepositChain, an
// in-process JSON-RPC server that answers the eth_* calls SubmitDeposit and
// ChainDepositStatus make and models the vault's getNonce counter. They pin
// the nonce flow without a real chain; the devnet integration test covers the
// same flow against anvil.

var (
	flowChainID = big.NewInt(31337)
	flowVault   = common.HexToAddress("0x00000000000000000000000000000000000000aa")
	flowToken   = common.HexToAddress("0x00000000000000000000000000000000000000bb")
	flowAccount = "0x000102030405060708090a0b0c0d0e0f10111213"
)

// fakeDepositChain is the state behind the fake RPC server.
type fakeDepositChain struct {
	mu        sync.Mutex
	custody   abi.ABI
	erc20     abi.ABI
	sequences map[string]uint64 // key.String() -> next sequence
	allowance *big.Int
	head      uint64

	// Scenario knobs.
	estimateErr error                                                       // returned by eth_estimateGas for deposit calls
	sendErr     error                                                       // returned by eth_sendRawTransaction
	revertMined bool                                                        // the deposit mines with status 0
	bumpOnMine  bool                                                        // with revertMined: another deposit consumed the nonce first
	depositLogs func(depositor common.Address, nonce *big.Int) []*types.Log // overrides the vault log

	calls    []string // "getNonce", "allowance", "send:approve", "send:deposit", ...
	receipts map[common.Hash]*types.Receipt
	pending  map[common.Hash]*types.Transaction
	txs      []*types.Transaction
}

func newFakeDepositChain(t *testing.T) *fakeDepositChain {
	t.Helper()
	custodyABI, err := CustodyMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	erc20ABI, err := MockERC20MetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	return &fakeDepositChain{
		custody:   *custodyABI,
		erc20:     *erc20ABI,
		sequences: map[string]uint64{},
		allowance: big.NewInt(0),
		head:      100,
		receipts:  map[common.Hash]*types.Receipt{},
		pending:   map[common.Hash]*types.Transaction{},
	}
}

func (c *fakeDepositChain) record(call string) {
	c.calls = append(c.calls, call)
}

func (c *fakeDepositChain) depositedLog(depositor common.Address, nonce *big.Int, emitter common.Address) *types.Log {
	event := c.custody.Events["Deposited"]
	data, err := event.Inputs.NonIndexed().Pack(depositor, common.Address{}, big.NewInt(1), nonce)
	if err != nil {
		panic(err)
	}
	return &types.Log{
		Address: emitter,
		Topics:  []common.Hash{event.ID, common.BytesToHash(common.FromHex(flowAccount)), {}},
		Data:    data,
	}
}

// fakeEthAPI is the "eth" RPC namespace served over fakeDepositChain.
type fakeEthAPI struct{ c *fakeDepositChain }

type fakeCallArgs struct {
	From  *common.Address `json:"from"`
	To    *common.Address `json:"to"`
	Data  hexutil.Bytes   `json:"data"`
	Input hexutil.Bytes   `json:"input"`
}

func (a fakeCallArgs) calldata() []byte {
	if len(a.Input) > 0 {
		return a.Input
	}
	return a.Data
}

func (api *fakeEthAPI) ChainId() *hexutil.Big { return (*hexutil.Big)(flowChainID) }

func (api *fakeEthAPI) BlockNumber() hexutil.Uint64 {
	api.c.mu.Lock()
	defer api.c.mu.Unlock()
	return hexutil.Uint64(api.c.head)
}

func (api *fakeEthAPI) GasPrice() *hexutil.Big { return (*hexutil.Big)(big.NewInt(1_000_000_000)) }

func (api *fakeEthAPI) GetTransactionCount(common.Address, json.RawMessage) hexutil.Uint64 {
	api.c.mu.Lock()
	defer api.c.mu.Unlock()
	return hexutil.Uint64(len(api.c.txs))
}

func (api *fakeEthAPI) GetCode(common.Address, json.RawMessage) hexutil.Bytes {
	return hexutil.Bytes{0x60, 0x00}
}

func (api *fakeEthAPI) GetBlockByNumber(json.RawMessage, bool) *types.Header {
	api.c.mu.Lock()
	defer api.c.mu.Unlock()
	return &types.Header{Number: new(big.Int).SetUint64(api.c.head), Difficulty: big.NewInt(0), GasLimit: 30_000_000}
}

func (api *fakeEthAPI) Call(args fakeCallArgs, _ *json.RawMessage) (hexutil.Bytes, error) {
	c := api.c
	c.mu.Lock()
	defer c.mu.Unlock()
	data := args.calldata()
	if len(data) < 4 {
		return nil, errors.New("fake: short calldata")
	}
	switch {
	case args.To != nil && *args.To == flowVault && string(data[:4]) == string(c.custody.Methods["getNonce"].ID):
		in, err := c.custody.Methods["getNonce"].Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		key := in[1].(*big.Int)
		c.record("getNonce")
		return c.custody.Methods["getNonce"].Outputs.Pack(ComposeNonce(key, c.sequences[key.String()]))
	case args.To != nil && *args.To == flowToken && string(data[:4]) == string(c.erc20.Methods["allowance"].ID):
		c.record("allowance")
		return c.erc20.Methods["allowance"].Outputs.Pack(c.allowance)
	}
	return nil, errors.New("fake: unexpected eth_call")
}

func (api *fakeEthAPI) EstimateGas(args fakeCallArgs, _ *json.RawMessage) (hexutil.Uint64, error) {
	c := api.c
	c.mu.Lock()
	defer c.mu.Unlock()
	if args.To != nil && *args.To == flowVault && c.estimateErr != nil {
		return 0, c.estimateErr
	}
	return 200_000, nil
}

func (api *fakeEthAPI) SendRawTransaction(raw hexutil.Bytes) (common.Hash, error) {
	c := api.c
	c.mu.Lock()
	defer c.mu.Unlock()
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(raw); err != nil {
		return common.Hash{}, err
	}
	step := "approve"
	if *tx.To() == flowVault {
		step = "deposit"
	}
	c.record("send:" + step)
	if c.sendErr != nil {
		return common.Hash{}, c.sendErr
	}
	c.txs = append(c.txs, tx)

	receipt := &types.Receipt{
		Status:      types.ReceiptStatusSuccessful,
		TxHash:      tx.Hash(),
		BlockNumber: new(big.Int).SetUint64(c.head),
		Logs:        []*types.Log{},
	}
	if step == "approve" {
		in, err := c.erc20.Methods["approve"].Inputs.Unpack(tx.Data()[4:])
		if err != nil {
			return common.Hash{}, err
		}
		c.allowance = in[1].(*big.Int)
	} else {
		in, err := c.custody.Methods["deposit"].Inputs.Unpack(tx.Data()[4:])
		if err != nil {
			return common.Hash{}, err
		}
		nonce := in[4].(*big.Int)
		key, seq := SplitNonce(nonce)
		from, err := types.Sender(types.LatestSignerForChainID(flowChainID), tx)
		if err != nil {
			return common.Hash{}, err
		}
		switch {
		case c.revertMined:
			receipt.Status = types.ReceiptStatusFailed
			if c.bumpOnMine {
				c.sequences[key.String()] = seq + 1
			}
		case c.depositLogs != nil:
			receipt.Logs = c.depositLogs(from, nonce)
			c.sequences[key.String()] = seq + 1
		default:
			receipt.Logs = []*types.Log{c.depositedLog(from, nonce, flowVault)}
			c.sequences[key.String()] = seq + 1
		}
	}
	for i, l := range receipt.Logs {
		l.TxHash = tx.Hash()
		l.BlockNumber = c.head
		l.Index = uint(i)
	}
	c.receipts[tx.Hash()] = receipt
	return tx.Hash(), nil
}

func (api *fakeEthAPI) GetTransactionReceipt(h common.Hash) *types.Receipt {
	api.c.mu.Lock()
	defer api.c.mu.Unlock()
	return api.c.receipts[h]
}

func (api *fakeEthAPI) GetTransactionByHash(h common.Hash) *types.Transaction {
	api.c.mu.Lock()
	defer api.c.mu.Unlock()
	return api.c.pending[h]
}

type flowAssets struct{}

func (flowAssets) ValidateAssetAddress(context.Context, string) error { return nil }
func (flowAssets) AssetDecimals(context.Context, string) (uint8, error) {
	return 18, nil
}

// newFlowDepositor starts the fake chain and returns a Depositor bound to it
// plus the depositor's address.
func newFlowDepositor(t *testing.T) (*Depositor, *fakeDepositChain, common.Address) {
	t.Helper()
	chain := newFakeDepositChain(t)
	srv := rpc.NewServer()
	if err := srv.RegisterName("eth", &fakeEthAPI{c: chain}); err != nil {
		t.Fatal(err)
	}
	rpcClient := rpc.DialInProc(srv)
	t.Cleanup(func() {
		rpcClient.Close()
		srv.Stop()
	})
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDepositor(ethclient.NewClient(rpcClient), flowVault, sign.NewKeySignerFromECDSA(key), flowAssets{})
	if err != nil {
		t.Fatal(err)
	}
	return d, chain, crypto.PubkeyToAddress(key.PublicKey)
}

func flowSubmit(t *testing.T, d *Depositor, asset string, key *big.Int) (core.SubmitDepositResult, error) {
	t.Helper()
	return d.SubmitDepositWithKey(context.Background(), asset, decimal.NewFromInt(1), core.DepositDestination{Account: flowAccount}, key)
}

func asSubmitError(t *testing.T, err error) *DepositSubmitError {
	t.Helper()
	var se *DepositSubmitError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v, want *DepositSubmitError", err)
	}
	return se
}

// Two sequential deposits use nonces n and n+1, and each returned ID is the
// hash of (chainid, vault, depositor, nonce) confirmed from the receipt log.
func TestSubmitDeposit_SequentialDepositsUseConsecutiveNonces(t *testing.T) {
	d, chain, depositor := newFlowDepositor(t)
	chain.sequences["0"] = 5

	for _, seq := range []uint64{5, 6} {
		res, err := d.SubmitDeposit(context.Background(), "", decimal.NewFromInt(1), core.DepositDestination{Account: flowAccount})
		if err != nil {
			t.Fatal(err)
		}
		want := DepositID(flowChainID, flowVault, depositor, ComposeNonce(big.NewInt(0), seq))
		if res.DepositID != want {
			t.Fatalf("sequence %d: DepositID = %s, want %s", seq, res.DepositID, want)
		}
		if res.TxHash != chain.txs[len(chain.txs)-1].Hash().Hex() {
			t.Fatalf("sequence %d: TxHash = %s, want the sent transaction's hash", seq, res.TxHash)
		}
	}
}

// SubmitDepositWithKey reads and uses the nonce of the requested key.
func TestSubmitDepositWithKey_UsesRequestedKey(t *testing.T) {
	d, chain, depositor := newFlowDepositor(t)

	res, err := flowSubmit(t, d, "", big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	nonce := new(big.Int).Lsh(big.NewInt(7), 64)
	if want := DepositID(flowChainID, flowVault, depositor, nonce); res.DepositID != want {
		t.Fatalf("DepositID = %s, want the ID of nonce 7<<64", res.DepositID)
	}
	if chain.sequences["7"] != 1 || chain.sequences["0"] != 0 {
		t.Fatalf("sequences = %v, want only key 7 advanced", chain.sequences)
	}
}

// For an ERC-20 the nonce is read before the allowance and the approve, and
// the approve is skipped when the allowance already covers the amount.
// A key outside [0, 2^192) is refused before any RPC. Without the check,
// 2^256 would be packed modulo 2^256 and deposit under key 0.
func TestSubmitDepositWithKey_RejectsOutOfRangeKey(t *testing.T) {
	for name, key := range map[string]*big.Int{
		"negative": big.NewInt(-1),
		"2^192":    new(big.Int).Lsh(big.NewInt(1), 192),
		"2^256":    new(big.Int).Lsh(big.NewInt(1), 256),
	} {
		t.Run(name, func(t *testing.T) {
			d, chain, _ := newFlowDepositor(t)
			if _, err := flowSubmit(t, d, "", key); err == nil {
				t.Fatal("SubmitDepositWithKey accepted an out-of-range key")
			}
			if len(chain.calls) != 0 || len(chain.txs) != 0 {
				t.Fatalf("calls = %v, txs = %d, want none", chain.calls, len(chain.txs))
			}
		})
	}
	d, _, _ := newFlowDepositor(t)
	if _, err := flowSubmit(t, d, "", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 192), big.NewInt(1))); err != nil {
		t.Fatalf("SubmitDepositWithKey(2^192-1) = %v, want success", err)
	}
}

func TestSubmitDeposit_ERC20ReadsNonceBeforeApprove(t *testing.T) {
	d, chain, _ := newFlowDepositor(t)

	if _, err := flowSubmit(t, d, flowToken.Hex(), nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"getNonce", "allowance", "send:approve", "send:deposit"}
	if len(chain.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", chain.calls, want)
	}
	for i := range want {
		if chain.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", chain.calls, want)
		}
	}

	// The first approve left an allowance of exactly the amount, which the
	// deposit did not spend in the fake; a second deposit sends no approve.
	chain.calls = nil
	if _, err := flowSubmit(t, d, flowToken.Hex(), nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range chain.calls {
		if c == "send:approve" {
			t.Fatalf("calls = %v, want no approve when the allowance covers the amount", chain.calls)
		}
	}
}

// A mined deposit whose receipt lacks the vault's Deposited log for the
// expected ID is an error carrying the hash and the ID, whether the receipt
// has no log or only a same-topic log from another contract.
func TestSubmitDeposit_EventNotFound(t *testing.T) {
	for name, logs := range map[string]func(*fakeDepositChain) func(common.Address, *big.Int) []*types.Log{
		"no log": func(*fakeDepositChain) func(common.Address, *big.Int) []*types.Log {
			return func(common.Address, *big.Int) []*types.Log { return []*types.Log{} }
		},
		"foreign emitter": func(c *fakeDepositChain) func(common.Address, *big.Int) []*types.Log {
			return func(depositor common.Address, nonce *big.Int) []*types.Log {
				return []*types.Log{c.depositedLog(depositor, nonce, common.HexToAddress("0xdead"))}
			}
		},
		"other nonce": func(c *fakeDepositChain) func(common.Address, *big.Int) []*types.Log {
			return func(depositor common.Address, nonce *big.Int) []*types.Log {
				return []*types.Log{c.depositedLog(depositor, new(big.Int).Add(nonce, big.NewInt(1)), flowVault)}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, chain, depositor := newFlowDepositor(t)
			chain.depositLogs = logs(chain)

			_, err := flowSubmit(t, d, "", nil)
			if !errors.Is(err, ErrDepositEventNotFound) {
				t.Fatalf("err = %v, want ErrDepositEventNotFound", err)
			}
			se := asSubmitError(t, err)
			if se.Step != "deposit" || se.TxHash != chain.txs[0].Hash().Hex() ||
				se.DepositID != DepositID(flowChainID, flowVault, depositor, big.NewInt(0)) ||
				se.Key.Sign() != 0 || se.Nonce.Sign() != 0 {
				t.Fatalf("error fields = %+v", se)
			}
		})
	}
}

// A deposit rejected with Custody's "Invalid nonce" before broadcast is
// ErrStaleNonce with the identity fields set and no transaction hash.
func TestSubmitDeposit_StaleNonceBeforeBroadcast(t *testing.T) {
	d, chain, depositor := newFlowDepositor(t)
	chain.sequences["0"] = 3
	chain.estimateErr = errors.New("execution reverted: Invalid nonce")

	_, err := flowSubmit(t, d, "", nil)
	if !errors.Is(err, ErrStaleNonce) {
		t.Fatalf("err = %v, want ErrStaleNonce", err)
	}
	se := asSubmitError(t, err)
	if se.Step != "deposit" || se.TxHash != "" || se.Nonce.Uint64() != 3 ||
		se.DepositID != DepositID(flowChainID, flowVault, depositor, big.NewInt(3)) {
		t.Fatalf("error fields = %+v", se)
	}
}

// A mined revert with no reason is ErrStaleNonce when a fresh getNonce shows
// the nonce was consumed, and a plain revert otherwise; both carry the hash.
func TestSubmitDeposit_MinedRevertClassification(t *testing.T) {
	for name, bumped := range map[string]bool{"nonce consumed": true, "nonce unused": false} {
		t.Run(name, func(t *testing.T) {
			d, chain, _ := newFlowDepositor(t)
			chain.revertMined = true
			chain.bumpOnMine = bumped

			_, err := flowSubmit(t, d, "", nil)
			se := asSubmitError(t, err)
			if se.TxHash != chain.txs[0].Hash().Hex() {
				t.Fatalf("TxHash = %q, want the mined transaction's hash", se.TxHash)
			}
			if errors.Is(err, ErrStaleNonce) != bumped {
				t.Fatalf("errors.Is(err, ErrStaleNonce) = %v, want %v (err = %v)", !bumped, bumped, err)
			}
		})
	}
}

// A broadcast failure keeps the already-known transaction hash.
func TestSubmitDeposit_SendFailureCarriesHash(t *testing.T) {
	d, chain, _ := newFlowDepositor(t)
	chain.sendErr = errors.New("connection reset")

	_, err := flowSubmit(t, d, "", nil)
	se := asSubmitError(t, err)
	if se.Step != "deposit" || se.TxHash == "" || se.DepositID == "" || se.Nonce == nil || se.Key == nil {
		t.Fatalf("error fields = %+v", se)
	}
}

// ChainDepositStatus: pending while unmined, pending below minConf, confirmed
// at depth, absent for a wrong ID, a foreign emitter or a reverted receipt,
// and an error for a malformed ID.
func TestChainDepositStatus(t *testing.T) {
	ctx := context.Background()
	d, chain, depositor := newFlowDepositor(t)
	res, err := flowSubmit(t, d, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	chain.head = 100 // mined at 100: one confirmation
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 3); err != nil || st != core.DepositPending {
		t.Fatalf("below depth: (%v, %v), want pending", st, err)
	}
	chain.head = 102
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 3); err != nil || st != core.DepositConfirmed {
		t.Fatalf("at depth: (%v, %v), want confirmed", st, err)
	}
	wrong := DepositID(flowChainID, flowVault, depositor, big.NewInt(99))
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, wrong, 1); err != nil || st != core.DepositAbsent {
		t.Fatalf("wrong ID: (%v, %v), want absent", st, err)
	}
	if _, err := d.ChainDepositStatus(ctx, res.TxHash, res.TxHash+"/0", 1); !errors.Is(err, ErrInvalidDepositID) {
		t.Fatalf("malformed ID: err = %v, want ErrInvalidDepositID", err)
	}

	// Same-topic log from another contract.
	hash := common.HexToHash(res.TxHash)
	receipt := chain.receipts[hash]
	receipt.Logs[0].Address = common.HexToAddress("0xdead")
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 1); err != nil || st != core.DepositAbsent {
		t.Fatalf("foreign emitter: (%v, %v), want absent", st, err)
	}
	receipt.Logs[0].Address = flowVault
	receipt.Status = types.ReceiptStatusFailed
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 1); err != nil || st != core.DepositAbsent {
		t.Fatalf("reverted: (%v, %v), want absent", st, err)
	}

	// Unmined but known to the node: pending.
	delete(chain.receipts, hash)
	chain.pending[hash] = chain.txs[0]
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 1); err != nil || st != core.DepositPending {
		t.Fatalf("unmined: (%v, %v), want pending", st, err)
	}
	delete(chain.pending, hash)
	if st, err := d.ChainDepositStatus(ctx, res.TxHash, res.DepositID, 1); err != nil || st != core.DepositAbsent {
		t.Fatalf("unknown: (%v, %v), want absent", st, err)
	}
}

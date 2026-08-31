package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

type quorumRPC struct {
	t                      *testing.T
	signersResult          string
	thresholdResult        string
	issuerSettingsResult   string
	signersSelector        string
	thresholdSelector      string
	issuerSettingsSelector string
	mu                     sync.Mutex
	callBlocks             []string
	issuerSettingsCalls    int
}

func newQuorumRPC(t *testing.T, signers []common.Address, threshold int) *quorumRPC {
	t.Helper()
	parsed, err := CustodyMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	signersResult, err := parsed.Methods["signers"].Outputs.Pack(signers)
	if err != nil {
		t.Fatal(err)
	}
	thresholdResult, err := parsed.Methods["threshold"].Outputs.Pack(big.NewInt(int64(threshold)))
	if err != nil {
		t.Fatal(err)
	}
	registryABI, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	issuerSettingsResult, err := registryABI.Methods["issuerSettings"].Outputs.Pack(signers, big.NewInt(int64(threshold)), big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	return &quorumRPC{
		t:                      t,
		signersResult:          hexutil.Encode(signersResult),
		thresholdResult:        hexutil.Encode(thresholdResult),
		issuerSettingsResult:   hexutil.Encode(issuerSettingsResult),
		signersSelector:        hexutil.Encode(parsed.Methods["signers"].ID),
		thresholdSelector:      hexutil.Encode(parsed.Methods["threshold"].ID),
		issuerSettingsSelector: hexutil.Encode(registryABI.Methods["issuerSettings"].ID),
	}
}

func (q *quorumRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		JSONRPC string            `json:"jsonrpc"`
		ID      json.RawMessage   `json:"id"`
		Method  string            `json:"method"`
		Params  []json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		q.t.Errorf("decode RPC request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var result any
	switch request.Method {
	case "eth_blockNumber":
		result = "0x2a"
	case "eth_call":
		var call struct {
			Input string `json:"input"`
			Data  string `json:"data"`
		}
		if len(request.Params) < 2 {
			q.t.Errorf("eth_call params = %d, want at least 2", len(request.Params))
			return
		}
		if err := json.Unmarshal(request.Params[0], &call); err != nil {
			q.t.Errorf("decode eth_call: %v", err)
			return
		}
		var block string
		if err := json.Unmarshal(request.Params[1], &block); err != nil {
			q.t.Errorf("decode eth_call block: %v", err)
			return
		}
		q.mu.Lock()
		q.callBlocks = append(q.callBlocks, block)
		q.mu.Unlock()
		input := call.Input
		if input == "" {
			input = call.Data
		}
		switch {
		case strings.HasPrefix(input, q.signersSelector):
			result = q.signersResult
		case strings.HasPrefix(input, q.thresholdSelector):
			result = q.thresholdResult
		case strings.HasPrefix(input, q.issuerSettingsSelector):
			q.mu.Lock()
			q.issuerSettingsCalls++
			q.mu.Unlock()
			result = q.issuerSettingsResult
		default:
			q.t.Errorf("unexpected eth_call input %q", input)
			result = "0x"
		}
	default:
		q.t.Errorf("unexpected RPC method %q", request.Method)
		result = nil
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
}

func TestFetchLiveIssuerQuorumUsesAtomicSettingsRead(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}
	rpcHandler := newQuorumRPC(t, signers, 2)
	server := httptest.NewServer(rpcHandler)
	defer server.Close()
	client, err := ethclient.Dial(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	registry, err := NewConfigRegistry(common.HexToAddress("0x000000000000000000000000000000000000beef"), client)
	if err != nil {
		t.Fatal(err)
	}
	gotSigners, gotThreshold, err := fetchLiveIssuerQuorum(context.Background(), registry, common.HexToAddress("0x000000000000000000000000000000000000b0b0"))
	if err != nil {
		t.Fatal(err)
	}
	if gotThreshold != 2 || len(gotSigners) != 2 || gotSigners[0] != signers[0] || gotSigners[1] != signers[1] {
		t.Fatalf("quorum = %v/%d, want %v/2", gotSigners, gotThreshold, signers)
	}
	rpcHandler.mu.Lock()
	defer rpcHandler.mu.Unlock()
	if rpcHandler.issuerSettingsCalls != 1 {
		t.Fatalf("issuerSettings calls = %d, want 1", rpcHandler.issuerSettingsCalls)
	}
}

func TestEVMPrepareSignatureValidatorsUseExactDigestAndPinnedQuorum(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}
	rpcHandler := newQuorumRPC(t, signers, 2)
	server := httptest.NewServer(rpcHandler)
	defer server.Close()
	client, err := ethclient.Dial(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	vault := common.HexToAddress("0x000000000000000000000000000000000000c057")
	custody, err := NewCustody(vault, client)
	if err != nil {
		t.Fatal(err)
	}

	withdrawal := &WithdrawalFinalizer{client: client, custody: custody, chainID: 1, vaultAddr: vault}
	withdrawalPacked, err := json.Marshal(evmPacked{
		To:    common.HexToAddress("0x0000000000000000000000000000000000000003").Hex(),
		Asset: common.Address{}.Hex(), Amount: "1", WithdrawalID: strings.Repeat("11", 32), Deadline: 123,
	})
	if err != nil {
		t.Fatal(err)
	}
	withdrawalValidator, err := withdrawal.PrepareSignatureValidator(context.Background(), withdrawalPacked)
	if err != nil {
		t.Fatal(err)
	}
	withdrawalDigest, err := withdrawal.digestFromPackedForTest(withdrawalPacked)
	if err != nil {
		t.Fatal(err)
	}
	if withdrawalValidator.Digest() != withdrawalDigest || withdrawalValidator.Threshold() != 2 {
		t.Fatalf("withdrawal prepared context = %x/%d, want %x/2", withdrawalValidator.Digest(), withdrawalValidator.Threshold(), withdrawalDigest)
	}

	rotation := &RotationFinalizer{client: client, custody: custody, chainID: 1, vaultAddr: vault}
	rotationPacked, err := json.Marshal(evmRotPacked{
		NewSigners: addrsToHex(signers), NewThreshold: 2, SignerNonce: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	rotationValidator, err := rotation.PrepareSignatureValidator(context.Background(), rotationPacked)
	if err != nil {
		t.Fatal(err)
	}
	rotationDigest, err := rotation.digestFromPacked(rotationPacked)
	if err != nil {
		t.Fatal(err)
	}
	if rotationValidator.Digest() != common.Hash(rotationDigest) || rotationValidator.Threshold() != 2 {
		t.Fatalf("rotation prepared context = %x/%d, want %x/2", rotationValidator.Digest(), rotationValidator.Threshold(), rotationDigest)
	}

	rpcHandler.mu.Lock()
	defer rpcHandler.mu.Unlock()
	if len(rpcHandler.callBlocks) != 4 {
		t.Fatalf("eth_call count = %d, want 4", len(rpcHandler.callBlocks))
	}
	for i, block := range rpcHandler.callBlocks {
		if block != "0x2a" {
			t.Fatalf("eth_call[%d] block = %q, want 0x2a", i, block)
		}
	}
}

func (f *WithdrawalFinalizer) digestFromPackedForTest(packed []byte) (common.Hash, error) {
	var p evmPacked
	if err := json.Unmarshal(packed, &p); err != nil {
		return common.Hash{}, fmt.Errorf("decode packed: %w", err)
	}
	return f.digest(p)
}

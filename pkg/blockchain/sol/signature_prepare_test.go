package sol

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"

	solcustody "github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

type configRPC struct {
	t           *testing.T
	programID   solana.PublicKey
	mu          sync.RWMutex
	data        string
	commitments []string
}

func newConfigRPC(t *testing.T, programID solana.PublicKey, cfg solcustody.Config) *configRPC {
	t.Helper()
	body, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	account := append(append([]byte(nil), solcustody.Account_Config[:]...), body...)
	return &configRPC{t: t, programID: programID, data: base64.StdEncoding.EncodeToString(account)}
}

func (s *configRPC) setConfig(cfg solcustody.Config) {
	s.t.Helper()
	body, err := cfg.Marshal()
	if err != nil {
		s.t.Fatal(err)
	}
	account := append(append([]byte(nil), solcustody.Account_Config[:]...), body...)
	s.mu.Lock()
	s.data = base64.StdEncoding.EncodeToString(account)
	s.mu.Unlock()
}

func (s *configRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.t.Errorf("decode RPC request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Method != "getAccountInfo" {
		s.t.Errorf("unexpected RPC method %q", request.Method)
	}
	if len(request.Params) != 2 {
		s.t.Errorf("getAccountInfo params = %d, want 2", len(request.Params))
	} else {
		var opts struct {
			Commitment string `json:"commitment"`
		}
		if err := json.Unmarshal(request.Params[1], &opts); err != nil {
			s.t.Errorf("decode getAccountInfo opts: %v", err)
		} else {
			s.mu.Lock()
			s.commitments = append(s.commitments, opts.Commitment)
			s.mu.Unlock()
		}
	}
	s.mu.RLock()
	data := s.data
	s.mu.RUnlock()
	result := map[string]any{
		"context": map[string]any{"slot": 1},
		"value": map[string]any{
			"data":       []any{data, "base64"},
			"executable": false,
			"lamports":   1,
			"owner":      s.programID.String(),
			"rentEpoch":  0,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
}

type signatureTestAssetResolver struct{}

func (signatureTestAssetResolver) ValidateAssetAddress(context.Context, string) error { return nil }
func (signatureTestAssetResolver) AssetDecimals(context.Context, string) (uint8, error) {
	return 9, nil
}

func TestSolanaPrepareSignatureValidatorsUseExactDigest(t *testing.T) {
	programID := solana.NewWallet().PublicKey()
	signers := []solana.PublicKey{solana.NewWallet().PublicKey(), solana.NewWallet().PublicKey()}
	config := newConfigRPC(t, programID, solcustody.Config{
		Signers: signers, Threshold: 2, RotationNonce: 7, ChainId: 1,
	})
	server := httptest.NewServer(config)
	defer server.Close()
	client := rpc.New(server.URL)

	withdrawal := &WithdrawalFinalizer{
		client: client, programID: programID, vaultPDA: VaultPDA(programID),
		chainID: 1, commitment: rpc.CommitmentConfirmed, assets: signatureTestAssetResolver{},
	}
	withdrawalPacked, err := json.Marshal(solPacked{
		To: solana.NewWallet().PublicKey().String(), Mint: solana.PublicKey{}.String(),
		Amount: 1, WithdrawalID: strings.Repeat("11", 32), FinalizedAt: 123, RotationNonce: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	withdrawalValidator, err := withdrawal.PrepareSignatureValidator(context.Background(), withdrawalPacked)
	if err != nil {
		t.Fatal(err)
	}
	withdrawalDigest, err := withdrawal.digestFromPacked(withdrawalPacked)
	if err != nil {
		t.Fatal(err)
	}
	if withdrawalValidator.Digest() != withdrawalDigest || withdrawalValidator.Threshold() != 2 {
		t.Fatalf("withdrawal prepared context = %x/%d, want %x/2", withdrawalValidator.Digest(), withdrawalValidator.Threshold(), withdrawalDigest)
	}

	rotation := &RotationFinalizer{
		client: client, programID: programID, configPDA: ConfigPDA(programID),
		chainID: 1, commitment: rpc.CommitmentConfirmed,
	}
	rotationSigners := append(append([]solana.PublicKey(nil), signers...), solana.NewWallet().PublicKey())
	rotationPacked, err := json.Marshal(rotPacked{
		NewSigners: []string{
			hex.EncodeToString(rotationSigners[0][:]),
			hex.EncodeToString(rotationSigners[1][:]),
			hex.EncodeToString(rotationSigners[2][:]),
		},
		NewThreshold: 2, RotationNonce: 7,
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
	if rotationValidator.Digest() != rotationDigest || rotationValidator.Threshold() != 2 {
		t.Fatalf("rotation prepared context = %x/%d, want %x/2", rotationValidator.Digest(), rotationValidator.Threshold(), rotationDigest)
	}

	config.setConfig(solcustody.Config{Signers: signers, Threshold: 2, RotationNonce: 8, ChainId: 1})
	if _, err := withdrawal.PrepareSignatureValidator(context.Background(), withdrawalPacked); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("old generation validator error = %v, want stale nonce", err)
	}
	bounds := core.WithdrawalTimeBounds{FinalizedAt: 123, ValidUntil: 3723}
	config.mu.Lock()
	config.commitments = nil
	config.mu.Unlock()
	repacked, err := withdrawal.Pack(context.Background(), &core.WithdrawalOp{
		Recipient: solana.NewWallet().PublicKey().String(),
		AssetURI:  "yellow://ynet/asset/0x0000000000000000000000000000000000001234/sol/0/0",
		Amount:    decimal.NewFromInt(1),
	}, [32]byte{1}, bounds)
	if err != nil {
		t.Fatal(err)
	}
	var retryPayload solPacked
	if err := json.Unmarshal(repacked, &retryPayload); err != nil {
		t.Fatal(err)
	}
	if retryPayload.RotationNonce != 8 {
		t.Fatalf("repacked rotation nonce = %d, want 8", retryPayload.RotationNonce)
	}
	config.mu.RLock()
	commitments := append([]string(nil), config.commitments...)
	config.mu.RUnlock()
	if len(commitments) != 1 || commitments[0] != string(rpc.CommitmentFinalized) {
		t.Fatalf("Pack config commitments = %v, want [finalized]", commitments)
	}
	if err := withdrawal.Validate(context.Background(), repacked, &core.WithdrawalOp{
		Recipient: retryPayload.To,
		AssetURI:  "yellow://ynet/asset/0x0000000000000000000000000000000000001234/sol/0/0",
		Amount:    decimal.NewFromInt(1),
	}, [32]byte{1}, bounds); err != nil {
		t.Fatalf("Validate repacked: %v", err)
	}
	config.mu.RLock()
	commitments = append([]string(nil), config.commitments...)
	config.mu.RUnlock()
	if len(commitments) != 1 {
		t.Fatalf("Validate performed a config read: commitments = %v", commitments)
	}
	if _, err := withdrawal.PrepareSignatureValidator(context.Background(), repacked); err != nil {
		t.Fatalf("repacked generation rejected: %v", err)
	}

	config.setConfig(solcustody.Config{Signers: signers, Threshold: 2, RotationNonce: 9, ChainId: 1})
	if _, err := withdrawal.PrepareSignatureValidator(context.Background(), repacked); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed pre-submit context error = %v, want stale nonce", err)
	}
}

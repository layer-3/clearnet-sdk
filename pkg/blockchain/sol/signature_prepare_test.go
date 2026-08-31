package sol

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	solcustody "github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

type configRPC struct {
	t         *testing.T
	programID solana.PublicKey
	data      string
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

func (s *configRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.t.Errorf("decode RPC request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Method != "getAccountInfo" {
		s.t.Errorf("unexpected RPC method %q", request.Method)
	}
	result := map[string]any{
		"context": map[string]any{"slot": 1},
		"value": map[string]any{
			"data":       []any{s.data, "base64"},
			"executable": false,
			"lamports":   1,
			"owner":      s.programID.String(),
			"rentEpoch":  0,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
}

func TestSolanaPrepareSignatureValidatorsUseExactDigest(t *testing.T) {
	programID := solana.NewWallet().PublicKey()
	signers := []solana.PublicKey{solana.NewWallet().PublicKey(), solana.NewWallet().PublicKey()}
	server := httptest.NewServer(newConfigRPC(t, programID, solcustody.Config{
		Signers: signers, Threshold: 2, SignerNonce: 7, ChainId: 1,
	}))
	defer server.Close()
	client := rpc.New(server.URL)

	withdrawal := &WithdrawalFinalizer{
		client: client, programID: programID, vaultPDA: VaultPDA(programID),
		chainID: 1, commitment: rpc.CommitmentConfirmed,
	}
	withdrawalPacked, err := json.Marshal(solPacked{
		To: solana.NewWallet().PublicKey().String(), Mint: solana.PublicKey{}.String(),
		Amount: 1, WithdrawalID: strings.Repeat("11", 32), Deadline: 123,
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
	rotationPacked, err := json.Marshal(rotPacked{
		NewSigners:   []string{hex.EncodeToString(signers[0][:]), hex.EncodeToString(signers[1][:])},
		NewThreshold: 2, SignerNonce: 7,
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
}

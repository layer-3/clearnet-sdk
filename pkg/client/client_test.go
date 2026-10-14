package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func sdkTestServer(t *testing.T, post func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/gateway/info":
			fmt.Fprintf(w, `{"networkId":"YDEV","chainId":1497646422,"virtualRouter":%q,"assets":[{"symbol":"YELLOW","decimals":18,"chainId":31337},{"symbol":"ETH","decimals":6,"chainId":31337}],"capabilities":["transfer-operation-status-v1","recent-transfers-v1"]}`, eip712.RouterHex)
		case strings.HasSuffix(r.URL.Path, "/balances"):
			fmt.Fprint(w, `{"address":"0x00000000000000000000000000000000000000aa","nonce":7,"balances":[{"asset":"YELLOW","symbol":"YELLOW","clearnetBalance":"1000000000000000000"}]}`)
		case r.URL.Path == "/v1/transfers" && r.Method == http.MethodPost:
			post(w, r)
		case strings.HasSuffix(r.URL.Path, "/transfers"):
			fmt.Fprint(w, `{"items":[],"retentionDays":30}`)
		case strings.HasPrefix(r.URL.Path, "/v1/operations/"):
			fmt.Fprintf(w, `{"operationId":%q,"kind":"transfer","status":"unknown"}`, strings.TrimPrefix(r.URL.Path, "/v1/operations/"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestUserClientPrepareSignSubmitPreservesUnknownOutcome(t *testing.T) {
	posts := 0
	var posted struct {
		To        string                 `json:"to"`
		MaxFee    string                 `json:"maxFee"`
		Nonce     uint64                 `json:"nonce"`
		Assets    []client.TransferAsset `json:"assets"`
		Signature string                 `json:"signature"`
	}
	server := sdkTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		posts++
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Errorf("decode transfer body: %v", err)
		}
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, `{"error":{"code":-32006,"message":"outcome unknown"}}`)
	})
	defer server.Close()
	c, err := client.New(server.URL, "YDEV")
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := sign.NewKeySignerFromECDSA(key)
	defer signer.Close()
	ctx := context.Background()
	info, err := c.Info(ctx)
	if err != nil || info.Assets[0].Symbol != "ETH" {
		t.Fatalf("validated sorted gateway info = %+v, %v", info, err)
	}
	to := "0x00000000000000000000000000000000000000bb"
	prepared, err := c.PrepareTransfer(ctx, signer, to, []client.TransferAmount{{Asset: "YELLOW", Amount: "1.25"}}, "0.01")
	if err != nil {
		t.Fatal(err)
	}
	if posts != 0 || prepared.Nonce != 7 || prepared.Assets[0].Amount != "1250000000000000000" || prepared.MaxFee != "10000000000000000" {
		t.Fatalf("unsigned preparation = %+v, posts=%d", prepared, posts)
	}
	signed, err := c.SignTransfer(ctx, signer, prepared)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Assets[0].Amount = "999"
	if signed.Intent.Assets[0].Amount != "1250000000000000000" {
		t.Fatal("prepared transfer mutation changed signed fields")
	}
	_, submitErr := c.SubmitTransfer(ctx, signed)
	if submitErr == nil {
		t.Fatal("expected an unknown-outcome error")
	}
	var subErr *client.SubmissionError
	if !errors.As(submitErr, &subErr) || !subErr.Submitted || !subErr.OutcomeUnknown || subErr.OperationID != signed.OperationID {
		t.Fatalf("submission error = %v", submitErr)
	}
	if posts != 1 || posted.MaxFee != "10000000000000000" || posted.Nonce != 7 || posted.To != to || posted.Assets[0].Amount != "1250000000000000000" {
		t.Fatalf("unsafe or changed transfer POST: %+v, posts=%d, error=%v", posted, posts, submitErr)
	}
	status, err := c.Operation(ctx, signed.OperationID)
	if err != nil || status.Status != "unknown" {
		t.Fatalf("status = %+v, %v", status, err)
	}
	page, err := c.Transfers(ctx, signed.Intent.From, client.TransferPageOptions{Limit: 25})
	if err != nil || page.RetentionDays != 30 || page.Items == nil {
		t.Fatalf("transfers = %+v, %v", page, err)
	}
}

func TestSignedTransferMutationIsRejectedBeforePOST(t *testing.T) {
	posts := 0
	server := sdkTestServer(t, func(w http.ResponseWriter, r *http.Request) { posts++ })
	defer server.Close()
	c, err := client.New(server.URL, "YDEV")
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := sign.NewKeySignerFromECDSA(key)
	defer signer.Close()
	intent, err := c.PrepareTransfer(context.Background(), signer, "0x00000000000000000000000000000000000000bb",
		[]client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "0.01")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := c.SignTransfer(context.Background(), signer, intent)
	if err != nil {
		t.Fatal(err)
	}
	signed.Intent.MaxFee = "0"
	if _, err := c.SubmitTransfer(context.Background(), signed); err == nil {
		t.Fatal("mutated signed body accepted")
	} else {
		var subErr *client.SubmissionError
		if !errors.As(err, &subErr) || subErr.Submitted || subErr.OutcomeUnknown {
			t.Fatalf("preflight mutation should be a definite local error: %v", err)
		}
	}
	if posts != 0 {
		t.Fatalf("mutated request reached the network %d times", posts)
	}
}

func TestSubmitTransferDistinguishesPendingFromSealed(t *testing.T) {
	posts := 0
	var operationID string
	server := sdkTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		posts++
		if posts == 1 {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"operationId":%q,"kind":"transfer","status":"pending"}`, operationID)
			return
		}
		fmt.Fprintf(w, `{"operationId":%q,"blockHash":"0x%s","entryIndex":0,"sealedAt":42,"stateRoot":"0x%s"}`,
			operationID, strings.Repeat("ab", 32), strings.Repeat("cd", 32))
	})
	defer server.Close()
	c, err := client.New(server.URL, "YDEV")
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := sign.NewKeySignerFromECDSA(key)
	defer signer.Close()
	intent, err := c.PrepareTransfer(context.Background(), signer, "0x00000000000000000000000000000000000000bb",
		[]client.TransferAmount{{Asset: "YELLOW", Amount: "1"}}, "0.01")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := c.SignTransfer(context.Background(), signer, intent)
	if err != nil {
		t.Fatal(err)
	}
	operationID = signed.OperationID
	pending, err := c.SubmitTransfer(context.Background(), signed)
	if err != nil || pending.Status != "pending" || posts != 1 {
		t.Fatalf("pending = %+v, posts=%d, err=%v", pending, posts, err)
	}
	sealed, err := c.SubmitTransfer(context.Background(), signed) // explicit replay of same authorization only
	if err != nil || sealed.Status != "sealed" || sealed.Receipt == nil || posts != 2 {
		t.Fatalf("sealed = %+v, posts=%d, err=%v", sealed, posts, err)
	}
}

func TestAmountInBaseUnitsRejectsRoundingAndUnsafeIntegers(t *testing.T) {
	for _, value := range []string{"", "-1", "+1", "1.123", "1e3", "0,1", "1.", strings.Repeat("9", 79)} {
		if _, err := client.AmountInBaseUnits(value, 2); err == nil {
			t.Errorf("accepted invalid amount %q", value)
		}
	}
	amount, err := client.AmountInBaseUnits("1.05", 2)
	if err != nil || amount.String() != "105" {
		t.Fatalf("1.05 with 2 decimals = %v, %v", amount, err)
	}
}

func TestNetworkAndTransportFailClosed(t *testing.T) {
	if _, err := client.New("http://example.com", "YDEV"); err == nil {
		t.Fatal("remote plaintext HTTP must require explicit opt-in")
	}
	server := sdkTestServer(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("unexpected POST") })
	defer server.Close()
	c, err := client.New(server.URL, "YTES")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Info(context.Background()); err == nil {
		t.Fatal("gateway with the wrong network must be rejected before signing")
	}
}

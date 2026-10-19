package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func TestNetworkFingerprintMatchesTypeScript(t *testing.T) {
	info := client.NetworkInfo{NetworkID: "YDEV", ChainID: 1497646422, VirtualRouter: eip712.RouterHex, Assets: []client.AssetInfo{
		{Symbol: "YELLOW", ChainID: 31337, Address: "0x236eB848C95b231299B4AA9f56c73D6893462720", Decimals: 18},
		{Symbol: "ETH", ChainID: 31337, Decimals: 18, Native: true},
	}}
	fingerprint, err := client.NetworkFingerprint(info)
	if err != nil || fingerprint != "5401edc080f28328873bad1792572d10b8cb482be574fcc8dace40136509501e" {
		t.Fatalf("fingerprint = %s, %v", fingerprint, err)
	}
	if info.Assets[0].Address != "0x236eB848C95b231299B4AA9f56c73D6893462720" {
		t.Fatal("fingerprinting mutated gateway metadata")
	}
}

func TestNetworkFingerprintEscapingGolden(t *testing.T) {
	var info client.NetworkInfo
	if err := json.Unmarshal([]byte(`{"networkId":"YDEV","chainId":1497646422,"virtualRouter":"0x00000000000000000000000000000000434C5200","assets":[{"symbol":"YELLOW","address":"<>&\u2028\u2029","decimals":18}]}`), &info); err != nil {
		t.Fatal(err)
	}
	got, err := client.NetworkFingerprint(info)
	if err != nil || got != "7ce75d8e8477b20bf1bff4bbb4d813153fc21aadf9c2181ce5f9205f179ce0cf" {
		t.Fatalf("escaped fingerprint = %s, %v", got, err)
	}
}

func TestPreparedTransferRejectsCatalogueDrift(t *testing.T) {
	changes := map[string]func(*client.NetworkInfo){
		"decimals":      func(i *client.NetworkInfo) { i.Assets[0].Decimals = 6 },
		"address":       func(i *client.NetworkInfo) { i.Assets[0].Address = "0x00000000000000000000000000000000000000cc" },
		"chain":         func(i *client.NetworkInfo) { i.Assets[0].ChainID = 1 },
		"native":        func(i *client.NetworkInfo) { i.Assets[0].Native = true },
		"fee precision": func(i *client.NetworkInfo) { i.Assets[1].Decimals = 6 },
		"removed asset": func(i *client.NetworkInfo) { i.Assets = i.Assets[1:] },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			info := client.NetworkInfo{NetworkID: "YDEV", ChainID: 1497646422, VirtualRouter: eip712.RouterHex,
				Capabilities: []string{client.CapabilityTransferStatus}, Assets: []client.AssetInfo{
					{Symbol: "USDC", ChainID: 31337, Decimals: 18}, {Symbol: "YELLOW", ChainID: 31337, Decimals: 18},
				}}
			var posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/v1/gateway/info":
					mu.Lock()
					defer mu.Unlock()
					_ = json.NewEncoder(w).Encode(info)
				case strings.HasSuffix(r.URL.Path, "/balances"):
					_ = json.NewEncoder(w).Encode(client.Balance{Nonce: 7})
				case r.Method == http.MethodPost:
					posts.Add(1)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
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
			intent, err := c.PrepareTransfer(ctx, signer, "0x00000000000000000000000000000000000000bb", []client.TransferAmount{{Asset: "USDC", Amount: "1"}}, "0.01")
			if err != nil {
				t.Fatal(err)
			}
			signed, err := c.SignTransfer(ctx, signer, intent)
			if err != nil {
				t.Fatal(err)
			}
			// A retained authorization must keep its guard across restart.
			raw, _ := json.Marshal(signed)
			var restored client.SignedTransfer
			if err := json.Unmarshal(raw, &restored); err != nil {
				t.Fatal(err)
			}
			legacy := *intent
			legacy.CatalogueFingerprint = ""
			if _, err := c.SignTransfer(ctx, signer, &legacy); err == nil {
				t.Fatal("missing catalogue accepted")
			}
			legacySigned := restored
			legacySigned.Intent.CatalogueFingerprint = ""
			if _, err := c.SubmitTransfer(ctx, &legacySigned); err == nil {
				t.Fatal("missing signed catalogue accepted")
			}
			mu.Lock()
			info.Assets[0], info.Assets[1] = info.Assets[1], info.Assets[0]
			info.VirtualRouter = strings.ToUpper(info.VirtualRouter)
			mu.Unlock()
			if _, err := c.SignTransfer(ctx, signer, intent); err != nil {
				t.Fatalf("equivalent catalogue rejected: %v", err)
			}
			mu.Lock()
			info.Assets[0], info.Assets[1] = info.Assets[1], info.Assets[0]
			change(&info)
			mu.Unlock()
			if _, err := c.SignTransfer(ctx, signer, intent); err == nil {
				t.Fatal("catalogue drift accepted for signing")
			}
			_, err = c.SubmitTransfer(ctx, &restored)
			var sub *client.SubmissionError
			if !errors.As(err, &sub) || sub.Submitted || sub.OutcomeUnknown || sub.OperationID != signed.OperationID {
				t.Fatalf("expected local catalogue refusal, got %v", err)
			}
			if posts.Load() != 0 {
				t.Fatal("catalogue drift reached POST")
			}
		})
	}
}

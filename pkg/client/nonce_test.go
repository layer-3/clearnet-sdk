package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

func TestSealedTransferAtNonce(t *testing.T) {
	const sender = "0x00000000000000000000000000000000000000aa"
	entry := client.TransferActivity{OperationID: "0x" + strings.Repeat("ab", 32), From: core.UserURI(sender), Nonce: 7, Direction: "out", Delivery: "sender_sealed", BlockHash: "0x" + strings.Repeat("cd", 32), EntryHash: "0x" + strings.Repeat("ef", 32), SealedAt: 42}
	for _, tt := range []struct {
		name                 string
		change               func(*client.TransferActivity)
		wantFound, wantError bool
	}{
		{"outgoing", func(*client.TransferActivity) {}, true, false},
		{"self", func(e *client.TransferActivity) { e.Direction = "self" }, true, false},
		{"other sender", func(e *client.TransferActivity) { e.From = "0x00000000000000000000000000000000000000bb" }, false, false},
		{"other nonce", func(e *client.TransferActivity) { e.Nonce = 8 }, false, false},
		{"incoming", func(e *client.TransferActivity) { e.Direction, e.Delivery = "in", "not_verified" }, false, true},
		{"missing ID", func(e *client.TransferActivity) { e.OperationID = "" }, false, true},
		{"missing block hash", func(e *client.TransferActivity) { e.BlockHash = "block" }, false, true},
		{"missing entry hash", func(e *client.TransferActivity) { e.EntryHash = "" }, false, true},
		{"missing seal time", func(e *client.TransferActivity) { e.SealedAt = 0 }, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			candidate := entry
			tt.change(&candidate)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Error("nonce lookup attempted a write")
				}
				if r.URL.Path == "/v1/gateway/info" {
					fmt.Fprintf(w, `{"networkId":"YDEV","chainId":1497646422,"virtualRouter":%q,"assets":[],"capabilities":["recent-transfers-v1"]}`, eip712.RouterHex)
					return
				}
				calls++
				if r.URL.Query().Get("limit") != "100" {
					t.Error("unbounded page")
				}
				if calls == 1 {
					next := "next-page"
					_ = json.NewEncoder(w).Encode(client.TransferPage{NextCursor: &next})
					return
				}
				if r.URL.Query().Get("cursor") != "next-page" {
					t.Error("wrong cursor")
				}
				_ = json.NewEncoder(w).Encode(client.TransferPage{Items: []client.TransferActivity{candidate}})
			}))
			defer server.Close()
			c, _ := client.New(server.URL, "YDEV")
			got, err := c.SealedTransferAtNonce(context.Background(), sender, 7)
			if (got != nil) != tt.wantFound || (err != nil) != tt.wantError || calls != 2 {
				t.Fatalf("got %+v / %v after %d pages", got, err, calls)
			}
		})
	}
}

func TestNonceLookupBoundsPagination(t *testing.T) {
	for _, repeat := range []bool{true, false} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/gateway/info" {
				fmt.Fprintf(w, `{"networkId":"YDEV","chainId":1497646422,"virtualRouter":%q,"assets":[],"capabilities":["recent-transfers-v1"]}`, eip712.RouterHex)
				return
			}
			calls++
			next := fmt.Sprint(calls)
			if repeat {
				next = "same"
			}
			_ = json.NewEncoder(w).Encode(client.TransferPage{NextCursor: &next})
		}))
		c, _ := client.New(server.URL, "YDEV")
		_, err := c.SealedTransferAtNonce(context.Background(), "0x00000000000000000000000000000000000000aa", 7)
		server.Close()
		if err == nil || (repeat && calls != 2) || (!repeat && calls != 10) {
			t.Fatal("unbounded pagination", err, calls)
		}
	}
}

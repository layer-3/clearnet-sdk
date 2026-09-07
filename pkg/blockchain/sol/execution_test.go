package sol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
)

type executionRPC struct {
	rpc.JSONRPCClient
	err error
}

func (c executionRPC) CallForInto(context.Context, any, string, []any) error { return c.err }

func TestVerifyExecutionPreservesErrors(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, errors.New("transport unavailable"), fmt.Errorf("wrapped: %w", rpc.ErrNotFound)} {
		f := &WithdrawalFinalizer{client: rpc.NewWithCustomRPCClient(executionRPC{err: cause})}
		_, executed, err := f.VerifyExecution(context.Background(), [32]byte{})
		if errors.Is(cause, rpc.ErrNotFound) {
			if err != nil || executed {
				t.Fatalf("absence: %v %v", executed, err)
			}
			continue
		}
		if executed || !errors.Is(err, cause) {
			t.Fatalf("cause %v: executed=%v err=%v", cause, executed, err)
		}
	}
}

func TestVerifyExecutionRPC(t *testing.T) {
	for _, tc := range []struct {
		name, result      string
		status            int
		rpcError          bool
		executed, wantErr bool
	}{
		{name: "present", result: `{"context":{"slot":42},"value":{"lamports":1,"owner":"11111111111111111111111111111111","data":["","base64"],"executable":false,"rentEpoch":0}}`, executed: true},
		{name: "absent", result: `{"context":{"slot":42},"value":null}`},
		{name: "null", result: `null`, wantErr: true},
		{name: "rate limit", status: 429, wantErr: true},
		{name: "rpc error", rpcError: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				var opts struct {
					Commitment     string
					MinContextSlot uint64
				}
				if len(req.Params) != 2 {
					t.Errorf("params: %s", req.Params)
					return
				}
				if err := json.Unmarshal(req.Params[1], &opts); err != nil {
					t.Error(err)
				}
				if req.Method != "getAccountInfo" || opts.Commitment != "finalized" || opts.MinContextSlot != 42 {
					t.Errorf("request: %+v %+v", req, opts)
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
					return
				}
				if tc.rpcError {
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32016,"message":"Minimum context slot has not been reached"}}`, req.ID)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, tc.result)
			}))
			defer srv.Close()
			f := &WithdrawalFinalizer{client: rpc.New(srv.URL), commitment: rpc.CommitmentFinalized}
			_, executed, err := f.VerifyExecutionAtSlot(context.Background(), [32]byte{}, 42)
			if executed != tc.executed || (err != nil) != tc.wantErr {
				t.Fatalf("executed=%v err=%v", executed, err)
			}
		})
	}
}

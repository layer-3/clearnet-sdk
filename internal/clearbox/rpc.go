package clearbox

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

type rpcRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if decodeBody(r, &raw) != nil {
		writeJSON(w, 200, rpcFailure(nil, -32700, "parse error"))
		return
	}
	if len(raw) > 0 && raw[0] == '[' {
		var batch []json.RawMessage
		if json.Unmarshal(raw, &batch) != nil || len(batch) == 0 || len(batch) > 100 {
			writeJSON(w, 200, rpcFailure(nil, -32600, "batch must contain 1 to 100 requests"))
			return
		}
		results := []any{}
		for _, call := range batch {
			if result := s.rpcOne(call); result != nil {
				results = append(results, result)
			}
		}
		if len(results) == 0 {
			w.WriteHeader(204)
		} else {
			writeJSON(w, 200, results)
		}
		return
	}
	if result := s.rpcOne(raw); result != nil {
		writeJSON(w, 200, result)
	} else {
		w.WriteHeader(204)
	}
}

func rpcFailure(id json.RawMessage, code int, message string) any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": rpcError{code, message}}
}

func (s *Server) rpcOne(raw json.RawMessage) any {
	var req rpcRequest
	if json.Unmarshal(raw, &req) != nil || req.JSONRPC != "2.0" || req.Method == "" {
		return rpcFailure(nil, -32600, "invalid request")
	}
	if len(req.ID) != 0 && req.ID[0] != '"' && req.ID[0] != '-' && (req.ID[0] < '0' || req.ID[0] > '9') && string(req.ID) != "null" {
		return rpcFailure(nil, -32600, "invalid request ID")
	}
	result, err := s.rpc(req.Method, req.Params)
	if len(req.ID) == 0 {
		return nil // JSON-RPC notifications have no response.
	}
	if err != nil {
		return rpcFailure(req.ID, err.Code, err.Message)
	}
	return map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}
}

func quantity(n uint64) string { return fmt.Sprintf("0x%x", n) }

func (s *Server) rpc(method string, params []json.RawMessage) (any, *rpcError) {
	bad := func(message string) (any, *rpcError) { return nil, &rpcError{-32602, message} }
	switch method {
	case "eth_chainId":
		return quantity(eip712.NetworkChainID(NetworkID).Uint64()), nil
	case "net_version":
		return eip712.NetworkChainID(NetworkID).String(), nil
	case "web3_clientVersion":
		return "clearbox/1", nil
	case "net_listening":
		return true, nil
	case "net_peerCount", "eth_gasPrice", "eth_maxPriorityFeePerGas":
		return "0x0", nil
	case "eth_syncing":
		return false, nil
	case "eth_accounts":
		return []string{}, nil // The server never signs or manages client keys.
	case "eth_blockNumber":
		if len(params) != 1 {
			return bad("expected [account]")
		}
		var raw string
		if json.Unmarshal(params[0], &raw) != nil {
			return bad("invalid account")
		}
		address, err := userAddress(raw)
		if err != nil {
			return bad(err.Error())
		}
		return quantity(uint64(len(s.accountBlocks(address)))), nil
	case "eth_getBalance", "eth_getTransactionCount", "eth_getCode":
		if len(params) != 2 {
			return bad("expected an address and block tag")
		}
		var raw, tag string
		if json.Unmarshal(params[0], &raw) != nil || json.Unmarshal(params[1], &tag) != nil {
			return bad("expected an address and block tag")
		}
		address, err := userAddress(raw)
		if err != nil || (tag != "latest" && tag != "pending") {
			return bad("expected a nonzero address and latest/pending block tag; historical state is not simulated")
		}
		a := s.account(address)
		switch method {
		case "eth_getBalance":
			return "0x" + a.Balances["YELLOW"].Text(16), nil
		case "eth_getTransactionCount":
			return quantity(a.Nonce), nil
		default:
			return "0x", nil
		}
	case "eth_getBlockByNumber":
		if len(params) < 2 || len(params) > 3 {
			return bad("expected [account, block tag, optional full-transactions boolean]")
		}
		var raw, tag string
		var full bool
		if json.Unmarshal(params[0], &raw) != nil || json.Unmarshal(params[1], &tag) != nil || (len(params) == 3 && json.Unmarshal(params[2], &full) != nil) {
			return bad("invalid account, block tag or full-transactions boolean")
		}
		address, err := userAddress(raw)
		if err != nil {
			return bad(err.Error())
		}
		blocks := s.accountBlocks(address)
		number := uint64(len(blocks))
		switch tag {
		case "earliest":
			number = 1
		case "latest", "pending":
		default:
			if !strings.HasPrefix(tag, "0x") || len(tag) < 3 {
				return bad("invalid block tag")
			}
			number, err = strconv.ParseUint(tag[2:], 16, 64)
			if err != nil {
				return bad("invalid block number")
			}
		}
		if number == 0 || number > uint64(len(blocks)) {
			return nil, nil
		}
		return blockRPC(blocks[number-1], number), nil
	case "eth_getBlockByHash":
		if len(params) < 1 || len(params) > 2 {
			return bad("expected [block hash, optional full-transactions boolean]")
		}
		var raw string
		var full bool
		if json.Unmarshal(params[0], &raw) != nil || (len(params) == 2 && json.Unmarshal(params[1], &full) != nil) {
			return bad("invalid block hash or full-transactions boolean")
		}
		hash, err := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
		if err != nil || len(hash) != 32 {
			return bad("invalid block hash")
		}
		for _, block := range s.blocks {
			if common.Hash(block.Hash()) == common.BytesToHash(hash) {
				for i, candidate := range s.accountBlocks(block.Entries[0].Account) {
					if candidate == block {
						return blockRPC(block, uint64(i+1)), nil
					}
				}
			}
		}
		return nil, nil
	default:
		return nil, &rpcError{-32601, "method " + method + " is not supported by clearbox"}
	}
}

// Clearnet numbers blocks per account, never as a global Ethereum chain.
func (s *Server) accountBlocks(address string) []*core.Block {
	anchor := core.ComputeAccountID(address)
	blocks := []*core.Block{}
	for _, block := range s.blocks {
		if block.Anchor == anchor {
			blocks = append(blocks, block)
		}
	}
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].SealedAt < blocks[j].SealedAt })
	return blocks
}

func blockRPC(block *core.Block, number uint64) map[string]any {
	entries := make([]any, len(block.Entries))
	for i, entry := range block.Entries {
		entries[i] = map[string]any{
			"type": uint8(entry.Type), "account": entry.Account, "nonce": quantity(entry.Nonce),
			"payload": "0x" + hex.EncodeToString(entry.Payload), "index": quantity(uint64(i)),
		}
	}
	return map[string]any{
		"number": quantity(number), "hash": common.Hash(block.Hash()).Hex(),
		"parentHash": common.Hash(block.Anchor).Hex(), "timestamp": quantity(uint64(block.SealedAt)),
		"stateRoot": common.Hash(block.StateRoot).Hex(), "transactions": entries,
		"miner": common.BytesToAddress(block.Anchor[12:]).Hex(), "gasUsed": "0x0", "gasLimit": "0x0",
	}
}

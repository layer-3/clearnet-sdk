package btc

import (
	"context"
	"errors"
	"strings"
)

// RPC is the bitcoind RPC surface the adapters depend on. It is supplied by the
// caller (mirroring how the EVM adapters take a caller-supplied
// *ethclient.Client), so the SDK carries no JSON-RPC client of its own. The
// block/raw-tx methods back the withdrawal-execution scan.
type RPC interface {
	ListUnspent(ctx context.Context, minConf int, addrs []string) ([]Unspent, error)
	GetTxOut(ctx context.Context, txid string, vout uint32, includeMempool bool) (*TxOut, error)
	SendRawTransaction(ctx context.Context, hexTx string) (string, error)
	EstimateSmartFeeSatPerVByte(ctx context.Context, confTarget int, fallbackRate int64) (int64, error)

	// For VerifyExecution: scan recent blocks for the OP_RETURN <withdrawalID>.
	GetBlockCount(ctx context.Context) (int64, error)
	GetBlockHash(ctx context.Context, height int64) (string, error)
	GetBlockTxids(ctx context.Context, blockHash string) ([]string, error)
	GetRawTransaction(ctx context.Context, txid string) (*RawTx, error)
}

// Unspent is a vault UTXO as reported by ListUnspent.
type Unspent struct {
	TxID          string
	Vout          uint32
	AmountSats    int64
	Confirmations int64
	ScriptPubKey  string
}

// TxOut is a single output as reported by GetTxOut.
type TxOut struct {
	AmountSats    int64
	ScriptPubKey  string
	Confirmations int64
}

// RawTx is a decoded transaction as reported by GetRawTransaction.
type RawTx struct {
	TxID          string
	Confirmations int64
	Vouts         []RawVout
}

// RawVout is one output of a RawTx, with its scriptPubKey hex.
type RawVout struct {
	ValueSats       int64
	ScriptPubKeyHex string
}

// isAlreadyKnown reports whether a SendRawTransaction error means the tx (or a
// prior attempt spending the same inputs) is already in the chain/mempool — the
// UTXO-model analogue of EVM's executed[withdrawalID] guard. It prefers the
// typed bitcoind error code (RPC_VERIFY_ALREADY_IN_CHAIN = -27,
// RPC_VERIFY_ERROR = -25 for spent/missing inputs) when the caller supplies
// *Client, and falls back to error-text matching for other RPC implementations.
func isAlreadyKnown(err error) bool {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		switch rpcErr.Code {
		case -27, -25:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already in block chain") ||
		strings.Contains(msg, "txn-already-known") ||
		strings.Contains(msg, "missingorspent") ||
		strings.Contains(msg, "missing inputs")
}

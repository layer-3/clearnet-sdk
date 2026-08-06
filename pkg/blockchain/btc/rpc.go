package btc

import (
	"context"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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

// isAlreadyKnown reports whether a SendRawTransaction error unambiguously says
// the exact submitted transaction is already in the chain/mempool. A generic
// RPC_VERIFY_ERROR (-25), including a missing-inputs error, is deliberately not
// sufficient: the inputs may have been spent by a different transaction.
func isAlreadyKnown(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == -27 { // RPC_VERIFY_ALREADY_IN_CHAIN
			return true
		}
		if rpcErr.Code == -25 { // RPC_VERIFY_ERROR always requires an exact lookup
			return false
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already in block chain") ||
		strings.Contains(msg, "txn-already-known")
}

// broadcastAlreadyAccepted reports whether a failed broadcast can safely be
// treated as an idempotent success for txID. Ambiguous verification failures
// are accepted only after an independent lookup returns that exact transaction.
func broadcastAlreadyAccepted(ctx context.Context, rpc RPC, txID string, broadcastErr error) bool {
	if isAlreadyKnown(broadcastErr) {
		return true
	}
	if !requiresBroadcastLookup(broadcastErr) {
		return false
	}

	raw, err := rpc.GetRawTransaction(ctx, txID)
	if err != nil || raw == nil {
		return false
	}
	localHash, err := chainhash.NewHashFromStr(txID)
	if err != nil {
		return false
	}
	foundHash, err := chainhash.NewHashFromStr(raw.TxID)
	return err == nil && foundHash.IsEqual(localHash)
}

func requiresBroadcastLookup(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) && rpcErr.Code == -25 { // RPC_VERIFY_ERROR
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "missingorspent") ||
		strings.Contains(msg, "missing inputs")
}

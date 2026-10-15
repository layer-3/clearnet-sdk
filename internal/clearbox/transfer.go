package clearbox

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

type transferRequest struct {
	To        string                 `json:"to"`
	Assets    []client.TransferAsset `json:"assets"`
	MaxFee    string                 `json:"maxFee"`
	Nonce     *uint64                `json:"nonce"`
	Signature string                 `json:"signature"`
}

type poolSummary struct {
	PoolID    string `json:"poolId"`
	TokenA    string `json:"tokenA"`
	TokenB    string `json:"tokenB"`
	ReserveA  string `json:"reserveA"`
	ReserveB  string `json:"reserveB"`
	SpotPrice string `json:"spotPrice"`
}

func pools() any {
	return struct {
		Items []poolSummary `json:"items"`
		Total int           `json:"total"`
	}{[]poolSummary{
		{"ETH/YELLOW", "ETH", "YELLOW", "100000000000000000000", "2485000000000000000000000", "24850"},
		{"USDC/YELLOW", "USDC", "YELLOW", "1000000000000", "10000000000000000000000000", "10"},
	}, 2}
}

// feeFor mirrors Clearnet's frozen per-leg fee rule. Prices are static local
// fixtures, represented as YELLOW base units per asset base unit.
func feeFor(assets []eip712.TransferAsset) *big.Int {
	fee := new(big.Int)
	for _, leg := range assets {
		price := big.NewInt(1)
		switch leg.Asset {
		case "ETH":
			price = big.NewInt(24850)
		case "USDC":
			price = integer("10000000000000")
		}
		part := new(big.Int).Mul(leg.Amount, price)
		part.Div(part, big.NewInt(1000)) // Fixed 10 bps, rounded down per leg.
		if part.Cmp(integer("100000000000000")) < 0 {
			part.SetString("100000000000000", 10)
		}
		fee.Add(fee, part)
	}
	return fee
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if decodeBody(r, &req) != nil || req.Nonce == nil || *req.Nonce == math.MaxUint64 {
		fail(w, 400, -32600, "invalid request body", "")
		return
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil {
		fail(w, 400, -32001, "invalid signature encoding", "signature")
		return
	}
	assets := make([]eip712.TransferAsset, len(req.Assets))
	for i, leg := range req.Assets {
		amount, err := uint256(leg.Amount)
		if err != nil || amount.Sign() <= 0 {
			fail(w, 400, -32602, fmt.Sprintf("assets[%d].amount must be a positive uint256 integer", i), "assets")
			return
		}
		assets[i] = eip712.TransferAsset{Asset: leg.Asset, Amount: amount}
	}
	assets, err = eip712.NormalizeTransferAssets(assets)
	if err != nil {
		fail(w, 400, -32602, err.Error(), "assets")
		return
	}
	maxFee, err := uint256(req.MaxFee)
	if err != nil {
		fail(w, 400, -32602, "invalid maxFee: "+err.Error(), "maxFee")
		return
	}
	to, err := userAddress(req.To)
	if err != nil {
		fail(w, 400, -32602, err.Error(), "to")
		return
	}
	chainID := eip712.NetworkChainID(NetworkID)
	recipient, err := eip712.TransferRecipientAddress(req.To)
	if err != nil {
		fail(w, 400, -32602, err.Error(), "to")
		return
	}
	signer, err := eip712.RecoverTransfer(chainID, recipient, assets, maxFee, *req.Nonce, sig)
	if err != nil {
		fail(w, 401, -32001, "signature recovery failed: "+err.Error(), "")
		return
	}
	from := signer.Hex()
	id, err := eip712.TransferOperationID(chainID, signer, req.To, assets, maxFee, *req.Nonce)
	if err != nil {
		fail(w, 400, -32602, err.Error(), "assets")
		return
	}
	if op := s.ops[id]; op != nil {
		s.writeOperation(w, op)
		return
	}
	ownerKey := fmt.Sprintf("%s:%d", from, *req.Nonce)
	if owner := s.nonceOwner[ownerKey]; owner != "" {
		fail(w, 409, -32009, "transfer operation conflicts with another signed intent", "")
		return
	}
	if len(s.ops) >= s.cfg.MaxOperations {
		fail(w, 429, -32008, "clearbox operation capacity reached; ask the operator to increase max-operations", "")
		return
	}
	sender := s.account(from)
	if *req.Nonce != sender.Nonce {
		fail(w, 409, -32002, fmt.Sprintf("nonce mismatch: expected %d, got %d", sender.Nonce, *req.Nonce), "nonce")
		return
	}
	activation := integer("10000000000000000000")
	if sender.Balances["YELLOW"].Cmp(activation) < 0 {
		fail(w, 422, -32003, "account not activated: hold at least 10 YELLOW to send", "")
		return
	}
	need := map[string]*big.Int{}
	for _, leg := range assets {
		if sender.Balances[leg.Asset] == nil {
			fail(w, 400, -32602, "unknown asset "+leg.Asset, "assets")
			return
		}
		need[leg.Asset] = new(big.Int).Set(leg.Amount)
	}
	fee := feeFor(assets)
	if fee.Cmp(maxFee) > 0 {
		fail(w, 422, -32003, fmt.Sprintf("fee %s exceeds maxFee %s", fee, maxFee), "maxFee")
		return
	}
	if need["YELLOW"] == nil {
		need["YELLOW"] = new(big.Int)
	}
	need["YELLOW"].Add(need["YELLOW"], fee)
	for asset, amount := range need {
		if sender.Balances[asset].Cmp(amount) < 0 {
			fail(w, 422, -32000, "insufficient "+asset+" balance", "assets")
			return
		}
	}
	if new(big.Int).Sub(sender.Balances["YELLOW"], need["YELLOW"]).Cmp(activation) < 0 {
		fail(w, 422, -32003, "post-operation balance below user minimum: leave at least 10 YELLOW", "assets")
		return
	}
	if s.cfg.PublicURL != "" && !s.traffic.writes.Allow() {
		if s.cfg.SealDelay > time.Second {
			w.Header().Set("Retry-After", fmt.Sprint(int64((s.cfg.SealDelay+time.Second-1)/time.Second)))
		}
		limited(w, "new transfer rate limit exceeded")
		return
	}
	// Admission and reservations are atomic under the server lock. Later nonces
	// cannot spend funds reserved by a transfer still waiting for its seal.
	for asset, amount := range need {
		sender.Balances[asset].Sub(sender.Balances[asset], amount)
	}
	for _, leg := range assets {
		sender.Clearing[leg.Asset].Add(sender.Clearing[leg.Asset], leg.Amount)
	}
	sender.Fees.Add(sender.Fees, fee)
	sender.Nonce++
	s.createAccount(to)
	s.nonceOwner[ownerKey] = id
	req.Assets = make([]client.TransferAsset, len(assets))
	for i, leg := range assets {
		req.Assets[i] = client.TransferAsset{Asset: leg.Asset, Amount: leg.Amount.String()}
	}
	now := s.now()
	op := &operation{
		Status: client.OperationStatus{OperationID: id, Kind: "transfer", Status: "pending", PendingUntil: now.Add(2 * time.Minute).Unix()},
		From:   from, To: to, Request: req, SealAt: now.Add(s.cfg.SealDelay), DeliverAt: now.Add(s.cfg.SealDelay + s.cfg.DeliveryDelay),
	}
	s.ops[id] = op
	s.order = append(s.order, id)
	s.dirty = true
	s.advance(now)
	s.writeOperation(w, op)
}

func (s *Server) writeOperation(w http.ResponseWriter, op *operation) {
	if op.Status.Status == "sealed" {
		writeJSON(w, 200, op.Status.Receipt)
	} else {
		writeJSON(w, 202, client.OperationStatus{OperationID: op.Status.OperationID, Kind: "transfer", Status: "pending"})
	}
}

// advance materializes due events on reads/writes, without timer goroutines that
// could outlive a test or process. Polling and balance reads drive the same clock.
func (s *Server) advance(now time.Time) {
	for _, id := range s.order {
		op := s.ops[id]
		if op.Status.Status == "pending" && !now.Before(op.SealAt) {
			s.seal(op)
		}
		if op.Status.Status == "sealed" && !op.Delivered && !now.Before(op.DeliverAt) {
			sender, recipient := s.account(op.From), s.account(op.To)
			for _, leg := range op.Request.Assets {
				amount := integer(leg.Amount)
				recipient.Balances[leg.Asset].Add(recipient.Balances[leg.Asset], amount)
				sender.Clearing[leg.Asset].Sub(sender.Clearing[leg.Asset], amount)
			}
			op.Delivered = true
			s.dirty = true
		}
	}
	cutoff := now.Add(-30 * 24 * time.Hour).Unix()
	for _, a := range s.accounts {
		end := sort.Search(len(a.History), func(i int) bool { return a.History[i].SealedAt < cutoff })
		if end < len(a.History) {
			s.dirty = true
		}
		a.History = a.History[:end]
	}
}

func (s *Server) seal(op *operation) {
	s.dirty = true
	sender := s.account(op.From)
	assets := make([]core.AssetTransfer, len(op.Request.Assets))
	for i, leg := range op.Request.Assets {
		assets[i] = core.AssetTransfer{Asset: core.AssetID(leg.Asset), Amount: decimal.NewFromBigInt(integer(leg.Amount), 0)}
	}
	entry := core.BlockEntry{Type: core.OpTransfer, Account: op.From, Nonce: *op.Request.Nonce, Payload: (&core.TransferOp{TxID: op.Status.OperationID, To: op.Request.To, Assets: assets}).Encode()}
	anchor := core.ComputeAccountID(op.From)
	block := &core.Block{
		Anchor: anchor, SealedAt: op.SealAt.Unix(), Entries: []core.BlockEntry{entry}, StateRoot: s.stateRoot(),
		Accounts: []core.AccountSnapshot{{AccountID: anchor, PrevBlockRef: sender.LastBlock, PostNonce: *op.Request.Nonce + 1}},
	}
	block.EntriesDigest = core.ComputeEntriesDigest(block.Entries)
	hash, entryHash := block.Hash(), core.ComputeEntryHash(entry)
	sender.LastBlock = hash
	s.blocks = append(s.blocks, block)
	receipt := &client.Receipt{OperationID: op.Status.OperationID, BlockHash: "0x" + hex.EncodeToString(hash[:]), EntryHash: "0x" + hex.EncodeToString(entryHash[:]), SealedAt: block.SealedAt, StateRoot: "0x" + hex.EncodeToString(block.StateRoot[:])}
	op.Status.Status, op.Status.Receipt, op.Status.PendingUntil = "sealed", receipt, 0
	item := client.TransferActivity{
		OperationID: op.Status.OperationID, BlockHash: receipt.BlockHash, EntryHash: receipt.EntryHash, EntryIndex: 0, SealedAt: receipt.SealedAt,
		From: op.From, To: op.Request.To, Nonce: *op.Request.Nonce, Assets: op.Request.Assets, Direction: "out", Delivery: "sender_sealed",
	}
	if op.From == op.To {
		item.Direction = "self"
	}
	s.addHistory(sender, item)
	if op.From != op.To {
		item.Direction, item.Delivery = "in", "not_verified"
		s.addHistory(s.account(op.To), item)
	}
}

func (s *Server) addHistory(a *account, item client.TransferActivity) {
	a.History = append(a.History, item)
	sort.Slice(a.History, func(i, j int) bool {
		x, y := a.History[i], a.History[j]
		if x.SealedAt != y.SealedAt {
			return x.SealedAt > y.SealedAt
		}
		return x.BlockHash > y.BlockHash
	})
}

// This commitment is intentionally a local snapshot hash, not a production
// sparse Merkle root or proof of validator finality. Block/entry hashes still
// use the SDK's genuine canonical CBOR encoders.
func (s *Server) stateRoot() [32]byte {
	balances := make([]client.Balance, 0, len(s.accounts))
	for _, address := range sortedAccounts(s.accounts) {
		balances = append(balances, s.balance(address))
	}
	b, _ := json.Marshal(balances)
	return crypto.Keccak256Hash(b)
}

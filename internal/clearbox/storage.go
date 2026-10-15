package clearbox

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	_ "github.com/mattn/go-sqlite3"
)

// One atomic SQLite checkpoint keeps the local simulator small: accounts,
// pending delivery, operations and block history always commit together. This
// is deliberately not the production Clearnet storage schema.
type checkpoint struct {
	Version        int                   `json:"version"`
	SeedAddress    string                `json:"seedAddress"`
	SeedPrivateKey string                `json:"seedPrivateKey"`
	Accounts       map[string]*account   `json:"accounts"`
	Operations     map[string]*operation `json:"operations"`
	Order          []string              `json:"order"`
	NonceOwner     map[string]string     `json:"nonceOwner"`
	Blocks         []*core.Block         `json:"blocks"`
}

func (s *Server) openStorage(path string) error {
	dsn := ":memory:"
	if path != "" && path != ":memory:" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
			return err
		}
		file, err := os.OpenFile(absolute, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		// The database contains a development private key. An existing file
		// could have been created with broader permissions than OpenFile's mode.
		if err := file.Chmod(0600); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		dsn = (&url.URL{Scheme: "file", Path: absolute}).String()
	}
	// Exclusive locking prevents two clearboxes from serving divergent in-memory
	// views of one file. WAL + FULL ensures acknowledged writes survive restart.
	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_synchronous=FULL&_locking_mode=EXCLUSIVE&_busy_timeout=1000")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	s.db = db
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS clearbox_state (id INTEGER PRIMARY KEY CHECK (id = 1), data BLOB NOT NULL)"); err != nil {
		db.Close()
		return fmt.Errorf("open clearbox database (another clearbox may own it): %w", err)
	}
	err = s.load()
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		db.Close()
		return fmt.Errorf("load clearbox database: %w", err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		db.Close()
		return fmt.Errorf("generate local seed account: %w", err)
	}
	s.SeedAddress = crypto.PubkeyToAddress(key.PublicKey).Hex()
	s.SeedPrivateKey = hex.EncodeToString(crypto.FromECDSA(key))
	s.createAccount(s.SeedAddress).Balances["YELLOW"] = integer("1000000000000000000000000") // 1,000,000 YELLOW, once per database.
	s.Created = true
	if err := s.save(); err != nil {
		db.Close()
		return err
	}
	return nil
}

func (s *Server) load() error {
	var raw []byte
	if err := s.db.QueryRow("SELECT data FROM clearbox_state WHERE id = 1").Scan(&raw); err != nil {
		return err
	}
	var state checkpoint
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	if state.Version != 1 || state.Accounts == nil || state.Operations == nil || state.NonceOwner == nil {
		return fmt.Errorf("unsupported or invalid clearbox checkpoint; database has not been reset")
	}
	key, err := crypto.HexToECDSA(state.SeedPrivateKey)
	if err != nil || crypto.PubkeyToAddress(key.PublicKey).Hex() != state.SeedAddress || state.Accounts[state.SeedAddress] == nil {
		return fmt.Errorf("invalid stored local seed account; database has not been reset")
	}
	if err := validateCheckpoint(state); err != nil {
		return fmt.Errorf("invalid clearbox checkpoint (database has not been reset): %w", err)
	}
	s.SeedAddress, s.SeedPrivateKey = state.SeedAddress, state.SeedPrivateKey
	s.accounts, s.ops, s.order, s.nonceOwner, s.blocks = state.Accounts, state.Operations, state.Order, state.NonceOwner, state.Blocks
	s.blockByHash = make(map[[32]byte]*core.Block, len(state.Blocks))
	for _, block := range state.Blocks {
		s.blockByHash[block.Hash()] = block
	}
	s.dirty = false
	return nil
}

// validateCheckpoint refuses corrupt or incompatible state at startup rather
// than panicking later during a balance read, block lookup or pending delivery.
// A future catalogue change must migrate stored balances or bump the format.
func validateCheckpoint(state checkpoint) error {
	for address, a := range state.Accounts {
		if a == nil || a.Balances == nil || a.Clearing == nil || a.Fees == nil || a.Fees.Sign() < 0 {
			return fmt.Errorf("account %s has missing balances, clearing or fees", address)
		}
		for _, asset := range catalog {
			balance, clearing := a.Balances[asset.Symbol], a.Clearing[asset.Symbol]
			if balance == nil || clearing == nil || balance.Sign() < 0 || clearing.Sign() < 0 {
				return fmt.Errorf("account %s has invalid %s balance or clearing (catalogue migration required)", address, asset.Symbol)
			}
		}
	}
	for id, op := range state.Operations {
		if op == nil || op.Request.Nonce == nil || op.Status.OperationID != id || op.Status.Kind != "transfer" ||
			(op.Status.Status != "pending" && op.Status.Status != "sealed") ||
			(op.Status.Status == "sealed" && op.Status.Receipt == nil) ||
			state.Accounts[op.From] == nil || state.Accounts[op.To] == nil || len(op.Request.Assets) == 0 {
			return fmt.Errorf("operation %s has invalid state or missing accounts", id)
		}
		for _, leg := range op.Request.Assets {
			amount, err := uint256(leg.Amount)
			if err != nil || amount.Sign() <= 0 || !opFromCatalog(leg.Asset) {
				return fmt.Errorf("operation %s has invalid asset %s amount", id, leg.Asset)
			}
		}
	}
	if len(state.Order) != len(state.Operations) {
		return fmt.Errorf("operation order does not match stored operations")
	}
	seen := make(map[string]bool, len(state.Order))
	for _, id := range state.Order {
		if seen[id] || state.Operations[id] == nil {
			return fmt.Errorf("operation order contains a duplicate or missing operation %s", id)
		}
		seen[id] = true
	}
	for owner, id := range state.NonceOwner {
		if state.Operations[id] == nil {
			return fmt.Errorf("nonce owner %s references missing operation %s", owner, id)
		}
	}
	for i, block := range state.Blocks {
		if block == nil || len(block.Entries) == 0 {
			return fmt.Errorf("block %d has no entries", i)
		}
	}
	return nil
}

func opFromCatalog(symbol string) bool {
	for _, asset := range catalog {
		if asset.Symbol == symbol {
			return true
		}
	}
	return false
}

func (s *Server) save() error {
	raw, err := json.Marshal(checkpoint{1, s.SeedAddress, s.SeedPrivateKey, s.accounts, s.ops, s.order, s.nonceOwner, s.blocks})
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO clearbox_state (id, data) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", raw)
	if err == nil {
		s.dirty = false
	}
	return err
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.db.Close()
}

// Package clearbox is a Yellow-specific local development simulator. It reuses
// the public SDK wire types, EIP-712 verification and block codecs; it is not a
// validator, custody provider, or production gateway.
package clearbox

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
)

const NetworkID = "YDEV"

var catalog = []client.AssetInfo{
	{Symbol: "ETH", ChainID: 31337, Decimals: 18, Native: true},
	{Symbol: "USDC", ChainID: 31337, Address: "0x0000000000000000000000000000000000000002", Decimals: 6},
	{Symbol: "YELLOW", ChainID: 31337, Address: "0x0000000000000000000000000000000000000003", Decimals: 18},
}

// Config selects SQLite storage and local timing. An empty path uses isolated
// in-memory SQLite for tests; zero delays settle synchronously.
type Config struct {
	DatabasePath      string
	SealDelay         time.Duration
	DeliveryDelay     time.Duration
	PublicURL         string
	TrustedProxies    []string
	RequestsPerSecond float64
	MaxOperations     int
}

type account struct {
	Nonce     uint64
	Balances  map[string]*big.Int
	Clearing  map[string]*big.Int
	Fees      *big.Int
	LastBlock [32]byte
	History   []client.TransferActivity
}

type operation struct {
	Status    client.OperationStatus
	From      string
	To        string
	Request   transferRequest
	SealAt    time.Time
	DeliverAt time.Time
	Delivered bool
}

// Server owns a single isolated ledger. It never connects to external services.
// SQLite commits every mutation before any response is released to the client.
type Server struct {
	// Generated once per database for LOCAL TESTING ONLY. Never reuse this
	// key for real funds, public networks or production. The command prints it
	// only when initializing a new database; it is never exposed over HTTP.
	SeedAddress    string
	SeedPrivateKey string
	Created        bool
	db             *sql.DB
	dirty          bool
	closed         bool
	unhealthy      bool
	traffic        *traffic
	inflight       chan struct{}
	mu             sync.Mutex
	cfg            Config
	now            func() time.Time
	accounts       map[string]*account
	ops            map[string]*operation
	order          []string
	nonceOwner     map[string]string
	blocks         []*core.Block
}

func New(cfg Config) (*Server, error) {
	if cfg.SealDelay < 0 || cfg.DeliveryDelay < 0 || cfg.SealDelay > time.Minute || cfg.DeliveryDelay > time.Minute {
		return nil, fmt.Errorf("seal and delivery delays must be between zero and one minute")
	}
	traffic, err := configure(&cfg)
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, traffic: traffic, inflight: make(chan struct{}, 16), now: time.Now, accounts: make(map[string]*account), ops: make(map[string]*operation), nonceOwner: make(map[string]string)}
	if err := s.openStorage(cfg.DatabasePath); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) account(address string) *account {
	a := s.accounts[address]
	if a == nil {
		a = &account{Balances: map[string]*big.Int{}, Clearing: map[string]*big.Int{}, Fees: new(big.Int), History: []client.TransferActivity{}}
		for _, asset := range catalog {
			a.Balances[asset.Symbol] = new(big.Int)
			a.Clearing[asset.Symbol] = new(big.Int)
		}
	}
	return a
}

func (s *Server) createAccount(address string) *account {
	a := s.account(address)
	s.accounts[address] = a
	s.dirty = true
	return a
}

func integer(raw string) *big.Int {
	v, _ := new(big.Int).SetString(raw, 10)
	return v
}

func uint256(raw string) (*big.Int, error) {
	if raw == "" {
		return nil, fmt.Errorf("must be a uint256 decimal integer")
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("must contain only decimal digits")
		}
	}
	v := integer(raw)
	if v == nil || v.BitLen() > 256 {
		return nil, fmt.Errorf("integer exceeds uint256")
	}
	return v, nil
}

// userAddress parses an address or canonical user URI for requests that need a
// real account, such as transfers and account-scoped block queries.
func userAddress(raw string) (string, error) { return parseUserAddress(raw, false) }

// readAddress parses an address for account-state reads. Wallet clients probe
// the zero address before a call — an implicit empty `from` makes cast and other
// EVM tooling send eth_getTransactionCount/eth_getBalance/eth_getCode for
// 0x000…000 first — so it resolves to an empty account instead of an error.
func readAddress(raw string) (string, error) { return parseUserAddress(raw, true) }

func parseUserAddress(raw string, allowZero bool) (string, error) {
	if core.IsURI(raw) {
		if err := core.ValidateURI(raw); err != nil {
			return "", err
		}
		_, kind, path, err := core.ParseURI(raw)
		if err != nil || kind != "user" {
			return "", fmt.Errorf("clearbox supports user accounts only")
		}
		raw = path
	}
	if !common.IsHexAddress(raw) {
		return "", fmt.Errorf("account must be a 20-byte Ethereum address")
	}
	address := common.HexToAddress(raw)
	if address == (common.Address{}) && !allowZero {
		return "", fmt.Errorf("account must be a nonzero 20-byte Ethereum address")
	}
	return address.Hex(), nil
}

// LoopbackListen rejects wildcard/nonlocal binds even if the command is
// accidentally started with production environment variables or flags.
func LoopbackListen(address string) (net.Listener, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if host == "localhost" {
		host = "127.0.0.1"
		_, port, _ := net.SplitHostPort(address)
		address = net.JoinHostPort(host, port)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("clearbox must listen on a loopback IP (127.0.0.1 or ::1)")
	}
	return net.Listen("tcp", address)
}

func localHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status, code int, message, field string) {
	body := map[string]any{"code": code, "message": message}
	if field != "" {
		body["field"] = field
	}
	writeJSON(w, status, map[string]any{"error": body})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Clearbox-Simulated", "true")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Restrict Host as well as the listener so DNS rebinding cannot turn the
	// local gateway into an endpoint accessible from an arbitrary site.
	if s.cfg.PublicURL == "" && !localHost(r.Host) {
		fail(w, 403, -32003, "clearbox accepts only a loopback Host", "")
		return
	}
	if s.cfg.PublicURL != "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Retry-After, X-Clearbox-Simulated")
	} else if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !localHost(u.Host) {
			fail(w, 403, -32003, "clearbox accepts only local browser origins", "")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		s.mu.Lock()
		unhealthy := s.closed || s.unhealthy
		operations := len(s.ops)
		s.mu.Unlock()
		if unhealthy {
			fail(w, 503, -32007, "clearbox storage is unavailable", "")
		} else {
			writeJSON(w, 200, map[string]any{"status": "ok", "networkId": NetworkID, "simulated": true, "operations": operations, "maxOperations": s.cfg.MaxOperations})
		}
		return
	}
	if s.cfg.PublicURL != "" && !s.traffic.allow(r) {
		limited(w, "request rate limit exceeded")
		return
	}
	select {
	case s.inflight <- struct{}{}:
		defer func() { <-s.inflight }()
	default:
		limited(w, "clearbox is busy; retry later")
		return
	}
	// Read outside the ledger lock: a slow upload must not block every account.
	if r.Body == nil {
		r.Body = http.NoBody
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		fail(w, 400, -32600, "cannot read request body (maximum 64 KiB)", "")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	buffer := s.response(r)
	for key, values := range buffer.header {
		w.Header()[key] = values
	}
	w.WriteHeader(buffer.status)
	_, _ = w.Write(buffer.body.Bytes())
}

func (s *Server) response(r *http.Request) *bufferedResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	buffer := &bufferedResponse{header: make(http.Header), status: 200}
	if s.closed {
		fail(buffer, 503, -32008, "clearbox is closed", "")
		return buffer
	}
	s.advance(s.now())
	s.serve(buffer, r)
	if s.dirty {
		if err := s.save(); err != nil {
			s.unhealthy = true
			slog.Error("clearbox checkpoint failed", "error", err)
			// Restore the last committed state so a failed commit cannot leave
			// spendable changes in memory or release a success response.
			if s.load() != nil {
				s.closed = true
			}
			buffer = &bufferedResponse{header: make(http.Header), status: 200}
			fail(buffer, 503, -32007, "cannot persist local ledger; query the operation before retrying", "")
			return buffer
		}
		s.unhealthy = false
	}
	return buffer
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *bufferedResponse) Header() http.Header         { return w.header }
func (w *bufferedResponse) WriteHeader(status int)      { w.status = status }
func (w *bufferedResponse) Write(b []byte) (int, error) { return w.body.Write(b) }

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/rpc" {
		s.handleRPC(w, r)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/gateway/info":
		rpcURL := s.cfg.PublicURL + "/rpc"
		if s.cfg.PublicURL == "" {
			rpcURL = "http://" + r.Host + "/rpc"
		}
		writeJSON(w, 200, s.info(rpcURL))
	case r.Method == http.MethodGet && r.URL.Path == "/v1/pools":
		writeJSON(w, 200, pools())
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/accounts/"):
		s.accountRoute(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/operations/"):
		id := strings.TrimPrefix(r.URL.Path, "/v1/operations/")
		if !eip712.ValidTransferOperationID(id) {
			fail(w, 400, -32602, "operation ID must be a lowercase 0x-prefixed 32-byte hash", "id")
			return
		}
		if op := s.ops[id]; op != nil {
			writeJSON(w, 200, op.Status)
		} else {
			writeJSON(w, 200, client.OperationStatus{OperationID: id, Kind: "transfer", Status: "unknown"})
		}
	case r.Method == http.MethodPost && r.URL.Path == "/v1/transfers":
		s.submit(w, r)
	default:
		fail(w, 404, -32601, "route not supported by clearbox", "")
	}
}

// zeroAddress is how the gateway projects an unconfigured contract address.
const zeroAddress = "0x0000000000000000000000000000000000000000"

func (s *Server) info(rpcURL string) client.NetworkInfo {
	return client.NetworkInfo{
		ChainID: eip712.NetworkChainID(NetworkID).Uint64(), NetworkID: NetworkID, RPCURL: rpcURL,
		L1ChainID: 31337, Confirmations: 1, Version: "clearbox-1",
		// Contract addresses and the virtual router are deployment metadata:
		// wallet clients fingerprint the gateway with them. Clearbox serves no
		// contract call surface, and unset addresses use the gateway's zero
		// address rather than an empty string.
		Contracts: client.GatewayContracts{
			Custody:  zeroAddress,
			Registry: zeroAddress,
			Token:    catalog[2].Address,
			Faucet:   zeroAddress,
		},
		Assets:        catalog,
		VirtualRouter: eip712.RouterHex,
		Capabilities:  []string{client.CapabilityTransferStatus, client.CapabilityRecentTransfers},
	}
}

func (s *Server) accountRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/accounts/"), "/")
	if len(parts) != 2 || (parts[1] != "balances" && parts[1] != "transfers") {
		fail(w, 404, -32601, "route not supported by clearbox", "")
		return
	}
	address, err := userAddress(parts[0])
	if err != nil {
		fail(w, 400, -32602, err.Error(), "address")
		return
	}
	if parts[1] == "balances" {
		writeJSON(w, 200, s.balance(address))
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			fail(w, 400, -32602, "limit must be between 1 and 100", "limit")
			return
		}
	}
	a := s.account(address)
	start := 0
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		var cursor historyCursor
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.AccountID != accountID(address) || cursor.SealedAt <= 0 || len(cursor.BlockHash) != 64 {
			fail(w, 400, -32602, "invalid transfer history cursor", "cursor")
			return
		}
		start = -1
		for i, item := range a.History {
			if strings.TrimPrefix(item.BlockHash, "0x") == cursor.BlockHash && item.SealedAt == cursor.SealedAt && item.EntryIndex == cursor.EntryIndex {
				start = i + 1
				break
			}
		}
		if start < 0 {
			fail(w, 400, -32602, "invalid transfer history cursor", "cursor")
			return
		}
	}
	end := min(start+limit, len(a.History))
	page := client.TransferPage{Items: a.History[start:end], RetentionDays: 30}
	if end < len(a.History) {
		last := a.History[end-1]
		b, _ := json.Marshal(historyCursor{accountID(address), last.SealedAt, strings.TrimPrefix(last.BlockHash, "0x"), last.EntryIndex})
		cursor := base64.RawURLEncoding.EncodeToString(b)
		page.NextCursor = &cursor
	}
	writeJSON(w, 200, page)
}

type historyCursor struct {
	AccountID  string `json:"accountId"`
	SealedAt   int64  `json:"sealedAt"`
	BlockHash  string `json:"blockHash"`
	EntryIndex uint64 `json:"entryIndex"`
}

func accountID(address string) string {
	id := core.ComputeAccountID(address)
	return hex.EncodeToString(id[:])
}

func (s *Server) balance(address string) client.Balance {
	a := s.account(address)
	result := client.Balance{Address: address, AccountType: "User", Nonce: a.Nonce, Balances: []client.BalanceEntry{}, Codes: map[string]map[string]string{}}
	for _, asset := range catalog {
		symbol := asset.Symbol
		result.Balances = append(result.Balances, client.BalanceEntry{Asset: symbol, Symbol: symbol, ClearnetBalance: a.Balances[symbol].String()})
		for code, value := range map[string]*big.Int{"1010": a.Balances[symbol], "2020": a.Clearing[symbol]} {
			if value.Sign() != 0 {
				if result.Codes[code] == nil {
					result.Codes[code] = map[string]string{}
				}
				result.Codes[code][symbol] = value.String()
			}
		}
	}
	if a.Fees.Sign() > 0 {
		result.Codes["3010"] = map[string]string{"YELLOW": a.Fees.String()}
	}
	return result
}

func decodeBody(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON value")
	}
	return nil
}

func sortedAccounts(accounts map[string]*account) []string {
	keys := make([]string, 0, len(accounts))
	for key := range accounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

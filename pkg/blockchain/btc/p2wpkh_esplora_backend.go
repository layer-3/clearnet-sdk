package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

const (
	maxEsploraResponseBytes int64 = 8 << 20
	// maxEsploraUTXORows bounds the number of per-transaction lookups made by
	// ListUnspent. It matches the legacy depositor's default MaxInputs policy:
	// callers with more fragmented funding must consolidate before depositing.
	maxEsploraUTXORows = defaultDepositorMaxInputs
)

// EsploraP2WPKHBackendConfig configures a reusable Esplora depositor backend.
// BaseURL and Network are required. A nil HTTPClient uses a client with a
// 30-second timeout. FixedFeeRateSatPerVByte skips fee discovery when positive;
// zero uses Esplora's fee-estimates endpoint.
type EsploraP2WPKHBackendConfig struct {
	BaseURL                 string
	Network                 *chaincfg.Params
	HTTPClient              *http.Client
	FixedFeeRateSatPerVByte int64
	// RequestTimeout bounds each Esplora HTTP request when positive. Zero
	// relies on the caller context and HTTPClient's timeout.
	RequestTimeout time.Duration
}

// EsploraP2WPKHBackend supplies reusable indexed public-chain access for a
// P2WPKHSender or Depositor.
type EsploraP2WPKHBackend struct {
	baseURL        *url.URL
	net            *chaincfg.Params
	http           *http.Client
	fixedFeeRate   int64
	requestTimeout time.Duration
}

var _ DepositorBackend = (*EsploraP2WPKHBackend)(nil)

// NewEsploraP2WPKHBackend constructs an Esplora depositor backend.
func NewEsploraP2WPKHBackend(cfg EsploraP2WPKHBackendConfig) (*EsploraP2WPKHBackend, error) {
	if cfg.Network == nil {
		return nil, fmt.Errorf("btc: Esplora P2WPKH network is required")
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("btc: invalid Esplora P2WPKH base URL %q", cfg.BaseURL)
	}
	if cfg.FixedFeeRateSatPerVByte < 0 {
		return nil, fmt.Errorf("btc: Esplora P2WPKH fixed fee rate must not be negative")
	}
	if cfg.RequestTimeout < 0 {
		return nil, fmt.Errorf("btc: Esplora P2WPKH request timeout must not be negative")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &EsploraP2WPKHBackend{
		baseURL:        base,
		net:            cfg.Network,
		http:           httpClient,
		fixedFeeRate:   cfg.FixedFeeRateSatPerVByte,
		requestTimeout: cfg.RequestTimeout,
	}, nil
}

func (b *EsploraP2WPKHBackend) endpoint(parts ...string) string {
	u := *b.baseURL
	for _, part := range parts {
		u.Path += "/" + url.PathEscape(part)
	}
	return u.String()
}

func (b *EsploraP2WPKHBackend) ListUnspent(ctx context.Context, address string, minConfirmations uint64) ([]UnspentOutput, error) {
	decodedAddress, err := btcutil.DecodeAddress(address, b.net)
	if err != nil {
		return nil, fmt.Errorf("btc: decode Esplora P2WPKH address: %w", err)
	}
	if !decodedAddress.IsForNet(b.net) {
		return nil, fmt.Errorf("btc: address is not for Esplora network %s", b.net.Name)
	}
	expectedScript, err := txscript.PayToAddrScript(decodedAddress)
	if err != nil {
		return nil, fmt.Errorf("btc: derive Esplora P2WPKH script: %w", err)
	}

	var rows []struct {
		TxID   string `json:"txid"`
		Vout   uint32 `json:"vout"`
		Value  int64  `json:"value"`
		Status struct {
			Confirmed   bool    `json:"confirmed"`
			BlockHeight *uint64 `json:"block_height"`
		} `json:"status"`
	}
	if err := b.getJSON(ctx, b.endpoint("address", address, "utxo"), &rows); err != nil {
		return nil, fmt.Errorf("btc: list Esplora P2WPKH UTXOs: %w", err)
	}
	if len(rows) > maxEsploraUTXORows {
		return nil, fmt.Errorf("btc: Esplora P2WPKH UTXO count %d exceeds maximum %d", len(rows), maxEsploraUTXORows)
	}
	tipHeight, err := b.tipHeight(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]UnspentOutput, 0, len(rows))
	for _, row := range rows {
		if !row.Status.Confirmed {
			continue
		}
		confirmations, err := esploraConfirmations(row.TxID, row.Status.BlockHeight, tipHeight)
		if err != nil {
			return nil, err
		}
		if confirmations < minConfirmations {
			continue
		}
		if row.Value <= 0 {
			return nil, fmt.Errorf("btc: Esplora UTXO %s:%d has invalid value %d", row.TxID, row.Vout, row.Value)
		}
		if err := validateHashString(row.TxID); err != nil {
			return nil, fmt.Errorf("btc: Esplora UTXO has invalid txid %q: %w", row.TxID, err)
		}
		var tx struct {
			Vout []struct {
				ScriptPubKey string `json:"scriptpubkey"`
				Value        int64  `json:"value"`
			} `json:"vout"`
		}
		if err := b.getJSON(ctx, b.endpoint("tx", row.TxID), &tx); err != nil {
			return nil, fmt.Errorf("btc: load Esplora funding transaction %s: %w", row.TxID, err)
		}
		if uint64(row.Vout) >= uint64(len(tx.Vout)) {
			return nil, fmt.Errorf("btc: Esplora UTXO %s:%d output is absent", row.TxID, row.Vout)
		}
		prevout := tx.Vout[row.Vout]
		if prevout.Value != row.Value {
			return nil, fmt.Errorf("btc: Esplora UTXO %s:%d value mismatch (%d != %d)", row.TxID, row.Vout, row.Value, prevout.Value)
		}
		script, err := hex.DecodeString(prevout.ScriptPubKey)
		if err != nil {
			return nil, fmt.Errorf("btc: Esplora UTXO %s:%d scriptPubKey: %w", row.TxID, row.Vout, err)
		}
		if !bytes.Equal(script, expectedScript) {
			return nil, fmt.Errorf("btc: Esplora UTXO %s:%d scriptPubKey does not match address %s", row.TxID, row.Vout, address)
		}
		out = append(out, UnspentOutput{
			TxID: row.TxID, Vout: row.Vout, AmountSats: row.Value,
			ScriptPubKey: script, Confirmations: confirmations,
		})
	}
	return out, nil
}

func (b *EsploraP2WPKHBackend) FeeRateSatPerVByte(ctx context.Context, confirmationTarget int) (int64, error) {
	if b.fixedFeeRate > 0 {
		return b.fixedFeeRate, nil
	}
	var estimates map[string]json.Number
	if err := b.getJSON(ctx, b.endpoint("fee-estimates"), &estimates); err != nil {
		return 0, fmt.Errorf("btc: load Esplora fee estimates: %w", err)
	}
	value, ok := estimates[strconv.Itoa(confirmationTarget)]
	if !ok {
		return 0, fmt.Errorf("%w: Esplora target %d absent", ErrP2WPKHFeeEstimateUnavailable, confirmationTarget)
	}
	rate, err := esploraFeeRateSatPerVByte(value)
	if err != nil {
		return 0, fmt.Errorf("btc: invalid Esplora fee estimate: %w", err)
	}
	return rate, nil
}

func (b *EsploraP2WPKHBackend) Broadcast(ctx context.Context, rawTx []byte) (string, error) {
	status, data, err := b.request(ctx, http.MethodPost, b.endpoint("tx"), []byte(hex.EncodeToString(rawTx)))
	if err != nil {
		return "", fmt.Errorf("btc: Esplora broadcast: %w", err)
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return "", esploraHTTPStatusError(status, data)
	}
	txID := strings.TrimSpace(string(data))
	if err := validateHashString(txID); err != nil {
		return "", fmt.Errorf("btc: Esplora broadcast returned invalid txid %q: %w", txID, err)
	}
	return txID, nil
}

func (b *EsploraP2WPKHBackend) GetTransactionConfirmations(ctx context.Context, txID string) (uint64, bool, error) {
	if err := validateHashString(txID); err != nil {
		return 0, false, fmt.Errorf("btc: invalid Esplora transaction ID %q: %w", txID, err)
	}
	statusCode, data, err := b.request(ctx, http.MethodGet, b.endpoint("tx", txID, "status"), nil)
	if err != nil {
		return 0, false, err
	}
	if statusCode == http.StatusNotFound {
		return 0, false, nil
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return 0, false, esploraHTTPStatusError(statusCode, data)
	}
	var status struct {
		Confirmed   bool    `json:"confirmed"`
		BlockHeight *uint64 `json:"block_height"`
	}
	if err := decodeEsploraJSON(data, &status); err != nil {
		return 0, false, fmt.Errorf("btc: decode Esplora transaction status: %w", err)
	}
	if !status.Confirmed {
		return 0, true, nil
	}
	tipHeight, err := b.tipHeight(ctx)
	if err != nil {
		return 0, false, err
	}
	confirmations, err := esploraConfirmations(txID, status.BlockHeight, tipHeight)
	if err != nil {
		return 0, false, err
	}
	return confirmations, true, nil
}

func (b *EsploraP2WPKHBackend) tipHeight(ctx context.Context) (uint64, error) {
	status, data, err := b.request(ctx, http.MethodGet, b.endpoint("blocks", "tip", "height"), nil)
	if err != nil {
		return 0, fmt.Errorf("btc: load Esplora tip height: %w", err)
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("btc: load Esplora tip height: %w", esploraHTTPStatusError(status, data))
	}
	text := strings.TrimSpace(string(data))
	height, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("btc: invalid Esplora tip height %q: %w", text, err)
	}
	return height, nil
}

func esploraConfirmations(id string, blockHeight *uint64, tipHeight uint64) (uint64, error) {
	if blockHeight == nil {
		return 0, fmt.Errorf("btc: confirmed Esplora transaction %s is missing block_height", id)
	}
	if *blockHeight > tipHeight {
		return 0, fmt.Errorf("btc: Esplora transaction %s block height %d exceeds tip %d", id, *blockHeight, tipHeight)
	}
	delta := tipHeight - *blockHeight
	if delta == math.MaxUint64 {
		return 0, fmt.Errorf("btc: Esplora transaction %s confirmation count overflows", id)
	}
	return delta + 1, nil
}

func esploraFeeRateSatPerVByte(value json.Number) (int64, error) {
	if len(value.String()) > 128 {
		return 0, fmt.Errorf("decimal fee rate is too long")
	}
	return exactDecimalToInt64(value, 0, true)
}

func (b *EsploraP2WPKHBackend) getJSON(ctx context.Context, endpoint string, dst any) error {
	status, data, err := b.request(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return esploraHTTPStatusError(status, data)
	}
	return decodeEsploraJSON(data, dst)
}

func decodeEsploraJSON(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode JSON: unexpected trailing value")
		}
		return fmt.Errorf("decode JSON trailing data: %w", err)
	}
	return nil
}

func (b *EsploraP2WPKHBackend) request(ctx context.Context, method, endpoint string, body []byte) (int, []byte, error) {
	requestCtx, cancel := optionalBackendTimeout(ctx, b.requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "text/plain")
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxEsploraResponseBytes+1))
	if err != nil {
		return 0, nil, err
	}
	if int64(len(data)) > maxEsploraResponseBytes {
		return 0, nil, fmt.Errorf("response exceeds %d bytes", maxEsploraResponseBytes)
	}
	return resp.StatusCode, data, nil
}

func esploraHTTPStatusError(status int, body []byte) error {
	return fmt.Errorf("Esplora HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

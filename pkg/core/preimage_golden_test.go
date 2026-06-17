package core

// Byte-exact golden vectors for the CBOR-based signing preimages
// introduced by Wave 2b of the CBOR migration (docs/plans/cbor-encoding.md
// §7 Wave 2b, ADR-009 §4). Complements the Solidity-pinned preimages
// captured under `testdata/goldens/solidity-preimages/` in Wave 0.
//
// Fixtures pinned by this test:
//
//   - Block.SigningMessage = canonical CBOR of BlockHeader{Anchor,
//     SealedAt, StateRoot, EntriesDigest, K, Accounts}.
//   - BlockEntry.Hash preimage = canonical CBOR of BlockEntry{Type,
//     Account, Nonce, Payload}, hashed by keccak256 at the call site.
//   - BlockEntry.Payload bytes = canonical CBOR of one typed op per
//     entry kind (TransferOp, SwapOp, WithdrawalOp, RepegOp,
//     SessionCloseOp, SessionChallengeOp).
//
// Regenerate fixtures after a coordinated change:
//
//	go test ./pkg/core/ -run TestGoldens_Preimages -update
//
// A change that alters any golden without a corresponding version-byte
// bump (ADR-009 §5) is a schema-family break and must be reviewed as
// such. The CI guard `scripts/ci/check-preimage-goldens.sh` enforces
// byte equality against committed goldens.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

var updatePreimageGoldens = flag.Bool("update", false, "regenerate testdata/goldens/preimages/* fixtures")

// cborMarshaler narrows the codec surface to the generated MarshalCBOR
// method. (In clearnet this lived in cbor_roundtrip_test.go, which did
// not move to the SDK.)
type cborMarshaler interface {
	MarshalCBOR(w io.Writer) error
}

// preimageGoldenRoot locates `testdata/goldens/preimages/` regardless of
// cwd. Mirrors the pattern used by clearing/anchor/smt_golden_test.go so
// the two fixture families coexist without surprise.
func preimageGoldenRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file = <repo>/pkg/core/preimage_golden_test.go
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(repoRoot, "testdata", "goldens", "preimages")
}

// writeOrComparePreimage pairs with the Wave-0 solidity-preimage helper
// but lives in the `core` package because these preimages are authored
// by core.
func writeOrComparePreimage(t *testing.T, base string, inputJSON []byte, goldenBytes []byte) {
	t.Helper()
	hexPath := base + ".golden.hex"
	jsonPath := base + ".input.json"
	wantHex := strings.ToLower(hex.EncodeToString(goldenBytes))

	if *updatePreimageGoldens {
		if err := os.MkdirAll(filepath.Dir(hexPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(hexPath), err)
		}
		if err := os.WriteFile(jsonPath, append(inputJSON, '\n'), 0o644); err != nil {
			t.Fatalf("write input: %v", err)
		}
		if err := os.WriteFile(hexPath, []byte(wantHex+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	raw, err := os.ReadFile(hexPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (re-run with -update to create)", hexPath, err)
	}
	got := strings.TrimSpace(string(raw))
	if got != wantHex {
		t.Fatalf("preimage drift at %s:\n  want (current Go): %s\n  have (on disk):    %s",
			hexPath, wantHex, got)
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("missing input fixture %s: %v", jsonPath, err)
	}
}

// cborBytes marshals a codec target to a fresh byte slice.
func cborBytes(t *testing.T, m cborMarshaler) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := m.MarshalCBOR(&buf); err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	return buf.Bytes()
}

// parseHex32Preimage decodes a 64-char hex string into a 32-byte array.
func parseHex32Preimage(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		t.Fatalf("bad 32-byte hex %q: %v (len=%d)", s, err, len(b))
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

// ---------------------------------------------------------------------------
// Fixture shapes (embedded in .input.json files for human traceability)
// ---------------------------------------------------------------------------

type blockHeaderVec struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Anchor        string                  `json:"anchorHex"`
	SealedAt      int64                   `json:"sealedAt"`
	StateRoot     string                  `json:"stateRootHex"`
	EntriesDigest string                  `json:"entriesDigestHex"`
	K             uint64                  `json:"k"`
	Accounts      []accountSnapshotVecRow `json:"accounts"`
	CBORBytes     int                     `json:"cborBytes"`
	DerivedHash   string                  `json:"derivedBlockHashHex"`
}

type accountSnapshotVecRow struct {
	AccountID    string `json:"accountIDHex"`
	PrevBlockRef string `json:"prevBlockRefHex"`
	PostNonce    uint64 `json:"postNonce"`
}

type entryHashVec struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	EntryType    uint8  `json:"entryType"`
	EntryTypeStr string `json:"entryTypeStr"`
	Account      string `json:"account"`
	Nonce        uint64 `json:"nonce"`
	PayloadHex   string `json:"payloadHex"`
	CBORBytes    int    `json:"cborBytes"`
	DerivedHash  string `json:"derivedEntryHashHex"`
}

type opPayloadVec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	OpType      string      `json:"opType"`
	Op          interface{} `json:"op"`
	CBORBytes   int         `json:"cborBytes"`
}

type finalizedWithdrawalHeaderVec struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	WithdrawalID string `json:"withdrawalIDHex"`
	BlockHash    string `json:"blockHashHex"`
	EntryIndex   uint64 `json:"entryIndex"`
	FinalizedAt  int64  `json:"finalizedAt"`
	CBORBytes    int    `json:"cborBytes"`
	DerivedHash  string `json:"derivedFinalizedHashHex"`
}

// snapshotRows converts []AccountSnapshot to the serializable vec shape.
func snapshotRows(accs []AccountSnapshot) []accountSnapshotVecRow {
	out := make([]accountSnapshotVecRow, len(accs))
	for i, a := range accs {
		out[i] = accountSnapshotVecRow{
			AccountID:    hex.EncodeToString(a.AccountID[:]),
			PrevBlockRef: hex.EncodeToString(a.PrevBlockRef[:]),
			PostNonce:    a.PostNonce,
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// TestGoldens_Preimages — the single top-level entry point.
// ---------------------------------------------------------------------------

func TestGoldens_Preimages(t *testing.T) {
	root := preimageGoldenRoot(t)

	t.Run("block_header", func(t *testing.T) {
		dir := filepath.Join(root, "block_header")

		// Case 1: empty Accounts — pure header bind.
		t.Run("empty_accounts", func(t *testing.T) {
			bh := BlockHeader{
				Anchor:        parseHex32Preimage(t, "1111111111111111111111111111111111111111111111111111111111111111"),
				SealedAt:      1700000000,
				StateRoot:     parseHex32Preimage(t, "2222222222222222222222222222222222222222222222222222222222222222"),
				EntriesDigest: parseHex32Preimage(t, "3333333333333333333333333333333333333333333333333333333333333333"),
				K:             7,
				Accounts:      nil,
			}
			got := cborBytes(t, &bh)
			blk := Block{
				Anchor:        bh.Anchor,
				SealedAt:      bh.SealedAt,
				StateRoot:     bh.StateRoot,
				EntriesDigest: bh.EntriesDigest,
				K:             bh.K,
				Accounts:      bh.Accounts,
			}
			blkHash := blk.Hash()

			meta := blockHeaderVec{
				Name:          "empty_accounts",
				Description:   "BlockHeader preimage (ADR-009 §4) with no AccountSnapshots. canonical-CBOR([]). Block.Hash = keccak256(preimage).",
				Anchor:        hex.EncodeToString(bh.Anchor[:]),
				SealedAt:      bh.SealedAt,
				StateRoot:     hex.EncodeToString(bh.StateRoot[:]),
				EntriesDigest: hex.EncodeToString(bh.EntriesDigest[:]),
				K:             bh.K,
				Accounts:      snapshotRows(bh.Accounts),
				CBORBytes:     len(got),
				DerivedHash:   hex.EncodeToString(blkHash[:]),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "empty_accounts"), js, got)
		})

		// Case 2: single AccountSnapshot — binds chain continuity (Q6).
		t.Run("single_account", func(t *testing.T) {
			bh := BlockHeader{
				Anchor:        parseHex32Preimage(t, "1111111111111111111111111111111111111111111111111111111111111111"),
				SealedAt:      1700000000,
				StateRoot:     parseHex32Preimage(t, "2222222222222222222222222222222222222222222222222222222222222222"),
				EntriesDigest: parseHex32Preimage(t, "3333333333333333333333333333333333333333333333333333333333333333"),
				K:             7,
				Accounts: []AccountSnapshot{
					{
						AccountID:    parseHex32Preimage(t, "4444444444444444444444444444444444444444444444444444444444444444"),
						PrevBlockRef: parseHex32Preimage(t, "5555555555555555555555555555555555555555555555555555555555555555"),
						PostNonce:    42,
					},
				},
			}
			got := cborBytes(t, &bh)
			blk := Block{
				Anchor:        bh.Anchor,
				SealedAt:      bh.SealedAt,
				StateRoot:     bh.StateRoot,
				EntriesDigest: bh.EntriesDigest,
				K:             bh.K,
				Accounts:      bh.Accounts,
			}
			blkHash := blk.Hash()

			meta := blockHeaderVec{
				Name:          "single_account",
				Description:   "BlockHeader preimage with one AccountSnapshot. Q6 binds per-account chain continuity (PrevBlockRef + PostNonce) under the BLS signature.",
				Anchor:        hex.EncodeToString(bh.Anchor[:]),
				SealedAt:      bh.SealedAt,
				StateRoot:     hex.EncodeToString(bh.StateRoot[:]),
				EntriesDigest: hex.EncodeToString(bh.EntriesDigest[:]),
				K:             bh.K,
				Accounts:      snapshotRows(bh.Accounts),
				CBORBytes:     len(got),
				DerivedHash:   hex.EncodeToString(blkHash[:]),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "single_account"), js, got)
		})
	})

	t.Run("entry_hash", func(t *testing.T) {
		dir := filepath.Join(root, "entry_hash")
		// Transfer entry: encode a known TransferOp, then pin EntryHash preimage.
		t.Run("transfer", func(t *testing.T) {
			op := NewSingleAssetTransferOp("tx-golden-transfer", "0xBob", "USDT", decimal.NewFromBigInt(big.NewInt(500_000), 0))
			e := BlockEntry{
				Type:    OpTransfer,
				Account: "0xAlice",
				Nonce:   1,
				Payload: op.Encode(),
			}
			preimage := cborBytes(t, &e)
			entryHash := crypto.Keccak256Hash(preimage)

			meta := entryHashVec{
				Name:         "transfer",
				Description:  "EntryHash preimage = canonical CBOR of BlockEntry tuple (ADR-009 §4). EntryHash = keccak256(preimage). Payload is the CBOR-encoded TransferOp.",
				EntryType:    uint8(e.Type),
				EntryTypeStr: "OpTransfer",
				Account:      e.Account,
				Nonce:        e.Nonce,
				PayloadHex:   hex.EncodeToString(e.Payload),
				CBORBytes:    len(preimage),
				DerivedHash:  hex.EncodeToString(entryHash[:]),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "transfer"), js, preimage)
		})
	})

	t.Run("op_payload", func(t *testing.T) {
		dir := filepath.Join(root, "op_payload")

		// TransferOp
		t.Run("transfer_single_asset", func(t *testing.T) {
			op := NewSingleAssetTransferOp("tx-golden-xfer-1", "0xBob", "USDT", decimal.NewFromBigInt(big.NewInt(1_000), 0))
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "transfer_single_asset",
				Description: "TransferOp payload = canonical CBOR of TransferOp tuple (ADR-009 §4); replaces the deleted hand-rolled op_encoding.go layout.",
				OpType:      "TransferOp",
				Op: map[string]interface{}{
					"txID": op.TxID, "to": op.To,
					"assets": []map[string]interface{}{{"asset": string(op.Assets[0].Asset), "amountDec": op.Assets[0].Amount.String()}},
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "transfer_single_asset"), js, got)
		})

		// SwapOp
		t.Run("swap", func(t *testing.T) {
			op := &SwapOp{
				TxID:      "tx-golden-swap",
				AssetIn:   "ETH",
				AssetOut:  "USDC",
				AmountIn:  decimal.NewFromBigInt(big.NewInt(1_000), 0),
				AmountOut: decimal.NewFromBigInt(big.NewInt(2_000), 0),
				PoolID:    "pool:ETH-USDC",
				Fee:       decimal.NewFromBigInt(big.NewInt(3), 0),
				FeeRate:   30,
				PriceEMA:  decimal.NewFromBigInt(big.NewInt(2_000), 0),
				SpotPrice: decimal.NewFromBigInt(big.NewInt(2_001), 0),
			}
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "swap",
				Description: "SwapOp payload = canonical CBOR of SwapOp tuple.",
				OpType:      "SwapOp",
				Op: map[string]interface{}{
					"txID": op.TxID, "assetIn": string(op.AssetIn), "assetOut": string(op.AssetOut),
					"amountInDec": op.AmountIn.String(), "amountOutDec": op.AmountOut.String(),
					"poolID": op.PoolID, "feeDec": op.Fee.String(), "feeRate": op.FeeRate,
					"priceEMADec": op.PriceEMA.String(), "spotPriceDec": op.SpotPrice.String(),
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "swap"), js, got)
		})

		// WithdrawalOp
		t.Run("withdrawal", func(t *testing.T) {
			op := &WithdrawalOp{
				Asset:         "USDT",
				L1Asset:       "0xA0b8000000000000000000000000000000000001",
				Amount:        decimal.NewFromBigInt(big.NewInt(10_000_000), 0),
				ChainID:       1,
				Recipient:     "0xRecipient",
				UserSignature: []byte{0xAA, 0xBB, 0xCC},
			}
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "withdrawal",
				Description: "WithdrawalOp payload = canonical CBOR of WithdrawalOp tuple.",
				OpType:      "WithdrawalOp",
				Op: map[string]interface{}{
					"asset": string(op.Asset), "l1Asset": op.L1Asset,
					"amountDec": op.Amount.String(), "chainID": op.ChainID, "recipient": op.Recipient,
					"userSignatureHex": hex.EncodeToString(op.UserSignature),
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "withdrawal"), js, got)
		})

		// RepegOp
		t.Run("repeg", func(t *testing.T) {
			op := &RepegOp{
				PoolID:        "pool:ETH-USDC",
				OldPriceScale: big.NewInt(2_000),
				NewPriceScale: big.NewInt(2_010),
				OldVirtPrice:  big.NewInt(1_000),
				NewVirtPrice:  big.NewInt(1_001),
				Epoch:         42,
				PriceEMA:      big.NewInt(2_005),
			}
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "repeg",
				Description: "RepegOp payload = canonical CBOR of RepegOp tuple.",
				OpType:      "RepegOp",
				Op: map[string]interface{}{
					"poolID": op.PoolID, "epoch": op.Epoch,
					"oldPriceScaleDec": op.OldPriceScale.String(), "newPriceScaleDec": op.NewPriceScale.String(),
					"oldVirtPriceDec": op.OldVirtPrice.String(), "newVirtPriceDec": op.NewVirtPrice.String(),
					"priceEMADec": op.PriceEMA.String(),
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "repeg"), js, got)
		})

		// SessionCloseOp
		t.Run("session_close", func(t *testing.T) {
			op := &SessionCloseOp{
				SessionID:     parseHex32Preimage(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Version:       42,
				UserAmount:    decimal.NewFromBigInt(big.NewInt(1_000_000), 0),
				ServiceAmount: decimal.NewFromBigInt(big.NewInt(500_000), 0),
				Cooperative:   true,
			}
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "session_close",
				Description: "SessionCloseOp payload = canonical CBOR of SessionCloseOp tuple.",
				OpType:      "SessionCloseOp",
				Op: map[string]interface{}{
					"sessionIDHex":  hex.EncodeToString(op.SessionID[:]),
					"version":       op.Version,
					"userAmountDec": op.UserAmount.String(), "serviceAmountDec": op.ServiceAmount.String(),
					"cooperative": op.Cooperative,
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "session_close"), js, got)
		})

		// SessionChallengeOp
		t.Run("session_challenge", func(t *testing.T) {
			op := &SessionChallengeOp{
				SessionID:       parseHex32Preimage(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				PreviousVersion: 5,
				NewVersion:      6,
				NewDeadline:     1700003600,
			}
			got := cborBytes(t, op)
			meta := opPayloadVec{
				Name:        "session_challenge",
				Description: "SessionChallengeOp payload = canonical CBOR of SessionChallengeOp tuple.",
				OpType:      "SessionChallengeOp",
				Op: map[string]interface{}{
					"sessionIDHex":    hex.EncodeToString(op.SessionID[:]),
					"previousVersion": op.PreviousVersion, "newVersion": op.NewVersion,
					"newDeadline": op.NewDeadline,
				},
				CBORBytes: len(got),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "session_challenge"), js, got)
		})
	})

	// FinalizedWithdrawal.SigningMessage() — the BLS finality preimage.
	// Per ADR-009 §4 this is the canonical-CBOR encoding of the four-field
	// header projection (Attestation deliberately excluded so the signature
	// is not self-referential).
	t.Run("finalized_withdrawal", func(t *testing.T) {
		dir := filepath.Join(root, "finalized_withdrawal")
		t.Run("header", func(t *testing.T) {
			fw := &FinalizedWithdrawal{
				WithdrawalID: parseHex32Preimage(t, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
				BlockHash:    parseHex32Preimage(t, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"),
				EntryIndex:   3,
				FinalizedAt:  1700004200,
			}
			got := fw.SigningMessage()
			finalizedHash := crypto.Keccak256Hash(got)

			meta := finalizedWithdrawalHeaderVec{
				Name:         "header",
				Description:  "FinalizedWithdrawal.SigningMessage() = canonical CBOR of FinalizedWithdrawalHeader{WithdrawalID, BlockHash, EntryIndex, FinalizedAt} (ADR-009 §4). Finality hash = keccak256(preimage).",
				WithdrawalID: hex.EncodeToString(fw.WithdrawalID[:]),
				BlockHash:    hex.EncodeToString(fw.BlockHash[:]),
				EntryIndex:   fw.EntryIndex,
				FinalizedAt:  fw.FinalizedAt,
				CBORBytes:    len(got),
				DerivedHash:  hex.EncodeToString(finalizedHash[:]),
			}
			js, _ := json.MarshalIndent(meta, "", "  ")
			writeOrComparePreimage(t, filepath.Join(dir, "header"), js, got)
		})
	})
}

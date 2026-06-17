package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
)

// BlockHeader is the signing-preimage projection of a sealed Block
// (docs/specs/protocol/data-structures.md §5.3). It pulls out the fields the
// BLS signature actually commits to — the six-tuple that becomes the
// canonical CBOR preimage now that Wave 2b has rewritten
// `Block.SigningMessage()` around the cbor-gen codec of this struct.
//
// `Accounts` is newly included in the preimage (Q6 in the CBOR migration
// plan / ADR-009 §4) so per-account chain continuity (PrevBlockRef /
// PostNonce) commits under the BLS signature — closing the latent forgery
// surface where the legacy 106-byte preimage did not bind chain-continuity
// metadata. `K` is `uint64` on the Go side for cbor-gen compatibility
// (W2-pre widen); the semantic bound is still 256 (DQE cap).
//
// Field order on this struct IS the wire encoding (cbor-gen tuple mode,
// ADR-009 §2). Append-only evolution: any insert/reorder/rename bumps the
// global schema-family version byte (ADR-009 §5).
type BlockHeader struct {
	Anchor        [32]byte          // AccountID of the first transaction (DHT center)
	SealedAt      int64             // Unix timestamp when block was sealed
	StateRoot     [32]byte          // Flat SMT root after applying ALL entries in order
	EntriesDigest [32]byte          // Flat hash binding entries to BLS signature (§2.2)
	K             uint64            // DQE-computed cluster size; semantic bound 256
	Accounts      []AccountSnapshot // Per-account chain continuity metadata (Q6)
}

// Block is an ordered batch of operations from accounts within a DHT neighborhood,
// attested by a single BLS threshold signature (ADR-005 §2).
type Block struct {
	// --- Header (covered by BLS signature) ---
	Anchor        [32]byte              `json:"Anchor"`                  // AccountID of the first transaction (DHT center)
	SealedAt      int64                 `json:"SealedAt"`                // Unix timestamp when block was sealed
	Entries       []BlockEntry          `json:"Entries"`                 // Ordered operations (canonical sort, §2.3)
	StateRoot     [32]byte              `json:"StateRoot"`               // Flat SMT root after applying ALL entries in order
	EntriesDigest [32]byte              `json:"EntriesDigest"`           // Flat hash binding entries to BLS signature (§2.2)
	K             uint64                `json:"K"`                       // DQE-computed cluster size for aggregate VaR (§5.2). Semantic bound of 256; wire type widened to uint64 for cbor-gen compatibility.
	Attestation   Attestation           `json:"Attestation"`             // Single BLS threshold signature for the block (un-embedded so cbor-gen tuple emitter produces stable output)
	Accounts      []AccountSnapshot     `json:"Accounts"`                // Per-account chain continuity metadata
	EventBloom    [BloomByteLength]byte `json:"event_bloom,omitempty"`   // Per-block event bloom filter (data_layer.md §5.2)
	AggregateVaR  *big.Int              `json:"aggregate_var,omitempty"` // Advisory VaR projection; validators recompute (§6.6)
	SealReason    string                `json:"seal_reason,omitempty"`   // "idle" | "window" | "entries" | "value"
}

// BlockEntry is an individual operation within a Block (ADR-005 §2).
type BlockEntry struct {
	Type    EntryType `json:"Type"`    // Transfer | Swap | Withdrawal | Mint | Burn | Repeg (LP ops use Transfer per §8.3)
	Account string    `json:"Account"` // Account whose state is mutated
	Nonce   uint64    `json:"Nonce"`   // Account nonce consumed by this entry
	Payload []byte    `json:"Payload"` // Canonical CBOR of the typed op (§5.3)
}

// AccountSnapshot preserves per-account chain continuity within a Block (ADR-005 §2.4).
type AccountSnapshot struct {
	AccountID    [32]byte `json:"AccountID"`    // keccak256(Address)
	PrevBlockRef [32]byte `json:"PrevBlockRef"` // Account.LastBlockHash before this block (back-pointer)
	PostNonce    uint64   `json:"PostNonce"`    // Account.Nonce after all entries for this account
}

// BlockState represents the lifecycle phase of a pending block (ADR-005 §4.4).
type BlockState uint8

const (
	BlockOpen    BlockState = iota // Proposer received first tx, block initialized
	BlockFilling                   // Accepting additional transactions
	BlockSealed                    // Sealed, BLS signing complete
)

// SigningMessage returns the canonical CBOR preimage covered by the BLS
// signature (ADR-009 §4, CBOR migration plan §7 Wave 2b).
//
// Preimage shape: canonical CBOR of `BlockHeader{Anchor, SealedAt,
// StateRoot, EntriesDigest, K, Accounts}`. `Accounts` (per-account
// `PrevBlockRef` + `PostNonce`) is newly bound under the signature (Q6).
//
// Byte output is frozen by goldens under `testdata/goldens/preimages/`
// (see `scripts/ci/check-preimage-goldens.sh`); any change to the field
// set or encoding rules trips the CI guard and requires a version-byte
// bump per ADR-009 §5.
func (b *Block) SigningMessage() []byte {
	header := BlockHeader{
		Anchor:        b.Anchor,
		SealedAt:      b.SealedAt,
		StateRoot:     b.StateRoot,
		EntriesDigest: b.EntriesDigest,
		K:             b.K,
		Accounts:      b.Accounts,
	}
	var buf bytes.Buffer
	if err := header.MarshalCBOR(&buf); err != nil {
		// BlockHeader's generated CBOR codec writes to a bytes.Buffer,
		// which cannot fail; a panic here means a programmer error
		// (e.g. a field type the codec cannot serialize was introduced).
		panic(fmt.Errorf("core: BlockHeader.MarshalCBOR: %w", err))
	}
	return buf.Bytes()
}

// Hash returns keccak256(SigningMessage) — the block identifier (ADR-005 §2.1).
// After sealing, Account.LastBlockHash = Hash() for each account in Accounts.
//
// Formula unchanged under Wave 2b; only the preimage layout moved from
// the 106-byte hand-rolled concat to canonical CBOR of `BlockHeader`.
func (b *Block) Hash() [32]byte {
	return crypto.Keccak256Hash(b.SigningMessage())
}

// ComputeEntriesDigest computes the flat hash over all entry hashes (ADR-005 §2.2).
//
//	EntriesDigest = keccak256(EntryHash_1 || EntryHash_2 || ... || EntryHash_n)
func ComputeEntriesDigest(entries []BlockEntry) [32]byte {
	if len(entries) == 0 {
		return [32]byte{}
	}
	buf := make([]byte, 0, len(entries)*32)
	for _, e := range entries {
		h := ComputeEntryHash(e)
		buf = append(buf, h[:]...)
	}
	return crypto.Keccak256Hash(buf)
}

// ComputeEntryHash computes the hash of a single entry (ADR-005 §2.2).
//
//	EntryHash = keccak256(canonical-CBOR(BlockEntry))
//
// The preimage is the cbor-gen tuple encoding of the entry struct
// (ADR-009 §4, CBOR migration plan §7 Wave 2b). `EntriesDigest`
// continues to be the flat keccak-concat of these per-entry hashes —
// only the per-entry preimage layout changed in Wave 2b.
func ComputeEntryHash(e BlockEntry) [32]byte {
	var buf bytes.Buffer
	if err := e.MarshalCBOR(&buf); err != nil {
		// See SigningMessage: a bytes.Buffer write cannot fail, so a
		// non-nil error here is a structural regression.
		panic(fmt.Errorf("core: BlockEntry.MarshalCBOR: %w", err))
	}
	return crypto.Keccak256Hash(buf.Bytes())
}

// CanonicalSort sorts entries in deterministic order (ADR-005 §2.3,
// data-structures.md §5.3.3).
// Primary: AccountID (= keccak256(URI)) by XOR distance from Anchor, ascending.
// Secondary: Nonce ascending within the same account.
func CanonicalSort(anchor [32]byte, entries []BlockEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		ai := ComputeAccountID(entries[i].Account)
		aj := ComputeAccountID(entries[j].Account)

		// XOR distance from anchor
		for k := 0; k < 32; k++ {
			di := ai[k] ^ anchor[k]
			dj := aj[k] ^ anchor[k]
			if di != dj {
				return di < dj
			}
		}
		// Same account — sort by nonce ascending
		return entries[i].Nonce < entries[j].Nonce
	})
}

// ValidateEntryOrder checks that entries are in canonical order (ADR-005 §2.3).
// Returns nil if ordering is valid.
func ValidateEntryOrder(anchor [32]byte, entries []BlockEntry) error {
	for i := 1; i < len(entries); i++ {
		ai := ComputeAccountID(entries[i-1].Account)
		aj := ComputeAccountID(entries[i].Account)

		for k := 0; k < 32; k++ {
			di := ai[k] ^ anchor[k]
			dj := aj[k] ^ anchor[k]
			if di < dj {
				break // correct order
			}
			if di > dj {
				return ErrNonCanonicalOrder
			}
			// equal byte — continue to next byte
			if k == 31 {
				// Same account — nonces must be strictly consecutive (ADR-005 §6).
				if entries[i-1].Nonce+1 != entries[i].Nonce {
					return ErrNonConsecutiveNonce
				}
			}
		}
	}
	return nil
}

// ValidateEntriesDigest recomputes EntriesDigest and checks it matches the block header.
func (b *Block) ValidateEntriesDigest() error {
	computed := ComputeEntriesDigest(b.Entries)
	if computed != b.EntriesDigest {
		return ErrEntriesDigestMismatch
	}
	return nil
}

// ---------------------------------------------------------------------------
// BlockEntry convenience methods
// ---------------------------------------------------------------------------

// Hash returns keccak256(Type || Account || Nonce || Payload) for this entry.
func (e *BlockEntry) Hash() [32]byte {
	return ComputeEntryHash(*e)
}

// DecodeOp decodes the entry payload into its typed operation struct.
// Returns one of *TransferOp, *SwapOp, *WithdrawalOp, *RepegOp,
// *SessionCloseOp, or *SessionChallengeOp.
func (e *BlockEntry) DecodeOp() (interface{}, error) {
	return DecodePayload(*e)
}

// DecodeTransferOp decodes the payload as a TransferOp.
// Returns an error if the entry type is not OpTransfer or the payload is malformed.
func (e *BlockEntry) DecodeTransferOp() (*TransferOp, error) {
	if e.Type != OpTransfer {
		return nil, fmt.Errorf("DecodeTransferOp: entry type is %d, want %d (OpTransfer)", e.Type, OpTransfer)
	}
	op := &TransferOp{}
	if err := op.Decode(e.Payload); err != nil {
		return nil, err
	}
	return op, nil
}

// DecodeSwapOp decodes the payload as a SwapOp.
// Returns an error if the entry type is not OpSwap or the payload is malformed.
func (e *BlockEntry) DecodeSwapOp() (*SwapOp, error) {
	if e.Type != OpSwap {
		return nil, fmt.Errorf("DecodeSwapOp: entry type is %d, want %d (OpSwap)", e.Type, OpSwap)
	}
	op := &SwapOp{}
	if err := op.Decode(e.Payload); err != nil {
		return nil, err
	}
	return op, nil
}

// DecodeWithdrawalOp decodes the payload as a WithdrawalOp.
// Returns an error if the entry type is not OpWithdrawal or the payload is malformed.
func (e *BlockEntry) DecodeWithdrawalOp() (*WithdrawalOp, error) {
	if e.Type != OpWithdrawal {
		return nil, fmt.Errorf("DecodeWithdrawalOp: entry type is %d, want %d (OpWithdrawal)", e.Type, OpWithdrawal)
	}
	op := &WithdrawalOp{}
	if err := op.Decode(e.Payload); err != nil {
		return nil, err
	}
	return op, nil
}

// DecodeBurnReceipt decodes the payload as a BurnReceipt (the on-block
// payload for OpBurn entries — the receipt IS the operation).
func (e *BlockEntry) DecodeBurnReceipt() (*BurnReceipt, error) {
	if e.Type != OpBurn {
		return nil, fmt.Errorf("DecodeBurnReceipt: entry type is %d, want %d (OpBurn)", e.Type, OpBurn)
	}
	v := &BurnReceipt{}
	if err := unmarshalPayload(e.Payload, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DecodeMintReceipt decodes the payload as a MintReceipt (the on-block
// payload for OpMint entries — the receipt IS the operation).
func (e *BlockEntry) DecodeMintReceipt() (*MintReceipt, error) {
	if e.Type != OpMint {
		return nil, fmt.Errorf("DecodeMintReceipt: entry type is %d, want %d (OpMint)", e.Type, OpMint)
	}
	v := &MintReceipt{}
	if err := unmarshalPayload(e.Payload, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DecodeRepegOp decodes the payload as a RepegOp.
// Returns an error if the entry type is not OpRepeg or the payload is malformed.
func (e *BlockEntry) DecodeRepegOp() (*RepegOp, error) {
	if e.Type != OpRepeg {
		return nil, fmt.Errorf("DecodeRepegOp: entry type is %d, want %d (OpRepeg)", e.Type, OpRepeg)
	}
	op := &RepegOp{}
	if err := op.Decode(e.Payload); err != nil {
		return nil, err
	}
	return op, nil
}

// ConsumesNonce reports whether applying this entry advances the account
// nonce. Receipt-driven entries (OpMint deposit credits, OpBurn
// withdrawal-execution burns) leave the account nonce unchanged — the
// authoritative trigger is the custody-signed receipt, not a user-signed
// transaction. Used by block validation to compute the correct pre-block
// nonce from snap.PostNonce when verifying entry order.
func (e *BlockEntry) ConsumesNonce() bool {
	return e.Type != OpMint && e.Type != OpBurn
}

// IsTransfer returns true if the entry type is OpTransfer.
func (e *BlockEntry) IsTransfer() bool { return e.Type == OpTransfer }

// IsSwap returns true if the entry type is OpSwap.
func (e *BlockEntry) IsSwap() bool { return e.Type == OpSwap }

// IsWithdrawal returns true if the entry type is OpWithdrawal.
func (e *BlockEntry) IsWithdrawal() bool { return e.Type == OpWithdrawal }

// AccountID returns keccak256(e.Account).
func (e *BlockEntry) AccountID() [32]byte {
	return ComputeAccountID(e.Account)
}

// Block validation errors (extracted from clearnet block_params.go — used by
// Block entry/digest verification above).
var (
	ErrNonCanonicalOrder     = errors.New("block: entries not in canonical order")
	ErrNonConsecutiveNonce   = errors.New("block: same-account entries must have consecutive nonces")
	ErrEntriesDigestMismatch = errors.New("block: entries digest does not match header")
)

// BlockEntryRef identifies a specific entry within a sealed block.
// This is the composite key used throughout the escrow lifecycle:
// recording, challenge, finalization, and BurnReceipt handling.
type BlockEntryRef struct {
	BlockHash  [32]byte // keccak256(SigningMessage) of the sealed block
	EntryIndex uint64   // Position of the entry in Block.Entries (widened from uint16 for cbor-gen compat, Wave 2 pre)
}

// String returns a compact hex:index representation for logging.
func (r BlockEntryRef) String() string {
	return fmt.Sprintf("%x:%d", r.BlockHash[:8], r.EntryIndex)
}

// Key returns a full-hex deduplication key for map lookups.
func (r BlockEntryRef) Key() string {
	return fmt.Sprintf("%x:%d", r.BlockHash, r.EntryIndex)
}

// RefFromBlock creates a BlockEntryRef from a sealed block and entry index.
func RefFromBlock(block *Block, entryIndex uint64) BlockEntryRef {
	return BlockEntryRef{
		BlockHash:  block.Hash(),
		EntryIndex: entryIndex,
	}
}

// BurnReceipt is returned by the custody layer after L1 execution of a
// withdrawal (ADR-005 §9.1, receipt-model amendment 2026-05-12). Provider
// ECDSA signatures (k-of-n against the configured custody signer directory
// — manifest custody.signers/threshold; future Registry-backed source)
// confirm the withdrawal was executed on-chain; the cluster validates the
// receipt and applies the second leg of the burn (DR 2010 / CR 1020).
type BurnReceipt struct {
	WithdrawalID  [32]byte // keccak256(accountId, blockHash, entryIndex, chainId, recipient, asset, amount, nonce)
	BlockEntryRef          // Block hash + entry index of the escrow entry
	L1TxHash      [32]byte // Transaction hash on the target chain
	Signatures    [][]byte // k-of-n provider ECDSA signatures over the receipt digest
}

// MintReceipt is issued by the custody layer after an L1 deposit confirms
// (ADR-005 §9.1, receipt-model amendment 2026-05-12). Provider ECDSA
// signatures (k-of-n against the configured custody signer directory —
// manifest custody.signers/threshold; future Registry-backed source) attest
// that the deposit landed and reached the configured confirmation depth;
// the cluster validates the receipt and credits the user account
// (DR 1010 / CR 2010).
//
// Idempotency is keyed by (ChainID, L1TxHash, LogIndex) so a re-issued
// receipt cannot produce a second credit. Clearnet does not watch the
// chain — receipts are the only deposit ingress.
type MintReceipt struct {
	ChainID     uint64   // EIP-155 chain id where the deposit landed
	L1TxHash    [32]byte // Transaction hash on the source chain
	LogIndex    uint64   // Log index of the Deposited event within the tx
	Account     string   // Clearnet account URI to credit
	Asset       string   // Asset symbol (matches on-chain symbol)
	Amount      *big.Int // Deposit amount (base units, must be > 0)
	BlockNumber uint64   // L1 block number for diagnostics / receipts
	Signatures  [][]byte // k-of-n provider ECDSA signatures over the receipt digest
}

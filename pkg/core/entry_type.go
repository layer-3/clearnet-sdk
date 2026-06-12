package core

import (
	"math/big"
	"math/bits"
)

// EntryType identifies the kind of state operation in a BlockEntry (ADR-005 §5.3.5).
type EntryType uint8

const (
	OpTransfer         EntryType = iota + 1 // SS5.3.5 — Transfer: debit sender, credit receiver (also LP operations per SS8.3)
	OpSwap                                  // SS8.1 — Swap: AMM execution result
	OpWithdrawal                            // SS7.3.1 — Withdrawal: fund lock for L1
	OpBurn                                  // ADR-005 §9.1 receipt-model — Withdrawal execution burn: DR 1020 / CR 2010 driven by a custody-signed BurnReceipt. Payload = canonical CBOR of the BurnReceipt.
	OpMint                                  // ADR-005 §9.1 receipt-model — Deposit credit: DR 1010 / CR 2010 driven by a custody-signed MintReceipt. Payload = canonical CBOR of the MintReceipt.
	OpRepeg                                 // SS8.1.2 — PriceScale adjustment
	OpSessionClose                          // service_sessions.md §4 — Cooperative or unilateral session close
	OpSessionChallenge                      // service_sessions.md §4 — Session dispute challenge
)

// Attestation holds the BLS threshold signature proving cluster consensus (ADR-005 §5.3).
// Embedded in every Block. Verified off-chain by clearing layer and custody layer.
//
// §7 coordination: Validators is []BLSPubKey (variable-length byte slices) so entries can
// hold full 128-byte BN254 G2 public keys (WS-A.2). Legacy NodeID-only entries remain
// parseable as 32-byte slices; WS-A.2 populates real G2 keys from core.Slot.BLSPubKey.
type Attestation struct {
	ThresholdSig []byte   `json:"ThresholdSig"` // Aggregated BLS signature (G1 point)
	Bitmask      [32]byte `json:"Bitmask"`      // Bitmask indicating which validators signed (up to 256 bits, SS5.3).
	Validators   [][]byte `json:"Validators"`   // BLS public keys (G2) of signing validators — 128 bytes each once WS-A.2 lands.
}

// SetBitmaskBit sets bit i (0-indexed, little-endian bit order) in a [32]byte bitmask.
// Bit i is stored at byte i/8, bit position i%8 within that byte.
func SetBitmaskBit(bm *[32]byte, i int) {
	if i < 0 || i >= 256 {
		return
	}
	bm[i/8] |= 1 << uint(i%8)
}

// GetBitmaskBit returns true if bit i is set in the bitmask.
func GetBitmaskBit(bm [32]byte, i int) bool {
	if i < 0 || i >= 256 {
		return false
	}
	return bm[i/8]&(1<<uint(i%8)) != 0
}

// BitmaskIsZero returns true if no bits are set in the bitmask.
func BitmaskIsZero(bm [32]byte) bool {
	return bm == [32]byte{}
}

// BitmaskBitLen returns the position of the highest set bit + 1 (0 if empty).
func BitmaskBitLen(bm [32]byte) int {
	for i := 31; i >= 0; i-- {
		if bm[i] != 0 {
			b := bm[i]
			bit := 7
			for b&(1<<uint(bit)) == 0 {
				bit--
			}
			return i*8 + bit + 1
		}
	}
	return 0
}

// BitmaskOnesCount returns the number of set bits in the bitmask.
func BitmaskOnesCount(bm [32]byte) int {
	count := 0
	for i := 0; i < 32; i++ {
		count += bits.OnesCount8(bm[i])
	}
	return count
}

// BitmaskToBigInt returns a big.Int representing the bitmask.
// The resulting big.Int uses big-endian byte order, so the little-endian
// bitmask bytes are reversed.
func BitmaskToBigInt(bm [32]byte) *big.Int {
	var be [32]byte
	for i := 0; i < 32; i++ {
		be[i] = bm[31-i]
	}
	return new(big.Int).SetBytes(be[:])
}

package sol

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

// withdrawDomain is mirrored byte-for-byte by the Anchor program
// (programs/custody/src/digest.rs).
const withdrawDomain = "YELLOW_CUSTODY_SOL_WITHDRAW_V1"

// WithdrawDigest computes the 32-byte digest the providers sign for a
// withdrawal, matching the program's `withdraw_digest`:
//
//	sha256(WITHDRAW_DOMAIN ‖ chainID(BE) ‖ programID ‖ vault
//	       ‖ to ‖ mint ‖ amount(BE) ‖ withdrawalID)
//
// mint == the zero pubkey denotes native SOL.
func WithdrawDigest(chainID uint64, programID, vault, to, mint solana.PublicKey, amount uint64, withdrawalID [32]byte) [32]byte {
	var u8 [8]byte
	h := sha256.New()
	h.Write([]byte(withdrawDomain))
	binary.BigEndian.PutUint64(u8[:], chainID)
	h.Write(u8[:])
	h.Write(programID[:])
	h.Write(vault[:])
	h.Write(to[:])
	h.Write(mint[:])
	binary.BigEndian.PutUint64(u8[:], amount)
	h.Write(u8[:])
	h.Write(withdrawalID[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

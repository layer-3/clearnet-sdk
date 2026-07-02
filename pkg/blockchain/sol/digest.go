package sol

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

// Domain separators, mirrored byte-for-byte by the Anchor program
// (chains/sol/contract/programs/custody/src/digest.rs).
//
// The withdraw domain is `_V2`: it appends an 8-byte big-endian deadline
// (unix seconds) to the preimage, so the digest is incompatible with the
// earlier `_V1` authorizations.
const (
	withdrawDomain = "YELLOW_CUSTODY_SOL_WITHDRAW_V2"
	rotateDomain   = "YELLOW_CUSTODY_SOL_ROTATE_V1"
)

// WithdrawDigest computes the 32-byte digest the providers sign for a
// withdrawal, matching the program's `withdraw_digest`:
//
//	sha256(WITHDRAW_DOMAIN ‖ chainID(BE) ‖ programID ‖ vault
//	       ‖ to ‖ mint ‖ amount(BE) ‖ withdrawalID ‖ deadline(BE))
//
// mint == the zero pubkey denotes native SOL. deadline is the unix-second time
// bound, encoded as an int64 big-endian so it matches Solana's
// Clock::unix_timestamp (i64) and the block SealedAt it derives from.
func WithdrawDigest(chainID uint64, programID, vault, to, mint solana.PublicKey, amount uint64, withdrawalID [32]byte, deadline int64) [32]byte {
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
	binary.BigEndian.PutUint64(u8[:], uint64(deadline))
	h.Write(u8[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// SignersCommitment computes sha256(newSigners ‖ newThreshold), matching the
// program's `signers_commitment`. It is the payload the rotation digest binds.
func SignersCommitment(newSigners []solana.PublicKey, newThreshold uint8) [32]byte {
	h := sha256.New()
	for _, s := range newSigners {
		h.Write(s[:])
	}
	h.Write([]byte{newThreshold})
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// RotateDigest computes the 32-byte digest the providers sign for a signer
// rotation, matching the program's `rotate_digest`:
//
//	sha256(ROTATE_DOMAIN ‖ chainID(BE) ‖ programID ‖ config
//	       ‖ signersCommitment ‖ signerNonce(BE))
//
// signerNonce is the on-chain Config.SignerNonce — the rotation replay token.
func RotateDigest(chainID uint64, programID, config solana.PublicKey, commitment [32]byte, signerNonce uint64) [32]byte {
	var u8 [8]byte
	h := sha256.New()
	h.Write([]byte(rotateDomain))
	binary.BigEndian.PutUint64(u8[:], chainID)
	h.Write(u8[:])
	h.Write(programID[:])
	h.Write(config[:])
	h.Write(commitment[:])
	binary.BigEndian.PutUint64(u8[:], signerNonce)
	h.Write(u8[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

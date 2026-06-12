package bls

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// VerifyClusterSignature verifies an aggregated BLS threshold signature
// against the set of validators indicated by the bitmask.
// The signature is expected in ABI-encoded format: (uint256 bitmask, uint256[2] sigma, uint256[4] apkG2).
//
// k is the DQE-computed signing quorum for this block (block.K). The
// threshold is floor(2k/3)+1, not floor(2r/3)+1 where r=len(validators) is
// the shard replication factor — in multi-Slot shards r > k, so a block
// sealed by exactly k validators must not be rejected for under-signing
// against r (ISSUE-035 WS-1).
//
// §7 coordination: validators is [][]byte (serialized BN254 G2 pubkeys,
// exactly 128 bytes each — ADR-008). The on-chain Slasher reconstructs
// apkG2 from the bitmask by summing those pubkeys; off-chain verification
// performs the same reconstruction and rejects entries of any other length.
// k=0 is rejected: every caller must supply the explicit signing quorum
// (F-CONSENSUS-001).
func VerifyClusterSignature(data []byte, signature []byte, bitmask [32]byte, k uint16, validators [][]byte) (bool, error) {
	if k == 0 {
		return false, errors.New("verify: k=0 not allowed; signing quorum must be explicit")
	}
	// Count signers via popcount on the full bitmask. We can't bound the
	// loop by len(validators) because some sealers (P2PBlockSealer) append
	// validators in partial-receipt order while bitmask bits track the
	// original cluster-member index — so a high-index signer can fall
	// outside a len(validators)-bounded iteration. Popcount is index-agnostic
	// and matches the on-chain Slasher's signer count.
	signerCount := core.BitmaskOnesCount(bitmask)
	threshold := (int(k)*2)/3 + 1
	if signerCount < threshold {
		return false, nil
	}

	// ABI-encoded format: (uint256 bitmask, uint256[2] sigma, uint256[4] apkG2).
	args := abi.Arguments{
		{Type: abiutil.Uint256},
		{Type: abiutil.Uint256Arr2},
		{Type: abiutil.Uint256Arr4},
	}

	values, err := args.Unpack(signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	if len(validators) == 0 {
		return false, nil
	}

	// ISSUE-043-01: reject tampered tuple-internal bitmask. AggregateSignatures
	// packs values[0]=0 today (it lives in Attestation.Bitmask instead), so the
	// invariant is zero OR equal to the outer bitmask — anything else is a
	// post-seal edit in a region the BLS signature does not cover.
	tupleBitmask := values[0].(*big.Int)
	if tupleBitmask.Sign() != 0 && tupleBitmask.Cmp(core.BitmaskToBigInt(bitmask)) != 0 {
		return false, nil
	}

	sigCoords := values[1].([2]*big.Int)
	apkCoords := values[2].([4]*big.Int)

	var sigma bn254.G1Affine
	sigma.X.SetBigInt(sigCoords[0])
	sigma.Y.SetBigInt(sigCoords[1])

	var apkG2 bn254.G2Affine
	apkG2.X.A1.SetBigInt(apkCoords[0]) // imaginary
	apkG2.X.A0.SetBigInt(apkCoords[1]) // real
	apkG2.Y.A1.SetBigInt(apkCoords[2]) // imaginary
	apkG2.Y.A0.SetBigInt(apkCoords[3]) // real

	// ISSUE-043-02: bind the outer bitmask to the aggregated apkG2.
	//
	// Spec invariant (ADR-008 §11): every set bit of the bitmask references
	// a valid index into Validators, i.e.
	//     highest_set_bit(bitmask) < len(Validators)
	// AND apkG2 == sum(Validators[i] for bit i set).
	//
	// Two sealer paths co-exist and produce compatible blocks under this
	// invariant:
	//   - cluster.SigningCoordinator.SealBlock writes ALL r shard
	//     members into Validators and a signers-only bitmask; typically
	//     popcount(bitmask) < len(Validators). This is the production
	//     path (`cmd/clearnode/main.go`).
	//   - node/service/block_sealer.go P2PBlockSealer writes only
	//     signers into Validators with popcount == len(Validators).
	//
	// Previously this function rejected when popcount != len(Validators),
	// which wrongly rejected every block sealed via SigningCoordinator
	// with threshold < r. The fix below aligns off-chain verification
	// with the on-chain Slasher reconstruction: range-check the bitmask
	// against the validator roster, then sum only the validators at
	// set bit positions.
	if len(validators) > 0 {
		// Bitmask is [32]byte; len(validators) is guarded by this check.
		// If any set bit i has i >= len(validators), the reconstruction
		// would dereference out of range.
		if core.BitmaskBitLen(bitmask) > len(validators) {
			return false, nil
		}
	}

	// ADR-008 §11 / F-CONSENSUS-001: every validator entry MUST be a full
	// 128-byte BN254 G2 pubkey. NodeID placeholders or any other length are
	// rejected. The sum of the bitmask-selected entries MUST match the
	// embedded apkG2 — mirrors the on-chain Slasher reconstruction.
	for i, v := range validators {
		if len(v) != core.BLSPubKeyG2Len {
			return false, fmt.Errorf("verify: validator %d has wrong length: got %d, want %d", i, len(v), core.BLSPubKeyG2Len)
		}
	}
	var expected bn254.G2Affine
	for i := 0; i < len(validators) && i < 256; i++ {
		if !core.GetBitmaskBit(bitmask, i) {
			continue
		}
		pub, dErr := DeserializeG2(validators[i])
		if dErr != nil {
			return false, fmt.Errorf("validator pubkey decode: %w", dErr)
		}
		expected.Add(&expected, &pub)
	}
	if !expected.Equal(&apkG2) {
		return false, nil
	}

	// A.1: hash the full input so verification matches Engine.Sign.
	msgHash := crypto.Keccak256Hash(data)

	return Verify(sigma, apkG2, msgHash)
}

// ComputeVaultWithdrawalID computes the frozen withdrawal ID used at the vault
// level after a withdrawal entry has been sealed into a block.
//
//	WithdrawalID = keccak256(accountId || blockHash || entryIndex || chainId || recipient || asset || amount || nonce)
//
// All integer fields use fixed-width big-endian encoding; amount is uint256.
// Including account identity and block location prevents economic-tuple
// collisions while still giving the vault a stable replay key for the exact
// finalized withdrawal.
func ComputeVaultWithdrawalID(accountID, blockHash [32]byte, entryIndex, chainID uint64, recipient, asset common.Address, amount *big.Int, nonce uint64) [32]byte {
	buf := make([]byte, 0, 32+32+8+8+20+20+32+8)
	buf = append(buf, accountID[:]...)
	buf = append(buf, blockHash[:]...)
	buf = appendUint64BE(buf, entryIndex)
	buf = appendUint64BE(buf, chainID)
	buf = append(buf, recipient.Bytes()...)
	buf = append(buf, asset.Bytes()...)
	buf = append(buf, uint256Bytes(amount)...)
	buf = appendUint64BE(buf, nonce)
	return crypto.Keccak256Hash(buf)
}

func appendUint64BE(buf []byte, v uint64) []byte {
	return append(buf,
		byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32),
		byte(v>>24), byte(v>>16), byte(v>>8), byte(v),
	)
}

func uint256Bytes(v *big.Int) []byte {
	var out [32]byte
	if v == nil || v.Sign() < 0 {
		return out[:]
	}
	b := v.Bytes()
	if len(b) > len(out) {
		b = b[len(b)-len(out):]
	}
	copy(out[len(out)-len(b):], b)
	return out[:]
}

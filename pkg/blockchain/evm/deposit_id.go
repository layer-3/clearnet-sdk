package evm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// nonceSequenceBits is the width of the sequence field in the 2D nonce: upper
// 192 bits are a caller-chosen key, lower 64 a consecutive per-(depositor, key)
// sequence (ERC-4337 EntryPoint layout).
const nonceSequenceBits = 64

// maxNonceKeyBits is the width of the key field (Custody.getNonce takes a
// uint192 key).
const maxNonceKeyBits = 192

// depositIDArguments is the abi.Arguments for (chainid uint256, vault
// address, depositor address, nonce uint256) — the exact tuple hashed into
// an EVM deposit ID (ADR-018, ISS-068). Built once at package init;
// abi.NewType never errors for these fixed elementary types.
var depositIDArguments = mustDepositIDArguments()

func mustDepositIDArguments() abi.Arguments {
	uint256Ty, err := abi.NewType("uint256", "", nil)
	if err != nil {
		panic(fmt.Sprintf("evm: build deposit ID uint256 type: %v", err))
	}
	addressTy, err := abi.NewType("address", "", nil)
	if err != nil {
		panic(fmt.Sprintf("evm: build deposit ID address type: %v", err))
	}
	return abi.Arguments{
		{Type: uint256Ty}, // chainid
		{Type: addressTy}, // vault
		{Type: addressTy}, // depositor
		{Type: uint256Ty}, // nonce
	}
}

// DepositID computes the EVM deposit ID (ISS-068, ADR-018 §Transaction IDs):
// "0x" + lowercase hex of keccak256(abi.encode(chainid, vault, depositor,
// nonce)). MintReceipts are signed over it, so it must stay byte-identical to
// the TypeScript implementation and the formula custody's IDeposit.sol
// documents; shared vectors in testdata/deposit_id_vectors.json at the
// repository root.
func DepositID(chainID *big.Int, vault, depositor common.Address, nonce *big.Int) string {
	encoded, err := depositIDArguments.Pack(chainID, vault, depositor, nonce)
	if err != nil {
		// Only reachable for a negative big.Int (nil panics inside Pack).
		// Values of 2^256 or more do not fail: they are packed modulo 2^256.
		panic(fmt.Sprintf("evm: pack deposit ID inputs: %v", err))
	}
	return "0x" + hex.EncodeToString(crypto.Keccak256(encoded))
}

// ErrInvalidDepositID is returned by ParseDepositID for anything other than
// the exact DepositID shape.
var ErrInvalidDepositID = errors.New("evm: deposit ID must be \"0x\" followed by 64 lowercase hex characters")

// ParseDepositID validates that id has exactly the DepositID shape ("0x" +
// 64 lowercase hex characters) and returns its 32 raw bytes. It rejects a
// "txHash/logIndex" shape. It cannot tell a deposit ID from a transaction hash
// (same shape), so callers keep them in separate fields (TxHash, DepositID).
func ParseDepositID(id string) ([32]byte, error) {
	var out [32]byte
	if len(id) != 66 || id[0] != '0' || id[1] != 'x' {
		return out, ErrInvalidDepositID
	}
	for _, c := range id[2:] {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		if !isDigit && !isLowerHex {
			return out, ErrInvalidDepositID
		}
	}
	if _, err := hex.Decode(out[:], []byte(id[2:])); err != nil {
		return out, ErrInvalidDepositID
	}
	return out, nil
}

// ComposeNonce packs key and sequence into the single uint256 nonce
// Custody.deposit expects: key << 64 | sequence. It returns an error unless key
// is non-nil and in [0, 2^192).
func ComposeNonce(key *big.Int, sequence uint64) (*big.Int, error) {
	if err := validateNonceKey(key); err != nil {
		return nil, err
	}
	nonce := new(big.Int).Lsh(key, nonceSequenceBits)
	return nonce.Or(nonce, new(big.Int).SetUint64(sequence)), nil
}

func validateNonceKey(key *big.Int) error {
	if key == nil {
		return errors.New("evm: nonce key is nil")
	}
	if key.Sign() < 0 || key.BitLen() > maxNonceKeyBits {
		return fmt.Errorf("evm: nonce key %s not in [0, 2^192)", key)
	}
	return nil
}

// SplitNonce reverses ComposeNonce: key is the upper 192 bits of nonce,
// sequence the lower 64.
func SplitNonce(nonce *big.Int) (key *big.Int, sequence uint64) {
	key = new(big.Int).Rsh(nonce, nonceSequenceBits)
	mask := new(big.Int).SetUint64(math.MaxUint64)
	sequence = new(big.Int).And(nonce, mask).Uint64()
	return key, sequence
}

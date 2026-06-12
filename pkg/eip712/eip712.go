// Package eip712 provides shared EIP-712 signature verification for both
// the gateway (HTTP->TCP relay) and the clearnode (TCP command verification).
package eip712

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/abiutil"
)

const (
	Name      = "Clearnet"
	Version   = "1"
	RouterHex = "0x00000000000000000000000000000000434C5200"
)

var (
	RouterAddr = common.HexToAddress(RouterHex)

	DomainTypeHash = crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))

	// Per ADR-009 §6.1, every user-authorized field of an operation MUST
	// appear in its EIP-712 typehash preimage. The schema here matches the
	// authoritative table in ADR-009 §6.1; `docs/specs/edge/gateway.md §5.2`
	// is the descriptive mirror. Any relay that could otherwise mutate a
	// listed field post-signing is closed by inclusion in the preimage.
	TransferAssetTypeHash = crypto.Keccak256Hash([]byte(
		"TransferAsset(string asset,uint256 amount)"))

	TransferTypeHash = crypto.Keccak256Hash([]byte(
		"Transfer(address to,TransferAsset[] assets,uint256 maxFee,uint64 nonce)TransferAsset(string asset,uint256 amount)"))

	SwapTypeHash = crypto.Keccak256Hash([]byte(
		"Swap(string assetIn,string assetOut,uint256 amountIn,uint256 minAmountOut,uint256 maxFee,uint64 nonce)"))

	WithdrawalTypeHash = crypto.Keccak256Hash([]byte(
		"Withdrawal(string asset,uint256 amount,uint256 chainId,address recipient,uint256 maxFee,uint64 nonce)"))
)

// TransferAsset is the EIP-712 projection of one TransferOp asset leg.
type TransferAsset struct {
	Asset  string
	Amount *big.Int
}

// NetworkChainID derives a numeric EVM chain ID from a 4-byte NetworkID string.
// "YDEV" -> 0x59444556 -> 1497646422.
func NetworkChainID(networkID string) *big.Int {
	if len(networkID) != 4 {
		return big.NewInt(0)
	}
	b := []byte(networkID)
	return new(big.Int).SetUint64(uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3]))
}

// ComputeDomainSeparator computes the EIP-712 domain separator for the given chain ID.
func ComputeDomainSeparator(chainID *big.Int) [32]byte {
	nameHash := crypto.Keccak256Hash([]byte(Name))
	versionHash := crypto.Keccak256Hash([]byte(Version))
	args := abi.Arguments{
		{Type: abiutil.Bytes32}, {Type: abiutil.Bytes32}, {Type: abiutil.Bytes32},
		{Type: abiutil.Uint256}, {Type: abiutil.Address},
	}
	packed, _ := args.Pack(DomainTypeHash, nameHash, versionHash, chainID, RouterAddr)
	return crypto.Keccak256Hash(packed)
}

// Digest computes keccak256("\x19\x01" || domainSeparator || structHash).
func Digest(domainSep [32]byte, structHash [32]byte) []byte {
	msg := make([]byte, 2+32+32)
	msg[0] = 0x19
	msg[1] = 0x01
	copy(msg[2:34], domainSep[:])
	copy(msg[34:66], structHash[:])
	return crypto.Keccak256(msg)
}

// RecoverSigner recovers the ECDSA signer from an EIP-712 digest and signature.
func RecoverSigner(digest []byte, sig []byte) (common.Address, error) {
	s := make([]byte, len(sig))
	copy(s, sig)
	if len(s) == 65 && s[64] >= 27 {
		s[64] -= 27
	}
	pubBytes, err := crypto.Ecrecover(digest, s)
	if err != nil {
		return common.Address{}, err
	}
	pub, err := crypto.UnmarshalPubkey(pubBytes)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}

// bigOrZero returns v if non-nil, else zero. Callers may pass nil for optional
// user-authorized bounds (e.g. maxFee, minAmountOut); they still bind under the
// signature — nil marshals to uint256(0), so a signature produced with nil
// cannot be replayed against a nonzero bound.
func bigOrZero(v *big.Int) *big.Int {
	if v == nil {
		return big.NewInt(0)
	}
	return v
}

// NormalizeTransferAssets returns a sorted, duplicate-free copy of the Transfer
// asset list used by the frozen EIP-712 Transfer schema.
func NormalizeTransferAssets(in []TransferAsset) ([]TransferAsset, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("transfer assets required")
	}
	out := make([]TransferAsset, len(in))
	for i, asset := range in {
		if asset.Asset == "" {
			return nil, fmt.Errorf("transfer asset %d missing Asset", i)
		}
		if asset.Amount == nil || asset.Amount.Sign() <= 0 {
			return nil, fmt.Errorf("transfer asset %s Amount must be > 0", asset.Asset)
		}
		out[i] = TransferAsset{Asset: asset.Asset, Amount: new(big.Int).Set(asset.Amount)}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Asset < out[j].Asset })
	for i := 1; i < len(out); i++ {
		if out[i-1].Asset == out[i].Asset {
			return nil, fmt.Errorf("duplicate transfer asset %s", out[i].Asset)
		}
	}
	return out, nil
}

func hashTransferAsset(asset TransferAsset) ([32]byte, error) {
	assetHash := crypto.Keccak256Hash([]byte(asset.Asset))
	args := abi.Arguments{
		{Type: abiutil.Bytes32}, {Type: abiutil.Bytes32}, {Type: abiutil.Uint256},
	}
	packed, err := args.Pack(TransferAssetTypeHash, assetHash, bigOrZero(asset.Amount))
	if err != nil {
		return [32]byte{}, err
	}
	return crypto.Keccak256Hash(packed), nil
}

// HashTransferAssets hashes the canonical EIP-712 array payload for
// TransferAsset[].
func HashTransferAssets(assets []TransferAsset) ([32]byte, error) {
	normalized, err := NormalizeTransferAssets(assets)
	if err != nil {
		return [32]byte{}, err
	}
	buf := make([]byte, 0, 32*len(normalized))
	for _, asset := range normalized {
		h, err := hashTransferAsset(asset)
		if err != nil {
			return [32]byte{}, err
		}
		buf = append(buf, h[:]...)
	}
	return crypto.Keccak256Hash(buf), nil
}

// TransferStructHash returns the EIP-712 struct hash for the canonical
// multi-asset Transfer schema.
func TransferStructHash(to common.Address, assets []TransferAsset, maxFee *big.Int, nonce uint64) ([32]byte, error) {
	assetsHash, err := HashTransferAssets(assets)
	if err != nil {
		return [32]byte{}, err
	}
	args := abi.Arguments{
		{Type: abiutil.Bytes32}, {Type: abiutil.Address}, {Type: abiutil.Bytes32},
		{Type: abiutil.Uint256}, {Type: abiutil.Uint64},
	}
	packed, err := args.Pack(TransferTypeHash, to, assetsHash, bigOrZero(maxFee), nonce)
	if err != nil {
		return [32]byte{}, err
	}
	return crypto.Keccak256Hash(packed), nil
}

// RecoverTransfer recovers the signer of a Transfer EIP-712 message.
// Per ADR-009 §6.1: Transfer(address to, TransferAsset[] assets, uint256 maxFee, uint64 nonce).
func RecoverTransfer(chainID *big.Int, to common.Address, assets []TransferAsset, maxFee *big.Int, nonce uint64, sig []byte) (common.Address, error) {
	domainSep := ComputeDomainSeparator(chainID)
	structHash, err := TransferStructHash(to, assets, maxFee, nonce)
	if err != nil {
		return common.Address{}, err
	}
	digest := Digest(domainSep, structHash)
	return RecoverSigner(digest, sig)
}

// RecoverSwap recovers the signer of a Swap EIP-712 message.
// Per ADR-009 §6.1: Swap(string assetIn, string assetOut, uint256 amountIn, uint256 minAmountOut, uint256 maxFee, uint64 nonce).
func RecoverSwap(chainID *big.Int, assetIn, assetOut string, amountIn, minAmountOut, maxFee *big.Int, nonce uint64, sig []byte) (common.Address, error) {
	domainSep := ComputeDomainSeparator(chainID)
	aiHash := crypto.Keccak256Hash([]byte(assetIn))
	aoHash := crypto.Keccak256Hash([]byte(assetOut))
	args := abi.Arguments{
		{Type: abiutil.Bytes32}, {Type: abiutil.Bytes32}, {Type: abiutil.Bytes32},
		{Type: abiutil.Uint256}, {Type: abiutil.Uint256}, {Type: abiutil.Uint256}, {Type: abiutil.Uint64},
	}
	packed, err := args.Pack(SwapTypeHash, aiHash, aoHash, bigOrZero(amountIn), bigOrZero(minAmountOut), bigOrZero(maxFee), nonce)
	if err != nil {
		return common.Address{}, err
	}
	structHash := crypto.Keccak256Hash(packed)
	digest := Digest(domainSep, structHash)
	return RecoverSigner(digest, sig)
}

// RecoverWithdrawal recovers the signer of a Withdrawal EIP-712 message.
// Per ADR-009 §6.1: Withdrawal(string asset, uint256 amount, uint256 chainId, address recipient, uint256 maxFee, uint64 nonce).
func RecoverWithdrawal(chainID *big.Int, asset string, amount *big.Int, targetChainID uint64, recipient common.Address, maxFee *big.Int, nonce uint64, sig []byte) (common.Address, error) {
	domainSep := ComputeDomainSeparator(chainID)
	assetHash := crypto.Keccak256Hash([]byte(asset))
	args := abi.Arguments{
		{Type: abiutil.Bytes32}, {Type: abiutil.Bytes32}, {Type: abiutil.Uint256},
		{Type: abiutil.Uint256}, {Type: abiutil.Address}, {Type: abiutil.Uint256}, {Type: abiutil.Uint64},
	}
	packed, err := args.Pack(WithdrawalTypeHash, assetHash, bigOrZero(amount), new(big.Int).SetUint64(targetChainID), recipient, bigOrZero(maxFee), nonce)
	if err != nil {
		return common.Address{}, err
	}
	structHash := crypto.Keccak256Hash(packed)
	digest := Digest(domainSep, structHash)
	return RecoverSigner(digest, sig)
}

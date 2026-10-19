package eip712

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func signedUint256(name string, value *big.Int) error {
	if value == nil || value.Sign() < 0 || value.BitLen() > 256 {
		return fmt.Errorf("%s must be a non-negative uint256", name)
	}
	return nil
}

// SwapDigest hashes all user-authorized Swap fields without a private key.
func SwapDigest(chainID *big.Int, assetIn, assetOut string, amountIn, minAmountOut, maxFee *big.Int, nonce uint64) ([32]byte, error) {
	if chainID == nil || chainID.Sign() <= 0 || chainID.BitLen() > 256 || assetIn == "" || assetOut == "" || assetIn == assetOut {
		return [32]byte{}, fmt.Errorf("invalid swap domain or asset pair")
	}
	for _, field := range []struct {
		name  string
		value *big.Int
	}{{"amountIn", amountIn}, {"minAmountOut", minAmountOut}, {"maxFee", maxFee}} {
		if err := signedUint256(field.name, field.value); err != nil {
			return [32]byte{}, err
		}
	}
	if amountIn.Sign() == 0 {
		return [32]byte{}, fmt.Errorf("amountIn must be positive")
	}
	structHash, err := swapStructHash(assetIn, assetOut, amountIn, minAmountOut, maxFee, nonce)
	if err != nil {
		return [32]byte{}, err
	}
	var digest [32]byte
	copy(digest[:], Digest(ComputeDomainSeparator(chainID), structHash))
	return digest, nil
}

// WithdrawalDigest hashes both the Yellow deployment domain and the signed
// destination L1 chain ID. They are different values and must not be conflated.
func WithdrawalDigest(chainID *big.Int, asset string, amount *big.Int, targetChainID uint64, recipient common.Address, maxFee *big.Int, nonce uint64) ([32]byte, error) {
	if chainID == nil || chainID.Sign() <= 0 || chainID.BitLen() > 256 || asset == "" || targetChainID == 0 || recipient == (common.Address{}) {
		return [32]byte{}, fmt.Errorf("invalid withdrawal domain, asset, target chain or recipient")
	}
	if err := signedUint256("amount", amount); err != nil {
		return [32]byte{}, err
	}
	if err := signedUint256("maxFee", maxFee); err != nil {
		return [32]byte{}, err
	}
	if amount.Sign() == 0 {
		return [32]byte{}, fmt.Errorf("withdrawal amount must be positive")
	}
	structHash, err := withdrawalStructHash(asset, amount, targetChainID, recipient, maxFee, nonce)
	if err != nil {
		return [32]byte{}, err
	}
	var digest [32]byte
	copy(digest[:], Digest(ComputeDomainSeparator(chainID), structHash))
	return digest, nil
}

// SignSwap and SignWithdrawal return canonical 65-byte Ethereum signatures
// with V in {27,28}, matching EIP-712 browser-wallet signatures.
func SignSwap(ctx context.Context, signer sign.Signer, chainID *big.Int, assetIn, assetOut string, amountIn, minAmountOut, maxFee *big.Int, nonce uint64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("swap signer required")
	}
	address, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	digest, err := SwapDigest(chainID, assetIn, assetOut, amountIn, minAmountOut, maxFee, nonce)
	if err != nil {
		return nil, err
	}
	sig, err := sign.SignEthDigest(ctx, signer, digest[:], address)
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}

func SignWithdrawal(ctx context.Context, signer sign.Signer, chainID *big.Int, asset string, amount *big.Int, targetChainID uint64, recipient common.Address, maxFee *big.Int, nonce uint64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("withdrawal signer required")
	}
	address, err := sign.EthAddress(signer)
	if err != nil {
		return nil, err
	}
	digest, err := WithdrawalDigest(chainID, asset, amount, targetChainID, recipient, maxFee, nonce)
	if err != nil {
		return nil, err
	}
	sig, err := sign.SignEthDigest(ctx, signer, digest[:], address)
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}

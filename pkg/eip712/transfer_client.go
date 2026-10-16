package eip712

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// TransferOperationDomain pins the additional, non-signing namespace used
// to identify one transfer across gateways. Changing it changes public IDs.
const TransferOperationDomain = "clearnet/transfer-operation/v1"

// ValidTransferOperationID checks the stable lowercase 0x-prefixed ID shape.
func ValidTransferOperationID(id string) bool {
	if len(id) != 66 || !strings.HasPrefix(id, "0x") || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id[2:])
	return err == nil
}

// TransferRecipientAddress maps a raw user address or canonical yellow URI
// into the EIP-712 address slot while leaving the original recipient unchanged
// in the routed request. Never silently turn malformed hex into address zero.
func TransferRecipientAddress(recipient string) (common.Address, error) {
	if core.IsURI(recipient) {
		if err := core.ValidateURI(recipient); err != nil {
			return common.Address{}, fmt.Errorf("transfer recipient: %w", err)
		}
		return core.URIToAddress(recipient)
	}
	if !common.IsHexAddress(recipient) {
		return common.Address{}, fmt.Errorf("transfer recipient must be an Ethereum address or canonical yellow URI")
	}
	addr := common.HexToAddress(recipient)
	if addr == (common.Address{}) {
		return common.Address{}, fmt.Errorf("transfer recipient must be nonzero")
	}
	return addr, nil
}

// TransferDigest is the frozen EIP-712 digest for a user-authorized transfer.
// maxFee is a required YELLOW-wei ceiling: zero authorizes no network fee.
func TransferDigest(chainID *big.Int, to common.Address, assets []TransferAsset, maxFee *big.Int, nonce uint64) ([32]byte, error) {
	if chainID == nil || chainID.Sign() <= 0 || chainID.BitLen() > 256 {
		return [32]byte{}, fmt.Errorf("transfer chain ID must be a positive uint256")
	}
	if maxFee == nil || maxFee.Sign() < 0 || maxFee.BitLen() > 256 {
		return [32]byte{}, fmt.Errorf("transfer maxFee must be a uint256 YELLOW-wei ceiling")
	}
	for i, asset := range assets {
		if asset.Amount == nil || asset.Amount.Sign() <= 0 || asset.Amount.BitLen() > 256 {
			return [32]byte{}, fmt.Errorf("transfer asset %d amount must be a positive uint256", i)
		}
	}
	structHash, err := TransferStructHash(to, assets, maxFee, nonce)
	if err != nil {
		return [32]byte{}, err
	}
	var digest [32]byte
	copy(digest[:], Digest(ComputeDomainSeparator(chainID), structHash))
	return digest, nil
}

// TransferOperationID identifies a signed transfer independently of the
// gateway process, HTTP request, signature recovery-ID encoding, or wall clock.
func TransferOperationID(chainID *big.Int, signer common.Address, recipient string, assets []TransferAsset, maxFee *big.Int, nonce uint64) (string, error) {
	to, err := TransferRecipientAddress(recipient)
	if err != nil {
		return "", err
	}
	digest, err := TransferDigest(chainID, to, assets, maxFee, nonce)
	if err != nil {
		return "", err
	}
	preimage := make([]byte, 0, len(TransferOperationDomain)+len(signer.Bytes())+len(digest))
	preimage = append(preimage, TransferOperationDomain...)
	preimage = append(preimage, signer.Bytes()...)
	preimage = append(preimage, digest[:]...)
	return "0x" + hex.EncodeToString(crypto.Keccak256(preimage)), nil
}

// SignTransfer signs once and returns the canonical, gateway-verifiable RSV
// signature and its operation ID. The caller must retain the same signed body
// after an uncertain submission rather than signing at a fresh nonce.
func SignTransfer(ctx context.Context, signer sign.Signer, chainID *big.Int, recipient string, assets []TransferAsset, maxFee *big.Int, nonce uint64) ([]byte, string, error) {
	if signer == nil {
		return nil, "", fmt.Errorf("transfer signer is required")
	}
	address, err := sign.EthAddress(signer)
	if err != nil {
		return nil, "", err
	}
	to, err := TransferRecipientAddress(recipient)
	if err != nil {
		return nil, "", err
	}
	digest, err := TransferDigest(chainID, to, assets, maxFee, nonce)
	if err != nil {
		return nil, "", err
	}
	id, err := TransferOperationID(chainID, address, recipient, assets, maxFee, nonce)
	if err != nil {
		return nil, "", err
	}
	sig, err := sign.SignEthDigest(ctx, signer, digest[:], address)
	if err != nil {
		return nil, "", err
	}
	// EIP-712 wallet signatures use V in {27,28}; the gateway accepts both
	// forms. Return one stable wire representation for independent clients.
	sig[64] += 27
	return sig, id, nil
}

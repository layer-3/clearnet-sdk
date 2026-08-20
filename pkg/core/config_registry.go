package core

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ConfigRegistryEvent is a normalized ConfigRegistry config event after chain
// confirmation. HasData distinguishes payload-carrying writes from
// checksum-only writes; consumers decide which issuers and keys they care about.
type ConfigRegistryEvent struct {
	Registry common.Address
	IssuerID common.Address
	Key      [32]byte
	Checksum [32]byte
	Data     []byte
	HasData  bool
	NewNonce *big.Int

	BlockNumber uint64
	LogIndex    uint
	TxHash      common.Hash
}

// ConfigRegistryCursor identifies the last ConfigRegistry log durably handled
// by a downstream store. Ordering is by (BlockNumber, LogIndex); TxHash is
// retained for audit/debug and idempotence.
type ConfigRegistryCursor struct {
	Registry    common.Address
	BlockNumber uint64
	LogIndex    uint
	TxHash      common.Hash
}

// ReceiptSignerSet is the current signer directory used to verify issuer
// receipts.
type ReceiptSignerSet struct {
	Signers   []common.Address
	Threshold int
}

// ReceiptSignerSource resolves receipt signers for a ConfigRegistry issuer.
type ReceiptSignerSource interface {
	LoadReceiptSigners(ctx context.Context, issuerID common.Address) (ReceiptSignerSet, error)
}

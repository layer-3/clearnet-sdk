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
	Epoch    uint64
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

// ReceiptSignerState is one consistent latest KEY_SIGNERS view used to prepare
// or verify issuer receipts. Epoch, signers, and threshold must come from the
// same resolved registry state.
type ReceiptSignerState struct {
	Epoch     uint64
	Signers   []common.Address
	Threshold int
}

// ReceiptSignerSource resolves latest receipt signer state for a ConfigRegistry issuer.
// Dynamic implementations backed by watchers or remote stores must fail closed
// when they cannot prove their data is fresh enough for their integration's
// safety policy. Implementations must return a positive threshold, distinct
// non-zero signer addresses, and at least threshold signers.
type ReceiptSignerSource interface {
	LoadLatestReceiptSignerState(ctx context.Context, issuerID common.Address) (ReceiptSignerState, error)
}

// Package main is the cbor-gen driver for pkg/core.
//
// It emits pkg/core/cbor_gen.go with struct-as-array (tuple) CBOR codecs
// for every exported wire type in the package, per ADR-009 §2
// (struct-as-array + append-only evolution). Ported from
// clearnet/clearing/core/gen, trimmed to the types that moved into the SDK.
//
// Invocation
//
//	go generate ./pkg/core/...
//
// The wrapper //go:generate directive lives in pkg/core/generate.go
// so the generator picks it up through the package root. Never call this
// main directly from tests or application code; it is a build-time tool.
//
// Evolution rules (ADR-009 §2):
//   - Field order in a target struct IS the wire encoding. Append only;
//     never insert, reorder, rename, or remove.
//   - When adding a new target type, add it at the end of the tuple list
//     below. The order here is not wire-significant (each struct is
//     independent), but stable ordering keeps generated-file diffs
//     minimal.
//   - Types with non-empty embedded or nested struct fields (e.g.
//     BurnReceipt embeds BlockEntryRef) serialize the nested struct in
//     its declared position; the nested type itself must also be a codec
//     target so cbor-gen can emit the nested MarshalCBOR call.
package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

func main() {
	// The //go:generate directive lives in pkg/core/generate.go, so
	// `go generate` runs this main with cwd = pkg/core/. That is why
	// the output path is "cbor_gen.go" (relative to cwd).
	if err := cbg.WriteTupleEncodersToFile(
		"cbor_gen.go",
		"core",
		// --- Block header / body ---
		core.BlockHeader{},
		core.Block{},
		core.BlockEntry{},
		core.AccountSnapshot{},
		core.Attestation{},
		// --- Transactions / ops ---
		core.AssetTransfer{},
		core.TransferOp{},
		core.SwapOp{},
		core.WithdrawalOp{},
		core.RepegOp{},
		core.SessionCloseOp{},
		core.SessionChallengeOp{},
		// --- Cross-cluster wire types ---
		core.BlockEntryRef{},
		core.BurnReceipt{},
		core.MintReceipt{},
		core.FinalizedWithdrawal{},
		core.FinalizedWithdrawalHeader{},
	); err != nil {
		panic(err)
	}
}

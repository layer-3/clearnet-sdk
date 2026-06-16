// Package sol implements the chain-agnostic adapter interfaces (see pkg/core)
// against Solana, over the custody Anchor program. The program bindings in
// ./custody are generated; the depositor/withdrawal-finalizer adapters and the
// Ed25519-precompile / digest helpers are hand-written on top of them.
package sol

// Regenerate the ./custody program bindings from the vendored Anchor IDL in
// ./artifacts (see ./artifacts/README.md to refresh the IDL itself).
//go:generate go run ./idl_refresher

// This file exists solely to anchor the `go generate` directive that
// produces pkg/core/cbor_gen.go. The generator is the main package
// at pkg/core/gen/; it reads the type list there and emits tuple-mode
// CBOR codecs per ADR-009 §2.
//
// Re-run whenever any target type's fields change:
//
//	go generate ./pkg/core/...
package core

//go:generate go run ./gen

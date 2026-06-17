// Package abiutil provides shared ABI type singletons parsed once at init.
// Seven packages previously declared their own abi.NewType("uint256", ...) etc.
// at package scope, silently discarding the error return 30+ times. This
// package centralises those declarations behind a must() wrapper.
package abiutil

import "github.com/ethereum/go-ethereum/accounts/abi"

// Scalar types used across the protocol.
var (
	Uint256 abi.Type
	Uint64  abi.Type
	Uint16  abi.Type
	Uint8   abi.Type
	Int64   abi.Type
	Bytes32 abi.Type
	Address abi.Type
	String  abi.Type
	Bool    abi.Type
	Bytes   abi.Type

	// Fixed-size array types used by the BLS signature ABI encoding.
	Uint256Arr2 abi.Type
	Uint256Arr4 abi.Type

	// Dynamic array type used by the signer-rotation digest.
	AddressArr abi.Type
)

func init() {
	Uint256 = must("uint256")
	Uint64 = must("uint64")
	Uint16 = must("uint16")
	Uint8 = must("uint8")
	Int64 = must("int64")
	Bytes32 = must("bytes32")
	Address = must("address")
	String = must("string")
	Bool = must("bool")
	Bytes = must("bytes")
	Uint256Arr2 = must("uint256[2]")
	Uint256Arr4 = must("uint256[4]")
	AddressArr = must("address[]")
}

func must(t string) abi.Type {
	ty, err := abi.NewType(t, "", nil)
	if err != nil {
		panic("abiutil: bad ABI type " + t + ": " + err.Error())
	}
	return ty
}

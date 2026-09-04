package marker

import "crypto/sha256"

// GenericDepositTagPreimage is the fixed string whose SHA256 is the tag on the
// single generic deposit script. It is a compile-time constant and is deliberately
// not configurable as it feeds the address every provider watches.
const GenericDepositTagPreimage = "yellow/btc/generic-deposit/v1"

// GenericDepositTagHex is SHA256(GenericDepositTagPreimage), pinned as a literal
// so a typo in the preimage cannot silently change the tag.
const GenericDepositTagHex = "791019971cfb071da60191897acda17879f6866754c6bdbee1a312a4ee33a8ac"

// GenericDepositTag returns the 32-byte tag.
func GenericDepositTag() [32]byte { return sha256.Sum256([]byte(GenericDepositTagPreimage)) }

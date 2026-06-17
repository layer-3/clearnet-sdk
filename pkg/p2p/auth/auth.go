// Package auth implements the /ynp/auth/1.0.0 libp2p handshake that lets a node
// prove who it is before a peer accepts its restricted streams (notably the
// burn/mint receipt protocols).
//
// The handshake is a single round trip: the Server sends a random nonce
// (AuthChallenge); the Client signs it and returns an AuthResponse. Two roles
// share the wire:
//
//   - Operator — the client signs keccak256(nonce) with a secp256k1 operator
//     key and sets Address. The server ecrecovers, matches the recovered
//     address to the claimed one, and checks it against an operator allow-list.
//     Proves the peer holds a key on the signer set; the gate for receipt
//     streams.
//
//   - Passive — the client leaves Address empty and signs a domain-separated
//     nonce with its libp2p identity key. The server verifies against the
//     connection's remote public key. Proves the peer controls the libp2p
//     identity it dials from — no operator key, no allow-list — for
//     gateway-safe / read paths.
//
// The package owns no host.Host: Server.Register installs a handler on a
// caller-built host (Server satisfies protocol.Registrar), and Client dials
// over one.
package auth

// maxAuthEnvelope caps a single challenge/response envelope read.
const maxAuthEnvelope = 64 << 10 // 64 KiB

// passiveAuthDomain separates passive-auth signatures from any other use of the
// libp2p identity key. It is part of the wire contract — both sides must agree
// on these exact bytes.
var passiveAuthDomain = []byte("ynp/libp2p-passive-auth/v1")

// Role is the outcome class of a successful handshake.
type Role uint8

const (
	// RolePassive means the peer proved control of its libp2p identity only.
	RolePassive Role = 1
	// RoleOperator means the peer proved control of an allow-listed operator key.
	RoleOperator Role = 2
)

func (r Role) String() string {
	switch r {
	case RolePassive:
		return "passive"
	case RoleOperator:
		return "operator"
	default:
		return "unknown"
	}
}

// Result holds the outcome of a successful authentication.
type Result struct {
	// Address is the recovered 0x-prefixed operator address; empty for passive.
	Address string
	Role    Role
}

func passiveAuthMessage(nonce [32]byte) []byte {
	msg := make([]byte, 0, len(passiveAuthDomain)+len(nonce))
	msg = append(msg, passiveAuthDomain...)
	msg = append(msg, nonce[:]...)
	return msg
}

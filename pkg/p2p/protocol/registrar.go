package protocol

import "github.com/libp2p/go-libp2p/core/host"

// Registrar is implemented by every p2p server: it installs that server's
// stream handlers on a caller-owned host. Keeping the contract here lets the
// wiring layer treat auth/receipt/… servers uniformly — register a slice of
// Registrars against one host without knowing their concrete types.
//
// GossipSub helpers (publish/subscribe over topics) deliberately do NOT
// implement Registrar: they register no stream handlers, which is the line
// between a stream protocol and a broadcast topic.
type Registrar interface {
	// Register installs the server's stream handlers on h. The caller owns h
	// and its lifecycle; Register only attaches handlers.
	Register(h host.Host)
}

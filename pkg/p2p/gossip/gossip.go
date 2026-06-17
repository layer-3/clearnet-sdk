// Package gossip is a generic GossipSub publish/subscribe toolset over any
// cborx-envelope payload. It is the type-parameterized alternative to the
// concrete pkg/p2p/pubsub (which bakes in *core.FinalizedWithdrawal): a
// Publisher[T]/Follower[T] works for any T whose cbor-gen codec lives on *T.
//
// Both ship side by side so the design can be chosen at review — generic reuse
// here vs. a concrete, no-generics surface in pubsub. Like pubsub, gossip is
// host-taking: the caller builds and owns the libp2p host and its connectivity;
// gossip owns only the GossipSub instance, topic, and subscription.
//
// Usage (constraint type inference fills the pointer type, so callers name only
// the value type):
//
//	pub, _ := gossip.NewPublisher[core.FinalizedWithdrawal](ctx, h, topic, nil)
//	_ = pub.Publish(ctx, &core.FinalizedWithdrawal{...})
//
//	fol, _ := gossip.NewFollower[core.FinalizedWithdrawal](ctx, h, topic, nil)
//	fol.SetHandler(func(fw *core.FinalizedWithdrawal) { ... })
//	go fol.Run(ctx)
package gossip

import (
	cbg "github.com/whyrusleeping/cbor-gen"
)

// maxMessageBytes caps the raw size of an inbound message before any CBOR
// allocation, keeping a malicious publisher from forcing large allocations per
// message. Matches the concrete pubsub follower's cap.
const maxMessageBytes = 128 * 1024

// message constrains *T to the cborx codec interfaces. cbor-gen emits
// Marshal/Unmarshal on the pointer receiver, so the constraint is expressed on
// *T. Its single type term (*T) gives the constraint a core type, which lets
// constraint type inference deduce M from T alone — callers write
// NewPublisher[T](...) without naming the pointer type.
type message[T any] interface {
	*T
	cbg.CBORMarshaler
	cbg.CBORUnmarshaler
}

// Handler receives each decoded payload. The Follower calls it synchronously on
// the consume goroutine; a slow handler backs up incoming messages.
type Handler[T any] func(v *T)

// Package pubsub is a generic GossipSub publish/subscribe toolset over any
// cborx-envelope payload: a Publisher[T]/Follower[T] works for any T whose
// cbor-gen codec lives on *T (e.g. *core.FinalizedWithdrawal).
//
// It is host-taking: the caller builds and owns the libp2p host and its
// connectivity; this package owns only the GossipSub instance, topic, and
// subscription.
//
// Usage (constraint type inference fills the pointer type, so callers name only
// the value type):
//
//	pub, _ := pubsub.NewPublisher[core.FinalizedWithdrawal](ctx, h, topic, nil)
//	_ = pub.Publish(ctx, &core.FinalizedWithdrawal{...})
//
//	fol, _ := pubsub.NewFollower[core.FinalizedWithdrawal](ctx, h, topic, nil)
//	fol.SetHandler(func(fw *core.FinalizedWithdrawal) { ... })
//	go fol.Run(ctx)
package pubsub

import (
	cbg "github.com/whyrusleeping/cbor-gen"
)

// maxMessageBytes caps the raw size of an inbound message before any CBOR
// allocation, keeping a malicious publisher from forcing large allocations per
// message.
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

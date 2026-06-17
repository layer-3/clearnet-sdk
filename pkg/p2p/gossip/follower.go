package gossip

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/log"
)

// Metrics captures Follower counters for ops dashboards.
type Metrics struct {
	mu            sync.Mutex
	Delivered     uint64 // payloads handed to the handler
	DecodeErrors  uint64 // envelope/CBOR decode failures
	OversizeDrops uint64 // dropped for exceeding the size cap
}

// Snapshot returns a copy of the current counters. Safe for concurrent use.
func (m *Metrics) Snapshot() Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Metrics{Delivered: m.Delivered, DecodeErrors: m.DecodeErrors, OversizeDrops: m.OversizeDrops}
}

func (m *Metrics) inc(field *uint64) {
	m.mu.Lock()
	*field++
	m.mu.Unlock()
}

// Follower subscribes to a topic on a caller-owned host and forwards each
// decoded value of type T to a Handler.
type Follower[T any, M message[T]] struct {
	host    host.Host
	sub     *pubsub.Subscription
	topic   *pubsub.Topic
	name    string
	logger  log.Logger
	metrics *Metrics

	handlerMu sync.RWMutex
	handler   Handler[T]
}

// NewFollower joins and subscribes to topic on h. A size-validator is attached
// before joining, so oversized messages are rejected by GossipSub and never
// reach the consume loop. Call Run to start consuming; register a handler with
// SetHandler before (or shortly after) Run. The caller owns h and its
// connectivity.
func NewFollower[T any, M message[T]](ctx context.Context, h host.Host, topic string, logger log.Logger) (*Follower[T, M], error) {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("gossipsub: %w", err)
	}
	f := &Follower[T, M]{
		host:    h,
		name:    topic,
		logger:  logger.WithName("p2p-gossip-follower").WithKV("topic", topic),
		metrics: &Metrics{},
	}
	if err := ps.RegisterTopicValidator(topic, f.validateSize); err != nil {
		return nil, fmt.Errorf("register validator %s: %w", topic, err)
	}
	t, err := ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("join %s: %w", topic, err)
	}
	sub, err := t.Subscribe()
	if err != nil {
		_ = t.Close()
		return nil, fmt.Errorf("subscribe %s: %w", topic, err)
	}
	f.topic = t
	f.sub = sub
	f.logger.Info("gossip follower started", "peer_id", h.ID().String())
	return f, nil
}

// SetHandler installs the handler invoked for each decoded value. Safe to call
// concurrently with Run.
func (f *Follower[T, M]) SetHandler(h Handler[T]) {
	f.handlerMu.Lock()
	f.handler = h
	f.handlerMu.Unlock()
}

// Metrics returns the Follower's counters handle.
func (f *Follower[T, M]) Metrics() *Metrics { return f.metrics }

// PeerID returns the host's libp2p peer ID.
func (f *Follower[T, M]) PeerID() peer.ID { return f.host.ID() }

// Run consumes the subscription until ctx is cancelled or the subscription
// closes. It blocks; run it in a goroutine.
func (f *Follower[T, M]) Run(ctx context.Context) {
	for {
		msg, err := f.sub.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				f.logger.Debug("subscription closed", "error", err)
			}
			return
		}
		if msg.ReceivedFrom == f.host.ID() {
			continue
		}
		f.handle(msg)
	}
}

// Close cancels the subscription and leaves the topic. It does not close the
// host — the caller owns that.
func (f *Follower[T, M]) Close() error {
	f.sub.Cancel()
	return f.topic.Close()
}

func (f *Follower[T, M]) validateSize(_ context.Context, from peer.ID, msg *pubsub.Message) bool {
	if len(msg.Data) > maxMessageBytes {
		f.metrics.inc(&f.metrics.OversizeDrops)
		f.logger.Warn("dropping oversize message", "from", from.ShortString(), "bytes", len(msg.Data))
		return false
	}
	return true
}

func (f *Follower[T, M]) handle(msg *pubsub.Message) {
	var v T
	var ver cborx.Version
	// M(&v) converts *T to the constrained pointer type, which satisfies
	// cborx's CBORUnmarshaler — decode in place into v.
	if err := cborx.ReadEnvelopeStrict(bytes.NewReader(msg.Data), &ver, M(&v)); err != nil {
		f.metrics.inc(&f.metrics.DecodeErrors)
		f.logger.Warn("decode payload failed", "error", err)
		return
	}
	f.handlerMu.RLock()
	h := f.handler
	f.handlerMu.RUnlock()
	if h == nil {
		f.logger.Warn("no handler registered; dropping payload")
		return
	}
	h(&v)
	f.metrics.inc(&f.metrics.Delivered)
}

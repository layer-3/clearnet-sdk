package pubsub

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
)

// maxFinalizedWithdrawalBytes caps the raw size of an inbound message before
// any CBOR allocation. A realistic envelope is a few KB (one Block + the BLS
// aggregate); 128 KiB leaves headroom for the largest plausible Block while
// keeping a malicious publisher from forcing megabyte allocations per message.
const maxFinalizedWithdrawalBytes = 128 * 1024

// WithdrawalHandler receives each FinalizedWithdrawal decoded from the topic.
// The Follower calls it synchronously on the consume goroutine; a slow handler
// backs up incoming messages.
type WithdrawalHandler func(fw *core.FinalizedWithdrawal)

// Metrics captures Follower counters for ops dashboards.
type Metrics struct {
	mu                   sync.Mutex
	DeliveredWithdrawals uint64 // handed to the handler
	DecodeErrors         uint64 // envelope/CBOR decode failures
	OversizeDrops        uint64 // dropped for exceeding the size cap
}

// Snapshot returns a copy of the current counters. Safe for concurrent use.
func (m *Metrics) Snapshot() Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Metrics{
		DeliveredWithdrawals: m.DeliveredWithdrawals,
		DecodeErrors:         m.DecodeErrors,
		OversizeDrops:        m.OversizeDrops,
	}
}

func (m *Metrics) inc(field *uint64) {
	m.mu.Lock()
	*field++
	m.mu.Unlock()
}

// Follower subscribes to a topic on a caller-owned host and forwards each
// decoded FinalizedWithdrawal to a handler.
type Follower struct {
	host    host.Host
	sub     *pubsub.Subscription
	topic   *pubsub.Topic
	name    string
	logger  log.Logger
	metrics *Metrics

	handlerMu sync.RWMutex
	handler   WithdrawalHandler
}

// NewFollower joins and subscribes to topic on h. A size-validator is attached
// before joining, so oversized messages are rejected by GossipSub and never
// reach the consume loop. Call Run to start consuming; register a handler with
// SetHandler before (or shortly after) Run — messages arriving with no handler
// are dropped with a warning. The caller owns h and its connectivity.
func NewFollower(ctx context.Context, h host.Host, topic string, logger log.Logger) (*Follower, error) {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("gossipsub: %w", err)
	}
	f := &Follower{
		host:    h,
		name:    topic,
		logger:  logger.WithName("p2p-pubsub-follower").WithKV("topic", topic),
		metrics: &Metrics{},
	}
	// Register the validator before Join so it is attached from the first
	// message. Oversized messages are rejected by GossipSub itself.
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
	f.logger.Info("pubsub follower started", "peer_id", h.ID().String())
	return f, nil
}

// SetHandler installs the handler invoked for each decoded withdrawal. Safe to
// call concurrently with Run.
func (f *Follower) SetHandler(h WithdrawalHandler) {
	f.handlerMu.Lock()
	f.handler = h
	f.handlerMu.Unlock()
}

// Metrics returns the Follower's counters handle.
func (f *Follower) Metrics() *Metrics { return f.metrics }

// PeerID returns the host's libp2p peer ID.
func (f *Follower) PeerID() peer.ID { return f.host.ID() }

// Run consumes the subscription until ctx is cancelled or the subscription
// closes. It blocks; run it in a goroutine.
func (f *Follower) Run(ctx context.Context) {
	for {
		msg, err := f.sub.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				f.logger.Debug("subscription closed", "error", err)
			}
			return
		}
		// Skip our own messages.
		if msg.ReceivedFrom == f.host.ID() {
			continue
		}
		f.handle(msg)
	}
}

// Close cancels the subscription and leaves the topic. It does not close the
// host — the caller owns that.
func (f *Follower) Close() error {
	f.sub.Cancel()
	return f.topic.Close()
}

// validateSize is the GossipSub topic validator: it rejects oversized messages
// before they propagate or reach the consume loop.
func (f *Follower) validateSize(_ context.Context, from peer.ID, msg *pubsub.Message) bool {
	if len(msg.Data) > maxFinalizedWithdrawalBytes {
		f.metrics.inc(&f.metrics.OversizeDrops)
		f.logger.Warn("dropping oversize message", "from", from.ShortString(), "bytes", len(msg.Data))
		return false
	}
	return true
}

func (f *Follower) handle(msg *pubsub.Message) {
	var fw core.FinalizedWithdrawal
	var v cborx.Version
	if err := cborx.ReadEnvelopeStrict(bytes.NewReader(msg.Data), &v, &fw); err != nil {
		f.metrics.inc(&f.metrics.DecodeErrors)
		f.logger.Warn("decode finalized withdrawal failed", "error", err)
		return
	}
	f.handlerMu.RLock()
	h := f.handler
	f.handlerMu.RUnlock()
	if h == nil {
		f.logger.Warn("no handler registered; dropping withdrawal")
		return
	}
	h(&fw)
	f.metrics.inc(&f.metrics.DeliveredWithdrawals)
}

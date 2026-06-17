package gossip

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

// Publisher joins a GossipSub topic and publishes values of type T. It does not
// subscribe: GossipSub only propagates once at least one subscriber is in the
// mesh, so a publisher-only node relies on its peers subscribing.
type Publisher[T any, M message[T]] struct {
	topic  *pubsub.Topic
	name   string
	logger *slog.Logger
}

// NewPublisher joins topic on h. The caller owns h and must keep it alive for
// the Publisher's lifetime.
func NewPublisher[T any, M message[T]](ctx context.Context, h host.Host, topic string, logger *slog.Logger) (*Publisher[T, M], error) {
	if logger == nil {
		logger = slog.Default()
	}
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("gossipsub: %w", err)
	}
	t, err := ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("join %s: %w", topic, err)
	}
	return &Publisher[T, M]{
		topic:  t,
		name:   topic,
		logger: logger.With("component", "p2p-gossip-publisher", "topic", topic),
	}, nil
}

// Publish emits v on the topic using the cborx V1 envelope. v is *T.
func (p *Publisher[T, M]) Publish(ctx context.Context, v M) error {
	var buf bytes.Buffer
	if err := cborx.WriteEnvelope(&buf, cborx.V1, v); err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	return p.topic.Publish(ctx, buf.Bytes())
}

// Topic returns the joined topic name.
func (p *Publisher[T, M]) Topic() string { return p.name }

// WaitForPeers blocks until at least minPeers subscribers have joined the topic
// mesh, ctx is cancelled, or timeout elapses.
func (p *Publisher[T, M]) WaitForPeers(ctx context.Context, minPeers int, timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(p.topic.ListPeers()) >= minPeers {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			return fmt.Errorf("only %d/%d peers joined within %s", len(p.topic.ListPeers()), minPeers, timeout)
		case <-tick.C:
		}
	}
}

// Close leaves the topic. It does not close the host — the caller owns that.
func (p *Publisher[T, M]) Close() error { return p.topic.Close() }

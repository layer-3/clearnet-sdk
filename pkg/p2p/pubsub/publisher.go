// Package pubsub provides GossipSub publish/subscribe helpers for the clearing
// layer's broadcast topics. A Publisher joins a topic and emits typed payloads;
// a Follower subscribes and forwards decoded payloads to a handler.
//
// Both are host-taking: the caller builds and owns the libp2p host (identity,
// listen addresses, resource limits) and is responsible for connectivity
// (dialing seed peers, peer discovery). These helpers own only the GossipSub
// instance, the topic, and — for the Follower — the subscription. Close
// releases those, never the host.
package pubsub

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// Publisher joins a GossipSub topic and publishes typed payloads to it. It does
// not subscribe: GossipSub only propagates once at least one subscriber is in
// the mesh, so a publisher-only node relies on its peers subscribing.
type Publisher struct {
	host   host.Host
	topic  *pubsub.Topic
	name   string
	logger *slog.Logger
}

// NewPublisher joins topic on h and returns a Publisher. The caller owns h and
// must keep it alive for the Publisher's lifetime.
func NewPublisher(ctx context.Context, h host.Host, topic string, logger *slog.Logger) (*Publisher, error) {
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
	return &Publisher{
		host:   h,
		topic:  t,
		name:   topic,
		logger: logger.With("component", "p2p-pubsub-publisher", "topic", topic),
	}, nil
}

// PublishFinalizedWithdrawal emits a single FinalizedWithdrawal on the topic
// using the cborx V1 envelope.
func (p *Publisher) PublishFinalizedWithdrawal(ctx context.Context, fw *core.FinalizedWithdrawal) error {
	var buf bytes.Buffer
	if err := cborx.WriteEnvelope(&buf, cborx.V1, fw); err != nil {
		return fmt.Errorf("encode finalized withdrawal: %w", err)
	}
	return p.topic.Publish(ctx, buf.Bytes())
}

// Topic returns the joined topic name.
func (p *Publisher) Topic() string { return p.name }

// WaitForPeers blocks until at least minPeers subscribers have joined the topic
// mesh, ctx is cancelled, or timeout elapses. GossipSub forwards only once mesh
// links form; publishing before then silently drops, so a publisher-only node
// should wait before its first broadcast.
func (p *Publisher) WaitForPeers(ctx context.Context, minPeers int, timeout time.Duration) error {
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
func (p *Publisher) Close() error { return p.topic.Close() }

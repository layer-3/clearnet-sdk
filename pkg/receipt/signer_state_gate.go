package receipt

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type ReceiptSignerStateGate interface {
	CheckReceiptSignerStateReady(ctx context.Context) error
}

type ReceiptSignerStateSubsystem string

const (
	ReceiptSignerStateConfigRegistryWatcher ReceiptSignerStateSubsystem = "config_registry_watcher"
)

type ReceiptSignerStateOnlineTracker interface {
	MarkReceiptSignerStateSubsystemOnline(ctx context.Context, subsystem ReceiptSignerStateSubsystem) error
	MarkReceiptSignerStateSubsystemOffline(ctx context.Context, subsystem ReceiptSignerStateSubsystem, reason error) error
}

type ReceiptSignerStateNotReadyError struct {
	Subsystem ReceiptSignerStateSubsystem
	Reason    error
}

func (e *ReceiptSignerStateNotReadyError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Reason == nil {
		return fmt.Sprintf("receipt signer state subsystem %s is offline", e.Subsystem)
	}
	return fmt.Sprintf("receipt signer state subsystem %s is offline: %v", e.Subsystem, e.Reason)
}

func (e *ReceiptSignerStateNotReadyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Reason
}

type InMemoryReceiptSignerStateGate struct {
	mu         sync.RWMutex
	subsystems map[ReceiptSignerStateSubsystem]error
}

func NewInMemoryReceiptSignerStateGate(subsystems ...ReceiptSignerStateSubsystem) (*InMemoryReceiptSignerStateGate, error) {
	if len(subsystems) == 0 {
		subsystems = []ReceiptSignerStateSubsystem{ReceiptSignerStateConfigRegistryWatcher}
	}
	m := make(map[ReceiptSignerStateSubsystem]error, len(subsystems))
	starting := errors.New("not started")
	for _, subsystem := range subsystems {
		if subsystem == "" {
			return nil, errors.New("receipt signer state gate: empty subsystem")
		}
		if _, dup := m[subsystem]; dup {
			return nil, fmt.Errorf("receipt signer state gate: duplicate subsystem %s", subsystem)
		}
		m[subsystem] = starting
	}
	return &InMemoryReceiptSignerStateGate{subsystems: m}, nil
}

func (g *InMemoryReceiptSignerStateGate) CheckReceiptSignerStateReady(context.Context) error {
	if g == nil {
		return errors.New("receipt signer state gate not configured")
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	for subsystem, reason := range g.subsystems {
		if reason != nil {
			return &ReceiptSignerStateNotReadyError{Subsystem: subsystem, Reason: reason}
		}
	}
	return nil
}

func (g *InMemoryReceiptSignerStateGate) MarkReceiptSignerStateSubsystemOnline(_ context.Context, subsystem ReceiptSignerStateSubsystem) error {
	if g == nil {
		return errors.New("receipt signer state gate not configured")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.subsystems[subsystem]; !ok {
		return fmt.Errorf("receipt signer state gate: unknown subsystem %s", subsystem)
	}
	g.subsystems[subsystem] = nil
	return nil
}

func (g *InMemoryReceiptSignerStateGate) MarkReceiptSignerStateSubsystemOffline(_ context.Context, subsystem ReceiptSignerStateSubsystem, reason error) error {
	if g == nil {
		return errors.New("receipt signer state gate not configured")
	}
	if reason == nil {
		return errors.New("receipt signer state gate: offline reason is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.subsystems[subsystem]; !ok {
		return fmt.Errorf("receipt signer state gate: unknown subsystem %s", subsystem)
	}
	g.subsystems[subsystem] = reason
	return nil
}

type ConfigRegistryWatcherReceiptSignerStateOnlineTracker struct {
	Tracker ReceiptSignerStateOnlineTracker
}

func (t ConfigRegistryWatcherReceiptSignerStateOnlineTracker) MarkConfigRegistryWatcherOnline(ctx context.Context) error {
	if t.Tracker == nil {
		return errors.New("receipt signer state online tracker not configured")
	}
	return t.Tracker.MarkReceiptSignerStateSubsystemOnline(ctx, ReceiptSignerStateConfigRegistryWatcher)
}

func (t ConfigRegistryWatcherReceiptSignerStateOnlineTracker) MarkConfigRegistryWatcherOffline(ctx context.Context, reason error) error {
	if t.Tracker == nil {
		return errors.New("receipt signer state online tracker not configured")
	}
	return t.Tracker.MarkReceiptSignerStateSubsystemOffline(ctx, ReceiptSignerStateConfigRegistryWatcher, reason)
}

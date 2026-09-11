package receipt

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryReceiptSignerStateGate(t *testing.T) {
	gate, err := NewInMemoryReceiptSignerStateGate(ReceiptSignerStateConfigRegistryWatcher)
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.CheckReceiptSignerStateReady(context.Background()); err == nil {
		t.Fatal("new gate unexpectedly ready")
	}
	if err := gate.MarkReceiptSignerStateSubsystemOnline(context.Background(), ReceiptSignerStateConfigRegistryWatcher); err != nil {
		t.Fatal(err)
	}
	if err := gate.CheckReceiptSignerStateReady(context.Background()); err != nil {
		t.Fatalf("online gate not ready: %v", err)
	}
	down := errors.New("rpc down")
	if err := gate.MarkReceiptSignerStateSubsystemOffline(context.Background(), ReceiptSignerStateConfigRegistryWatcher, down); err != nil {
		t.Fatal(err)
	}
	err = gate.CheckReceiptSignerStateReady(context.Background())
	var notReady *ReceiptSignerStateNotReadyError
	if !errors.As(err, &notReady) || notReady.Subsystem != ReceiptSignerStateConfigRegistryWatcher || !errors.Is(err, down) {
		t.Fatalf("not-ready error = %v", err)
	}
	if err := gate.MarkReceiptSignerStateSubsystemOffline(context.Background(), ReceiptSignerStateConfigRegistryWatcher, nil); err == nil {
		t.Fatal("offline without reason unexpectedly succeeded")
	}
	if err := gate.MarkReceiptSignerStateSubsystemOnline(context.Background(), ReceiptSignerStateSubsystem("unknown")); err == nil {
		t.Fatal("unknown subsystem unexpectedly succeeded")
	}
}

func TestConfigRegistryWatcherReceiptSignerStateOnlineTracker(t *testing.T) {
	gate, err := NewInMemoryReceiptSignerStateGate(ReceiptSignerStateConfigRegistryWatcher)
	if err != nil {
		t.Fatal(err)
	}
	tracker := ConfigRegistryWatcherReceiptSignerStateOnlineTracker{Tracker: gate}
	if err := tracker.MarkConfigRegistryWatcherOnline(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := gate.CheckReceiptSignerStateReady(context.Background()); err != nil {
		t.Fatalf("gate not ready after watcher online: %v", err)
	}
	reason := errors.New("stopped")
	if err := tracker.MarkConfigRegistryWatcherOffline(context.Background(), reason); err != nil {
		t.Fatal(err)
	}
	if err := gate.CheckReceiptSignerStateReady(context.Background()); !errors.Is(err, reason) {
		t.Fatalf("gate error = %v, want %v", err, reason)
	}
}

func TestInMemoryReceiptSignerStateGateZeroValueFailsClosed(t *testing.T) {
	var gate InMemoryReceiptSignerStateGate
	if err := gate.CheckReceiptSignerStateReady(context.Background()); err == nil {
		t.Fatal("zero-value gate unexpectedly reported ready")
	}
}

func TestInMemoryReceiptSignerStateGateReportsOfflineSubsystemsDeterministically(t *testing.T) {
	first := ReceiptSignerStateSubsystem("first")
	second := ReceiptSignerStateSubsystem("second")
	gate, err := NewInMemoryReceiptSignerStateGate(first, second)
	if err != nil {
		t.Fatal(err)
	}
	firstErr := errors.New("first down")
	secondErr := errors.New("second down")
	if err := gate.MarkReceiptSignerStateSubsystemOffline(context.Background(), first, firstErr); err != nil {
		t.Fatal(err)
	}
	if err := gate.MarkReceiptSignerStateSubsystemOffline(context.Background(), second, secondErr); err != nil {
		t.Fatal(err)
	}
	err = gate.CheckReceiptSignerStateReady(context.Background())
	var notReady *ReceiptSignerStateNotReadyError
	if !errors.As(err, &notReady) || notReady.Subsystem != first || !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("not-ready error = %v, want ordered aggregate", err)
	}
}

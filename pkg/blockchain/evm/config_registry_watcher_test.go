package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

type fakeHead struct {
	head uint64
	err  error
}

func (f fakeHead) BlockNumber(context.Context) (uint64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.head, nil
}

type fakeConfigRegistryWatcherReader struct {
	committed []*ConfigRegistryConfigCommitted
	withData  []*ConfigRegistryConfigWithDataCommitted
	logs      map[common.Hash][]types.Log

	committedRanges []filterRange
	withDataRanges  []filterRange

	committedErr     error
	committedErrOnce error
	withDataErr      error
	logsErr          error
}

type filterRange struct {
	start uint64
	end   *uint64
}

func (f *fakeConfigRegistryWatcherReader) FilterConfigCommittedEvents(_ context.Context, from, to uint64) ([]*ConfigRegistryConfigCommitted, error) {
	if f.committedErrOnce != nil {
		err := f.committedErrOnce
		f.committedErrOnce = nil
		return nil, err
	}
	if f.committedErr != nil {
		return nil, f.committedErr
	}
	end := to
	f.committedRanges = append(f.committedRanges, filterRange{start: from, end: &end})
	var out []*ConfigRegistryConfigCommitted
	for _, ev := range f.committed {
		if ev != nil && ev.Raw.BlockNumber >= from && ev.Raw.BlockNumber <= to {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeConfigRegistryWatcherReader) FilterConfigWithDataCommittedEvents(_ context.Context, from, to uint64) ([]*ConfigRegistryConfigWithDataCommitted, error) {
	if f.withDataErr != nil {
		return nil, f.withDataErr
	}
	end := to
	f.withDataRanges = append(f.withDataRanges, filterRange{start: from, end: &end})
	var out []*ConfigRegistryConfigWithDataCommitted
	for _, ev := range f.withData {
		if ev != nil && ev.Raw.BlockNumber >= from && ev.Raw.BlockNumber <= to {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeConfigRegistryWatcherReader) TransactionLogs(_ context.Context, txHash common.Hash) ([]types.Log, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	return f.logs[txHash], nil
}

type fakeConfigRegistryCursorSource struct {
	cursor core.ConfigRegistryCursor
	ok     bool
	err    error
}

func (f fakeConfigRegistryCursorSource) LatestConfigRegistryCursor(context.Context, common.Address) (core.ConfigRegistryCursor, bool, error) {
	return f.cursor, f.ok, f.err
}

type captureConfigRegistryHandler struct {
	events []core.ConfigRegistryEvent
	err    error
}

func (h *captureConfigRegistryHandler) HandleConfigRegistryEvent(_ context.Context, ev core.ConfigRegistryEvent) error {
	if h.err != nil {
		return h.err
	}
	h.events = append(h.events, ev)
	return nil
}

type channelConfigRegistryHandler struct {
	events chan core.ConfigRegistryEvent
}

func (h channelConfigRegistryHandler) HandleConfigRegistryEvent(_ context.Context, ev core.ConfigRegistryEvent) error {
	h.events <- ev
	return nil
}

type captureConfigRegistryWatcherOnlineTracker struct {
	mu         sync.Mutex
	online     int
	offline    []error
	onlineErr  error
	offlineErr error
}

func (t *captureConfigRegistryWatcherOnlineTracker) MarkConfigRegistryWatcherOnline(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.onlineErr != nil {
		return t.onlineErr
	}
	t.online++
	return nil
}

func (t *captureConfigRegistryWatcherOnlineTracker) MarkConfigRegistryWatcherOffline(_ context.Context, reason error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.offlineErr != nil {
		return t.offlineErr
	}
	t.offline = append(t.offline, reason)
	return nil
}

func (t *captureConfigRegistryWatcherOnlineTracker) snapshot() (int, []error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	offline := append([]error(nil), t.offline...)
	return t.online, offline
}

func TestConfigRegistryWatcher_InitCursorFromCursor(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 200}, registry, &fakeConfigRegistryWatcherReader{}, 12, handler)
	if err != nil {
		t.Fatal(err)
	}
	cur := core.ConfigRegistryCursor{Registry: registry, BlockNumber: 123, LogIndex: 4, TxHash: common.HexToHash("0xabc")}
	w.SetCursorSource(fakeConfigRegistryCursorSource{cursor: cur, ok: true})
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if got, ok := w.Cursor(); !ok || got != cur {
		t.Fatalf("cursor: ok=%v got=%+v want=%+v", ok, got, cur)
	}
	if wm := w.Watermark(); wm != 123 {
		t.Fatalf("watermark: got %d want 123", wm)
	}
}

func TestConfigRegistryWatcher_InitCursorUsesLookbackWithoutCursor(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 200}, registry, &fakeConfigRegistryWatcherReader{}, 12, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(50)
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	// confirmed = 188; start = 188 - 50.
	if wm := w.Watermark(); wm != 138 {
		t.Fatalf("watermark: got %d want 138", wm)
	}
	if _, ok := w.Cursor(); ok {
		t.Fatal("cursor should not be set without persisted cursor")
	}
}

func TestConfigRegistryWatcher_WatchPollsImmediately(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	ev := committedEvent(10, 2, issuer, key, [32]byte{0xAA}, 1)
	reader := &fakeConfigRegistryWatcherReader{
		committed: []*ConfigRegistryConfigCommitted{ev},
	}
	reader.addConfigSetLog(issuer, ev, 7)
	handler := channelConfigRegistryHandler{events: make(chan core.ConfigRegistryEvent, 1)}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	w.SetPollInterval(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 1)
	go func() {
		errs <- w.Watch(ctx)
	}()

	select {
	case got := <-handler.events:
		if got.BlockNumber != ev.Raw.BlockNumber || got.LogIndex != ev.Raw.Index || got.Epoch != 7 {
			t.Fatalf("event = %+v, want block %d log %d epoch 7", got, ev.Raw.BlockNumber, ev.Raw.Index)
		}
	case <-time.After(time.Second):
		t.Fatal("Watch did not poll before first ticker")
	}

	cancel()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("Watch returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Watch did not stop after context cancellation")
	}
}

func TestConfigRegistryWatcher_OnlineTracker(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	handler := &captureConfigRegistryHandler{}
	tracker := &captureConfigRegistryWatcherOnlineTracker{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, &fakeConfigRegistryWatcherReader{}, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	w.SetPollInterval(time.Hour)
	w.SetOnlineTracker(tracker)

	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		errs <- w.Watch(ctx)
	}()
	deadline := time.After(time.Second)
	for {
		online, _ := tracker.snapshot()
		if online > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watcher did not mark online")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("Watch returned error: %v", err)
	}
	_, offline := tracker.snapshot()
	if len(offline) < 2 {
		t.Fatalf("offline marks = %d, want startup and stop", len(offline))
	}

	pollErr := errors.New("rpc down")
	tracker = &captureConfigRegistryWatcherOnlineTracker{}
	w, err = newConfigRegistryWatcher(fakeHead{head: 20}, registry, &fakeConfigRegistryWatcherReader{committedErr: pollErr}, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetPollInterval(time.Hour)
	w.SetOnlineTracker(tracker)
	ctx, cancel = context.WithCancel(context.Background())
	errs = make(chan error, 1)
	go func() {
		errs <- w.Watch(ctx)
	}()
	deadline = time.After(time.Second)
	for {
		_, offline = tracker.snapshot()
		if len(offline) >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watcher did not mark offline after poll failure")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("Watch returned error after poll failure: %v", err)
	}
	_, offline = tracker.snapshot()
	if !errors.Is(offline[1], pollErr) {
		t.Fatalf("poll offline reason = %v, want %v", offline[1], pollErr)
	}
}

func TestConfigRegistryWatcher_OnlineTrackerRecoversAfterTransientPollFailure(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	tracker := &captureConfigRegistryWatcherOnlineTracker{}
	reader := &fakeConfigRegistryWatcherReader{committedErrOnce: errors.New("temporary rpc failure")}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, &captureConfigRegistryHandler{})
	if err != nil {
		t.Fatal(err)
	}
	w.SetPollInterval(10 * time.Millisecond)
	w.SetOnlineTracker(tracker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 1)
	go func() { errs <- w.Watch(ctx) }()
	deadline := time.After(time.Second)
	for {
		online, offline := tracker.snapshot()
		if online > 0 && len(offline) >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("watcher did not recover: online=%d offline=%d", online, len(offline))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("Watch returned error: %v", err)
	}
}

func TestConfigRegistryWatcher_OnlineTrackerErrors(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	trackerErr := errors.New("tracker unavailable")
	tracker := &captureConfigRegistryWatcherOnlineTracker{offlineErr: trackerErr}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, &fakeConfigRegistryWatcherReader{}, 5, &captureConfigRegistryHandler{})
	if err != nil {
		t.Fatal(err)
	}
	w.SetOnlineTracker(tracker)
	if err := w.Watch(context.Background()); !errors.Is(err, trackerErr) {
		t.Fatalf("Watch error = %v, want %v", err, trackerErr)
	}

	tracker = &captureConfigRegistryWatcherOnlineTracker{onlineErr: trackerErr}
	w, err = newConfigRegistryWatcher(fakeHead{head: 20}, registry, &fakeConfigRegistryWatcherReader{}, 5, &captureConfigRegistryHandler{})
	if err != nil {
		t.Fatal(err)
	}
	w.SetOnlineTracker(tracker)
	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() { errs <- w.Watch(ctx) }()
	deadline := time.After(time.Second)
	for {
		_, offline := tracker.snapshot()
		if len(offline) > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watcher did not record tracker online failure")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("Watch returned error after online tracker failure: %v", err)
	}
}

func TestConfigRegistryWatcher_PollDeliversOrderedEventsAndSkipsCursorLog(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var keyA, keyB, keyC [32]byte
	keyA[31] = 0xA
	keyB[31] = 0xB
	keyC[31] = 0xC
	data := []byte("signers")
	reader := &fakeConfigRegistryWatcherReader{
		committed: []*ConfigRegistryConfigCommitted{
			committedEvent(10, 2, issuer, keyB, [32]byte{0xBB}, 2),
			committedEvent(10, 4, issuer, keyC, [32]byte{0xCC}, 4),
		},
		withData: []*ConfigRegistryConfigWithDataCommitted{
			withDataEvent(10, 3, issuer, keyA, data, 3),
		},
	}
	reader.addConfigSetLog(issuer, reader.committed[0], 20)
	reader.addConfigSetWithDataLog(issuer, reader.withData[0], 21)
	reader.addConfigSetLog(issuer, reader.committed[1], 22)
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetCursorSource(fakeConfigRegistryCursorSource{
		cursor: core.ConfigRegistryCursor{Registry: registry, BlockNumber: 10, LogIndex: 2, TxHash: common.HexToHash("0x2")},
		ok:     true,
	})
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce: %v", err)
	}
	if len(handler.events) != 2 {
		t.Fatalf("events: got %d want 2", len(handler.events))
	}
	if handler.events[0].LogIndex != 3 || !handler.events[0].HasData {
		t.Fatalf("event[0]: got log=%d hasData=%v", handler.events[0].LogIndex, handler.events[0].HasData)
	}
	if handler.events[0].Epoch != 21 {
		t.Fatalf("event[0] epoch: got %d want 21", handler.events[0].Epoch)
	}
	if handler.events[1].LogIndex != 4 || handler.events[1].HasData {
		t.Fatalf("event[1]: got log=%d hasData=%v", handler.events[1].LogIndex, handler.events[1].HasData)
	}
	if handler.events[1].Epoch != 22 {
		t.Fatalf("event[1] epoch: got %d want 22", handler.events[1].Epoch)
	}
	if len(reader.committedRanges) != 1 || reader.committedRanges[0].start != 10 || *reader.committedRanges[0].end != 15 {
		t.Fatalf("committed range: %+v", reader.committedRanges)
	}
	if cur, ok := w.Cursor(); !ok || cur.BlockNumber != 10 || cur.LogIndex != 4 {
		t.Fatalf("cursor: ok=%v cur=%+v", ok, cur)
	}
	if wm := w.Watermark(); wm != 15 {
		t.Fatalf("watermark: got %d want 15", wm)
	}
}

func TestConfigRegistryWatcher_PollChunksRestoredCursorByLookback(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	ev := committedEvent(132, 1, issuer, key, [32]byte{0xAA}, 1)
	reader := &fakeConfigRegistryWatcherReader{
		committed: []*ConfigRegistryConfigCommitted{ev},
	}
	reader.addConfigSetLog(issuer, ev, 3)
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 140}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(10)
	w.SetCursorSource(fakeConfigRegistryCursorSource{
		cursor: core.ConfigRegistryCursor{Registry: registry, BlockNumber: 100, LogIndex: 0},
		ok:     true,
	})
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce: %v", err)
	}
	want := []filterRange{
		{start: 100, end: uint64ptr(109)},
		{start: 110, end: uint64ptr(119)},
		{start: 120, end: uint64ptr(129)},
		{start: 130, end: uint64ptr(135)},
	}
	if len(reader.committedRanges) != len(want) {
		t.Fatalf("committed ranges: got %d want %d (%+v)", len(reader.committedRanges), len(want), reader.committedRanges)
	}
	for i, got := range reader.committedRanges {
		if got.start != want[i].start || *got.end != *want[i].end {
			t.Fatalf("committed range[%d]: start=%d end=%d want start=%d end=%d", i, got.start, *got.end, want[i].start, *want[i].end)
		}
	}
	if len(handler.events) != 1 || handler.events[0].BlockNumber != 132 {
		t.Fatalf("events: %+v, want block 132", handler.events)
	}
	if wm := w.Watermark(); wm != 135 {
		t.Fatalf("watermark: got %d want 135", wm)
	}
}

func uint64ptr(v uint64) *uint64 { return &v }

func TestConfigRegistryWatcher_HandlerErrorDoesNotAdvancePastFailedEvent(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	reader := &fakeConfigRegistryWatcherReader{
		committed: []*ConfigRegistryConfigCommitted{committedEvent(10, 2, issuer, key, [32]byte{0xAA}, 1)},
	}
	reader.addConfigSetLog(issuer, reader.committed[0], 9)
	handler := &captureConfigRegistryHandler{err: fmt.Errorf("store failed")}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetCursorSource(fakeConfigRegistryCursorSource{
		cursor: core.ConfigRegistryCursor{Registry: registry, BlockNumber: 10, LogIndex: 1},
		ok:     true,
	})
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err == nil {
		t.Fatal("expected handler error")
	}
	if cur, ok := w.Cursor(); !ok || cur.BlockNumber != 10 || cur.LogIndex != 1 {
		t.Fatalf("cursor advanced after handler error: ok=%v cur=%+v", ok, cur)
	}
	if wm := w.Watermark(); wm != 10 {
		t.Fatalf("watermark advanced after handler error: got %d want 10", wm)
	}
}

func TestConfigRegistryWatcher_MissingConfigEpochEventFails(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	reader := &fakeConfigRegistryWatcherReader{
		committed: []*ConfigRegistryConfigCommitted{committedEvent(10, 2, issuer, key, [32]byte{0xAA}, 1)},
	}
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err == nil {
		t.Fatal("expected missing IConfig event error")
	}
	if len(handler.events) != 0 {
		t.Fatalf("handler received %d events despite missing epoch event", len(handler.events))
	}
}

func TestConfigRegistryWatcher_WrongDataConfigEpochEventFails(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	ev := withDataEvent(10, 2, issuer, key, []byte("signers"), 1)
	reader := &fakeConfigRegistryWatcherReader{withData: []*ConfigRegistryConfigWithDataCommitted{ev}}
	reader.addConfigSetWithDataLogFor(issuer, ev.Raw, ev.Key, ev.Checksum, []byte("different"), 1)
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err == nil {
		t.Fatal("expected wrong ConfigSetWithData payload to fail matching")
	}
	if len(handler.events) != 0 {
		t.Fatalf("handler received %d events despite wrong epoch event", len(handler.events))
	}
}

func TestConfigRegistryWatcher_EmptyWithDataPayloadResolvesEpoch(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	ev := withDataEvent(10, 2, issuer, key, nil, 1)
	reader := &fakeConfigRegistryWatcherReader{withData: []*ConfigRegistryConfigWithDataCommitted{ev}}
	reader.addConfigSetWithDataLog(issuer, ev, 11)
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce: %v", err)
	}
	if len(handler.events) != 1 {
		t.Fatalf("events: got %d want 1", len(handler.events))
	}
	if !handler.events[0].HasData || handler.events[0].Data != nil || handler.events[0].Epoch != 11 {
		t.Fatalf("event = %+v, want with-data empty payload at epoch 11", handler.events[0])
	}
}

func TestConfigRegistryWatcher_WithDataChecksumMismatch(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	var key [32]byte
	key[31] = 0xA
	ev := withDataEvent(10, 2, issuer, key, []byte("signers"), 1)
	ev.Checksum = [32]byte{0x99}
	reader := &fakeConfigRegistryWatcherReader{withData: []*ConfigRegistryConfigWithDataCommitted{ev}}
	handler := &captureConfigRegistryHandler{}
	w, err := newConfigRegistryWatcher(fakeHead{head: 20}, registry, reader, 5, handler)
	if err != nil {
		t.Fatal(err)
	}
	w.SetInitialLookback(20)
	if err := w.initCursor(context.Background()); err != nil {
		t.Fatalf("init cursor: %v", err)
	}
	if err := w.pollOnce(context.Background()); err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if len(handler.events) != 0 {
		t.Fatalf("handler received %d events despite checksum mismatch", len(handler.events))
	}
}

func committedEvent(block uint64, index uint, issuer common.Address, key [32]byte, checksum [32]byte, nonce int64) *ConfigRegistryConfigCommitted {
	return &ConfigRegistryConfigCommitted{
		IssuerId: issuer,
		Key:      key,
		Checksum: checksum,
		NewNonce: big.NewInt(nonce),
		Raw: types.Log{
			BlockNumber: block,
			Index:       index,
			TxHash:      common.BigToHash(big.NewInt(int64(index))),
		},
	}
}

func withDataEvent(block uint64, index uint, issuer common.Address, key [32]byte, data []byte, nonce int64) *ConfigRegistryConfigWithDataCommitted {
	return &ConfigRegistryConfigWithDataCommitted{
		IssuerId: issuer,
		Key:      key,
		Checksum: crypto.Keccak256Hash(data),
		Data:     data,
		NewNonce: big.NewInt(nonce),
		Raw: types.Log{
			BlockNumber: block,
			Index:       index,
			TxHash:      common.BigToHash(big.NewInt(int64(index))),
		},
	}
}

func (f *fakeConfigRegistryWatcherReader) addConfigSetLog(issuer common.Address, ev *ConfigRegistryConfigCommitted, epoch uint64) {
	f.addConfigSetLogFor(issuer, ev.Raw, ev.Key, ev.Checksum, epoch)
}

func (f *fakeConfigRegistryWatcherReader) addConfigSetLogFor(issuer common.Address, registryRaw types.Log, key [32]byte, checksum [32]byte, epoch uint64) {
	abi, err := IConfigMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	data, err := abi.Events["ConfigSet"].Inputs.NonIndexed().Pack(checksum, epoch)
	if err != nil {
		panic(err)
	}
	f.addTxLog(registryRaw.TxHash, types.Log{
		Address:     issuer,
		Topics:      []common.Hash{abi.Events["ConfigSet"].ID, common.BytesToHash(key[:]), common.BytesToHash(common.HexToAddress("0x000000000000000000000000000000000000c0de").Bytes())},
		Data:        data,
		BlockNumber: registryRaw.BlockNumber,
		TxHash:      registryRaw.TxHash,
		Index:       registryRaw.Index - 1,
	})
}

func (f *fakeConfigRegistryWatcherReader) addConfigSetWithDataLog(issuer common.Address, ev *ConfigRegistryConfigWithDataCommitted, epoch uint64) {
	f.addConfigSetWithDataLogFor(issuer, ev.Raw, ev.Key, ev.Checksum, ev.Data, epoch)
}

func (f *fakeConfigRegistryWatcherReader) addConfigSetWithDataLogFor(issuer common.Address, registryRaw types.Log, key [32]byte, checksum [32]byte, payload []byte, epoch uint64) {
	abi, err := IConfigMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	data, err := abi.Events["ConfigSetWithData"].Inputs.NonIndexed().Pack(checksum, epoch, payload)
	if err != nil {
		panic(err)
	}
	f.addTxLog(registryRaw.TxHash, types.Log{
		Address:     issuer,
		Topics:      []common.Hash{abi.Events["ConfigSetWithData"].ID, common.BytesToHash(key[:]), common.BytesToHash(common.HexToAddress("0x000000000000000000000000000000000000c0de").Bytes())},
		Data:        data,
		BlockNumber: registryRaw.BlockNumber,
		TxHash:      registryRaw.TxHash,
		Index:       registryRaw.Index - 1,
	})
}

func (f *fakeConfigRegistryWatcherReader) addTxLog(tx common.Hash, log types.Log) {
	if f.logs == nil {
		f.logs = make(map[common.Hash][]types.Log)
	}
	f.logs[tx] = append(f.logs[tx], log)
}

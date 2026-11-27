package positions

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type fakeGateway struct {
	mu    sync.Mutex
	calls []uuid.UUID
	err   error
	delay time.Duration
}

func (f *fakeGateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, tripID)
	f.mu.Unlock()
	return f.err
}

func (f *fakeGateway) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestWorker_ProcessesTasks(t *testing.T) {
	const n = 3
	q := NewQueue(10)
	gw := &fakeGateway{}
	w := NewWorker(newTestLogger(), q, gw, nil, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()

	for i := 0; i < n; i++ {
		q.TryEnqueue(model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()})
	}

	deadline := time.Now().Add(2 * time.Second)
	for gw.callCount() < n && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if gw.callCount() != n {
		t.Errorf("gateway calls = %d, want %d", gw.callCount(), n)
	}

	q.Close()
	<-done
}

func TestWorker_ErrorDoesNotStop(t *testing.T) {
	q := NewQueue(10)
	gw := &fakeGateway{err: errors.New("push down")}
	w := NewWorker(newTestLogger(), q, gw, nil, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()

	q.TryEnqueue(model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()})
	q.TryEnqueue(model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()})

	deadline := time.Now().Add(2 * time.Second)
	for gw.callCount() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if gw.callCount() != 2 {
		t.Errorf("gateway calls = %d, want 2 (error must not stop worker)", gw.callCount())
	}

	q.Close()
	<-done
}

func TestWorker_CancelSkipsPush(t *testing.T) {
	q := NewQueue(10)
	gw := &fakeGateway{}
	w := NewWorker(newTestLogger(), q, gw, nil, time.Second)

	id := uuid.New()
	q.TryEnqueue(model.StaleTrip{TripID: id, DriverID: uuid.New()})
	q.Cancel(id)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	if gw.callCount() != 0 {
		t.Errorf("gateway calls = %d, want 0 (task was cancelled)", gw.callCount())
	}

	q.Close()
	<-done
}

type countingGateway struct {
	inner     *fakeGateway
	active    *int32
	maxActive *int32
}

func (c *countingGateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	cur := atomic.AddInt32(c.active, 1)
	defer atomic.AddInt32(c.active, -1)

	for {
		old := atomic.LoadInt32(c.maxActive)
		if cur <= old || atomic.CompareAndSwapInt32(c.maxActive, old, cur) {
			break
		}
	}
	return c.inner.RequestPosition(ctx, driverID, tripID)
}

func TestWorker_PoolProcessesN(t *testing.T) {
	const n = 20
	const workers = 4
	q := NewQueue(n)
	gw := &fakeGateway{delay: 10 * time.Millisecond}

	var active, maxActive int32
	w := NewWorker(newTestLogger(), q, &countingGateway{inner: gw, active: &active, maxActive: &maxActive}, nil, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wgDone := Start(ctx, workers, w)

	for i := 0; i < n; i++ {
		q.TryEnqueue(model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()})
	}

	deadline := time.Now().Add(5 * time.Second)
	for gw.callCount() < n && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	q.Close()
	wgDone()

	if gw.callCount() != n {
		t.Errorf("gateway calls = %d, want %d", gw.callCount(), n)
	}
	if atomic.LoadInt32(&maxActive) > workers {
		t.Errorf("max active = %d, want <= %d", maxActive, workers)
	}
}

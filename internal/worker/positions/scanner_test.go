package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type fakeRepo struct {
	mu    sync.Mutex
	trips []model.StaleTrip
	err   error
	calls int
}

func (f *fakeRepo) FindStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]model.StaleTrip, len(f.trips))
	copy(out, f.trips)
	return out, nil
}

func (f *fakeRepo) setTrips(t []model.StaleTrip) {
	f.mu.Lock()
	f.trips = t
	f.mu.Unlock()
}

func (f *fakeRepo) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestScanner_TickDoesNotOverlap(t *testing.T) {
	repo := &fakeRepo{}
	repo.setTrips([]model.StaleTrip{{TripID: uuid.New(), DriverID: uuid.New()}})

	q := NewQueue(10)
	sc := NewScanner(newTestLogger(), repo, q, nil, 10*time.Millisecond, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Millisecond)
	defer cancel()

	sc.Run(ctx)

	if repo.callCount() == 0 {
		t.Error("scanner did not call repo.FindStale")
	}
}

func TestScanner_EnqueuesStaleTrips(t *testing.T) {
	repo := &fakeRepo{}
	id1, id2 := uuid.New(), uuid.New()
	repo.setTrips([]model.StaleTrip{
		{TripID: id1, DriverID: uuid.New()},
		{TripID: id2, DriverID: uuid.New()},
	})

	q := NewQueue(10)
	sc := NewScanner(newTestLogger(), repo, q, nil, time.Hour, time.Minute)

	sc.scan(context.Background())

	if q.Len() != 2 {
		t.Errorf("queue len = %d, want 2", q.Len())
	}
}

func TestScanner_FindStaleErrorDoesNotPanic(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db down")}
	q := NewQueue(10)
	sc := NewScanner(newTestLogger(), repo, q, nil, time.Hour, time.Minute)

	sc.scan(context.Background())

	if q.Len() != 0 {
		t.Errorf("queue len = %d, want 0", q.Len())
	}
}

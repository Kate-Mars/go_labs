package positions

import (
	"context"
	"sync"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type Queue struct {
	mu       sync.Mutex
	items    chan model.StaleTrip
	inflight map[uuid.UUID]struct{}
}

func NewQueue(size int) *Queue {
	return &Queue{
		items:    make(chan model.StaleTrip, size),
		inflight: make(map[uuid.UUID]struct{}, size),
	}
}

func (q *Queue) TryEnqueue(t model.StaleTrip) string {
	q.mu.Lock()
	if _, ok := q.inflight[t.TripID]; ok {
		q.mu.Unlock()
		return "duplicate"
	}
	select {
	case q.items <- t:
		q.inflight[t.TripID] = struct{}{}
		q.mu.Unlock()
		return "queued"
	default:
		q.mu.Unlock()
		return "dropped"
	}
}

func (q *Queue) Cancel(tripID uuid.UUID) bool {
	q.mu.Lock()
	_, existed := q.inflight[tripID]
	delete(q.inflight, tripID)
	q.mu.Unlock()
	return existed
}

func (q *Queue) Take(ctx context.Context) (model.StaleTrip, bool) {
	select {
	case t, ok := <-q.items:
		return t, ok
	case <-ctx.Done():
		return model.StaleTrip{}, false
	}
}

func (q *Queue) IsInflight(tripID uuid.UUID) bool {
	q.mu.Lock()
	_, ok := q.inflight[tripID]
	q.mu.Unlock()
	return ok
}

func (q *Queue) Done(tripID uuid.UUID) bool {
	q.mu.Lock()
	_, existed := q.inflight[tripID]
	delete(q.inflight, tripID)
	q.mu.Unlock()
	return existed
}

func (q *Queue) Len() int {
	return len(q.items)
}

func (q *Queue) Close() {
	close(q.items)
}

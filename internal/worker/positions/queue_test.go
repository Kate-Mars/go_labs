package positions

import (
	"context"
	"testing"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func TestQueue_Deduplication(t *testing.T) {
	q := NewQueue(10)
	id := uuid.New()
	task := model.StaleTrip{TripID: id}

	if got := q.TryEnqueue(task); got != "queued" {
		t.Errorf("first enqueue = %q, want queued", got)
	}
	if got := q.TryEnqueue(task); got != "duplicate" {
		t.Errorf("second enqueue = %q, want duplicate", got)
	}
}

func TestQueue_DroppedWhenFull(t *testing.T) {
	q := NewQueue(1)
	if got := q.TryEnqueue(model.StaleTrip{TripID: uuid.New()}); got != "queued" {
		t.Fatalf("first enqueue = %q, want queued", got)
	}
	if got := q.TryEnqueue(model.StaleTrip{TripID: uuid.New()}); got != "dropped" {
		t.Errorf("second enqueue = %q, want dropped", got)
	}
}

func TestQueue_Cancel(t *testing.T) {
	q := NewQueue(10)
	id := uuid.New()
	q.TryEnqueue(model.StaleTrip{TripID: id})

	if !q.IsInflight(id) {
		t.Fatal("expected inflight")
	}
	if !q.Cancel(id) {
		t.Error("Cancel = false, want true")
	}
	if q.IsInflight(id) {
		t.Error("still inflight after Cancel")
	}
	if q.Cancel(id) {
		t.Error("second Cancel = true, want false")
	}
}

func TestQueue_TakeRespectsContext(t *testing.T) {
	q := NewQueue(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, ok := q.Take(ctx); ok {
		t.Error("Take with cancelled ctx returned task")
	}
}

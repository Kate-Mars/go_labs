package positions

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func TestManager_StartStop(t *testing.T) {
	repo := &fakeRepo{}
	repo.setTrips([]model.StaleTrip{
		{TripID: uuid.New(), DriverID: uuid.New()},
		{TripID: uuid.New(), DriverID: uuid.New()},
	})
	gw := &fakeGateway{}

	m := NewManager(
		newTestLogger(),
		repo,
		gw,
		nil,
		20*time.Millisecond,
		time.Minute,
		2,
		10,
		time.Second,
	)

	ctx := context.Background()
	m.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for gw.callCount() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if gw.callCount() < 2 {
		t.Errorf("gateway calls = %d, want >= 2", gw.callCount())
	}
}

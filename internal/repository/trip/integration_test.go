//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres/transaction"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	repo "github.com/Kate-Mars/go_labs/internal/repository/trip"
)

func setup(t *testing.T) *repo.Repo {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	var exists bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'trips')`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check schema: %v", err)
	}
	if !exists {
		t.Fatalf("trips table not found — run `make migrate` first")
	}

	return repo.New(pool)
}

func newTrip(driver uuid.UUID) model.Trip {
	return model.Trip{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		DriverID:       driver,
		StartLatitude:  59.9,
		StartLongitude: 30.3,
		EndLatitude:    59.9,
		EndLongitude:   30.4,
		Price:          100,
		Status:         model.StatusActive,
		StartedAt:      time.Now().UTC(),
	}
}

func TestRepo_CreateAndGet(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	driver := uuid.New()
	tr := newTrip(driver)

	if err := r.Create(ctx, tr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByID(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != tr.ID {
		t.Errorf("ID = %v, want %v", got.ID, tr.ID)
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	r := setup(t)
	_, err := r.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, trip.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepo_DriverBusy(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	driver := uuid.New()

	if err := r.Create(ctx, newTrip(driver)); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	err := r.Create(ctx, newTrip(driver))
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Errorf("second Create err = %v, want ErrDriverBusy", err)
	}
}

func TestRepo_Finish(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	if err := r.Create(ctx, tr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := r.Finish(ctx, tr.ID, time.Now().UTC()); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	// повторный Finish → ErrAlreadyDone
	if err := r.Finish(ctx, tr.ID, time.Now().UTC()); !errors.Is(err, trip.ErrAlreadyDone) {
		t.Errorf("second Finish err = %v, want ErrAlreadyDone", err)
	}
}

func TestRepo_SavePosition_UpdatesLastPositionAt(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	if err := r.Create(ctx, tr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Truncate до микросекунд: PostgreSQL TIMESTAMPTZ хранит микросекунды,
	// time.Now() даёт наносекунды — без Truncate сравнение упадёт.
	older := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	newer := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := r.SavePosition(ctx, model.Position{
		TripID: tr.ID, Latitude: 59.9, Longitude: 30.3, RecordedAt: newer,
	}); err != nil {
		t.Fatalf("SavePosition newer: %v", err)
	}

	// старая координата не должна откатить last_position_at
	if _, err := r.SavePosition(ctx, model.Position{
		TripID: tr.ID, Latitude: 59.8, Longitude: 30.2, RecordedAt: older,
	}); err != nil {
		t.Fatalf("SavePosition older: %v", err)
	}

	got, err := r.GetByID(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastPositionAt == nil {
		t.Fatal("LastPositionAt = nil, want non-nil")
	}
	if !got.LastPositionAt.Equal(newer) {
		t.Errorf("LastPositionAt = %v, want %v", got.LastPositionAt, newer)
	}
}

func TestRepo_SavePosition_ListPositions(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	if err := r.Create(ctx, tr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	times := []time.Time{now.Add(-2 * time.Minute), now.Add(-time.Minute), now}
	for i, at := range times {
		if _, err := r.SavePosition(ctx, model.Position{
			TripID: tr.ID, Latitude: 59.9 + float64(i), Longitude: 30.3, RecordedAt: at,
		}); err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
	}

	got, err := r.ListPositions(ctx, tr.ID)
	if err != nil {
		t.Fatalf("ListPositions: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].RecordedAt.Before(got[i-1].RecordedAt) {
			t.Errorf("positions not sorted by recorded_at at %d", i)
		}
	}
}

func TestRepo_FindStale(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	// свежая поездка — не должна попасть в выборку
	fresh := newTrip(uuid.New())
	if err := r.Create(ctx, fresh); err != nil {
		t.Fatalf("Create fresh: %v", err)
	}
	if _, err := r.SavePosition(ctx, model.Position{
		TripID: fresh.ID, Latitude: 59.9, Longitude: 30.3, RecordedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SavePosition fresh: %v", err)
	}

	// старая поездка — должна попасть
	stale := newTrip(uuid.New())
	stale.StartedAt = time.Now().UTC().Add(-2 * time.Hour)
	if err := r.Create(ctx, stale); err != nil {
		t.Fatalf("Create stale: %v", err)
	}

	// завершённая — не должна
	done := newTrip(uuid.New())
	done.StartedAt = time.Now().UTC().Add(-2 * time.Hour)
	if err := r.Create(ctx, done); err != nil {
		t.Fatalf("Create done: %v", err)
	}
	if err := r.Finish(ctx, done.ID, time.Now().UTC()); err != nil {
		t.Fatalf("Finish done: %v", err)
	}

	stales, err := r.FindStale(ctx, time.Minute)
	if err != nil {
		t.Fatalf("FindStale: %v", err)
	}

	var foundStale bool
	for _, s := range stales {
		if s.TripID == stale.ID {
			foundStale = true
		}
		if s.TripID == fresh.ID {
			t.Errorf("fresh trip %v must not be in stale list", fresh.ID)
		}
		if s.TripID == done.ID {
			t.Errorf("completed trip %v must not be in stale list", done.ID)
		}
	}
	if !foundStale {
		t.Errorf("stale trip %v not found in FindStale result", stale.ID)
	}
}

func TestTxManager_Rollback_TwoInserts(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	r := repo.New(pool)
	txMgr := transaction.NewManager(pool)

	tr := newTrip(uuid.New())

	err = txMgr.Do(ctx, func(ctx context.Context) error {
		if err := r.Create(ctx, tr); err != nil {
			return err
		}
		if err := r.AppendStatusChange(ctx, model.StatusChange{
			TripID:   tr.ID,
			ToStatus: model.StatusActive,
			Reason:   "trip created",
		}); err != nil {
			return err
		}
		return errors.New("simulated failure after both inserts")
	})
	if err == nil {
		t.Fatal("expected error from Do")
	}

	_, getErr := r.GetByID(ctx, tr.ID)
	if !errors.Is(getErr, trip.ErrNotFound) {
		t.Errorf("after rollback: trips.GetByID err = %v, want ErrNotFound", getErr)
	}

	var histCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM trip_status_history WHERE trip_id = $1`, tr.ID,
	).Scan(&histCount); err != nil {
		t.Fatalf("count trip_status_history: %v", err)
	}
	if histCount != 0 {
		t.Errorf("after rollback: trip_status_history count = %d, want 0", histCount)
	}
}

func TestTxManager_Rollback_SecondInsertFails(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	r := repo.New(pool)
	txMgr := transaction.NewManager(pool)

	tr := newTrip(uuid.New())

	err = txMgr.Do(ctx, func(ctx context.Context) error {
		if err := r.Create(ctx, tr); err != nil {
			return err
		}
		return r.AppendStatusChange(ctx, model.StatusChange{
			TripID:   tr.ID,
			ToStatus: model.Status("bogus"),
			Reason:   "should fail",
		})
	})
	if err == nil {
		t.Fatal("expected error from Do (second insert must fail)")
	}

	_, getErr := r.GetByID(ctx, tr.ID)
	if !errors.Is(getErr, trip.ErrNotFound) {
		t.Errorf("after rollback: trips.GetByID err = %v, want ErrNotFound", getErr)
	}

	var histCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM trip_status_history WHERE trip_id = $1`, tr.ID,
	).Scan(&histCount); err != nil {
		t.Fatalf("count trip_status_history: %v", err)
	}
	if histCount != 0 {
		t.Errorf("after rollback: trip_status_history count = %d, want 0", histCount)
	}
}

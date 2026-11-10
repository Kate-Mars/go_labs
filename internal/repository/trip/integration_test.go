//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Kate-Mars/go_labs/internal/infra/postgres/transaction"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
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

func TestTxManager_Rollback(t *testing.T) {
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

	// внутри Do: Create успешен, потом ошибка → всё должно откатиться
	err = txMgr.Do(ctx, func(ctx context.Context) error {
		if err := r.Create(ctx, tr); err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	if err == nil {
		t.Fatal("expected error from Do")
	}

	// проверяем: поездки нет
	_, getErr := r.GetByID(ctx, tr.ID)
	if !errors.Is(getErr, trip.ErrNotFound) {
		t.Errorf("after rollback: GetByID err = %v, want ErrNotFound", getErr)
	}
}

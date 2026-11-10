//go:build integration

package idempotency_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	repo "github.com/Kate-Mars/go_labs/internal/repository/idempotency"
)

func setup(t *testing.T) (*pgxpool.Pool, *repo.Repo) {
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

	return pool, repo.New(pool)
}

func TestRepo_InsertIfAbsent_And_Get(t *testing.T) {
	pool, r := setup(t)
	ctx := context.Background()

	key := uuid.New()
	hash := []byte("hash1")

	inserted, err := r.InsertIfAbsent(ctx, model.IdempotencyRecord{
		Key:         key,
		RequestHash: hash,
	})
	if err != nil {
		t.Fatalf("InsertIfAbsent: %v", err)
	}
	if !inserted {
		t.Errorf("first InsertIfAbsent = false, want true")
	}

	inserted, err = r.InsertIfAbsent(ctx, model.IdempotencyRecord{
		Key:         key,
		RequestHash: []byte("hash-other"),
	})
	if err != nil {
		t.Fatalf("second InsertIfAbsent: %v", err)
	}
	if inserted {
		t.Errorf("second InsertIfAbsent = true, want false")
	}

	got, err := r.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if string(got.RequestHash) != "hash1" {
		t.Errorf("RequestHash = %q, want hash1", got.RequestHash)
	}
	if got.TripID != nil {
		t.Errorf("TripID = %v, want nil", got.TripID)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM idempotency_keys WHERE key = $1", key)
	})
}

func TestRepo_SetTripID(t *testing.T) {
	pool, r := setup(t)
	ctx := context.Background()

	key := uuid.New()
	if _, err := r.InsertIfAbsent(ctx, model.IdempotencyRecord{
		Key:         key,
		RequestHash: []byte("h"),
	}); err != nil {
		t.Fatalf("InsertIfAbsent: %v", err)
	}

	tripID := uuid.New()
	driverID := uuid.New()
	userID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO trips (
		id, user_id, driver_id,
		start_latitude, start_longitude, end_latitude, end_longitude,
		price, status, started_at
	) VALUES ($1,$2,$3,59.9,30.3,59.9,30.4,100,'active',now())`,
		tripID, userID, driverID,
	)
	if err != nil {
		t.Fatalf("seed trip: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM idempotency_keys WHERE key = $1", key)
		_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", tripID)
	})

	if err := r.SetTripID(ctx, key, tripID); err != nil {
		t.Fatalf("SetTripID: %v", err)
	}

	got, err := r.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TripID == nil || *got.TripID != tripID {
		t.Errorf("TripID = %v, want %v", got.TripID, tripID)
	}
}

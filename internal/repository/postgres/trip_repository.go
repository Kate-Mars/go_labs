package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

const tripsTable = "trips"
const statusHistoryTable = "trip_status_history"
const pgUniqueViolation = "23505" // error of ununique object
const driverActiveUniqIndex = "trips_one_active_per_driver_uniq"

type TripRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

// NewTripRepository(pool *pgxpool.Pool) *TripRepository

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// (r *TripRepository) exec(ctx context.Context) DBTX

func (r *TripRepository) exec(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

// (r *TripRepository) Create(ctx context.Context, t *trip.Trip) error

func (r *TripRepository) Create(ctx context.Context, t *trip.Trip) error {
	q, args, err := r.sb.
		Insert(tripsTable).
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status",
			"started_at", "finished_at",
		).
		Values(
			t.ID, t.UserID, t.DriverID,
			t.StartLatitude, t.StartLongitude,
			t.EndLatitude, t.EndLongitude,
			t.Price, string(t.Status),
			t.StartedAt, t.FinishedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	_, err = r.exec(ctx).Exec(ctx, q, args...)
	if err != nil {
		if isDriverBusy(err) {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

// (r *TripRepository) Ping(ctx context.Context) error

func (r *TripRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*trip.Trip, error)

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	q, args, err := r.sb.
		Select(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status",
			"started_at", "finished_at",
			"created_at", "updated_at",
		).
		From(tripsTable).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip: %w", err)
	}

	row := r.exec(ctx).QueryRow(ctx, q, args...)
	t, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trip.ErrNotFound
		}
		return nil, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

// (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, now time.Time) error

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, now time.Time) error {
	q, args, err := r.sb.
		Update(tripsTable).
		Set("status", string(trip.StatusCompleted)).
		Set("finished_at", now).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": string(trip.StatusActive)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip: %w", err)
	}

	tag, err := r.exec(ctx).Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}

	if tag.RowsAffected() == 0 {
		current, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if current.Status == trip.StatusCompleted {
			return trip.ErrAlreadyDone
		}
		return trip.ErrAlreadyDone
	}
	return nil
}

// (r *TripRepository) AppendStatusChange(ctx context.Context, ch trip.StatusChange) error

func (r *TripRepository) AppendStatusChange(ctx context.Context, ch trip.StatusChange) error {
	var from *string
	if ch.FromStatus != nil {
		s := string(*ch.FromStatus)
		from = &s
	}

	q, args, err := r.sb.
		Insert(statusHistoryTable).
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(ch.TripID, from, string(ch.ToStatus), ch.Reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert status history: %w", err)
	}

	if _, err := r.exec(ctx).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}
	return nil
}

// --- helpers ---

// scanTrip(row pgx.Row) (*trip.Trip, error)

func scanTrip(row pgx.Row) (*trip.Trip, error) {
	var t trip.Trip
	var status string
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude,
		&t.Price, &status,
		&t.StartedAt, &t.FinishedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Status = trip.Status(status)
	return &t, nil
}

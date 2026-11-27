package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func (r *Repo) SavePosition(ctx context.Context, p model.Position) (model.Position, error) {
	insertQ, insertArgs, err := r.sb.
		Insert(tablePositions).
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(p.TripID, p.Latitude, p.Longitude, p.RecordedAt).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return model.Position{}, fmt.Errorf("build insert position: %w", err)
	}

	row := r.exec(ctx).QueryRow(ctx, insertQ, insertArgs...)
	if err := row.Scan(&p.ID, &p.CreatedAt); err != nil {
		return model.Position{}, fmt.Errorf("insert position: %w", err)
	}

	updateQ, updateArgs, err := r.sb.
		Update(tableTrips).
		Set("last_position_at", p.RecordedAt).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": p.TripID}).
		Where(squirrel.Or{
			squirrel.Eq{"last_position_at": nil},
			squirrel.Expr("last_position_at < ?", p.RecordedAt),
		}).
		ToSql()
	if err != nil {
		return model.Position{}, fmt.Errorf("build update last_position_at: %w", err)
	}

	if _, err := r.exec(ctx).Exec(ctx, updateQ, updateArgs...); err != nil {
		return model.Position{}, fmt.Errorf("update last_position_at: %w", err)
	}

	return p, nil
}

func (r *Repo) ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.Position, error) {
	q, args, err := r.sb.
		Select("id", "trip_id", "latitude", "longitude", "recorded_at", "created_at").
		From(tablePositions).
		Where(squirrel.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list positions: %w", err)
	}

	rows, err := r.exec(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query positions: %w", err)
	}
	defer rows.Close()

	var out []model.Position
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.ID, &p.TripID, &p.Latitude, &p.Longitude, &p.RecordedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) FindStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	q, args, err := r.sb.
		Select("id", "driver_id", "COALESCE(last_position_at, started_at) AS stale_since").
		From(tableTrips).
		Where(squirrel.Eq{"status": statusActive}).
		Where("COALESCE(last_position_at, started_at) < now() - make_interval(secs => ?)", staleAfter.Seconds()).
		OrderBy("COALESCE(last_position_at, started_at) ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find stale: %w", err)
	}

	rows, err := r.exec(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query stale trips: %w", err)
	}
	defer rows.Close()

	var out []model.StaleTrip
	for rows.Next() {
		var s model.StaleTrip
		if err := rows.Scan(&s.TripID, &s.DriverID, &s.StaleSince); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) CheckTripActive(ctx context.Context, tripID uuid.UUID) error {
	q, args, err := r.sb.
		Select("status").
		From(tableTrips).
		Where(squirrel.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build check trip: %w", err)
	}

	var status string
	if err := r.exec(ctx).QueryRow(ctx, q, args...).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return trip.ErrNotFound
		}
		return fmt.Errorf("check trip: %w", err)
	}
	if status != statusActive {
		return trip.ErrAlreadyDone
	}
	return nil
}

func (r *Repo) CheckTripActiveOrCompleted(ctx context.Context, tripID uuid.UUID) error {
	q, args, err := r.sb.
		Select("1").
		From(tableTrips).
		Where(squirrel.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build check trip exists: %w", err)
	}

	var exists int
	if err := r.exec(ctx).QueryRow(ctx, q, args...).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return trip.ErrNotFound
		}
		return fmt.Errorf("check trip exists: %w", err)
	}
	return nil
}

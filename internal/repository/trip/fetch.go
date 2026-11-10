package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	q, args, err := r.sb.
		Select(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status",
			"started_at", "finished_at",
			"created_at", "updated_at",
		).
		From(tableTrips).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select trip: %w", err)
	}

	row := r.exec(ctx).QueryRow(ctx, q, args...)
	t, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Trip{}, trip.ErrNotFound
		}
		return model.Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

func scanTrip(row pgx.Row) (model.Trip, error) {
	var (
		t      model.Trip
		status string
	)
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude,
		&t.Price, &status,
		&t.StartedAt, &t.FinishedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return model.Trip{}, err
	}
	t.Status = model.Status(status)
	return t, nil
}

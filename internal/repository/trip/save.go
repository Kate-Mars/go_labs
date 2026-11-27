package trip

import (
	"context"
	"fmt"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func (r *Repo) Create(ctx context.Context, t model.Trip) error {
	q, args, err := r.sb.
		Insert(tableTrips).
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
		if isUniqueViolation(err, driverActiveUniqIndex) {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repo) AppendStatusChange(ctx context.Context, ch model.StatusChange) error {
	var from *string
	if ch.FromStatus != nil {
		s := string(*ch.FromStatus)
		from = &s
	}

	q, args, err := r.sb.
		Insert(tableStatusHistory).
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

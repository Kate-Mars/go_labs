package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

func (r *Repo) Finish(ctx context.Context, id uuid.UUID, at time.Time) error {
	q, args, err := r.sb.
		Update(tableTrips).
		Set("status", "completed").
		Set("finished_at", at).
		Set("updated_at", at).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": "active"}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip: %w", err)
	}

	tag, err := r.exec(ctx).Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}

	if tag.RowsAffected() == 0 {
		if _, err := r.GetByID(ctx, id); err != nil {
			return err
		}
		return trip.ErrAlreadyDone
	}
	return nil
}

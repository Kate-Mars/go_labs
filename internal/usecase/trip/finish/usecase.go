package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type Usecase struct {
	tripRepo  tripRepository
	txManager txManager
	metrics   metrics
}

func New(tripRepo tripRepository, txManager txManager, m metrics) *Usecase {
	return &Usecase{tripRepo: tripRepo, txManager: txManager, metrics: m}
}

func (u *Usecase) Do(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	if id == (uuid.UUID{}) {
		return model.Trip{}, &trip.ValidationError{Field: "trip_id", Message: "is required"}
	}

	now := time.Now().UTC()

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Finish(ctx, id, now); err != nil {
			return err
		}
		active := model.StatusActive
		return u.tripRepo.AppendStatusChange(ctx, model.StatusChange{
			TripID:     id,
			FromStatus: &active,
			ToStatus:   model.StatusCompleted,
			Reason:     "trip finished",
		})
	})
	if err != nil {
		return model.Trip{}, err
	}

	t, err := u.tripRepo.GetByID(ctx, id)
	if err != nil {
		return model.Trip{}, err
	}

	u.metrics.IncCompleted(ctx)
	return t, nil
}

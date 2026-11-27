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
		m, err := u.tripRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		dt := toDomain(m)
		if err := dt.Finish(now); err != nil {
			return err
		}

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

func toDomain(m model.Trip) trip.Trip {
	return trip.Trip{
		ID:             m.ID,
		UserID:         m.UserID,
		DriverID:       m.DriverID,
		StartLatitude:  m.StartLatitude,
		StartLongitude: m.StartLongitude,
		EndLatitude:    m.EndLatitude,
		EndLongitude:   m.EndLongitude,
		Price:          m.Price,
		Status:         trip.Status(m.Status),
		StartedAt:      m.StartedAt,
		FinishedAt:     m.FinishedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

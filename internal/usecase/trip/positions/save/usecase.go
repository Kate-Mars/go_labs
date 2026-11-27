package save

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type Input struct {
	TripID     uuid.UUID
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

type Usecase struct {
	tripRepo  tripRepository
	txManager txManager
	metrics   metrics
	now       Now
}

func New(tripRepo tripRepository, txManager txManager, m metrics, now Now) *Usecase {
	if now == nil {
		now = time.Now
	}
	return &Usecase{tripRepo: tripRepo, txManager: txManager, metrics: m, now: now}
}

func (u *Usecase) Do(ctx context.Context, in Input) (model.Position, error) {
	if err := validate(in, u.now()); err != nil {
		return model.Position{}, err
	}

	var saved model.Position
	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.CheckTripActive(ctx, in.TripID); err != nil {
			return err
		}
		p, err := u.tripRepo.SavePosition(ctx, model.Position{
			TripID:     in.TripID,
			Latitude:   in.Latitude,
			Longitude:  in.Longitude,
			RecordedAt: in.RecordedAt,
		})
		if err != nil {
			return err
		}
		saved = p
		return nil
	})
	if err != nil {
		return model.Position{}, err
	}

	u.metrics.IncPositionSaved(ctx)
	return saved, nil
}

func validate(in Input, now time.Time) error {
	if in.TripID == (uuid.UUID{}) {
		return &trip.ValidationError{Field: "trip_id", Message: "is required"}
	}
	if in.Latitude < -90 || in.Latitude > 90 {
		return &trip.ValidationError{Field: "latitude", Message: "must be in [-90, 90]"}
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return &trip.ValidationError{Field: "longitude", Message: "must be in [-180, 180]"}
	}
	if in.RecordedAt.IsZero() {
		return &trip.ValidationError{Field: "recorded_at", Message: "is required"}
	}
	if in.RecordedAt.After(now.Add(time.Minute)) {
		return &trip.ValidationError{Field: "recorded_at", Message: "must not be in the future"}
	}
	return nil
}

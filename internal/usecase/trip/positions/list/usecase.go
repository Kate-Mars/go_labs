package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type Usecase struct {
	tripRepo tripRepository
}

func New(tripRepo tripRepository) *Usecase {
	return &Usecase{tripRepo: tripRepo}
}

func (u *Usecase) Do(ctx context.Context, tripID uuid.UUID) ([]model.Position, error) {
	if tripID == (uuid.UUID{}) {
		return nil, &trip.ValidationError{Field: "trip_id", Message: "is required"}
	}
	if err := u.tripRepo.CheckTripActiveOrCompleted(ctx, tripID); err != nil {
		return nil, err
	}
	return u.tripRepo.ListPositions(ctx, tripID)
}

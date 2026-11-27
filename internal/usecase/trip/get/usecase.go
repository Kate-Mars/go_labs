package get

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

func (u *Usecase) Do(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	if id == (uuid.UUID{}) {
		return model.Trip{}, &trip.ValidationError{Field: "trip_id", Message: "is required"}
	}
	return u.tripRepo.GetByID(ctx, id)
}

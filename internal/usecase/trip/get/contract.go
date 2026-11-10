package get

import (
	"context"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

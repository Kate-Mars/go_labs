package create

import (
	"context"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type tripRepository interface {
	Create(ctx context.Context, t model.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	AppendStatusChange(ctx context.Context, change model.StatusChange) error
}

type idempotencyRepository interface {
	InsertIfAbsent(ctx context.Context, rec model.IdempotencyRecord) (bool, error)

	// Get читает ключ. Возвращает nil, если ключа нет.
	Get(ctx context.Context, key uuid.UUID) (*model.IdempotencyRecord, error)

	// SetTripID проставляет trip_id после создания поездки.
	SetTripID(ctx context.Context, key, tripID uuid.UUID) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

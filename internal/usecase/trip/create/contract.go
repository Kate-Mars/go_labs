//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type metrics interface {
	IncCreated(ctx context.Context)
}

type tripRepository interface {
	Create(ctx context.Context, t model.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	AppendStatusChange(ctx context.Context, change model.StatusChange) error
}

type idempotencyRepository interface {
	InsertIfAbsent(ctx context.Context, rec model.IdempotencyRecord) (bool, error)

	Get(ctx context.Context, key uuid.UUID) (*model.IdempotencyRecord, error)

	SetTripID(ctx context.Context, key, tripID uuid.UUID) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

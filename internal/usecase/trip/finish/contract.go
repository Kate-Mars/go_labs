//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type tripRepository interface {
	Finish(ctx context.Context, id uuid.UUID, at time.Time) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	AppendStatusChange(ctx context.Context, change model.StatusChange) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

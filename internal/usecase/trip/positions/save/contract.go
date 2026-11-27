//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package save

import (
	"context"
	"time"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type tripRepository interface {
	CheckTripActive(ctx context.Context, tripID uuid.UUID) error
	SavePosition(ctx context.Context, p model.Position) (model.Position, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type metrics interface {
	IncPositionSaved(ctx context.Context)
}

type Now func() time.Time

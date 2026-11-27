//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package list

import (
	"context"

	"github.com/google/uuid"

	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type tripRepository interface {
	CheckTripActiveOrCompleted(ctx context.Context, tripID uuid.UUID) error
	ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.Position, error)
}

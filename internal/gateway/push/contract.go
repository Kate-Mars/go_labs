//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package push

import (
	"context"

	"github.com/google/uuid"
)

type Gateway interface {
	RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
}

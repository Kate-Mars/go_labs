package create

import (
	"errors"

	"github.com/google/uuid"
)

var (
	zeroUUID = uuid.UUID{}

	errLatRange = errors.New("latitude out of range [-90, 90]")
	errLngRange = errors.New("longitude out of range [-180, 180]")
)

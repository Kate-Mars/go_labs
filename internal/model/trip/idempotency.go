package trip

import "github.com/google/uuid"

type IdempotencyRecord struct {
	Key         uuid.UUID
	TripID      *uuid.UUID
	RequestHash []byte
}

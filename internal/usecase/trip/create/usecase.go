package create

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

// Input — вход usecase создания поездки.
type Input struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64

	// IdempotencyKey — если задан, usecase работает идемпотентно.
	IdempotencyKey *uuid.UUID
}

// Output — что usecase возвращает.
type Output struct {
	Trip    model.Trip
	Created bool // true — создана только что (201), false — повтор (200).
}

// Usecase — сценарий создания поездки.
type Usecase struct {
	tripRepo  tripRepository
	idemRepo  idempotencyRepository
	txManager txManager
	metrics   metrics
}

func New(tripRepo tripRepository, idemRepo idempotencyRepository, txManager txManager, m metrics) *Usecase {
	return &Usecase{
		tripRepo:  tripRepo,
		idemRepo:  idemRepo,
		txManager: txManager,
		metrics:   m,
	}
}

// Do выполняет сценарий создания поездки.
func (u *Usecase) Do(ctx context.Context, in Input) (Output, error) {
	if err := validate(in); err != nil {
		return Output{}, err
	}

	if in.IdempotencyKey != nil {
		return u.doIdempotent(ctx, in, *in.IdempotencyKey)
	}
	return u.doPlain(ctx, in)
}

func (u *Usecase) doPlain(ctx context.Context, in Input) (Output, error) {
	now := time.Now().UTC()
	t := toModelTrip(in, now)

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Create(ctx, t); err != nil {
			return err
		}
		return u.tripRepo.AppendStatusChange(ctx, model.StatusChange{
			TripID:   t.ID,
			ToStatus: model.StatusActive,
			Reason:   "trip created",
		})
	})
	if err != nil {
		return Output{}, err
	}

	u.metrics.IncCreated(ctx)
	return Output{Trip: t, Created: true}, nil
}

func (u *Usecase) doIdempotent(ctx context.Context, in Input, key uuid.UUID) (Output, error) {
	requestHash := hashInput(in)

	var (
		result  model.Trip
		created bool
	)

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		inserted, err := u.idemRepo.InsertIfAbsent(ctx, model.IdempotencyRecord{
			Key:         key,
			RequestHash: requestHash,
		})
		if err != nil {
			return err
		}

		if inserted {
			now := time.Now().UTC()
			t := toModelTrip(in, now)
			if err := u.tripRepo.Create(ctx, t); err != nil {
				return err
			}
			if err := u.tripRepo.AppendStatusChange(ctx, model.StatusChange{
				TripID:   t.ID,
				ToStatus: model.StatusActive,
				Reason:   "trip created",
			}); err != nil {
				return err
			}
			if err := u.idemRepo.SetTripID(ctx, key, t.ID); err != nil {
				return err
			}
			result = t
			created = true
			return nil
		}

		// Ключ занят — читаем запись.
		rec, err := u.idemRepo.Get(ctx, key)
		if err != nil {
			return err
		}
		if rec == nil || rec.TripID == nil {
			return trip.ErrIdempotencyConflict
		}
		if !bytes.Equal(rec.RequestHash, requestHash) {
			return trip.ErrIdempotencyConflict
		}

		t, err := u.tripRepo.GetByID(ctx, *rec.TripID)
		if err != nil {
			return err
		}
		result = t
		return nil
	})
	if err != nil {
		return Output{}, err
	}

	// Увеличиваем счётчик только при создании, не при повторе.
	if created {
		u.metrics.IncCreated(ctx)
	}

	return Output{Trip: result, Created: created}, nil
}

func toModelTrip(in Input, now time.Time) model.Trip {
	return model.Trip{
		ID:             uuid.New(),
		UserID:         in.UserID,
		DriverID:       in.DriverID,
		StartLatitude:  in.StartLatitude,
		StartLongitude: in.StartLongitude,
		EndLatitude:    in.EndLatitude,
		EndLongitude:   in.EndLongitude,
		Price:          in.Price,
		Status:         model.StatusActive,
		StartedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func hashInput(in Input) []byte {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%f|%f|%f|%f|%d",
		in.UserID, in.DriverID,
		in.StartLatitude, in.StartLongitude,
		in.EndLatitude, in.EndLongitude,
		in.Price,
	)
	return h.Sum(nil)
}

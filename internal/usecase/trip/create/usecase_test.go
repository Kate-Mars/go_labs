package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
)

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := NewMocktxManager(ctrl)

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})

	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)

	uc := create.New(tripRepo, idemRepo, txManager)

	out, err := uc.Do(context.Background(), create.Input{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Price:    100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Created {
		t.Errorf("Created = false, want true")
	}
	if out.Trip.Price != 100 {
		t.Errorf("Price = %d, want 100", out.Trip.Price)
	}
}

func TestCreate_DriverBusy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := NewMocktxManager(ctrl)

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})

	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

	uc := create.New(tripRepo, idemRepo, txManager)

	_, err := uc.Do(context.Background(), create.Input{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Price:    100,
	})
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Errorf("err = %v, want ErrDriverBusy", err)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := NewMocktxManager(ctrl)

	uc := create.New(tripRepo, idemRepo, txManager)

	_, err := uc.Do(context.Background(), create.Input{
		DriverID: uuid.New(),
		Price:    100,
	})

	var ve *trip.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("err = %v, want ValidationError", err)
	}
}

func TestCreate_WithIdempotency_NewKey(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := NewMocktxManager(ctrl)

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})

	idemRepo.EXPECT().InsertIfAbsent(gomock.Any(), gomock.Any()).Return(true, nil)
	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
	idemRepo.EXPECT().SetTripID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	key := uuid.New()
	uc := create.New(tripRepo, idemRepo, txManager)

	out, err := uc.Do(context.Background(), create.Input{
		UserID:         uuid.New(),
		DriverID:       uuid.New(),
		Price:          100,
		IdempotencyKey: &key,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Created {
		t.Errorf("Created = false, want true")
	}
}

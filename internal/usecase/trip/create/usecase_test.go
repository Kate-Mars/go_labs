package create_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	"github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
)

// helper: txManager.Do просто вызывает fn
func passthroughDo(ctrl *gomock.Controller) *MocktxManager {
	m := NewMocktxManager(ctrl)
	m.EXPECT().Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
	return m
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().IncCreated(gomock.Any()).Times(1)

	uc := create.New(tripRepo, idemRepo, txManager, metrics)

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
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

	uc := create.New(tripRepo, idemRepo, txManager, metrics)

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
	metrics := NewMockmetrics(ctrl)

	uc := create.New(tripRepo, idemRepo, txManager, metrics)

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
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	idemRepo.EXPECT().InsertIfAbsent(gomock.Any(), gomock.Any()).Return(true, nil)
	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
	idemRepo.EXPECT().SetTripID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().IncCreated(gomock.Any()).Times(1)

	key := uuid.New()
	uc := create.New(tripRepo, idemRepo, txManager, metrics)

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

func TestCreate_WithIdempotency_Replay(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	key := uuid.New()
	tripID := uuid.New()
	existing := &model.IdempotencyRecord{
		Key:    key,
		TripID: &tripID,
		RequestHash: hashOf(create.Input{
			UserID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			DriverID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Price:    100,
		}),
	}
	idemRepo.EXPECT().InsertIfAbsent(gomock.Any(), gomock.Any()).Return(false, nil)
	idemRepo.EXPECT().Get(gomock.Any(), key).Return(existing, nil)

	tripRepo.EXPECT().GetByID(gomock.Any(), tripID).Return(model.Trip{
		ID:     tripID,
		Status: model.StatusActive,
		Price:  100,
	}, nil)

	uc := create.New(tripRepo, idemRepo, txManager, metrics)

	out, err := uc.Do(context.Background(), create.Input{
		UserID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Price:          100,
		IdempotencyKey: &key,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Created {
		t.Errorf("Created = true, want false (это повтор)")
	}
	if out.Trip.ID != tripID {
		t.Errorf("Trip.ID = %v, want %v", out.Trip.ID, tripID)
	}
}

// TestCreate_PriceZeroIsValid — price=0 валиден (граница "price >= 0").
func TestCreate_PriceZeroIsValid(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	idemRepo := NewMockidempotencyRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().IncCreated(gomock.Any()).Times(1)

	uc := create.New(tripRepo, idemRepo, txManager, metrics)

	_, err := uc.Do(context.Background(), create.Input{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Price:    0,
	})
	if err != nil {
		t.Fatalf("Price=0 should be valid, got err: %v", err)
	}
}

// TestCreate_CoordsBoundaries — координаты на границах допустимы.
func TestCreate_CoordsBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		lat, lng float64
	}{
		{"lat=-90", -90, 30},
		{"lat=90", 90, 30},
		{"lng=-180", 59, -180},
		{"lng=180", 59, 180},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			tripRepo := NewMocktripRepository(ctrl)
			idemRepo := NewMockidempotencyRepository(ctrl)
			txManager := passthroughDo(ctrl)
			metrics := NewMockmetrics(ctrl)

			tripRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
			metrics.EXPECT().IncCreated(gomock.Any()).Times(1)

			uc := create.New(tripRepo, idemRepo, txManager, metrics)

			_, err := uc.Do(context.Background(), create.Input{
				UserID:         uuid.New(),
				DriverID:       uuid.New(),
				StartLatitude:  tc.lat,
				StartLongitude: tc.lng,
				EndLatitude:    0,
				EndLongitude:   0,
				Price:          100,
			})
			if err != nil {
				t.Errorf("boundary %s should be valid, got err: %v", tc.name, err)
			}
		})
	}
}

func hashOf(in create.Input) []byte {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%f|%f|%f|%f|%d",
		in.UserID, in.DriverID,
		in.StartLatitude, in.StartLongitude,
		in.EndLatitude, in.EndLongitude,
		in.Price,
	)
	return h.Sum(nil)
}

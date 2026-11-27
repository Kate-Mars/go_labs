package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	"github.com/Kate-Mars/go_labs/internal/usecase/trip/get"
)

func TestGet_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)

	wantID := uuid.New()
	wantTrip := model.Trip{
		ID:     wantID,
		Status: model.StatusActive,
		Price:  100,
	}

	tripRepo.EXPECT().
		GetByID(gomock.Any(), wantID).
		Return(wantTrip, nil)

	uc := get.New(tripRepo)

	got, err := uc.Do(context.Background(), wantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != wantTrip.ID {
		t.Errorf("ID = %v, want %v", got.ID, wantTrip.ID)
	}
	if got.Price != wantTrip.Price {
		t.Errorf("Price = %d, want %d", got.Price, wantTrip.Price)
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)

	id := uuid.New()
	tripRepo.EXPECT().
		GetByID(gomock.Any(), id).
		Return(model.Trip{}, trip.ErrNotFound)

	uc := get.New(tripRepo)

	_, err := uc.Do(context.Background(), id)
	if !errors.Is(err, trip.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestGet_EmptyID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)

	uc := get.New(tripRepo)

	_, err := uc.Do(context.Background(), uuid.UUID{})

	var ve *trip.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("err = %v, want ValidationError", err)
	}
}

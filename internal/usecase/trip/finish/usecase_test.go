package finish_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	"github.com/Kate-Mars/go_labs/internal/usecase/trip/finish"
)

func passthroughDo(ctrl *gomock.Controller) *MocktxManager {
	m := NewMocktxManager(ctrl)
	m.EXPECT().Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
	return m
}

func TestFinish_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	id := uuid.New()

	tripRepo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(nil)
	tripRepo.EXPECT().AppendStatusChange(gomock.Any(), gomock.Any()).Return(nil)
	tripRepo.EXPECT().GetByID(gomock.Any(), id).Return(model.Trip{
		ID:     id,
		Status: model.StatusCompleted,
		Price:  100,
	}, nil)
	metrics.EXPECT().IncCompleted(gomock.Any()).Times(1)

	uc := finish.New(tripRepo, txManager, metrics)

	got, err := uc.Do(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != model.StatusCompleted {
		t.Errorf("Status = %v, want completed", got.Status)
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
}

func TestFinish_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	id := uuid.New()

	tripRepo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(trip.ErrNotFound)

	uc := finish.New(tripRepo, txManager, metrics)

	_, err := uc.Do(context.Background(), id)
	if !errors.Is(err, trip.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestFinish_AlreadyDone(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	txManager := passthroughDo(ctrl)
	metrics := NewMockmetrics(ctrl)

	id := uuid.New()

	tripRepo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(trip.ErrAlreadyDone)

	uc := finish.New(tripRepo, txManager, metrics)

	_, err := uc.Do(context.Background(), id)
	if !errors.Is(err, trip.ErrAlreadyDone) {
		t.Errorf("err = %v, want ErrAlreadyDone", err)
	}
}

func TestFinish_EmptyID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tripRepo := NewMocktripRepository(ctrl)
	txManager := NewMocktxManager(ctrl)
	metrics := NewMockmetrics(ctrl)

	uc := finish.New(tripRepo, txManager, metrics)

	_, err := uc.Do(context.Background(), uuid.UUID{})

	var ve *trip.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("err = %v, want ValidationError", err)
	}
}

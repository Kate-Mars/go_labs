package get_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/http/trip/get"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

var errTest = errors.New("boom")

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler_Get_200(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{
			ID:     tripID,
			Status: model.StatusActive,
			Price:  100,
		}, nil)

	h := get.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req, tripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(rec.Body.String(), tripID.String()) {
		t.Errorf("body does not contain trip id: %s", rec.Body.String())
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, trip.ErrNotFound)

	h := get.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req, tripID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	if !strings.Contains(rec.Body.String(), `"trip_not_found"`) {
		t.Errorf("body missing trip_not_found code: %s", rec.Body.String())
	}
}

func TestHandler_Get_InternalError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, errTest)

	h := get.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req, tripID)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), errTest.Error()) {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

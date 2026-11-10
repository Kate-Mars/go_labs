package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/http/trip/finish"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

var errTest = errors.New("boom")

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler_Finish_200(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	now := time.Now().UTC()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{
			ID:         tripID,
			Status:     model.StatusCompleted,
			Price:      100,
			FinishedAt: &now,
		}, nil)

	h := finish.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	h.Finish(rec, req, tripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"completed"`) {
		t.Errorf("body missing completed status: %s", rec.Body.String())
	}
}

func TestHandler_Finish_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, trip.ErrNotFound)

	h := finish.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	h.Finish(rec, req, tripID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	if !strings.Contains(rec.Body.String(), `"trip_not_found"`) {
		t.Errorf("body missing trip_not_found: %s", rec.Body.String())
	}
}

func TestHandler_Finish_AlreadyDone(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, trip.ErrAlreadyDone)

	h := finish.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	h.Finish(rec, req, tripID)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"trip_completed"`) {
		t.Errorf("body missing trip_completed: %s", rec.Body.String())
	}
}

func TestHandler_Finish_InternalError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, errTest)

	h := finish.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	h.Finish(rec, req, tripID)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), errTest.Error()) {
		t.Errorf("body leaks internal error: %s", rec.Body.String())
	}
}

type countingHandler struct {
	mu     sync.Mutex
	errors int
}

func (h *countingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *countingHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelError {
		h.mu.Lock()
		h.errors++
		h.mu.Unlock()
	}
	return nil
}
func (h *countingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *countingHandler) WithGroup(string) slog.Handler      { return h }

func (h *countingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.errors
}

func TestHandler_Finish_InternalError_LogsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	tripID := uuid.New()
	mockUC.EXPECT().
		Do(gomock.Any(), tripID).
		Return(model.Trip{}, errTest)

	logHandler := &countingHandler{}
	h := finish.NewHandler(slog.New(logHandler), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	h.Finish(rec, req, tripID)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if logHandler.count() == 0 {
		t.Errorf("expected at least one Error log, got 0")
	}
}

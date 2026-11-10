package create_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/trip/create"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	uc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler_Create_201(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	mockUC.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(uc.Output{
			Trip: model.Trip{
				ID:     uuid.New(),
				Status: model.StatusActive,
				Price:  100,
			},
			Created: true,
		}, nil)

	h := create.NewHandler(newLogger(), mockUC)

	body := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.4},
		"price":100
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") == "" {
		t.Errorf("Location header missing")
	}
}

func TestHandler_Create_DriverBusy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	mockUC.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(uc.Output{}, trip.ErrDriverBusy)

	h := create.NewHandler(newLogger(), mockUC)

	body := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.4},
		"price":100
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"driver_busy"`) {
		t.Errorf("body missing driver_busy code: %s", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)
	// Do НЕ должен вызываться

	h := create.NewHandler(newLogger(), mockUC)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader([]byte(`{not json`)))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// на всякий случай: ошибка usecase → 500
func TestHandler_Create_InternalError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	mockUC.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(uc.Output{}, errors.New("boom"))

	h := create.NewHandler(newLogger(), mockUC)

	body := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.4},
		"price":100
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	// НЕ должно быть деталей ошибки в теле
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body leaks internal error details: %s", rec.Body.String())
	}
}

var _ = context.Background

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

// TestHandler_Create_WithIdempotencyKey — ключ из params передаётся в usecase.
func TestHandler_Create_WithIdempotencyKey(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	key := uuid.New()

	mockUC.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in uc.Input) (uc.Output, error) {
			if in.IdempotencyKey == nil {
				t.Errorf("IdempotencyKey = nil, want &%v", key)
			} else if *in.IdempotencyKey != key {
				t.Errorf("IdempotencyKey = %v, want %v", *in.IdempotencyKey, key)
			}
			return uc.Output{
				Trip:    model.Trip{ID: uuid.New(), Status: model.StatusActive, Price: 100},
				Created: true,
			}, nil
		})

	h := create.NewHandler(newLogger(), mockUC)

	body := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.4},
		"price":100
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{IdempotencyKey: &key})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
}

// TestHandler_Create_InternalError_LogsError — при 500 пишется Error-лог.
func TestHandler_Create_InternalError_LogsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockUC := NewMockusecase(ctrl)

	mockUC.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(uc.Output{}, errors.New("boom"))

	logHandler := &countingHandler{}
	h := create.NewHandler(slog.New(logHandler), mockUC)

	body := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.4},
		"price":100
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if logHandler.count() == 0 {
		t.Errorf("expected at least one Error log, got 0")
	}
}

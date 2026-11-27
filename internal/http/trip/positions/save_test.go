package positions_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/http/trip/positions"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	saveuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/positions/save"
)

type fakeUC struct{}

func (f *fakeUC) Do(ctx context.Context, in saveuc.Input) (model.Position, error) {
	return model.Position{ID: 1, TripID: in.TripID}, nil
}

type fakeCanceller struct {
	called bool
}

func (f *fakeCanceller) Cancel(tripID uuid.UUID) bool {
	f.called = true
	return true
}

func TestSaveHandler_CancelsOnSuccess(t *testing.T) {
	tripID := uuid.New()
	c := &fakeCanceller{}
	h := positions.NewSaveHandler(slog.Default(), &fakeUC{}, c)

	body := `{"coordinates":{"latitude":59.9,"longitude":30.3},"recorded_at":"2026-11-27T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/positions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateTripPosition(rec, req, tripID)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
	if !c.called {
		t.Error("expected Cancel to be called after successful save")
	}
}

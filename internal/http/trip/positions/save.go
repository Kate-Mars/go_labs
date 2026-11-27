//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package positions

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
	saveuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/positions/save"
)

type saveUsecase interface {
	Do(ctx context.Context, in saveuc.Input) (model.Position, error)
}

type positionCanceller interface {
	Cancel(tripID uuid.UUID) bool
}

type SaveHandler struct {
	log    *slog.Logger
	uc     saveUsecase
	cancel positionCanceller
}

func NewSaveHandler(log *slog.Logger, u saveUsecase, cancel positionCanceller) *SaveHandler {
	return &SaveHandler{log: log, uc: u, cancel: cancel}
}

func (h *SaveHandler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.invalidRequest(w, r, "cannot read request body")
		return
	}
	defer r.Body.Close()

	var req api.PositionData
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		h.invalidRequest(w, r, "invalid JSON")
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		h.invalidRequest(w, r, "unexpected data after JSON body")
		return
	}

	in := saveuc.Input{
		TripID:     tripID,
		Latitude:   req.Coordinates.Latitude,
		Longitude:  req.Coordinates.Longitude,
		RecordedAt: req.RecordedAt,
	}

	p, err := h.uc.Do(r.Context(), in)
	if err != nil {
		status, prob := domainerror.Problem(err)
		common.Problem(w, h.log, r, status, prob)
		return
	}

	if h.cancel != nil && h.cancel.Cancel(tripID) {
		h.log.Debug("cancelled stale position request", "trip_id", tripID)
	}

	common.JSON(w, h.log, http.StatusCreated, common.ToAPIPosition(p))
}

func (h *SaveHandler) invalidRequest(w http.ResponseWriter, r *http.Request, detail string) {
	common.Problem(w, h.log, r, http.StatusBadRequest, api.Problem{
		Type:   "https://tripgo.example/problems/invalid-request",
		Title:  "Invalid request",
		Status: http.StatusBadRequest,
		Code:   "invalid_request",
		Detail: ptr(detail),
	})
}

func ptr(s string) *string { return &s }

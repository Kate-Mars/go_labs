package create

import (
	"encoding/json"
	"net/http"

	"log/slog"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
	uc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
	"github.com/google/uuid"
)

type Handler struct {
	log *slog.Logger
	uc  *uc.Usecase
}

func NewHandler(log *slog.Logger, uc *uc.Usecase) *Handler {
	return &Handler{log: log, uc: uc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var req api.TripData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.Problem(w, h.log, r, 400, api.Problem{
			Type:   "https://tripgo.example/problems/invalid-request",
			Title:  "Invalid request",
			Status: 400,
			Code:   "invalid_request",
			Detail: ptr("Request body is not valid JSON"),
		})
		return
	}

	in := uc.Input{
		UserID:         uuid.UUID(req.UserId),
		DriverID:       uuid.UUID(req.DriverId),
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
	}
	if params.IdempotencyKey != nil {
		key := uuid.UUID(*params.IdempotencyKey)
		in.IdempotencyKey = &key
	}

	out, err := h.uc.Do(r.Context(), in)
	if err != nil {
		status, p := domainerror.Problem(err)
		if status == http.StatusInternalServerError {
			h.log.Error("create trip failed", "err", err)
		}
		common.Problem(w, h.log, r, status, p)
		return
	}

	resp := common.ToAPITrip(out.Trip)
	status := http.StatusCreated
	if !out.Created {
		status = http.StatusOK
	}
	w.Header().Set("Location", "/api/v1/trips/"+resp.Id.String())
	common.JSON(w, h.log, status, resp)
}

func ptr(s string) *string { return &s }

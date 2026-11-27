//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
	uc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
)

type usecase interface {
	Do(ctx context.Context, in uc.Input) (uc.Output, error)
}

type Handler struct {
	log *slog.Logger
	uc  usecase
}

func NewHandler(log *slog.Logger, u usecase) *Handler {
	return &Handler{log: log, uc: u}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		invalidRequest(w, h.log, r, "cannot read request body")
		return
	}
	defer r.Body.Close()

	// Первый декодер: в map, чтобы отличить missing/null от нулевого значения.
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&raw); err != nil {
		invalidRequest(w, h.log, r, "malformed JSON")
		return
	}
	// Trailing JSON — на ТОМ ЖЕ декодере.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		invalidRequest(w, h.log, r, "unexpected data after JSON body")
		return
	}

	if !requiredFields(raw) {
		invalidRequest(w, h.log, r, "required fields are missing or null")
		return
	}

	// Второй декодер: в структуру, со строгой проверкой лишних полей.
	var req api.TripData
	strict := json.NewDecoder(bytes.NewReader(body))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&req); err != nil {
		invalidRequest(w, h.log, r, "unknown or invalid fields")
		return
	}

	in := uc.Input{
		UserID:         req.UserId,
		DriverID:       req.DriverId,
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
	}
	if params.IdempotencyKey != nil {
		in.IdempotencyKey = params.IdempotencyKey
	}

	out, err := h.uc.Do(r.Context(), in)
	if err != nil {
		status, p := domainerror.Problem(err)
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

func requiredFields(raw map[string]json.RawMessage) bool {
	if raw == nil {
		return false
	}

	for _, field := range []string{"user_id", "driver_id", "price", "start_point", "end_point"} {
		if !present(raw, field) {
			return false
		}
	}

	for _, field := range []string{"start_point", "end_point"} {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(raw[field], &point); err != nil {
			return false
		}
		if !present(point, "latitude") || !present(point, "longitude") {
			return false
		}
	}

	return true
}

func present(raw map[string]json.RawMessage, key string) bool {
	value, exists := raw[key]
	if !exists || len(value) == 0 {
		return false
	}
	return string(value) != "null"
}

func invalidRequest(w http.ResponseWriter, log *slog.Logger, r *http.Request, detail string) {
	common.Problem(
		w,
		log,
		r,
		http.StatusBadRequest,
		api.Problem{
			Type:   "https://tripgo.example/problems/invalid-request",
			Title:  "Invalid request",
			Status: http.StatusBadRequest,
			Code:   "invalid_request",
			Detail: ptr(detail),
		},
	)
}

func ptr(s string) *string { return &s }

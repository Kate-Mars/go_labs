package domainerror

import (
	"errors"
	"net/http"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	api "github.com/Kate-Mars/go_labs/internal/generated"
)

func Problem(err error) (status int, body api.Problem) {
	var valErr *trip.ValidationError
	if errors.As(err, &valErr) {
		return http.StatusBadRequest, api.Problem{
			Type:   "https://tripgo.example/problems/invalid-request",
			Title:  "Invalid request",
			Status: http.StatusBadRequest,
			Code:   "invalid_request",
			Detail: ptr(valErr.Error()),
		}
	}

	switch {
	case errors.Is(err, trip.ErrNotFound):
		return http.StatusNotFound, mk(http.StatusNotFound, "Trip not found", "trip_not_found")
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, mk(http.StatusConflict, "Driver busy", "driver_busy")
	case errors.Is(err, trip.ErrAlreadyDone):
		return http.StatusConflict, mk(http.StatusConflict, "Trip completed", "trip_completed")
	case errors.Is(err, trip.ErrIdempotencyConflict):
		return http.StatusConflict, mk(http.StatusConflict, "Idempotency conflict", "idempotency_conflict")
	}

	return http.StatusInternalServerError, mk(http.StatusInternalServerError, "Internal Server Error", "internal_error")
}

func mk(status int, title, code string) api.Problem {
	return api.Problem{
		Type:   "https://tripgo.example/problems/" + code,
		Title:  title,
		Status: int32(status),
		Code:   code,
	}
}

func ptr(s string) *string { return &s }

package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

const problemTypeBase = "https://tripgo.example/problems/"

type problemSpec struct {
	status int
	title  string
	code   string
	slug   string
}

var (
	specInvalidRequest = problemSpec{
		status: http.StatusBadRequest,
		title:  "Invalid request",
		code:   "invalid_request",
		slug:   "invalid-request",
	}
	specTripNotFound = problemSpec{
		status: http.StatusNotFound,
		title:  "Trip not found",
		code:   "trip_not_found",
		slug:   "trip-not-found",
	}
	specTripCompleted = problemSpec{
		status: http.StatusConflict,
		title:  "Trip completed",
		code:   "trip_completed",
		slug:   "trip-completed",
	}
	specDriverBusy = problemSpec{
		status: http.StatusConflict,
		title:  "Driver busy",
		code:   "driver_busy",
		slug:   "driver-busy",
	}
	specInternal = problemSpec{
		status: http.StatusInternalServerError,
		title:  "Internal Server Error",
		code:   "internal_error",
		slug:   "internal-error",
	}
)

// writeProblem(w http.ResponseWriter, log *slog.Logger, r *http.Request, spec problemSpec, detail string)

func writeProblem(w http.ResponseWriter, log *slog.Logger, r *http.Request, spec problemSpec, detail string) {
	instance := r.URL.Path
	p := api.Problem{
		Type:     problemTypeBase + spec.slug,
		Title:    spec.title,
		Status:   int32(spec.status),
		Code:     spec.code,
		Detail:   &detail,
		Instance: &instance,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(spec.status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Error("encode problem", "err", err)
	}
}

// writeDomainError(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error) bool

func writeDomainError(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, trip.ErrNotFound):
		writeProblem(w, log, r, specTripNotFound, "Trip was not found")
		return true
	case errors.Is(err, trip.ErrDriverBusy):
		writeProblem(w, log, r, specDriverBusy, "Driver already has an active trip")
		return true
	case errors.Is(err, trip.ErrAlreadyDone):
		writeProblem(w, log, r, specTripCompleted, "Operation is not allowed for a completed trip")
		return true
	default:
		return false
	}
}

// writeInternal(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error)

func writeInternal(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error) {
	log.Error("internal error", "err", err, "path", r.URL.Path)
	writeProblem(w, log, r, specInternal, "Internal server error")
}

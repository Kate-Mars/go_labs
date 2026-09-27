package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/Kate-Mars/go_labs/internal/repository/postgres"
)

type Handlers struct {
	log       *slog.Logger
	repo      *postgres.TripRepository
	txManager postgres.TxManager
}

func NewHandlers(log *slog.Logger, repo *postgres.TripRepository, txManager postgres.TxManager) *Handlers {
	return &Handlers{
		log:       log,
		repo:      repo,
		txManager: txManager,
	}
}

// --- Service ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.repo.Ping(ctx); err != nil {
		h.log.Warn("ready: db ping failed", "err", err)
		writeJSON(w, h.log, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// --- Trips ---

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var body api.TripData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeProblem(w, h.log, r, specInvalidRequest, "Request body is not valid JSON")
		return
	}

	if err := validateTripData(body); err != nil {
		writeProblem(w, h.log, r, specInvalidRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	t := &trip.Trip{
		ID:             uuid.New(),
		UserID:         uuid.UUID(body.UserId),
		DriverID:       uuid.UUID(body.DriverId),
		StartLatitude:  body.StartPoint.Latitude,
		StartLongitude: body.StartPoint.Longitude,
		EndLatitude:    body.EndPoint.Latitude,
		EndLongitude:   body.EndPoint.Longitude,
		Price:          body.Price,
		Status:         trip.StatusActive,
		StartedAt:      now,
		FinishedAt:     nil,
	}

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := h.repo.Create(ctx, t); err != nil {
			return err
		}
		return h.repo.AppendStatusChange(ctx, trip.StatusChange{
			TripID:     t.ID,
			FromStatus: nil,
			ToStatus:   trip.StatusActive,
			Reason:     "trip created",
		})
	})
	if err != nil {
		if writeDomainError(w, h.log, r, err) {
			return
		}
		writeInternal(w, h.log, r, err)
		return
	}

	saved, err := h.repo.GetByID(r.Context(), t.ID)
	if err != nil {
		writeInternal(w, h.log, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+saved.ID.String())
	writeJSON(w, h.log, http.StatusCreated, toAPITrip(saved))
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	t, err := h.repo.GetByID(r.Context(), tripId)
	if err != nil {
		if writeDomainError(w, h.log, r, err) {
			return
		}
		writeInternal(w, h.log, r, err)
		return
	}
	writeJSON(w, h.log, http.StatusOK, toAPITrip(t))
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	now := time.Now().UTC()

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := h.repo.Finish(ctx, tripId, now); err != nil {
			return err
		}
		active := trip.StatusActive
		return h.repo.AppendStatusChange(ctx, trip.StatusChange{
			TripID:     tripId,
			FromStatus: &active,
			ToStatus:   trip.StatusCompleted,
			Reason:     "trip finished",
		})
	})
	if err != nil {
		if writeDomainError(w, h.log, r, err) {
			return
		}
		writeInternal(w, h.log, r, err)
		return
	}

	t, err := h.repo.GetByID(r.Context(), tripId)
	if err != nil {
		writeInternal(w, h.log, r, err)
		return
	}
	writeJSON(w, h.log, http.StatusOK, toAPITrip(t))
}

// --- Positions ---

func (h *Handlers) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

func (h *Handlers) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, log *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("encode response", "err", err)
	}
}

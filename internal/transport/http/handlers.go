package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handlers struct {
	log  *slog.Logger
	pool *pgxpool.Pool
}

func NewHandlers(log *slog.Logger, pool *pgxpool.Pool) *Handlers {
	return &Handlers{log: log, pool: pool}
}

// --- Service ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		h.log.Warn("ready: db ping failed", "err", err)
		writeJSON(w, h.log, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// --- Trips ---

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
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

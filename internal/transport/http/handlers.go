package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/repository/postgres"
)

type Handlers struct {
	log       *slog.Logger
	repo      *postgres.TripRepository
	idemRepo  *postgres.IdempotencyRepository
	txManager postgres.TxManager
}

func NewHandlers(
	log *slog.Logger,
	repo *postgres.TripRepository,
	idemRepo *postgres.IdempotencyRepository,
	txManager postgres.TxManager,
) *Handlers {
	return &Handlers{
		log:       log,
		repo:      repo,
		idemRepo:  idemRepo,
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
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeProblem(w, h.log, r, specInvalidRequest, "Failed to read request body")
		return
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var body api.TripData
	if err := dec.Decode(&body); err != nil {
		writeProblem(w, h.log, r, specInvalidRequest, "Request body is not valid JSON")
		return
	}

	if err := validateTripData(body); err != nil {
		writeProblem(w, h.log, r, specInvalidRequest, err.Error())
		return
	}

	if params.IdempotencyKey != nil {
		h.createTripIdempotent(w, r, body, *params.IdempotencyKey)
		return
	}

	h.createTripPlain(w, r, body)
}

func (h *Handlers) createTripPlain(w http.ResponseWriter, r *http.Request, body api.TripData) {
	var result *trip.Trip

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		t, err := h.createTripInTx(ctx, body)
		if err != nil {
			return err
		}
		result = t
		return nil
	})
	if err != nil {
		if writeDomainError(w, h.log, r, err) {
			return
		}
		writeInternal(w, h.log, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	writeJSON(w, h.log, http.StatusCreated, toAPITrip(result))
}

// (h *Handlers) createTripIdempotent

func (h *Handlers) createTripIdempotent(
	w http.ResponseWriter,
	r *http.Request,
	body api.TripData,
	key uuid.UUID,
) {
	requestHash := sha256.Sum256(mustJSON(body))

	const spName = "idem_insert"

	var (
		result *trip.Trip
		replay bool
	)

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := h.idemRepo.Savepoint(ctx, spName); err != nil {
			return err
		}

		insertErr := h.idemRepo.Insert(ctx, postgres.IdempotencyRecord{
			Key:         key,
			TripID:      nil,
			RequestHash: requestHash[:],
		})
		if insertErr == nil {
			t, err := h.createTripInTx(ctx, body)
			if err != nil {
				return err
			}
			if err := h.idemRepo.SetTripID(ctx, key, t.ID); err != nil {
				return err
			}
			result = t
			return nil
		}

		if rbErr := h.idemRepo.RollbackToSavepoint(ctx, spName); rbErr != nil {
			return rbErr
		}

		if !errors.Is(insertErr, postgres.ErrIdempotencyKeyExists) {
			return insertErr
		}

		rec, err := h.idemRepo.Get(ctx, key)
		if err != nil {
			return err
		}
		if rec == nil {
			return trip.ErrIdempotencyConflict
		}

		if !bytes.Equal(rec.RequestHash, requestHash[:]) {
			return trip.ErrIdempotencyConflict
		}
		if rec.TripID == nil {
			return trip.ErrIdempotencyConflict
		}

		t, err := h.repo.GetByID(ctx, *rec.TripID)
		if err != nil {
			return err
		}
		result = t
		replay = true
		return nil
	})
	if err != nil {
		if writeDomainError(w, h.log, r, err) {
			return
		}
		writeInternal(w, h.log, r, err)
		return
	}

	if replay {
		writeJSON(w, h.log, http.StatusOK, toAPITrip(result))
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	writeJSON(w, h.log, http.StatusCreated, toAPITrip(result))
}

func (h *Handlers) createTripInTx(ctx context.Context, body api.TripData) (*trip.Trip, error) {
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
	if err := h.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	if err := h.repo.AppendStatusChange(ctx, trip.StatusChange{
		TripID:     t.ID,
		FromStatus: nil,
		ToStatus:   trip.StatusActive,
		Reason:     "trip created",
	}); err != nil {
		return nil, err
	}
	return t, nil
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

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

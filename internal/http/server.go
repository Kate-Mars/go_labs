package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/trip/create"
	"github.com/Kate-Mars/go_labs/internal/http/trip/finish"
	"github.com/Kate-Mars/go_labs/internal/http/trip/get"
)

type Server struct {
	log    *slog.Logger
	create *create.Handler
	get    *get.Handler
	finish *finish.Handler
}

func NewServer(
	log *slog.Logger,
	createH *create.Handler,
	getH *get.Handler,
	finishH *finish.Handler,
) *Server {
	return &Server{
		log:    log,
		create: createH,
		get:    getH,
		finish: finishH,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.create.Create(w, r, params)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.get.Get(w, r, tripId)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.finish.Finish(w, r, tripId)
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// ЛР3
func (s *Server) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	http.NotFound(w, r)
}

// ЛР3
func (s *Server) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	http.NotFound(w, r)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		s.log.Error("encode response", "err", err)
	}
}

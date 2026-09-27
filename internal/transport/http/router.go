package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/Kate-Mars/go_labs/internal/repository/postgres"
)

func NewRouter(log *slog.Logger, repo *postgres.TripRepository, txManager postgres.TxManager) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	handlers := NewHandlers(log, repo, txManager)
	api.HandlerFromMux(handlers, r)

	return r
}

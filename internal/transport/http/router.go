package http

import (
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(log *slog.Logger, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	handlers := NewHandlers(log, pool)
	api.HandlerFromMux(handlers, r)

	return r
}

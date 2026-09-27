package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Kate-Mars/go_labs/internal/api"
)

func NewRouter(log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	handlers := NewHandlers(log)
	api.HandlerFromMux(handlers, r)

	return r
}

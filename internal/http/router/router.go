package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Kate-Mars/go_labs/internal/generated"
)

func NewRouter(log *slog.Logger, s api.ServerInterface) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	api.HandlerFromMux(s, r)

	return r
}

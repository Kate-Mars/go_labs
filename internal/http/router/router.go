package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/middleware"
)

func NewRouter(log *slog.Logger, s api.ServerInterface, m *middleware.Metrics) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(log))
	r.Use(middleware.Tracing())
	r.Use(middleware.Logging(log))
	r.Use(m.Middleware())

	api.HandlerFromMux(s, r)

	return r
}

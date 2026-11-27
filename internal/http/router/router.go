package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/middleware"
)

func NewRouter(log *slog.Logger, s api.ServerInterface, m *middleware.Metrics) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logging(log))
	r.Use(m.Middleware())
	r.Use(middleware.Tracing())
	r.Use(middleware.Recover(log))

	api.HandlerWithOptions(s, api.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: errorHandler(log),
	})

	return r
}

func errorHandler(log *slog.Logger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		common.Problem(
			w,
			log,
			r,
			http.StatusBadRequest,
			api.Problem{
				Type:   "https://tripgo.example/problems/invalid-request",
				Title:  "Invalid request",
				Status: http.StatusBadRequest,
				Code:   "invalid_request",
				Detail: stringPtr("Invalid request parameter: " + err.Error()),
			},
		)
	}
}

func stringPtr(s string) *string {
	return &s
}

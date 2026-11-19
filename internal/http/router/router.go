package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/middleware"
	"github.com/Kate-Mars/go_labs/internal/http/common"
)

func NewRouter(log *slog.Logger, s api.ServerInterface, m *middleware.Metrics) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(log))
	r.Use(middleware.Tracing())
	r.Use(middleware.Logging(log))
	r.Use(m.Middleware())

	api.HandlerWithOptions(s, api.ChiServerOptions{
	    BaseRouter: r,
	    ErrorHandlerFunc: func(
	        w http.ResponseWriter,
	        req *http.Request,
	        err error,
	    ) {
	        common.Problem(
	            w,
	            log,
	            req,
	            http.StatusBadRequest,
	            api.Problem{
	                Type: "https://tripgo.example/problems/invalid-request",
	                Title: "Invalid request",
	                Status: http.StatusBadRequest,
	                Code: "invalid_request",
	                Detail: stringPtr("Invalid request parameter: " + err.Error()),
	            },
	        )
	    },
	})

	return r
}

func stringPtr(s string) *string {
    return &s
}

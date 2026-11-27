package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	api "github.com/Kate-Mars/go_labs/internal/generated"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					span := trace.SpanFromContext(r.Context())
					span.RecordError(fmt.Errorf("panic: %v", p))
					span.SetStatus(codes.Error, "panic")
					span.SetAttributes(
						attribute.Int("http.response.status_code", http.StatusInternalServerError),
					)

					sc := span.SpanContext()
					log.Error("panic recovered",
						"panic", p,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
						"route", r.Pattern,
						"trace_id", sc.TraceID().String(),
						"span_id", sc.SpanID().String(),
					)

					body := api.Problem{
						Type:   "https://tripgo.example/problems/internal-error",
						Title:  "Internal Server Error",
						Status: http.StatusInternalServerError,
						Code:   "internal_error",
					}
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(body)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

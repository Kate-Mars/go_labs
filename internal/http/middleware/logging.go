package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)
			duration := time.Since(start)

			sc := trace.SpanContextFromContext(r.Context())
			traceID := ""
			spanID := ""
			if sc.HasTraceID() {
				traceID = sc.TraceID().String()
			}
			if sc.HasSpanID() {
				spanID = sc.SpanID().String()
			}

			level := slog.LevelInfo
			if ww.status >= 500 {
				level = slog.LevelError
			}

			log.Log(r.Context(), level, "request handled",
				"service", "trip-service",
				"method", r.Method,
				"route", r.Pattern,
				"status", ww.status,
				"duration_ms", duration.Milliseconds(),
				"trace_id", traceID,
				"span_id", spanID,
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

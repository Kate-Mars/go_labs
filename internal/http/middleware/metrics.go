package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Metrics struct {
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("http-server")

	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		requestDuration: duration,
		activeRequests:  active,
	}, nil
}

func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()

			activeAttrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
			}
			m.activeRequests.Add(ctx, 1, metric.WithAttributes(activeAttrs...))
			defer m.activeRequests.Add(ctx, -1, metric.WithAttributes(activeAttrs...))

			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)

			route := r.Pattern
			if route == "" {
				route = "unknown"
			}

			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.SetName(r.Method + " " + route)
			}

			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
				attribute.String("http.response.status_code", strconv.Itoa(ww.status)),
			}
			m.requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
		})
	}
}

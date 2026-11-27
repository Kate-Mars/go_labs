package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func Tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + r.Pattern
			}),
		)
	}
}

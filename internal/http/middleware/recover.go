package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	api "github.com/Kate-Mars/go_labs/internal/generated"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					log.Error("panic recovered",
						"panic", p,
						"stack", string(debug.Stack()),
						"path", r.URL.Path,
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

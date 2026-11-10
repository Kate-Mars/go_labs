package common

import (
	"encoding/json"
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/generated"
)

func JSON(w http.ResponseWriter, log *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("encode response", "err", err)
	}
}

func Problem(w http.ResponseWriter, log *slog.Logger, r *http.Request, status int, p api.Problem) {
	if p.Instance == nil {
		inst := r.URL.Path
		p.Instance = &inst
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Error("encode problem", "err", err)
	}
}

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

// usecase — что хендлеру нужно от слоя сценариев.
type usecase interface {
	Do(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

type Handler struct {
	log *slog.Logger
	uc  usecase
}

func NewHandler(log *slog.Logger, u usecase) *Handler {
	return &Handler{log: log, uc: u}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	t, err := h.uc.Do(r.Context(), tripID)
	if err != nil {
		status, p := domainerror.Problem(err)
		if status == http.StatusInternalServerError {
			h.log.Error("get trip failed", "err", err, "trip_id", tripID)
		}
		common.Problem(w, h.log, r, status, p)
		return
	}

	common.JSON(w, h.log, http.StatusOK, common.ToAPITrip(t))
}

package positions

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

type listUsecase interface {
	Do(ctx context.Context, tripID uuid.UUID) ([]model.Position, error)
}

type ListHandler struct {
	log *slog.Logger
	uc  listUsecase
}

func NewListHandler(log *slog.Logger, u listUsecase) *ListHandler {
	return &ListHandler{log: log, uc: u}
}

func (h *ListHandler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	positions, err := h.uc.Do(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err)
		common.Problem(w, h.log, r, status, prob)
		return
	}

	resp := api.TripPositionsResponse{
		Positions: make([]api.TripPosition, 0, len(positions)),
	}
	for _, p := range positions {
		resp.Positions = append(resp.Positions, common.ToAPIPosition(p))
	}
	common.JSON(w, h.log, http.StatusOK, resp)
}

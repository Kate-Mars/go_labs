package finish

import (
	"log/slog"
	"net/http"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
	finishuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/finish"
)

type Handler struct {
	log *slog.Logger
	uc  *finishuc.Usecase
}

func NewHandler(log *slog.Logger, uc *finishuc.Usecase) *Handler {
	return &Handler{log: log, uc: uc}
}

func (h *Handler) Finish(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	t, err := h.uc.Do(r.Context(), tripID)
	if err != nil {
		status, p := domainerror.Problem(err)
		if status == http.StatusInternalServerError {
			h.log.Error("finish trip failed", "err", err, "trip_id", tripID)
		}
		common.Problem(w, h.log, r, status, p)
		return
	}

	common.JSON(w, h.log, http.StatusOK, common.ToAPITrip(t))
}

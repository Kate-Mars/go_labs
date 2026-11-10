package domainerror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
	"github.com/Kate-Mars/go_labs/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "validation error",
			err:        &trip.ValidationError{Field: "user_id", Message: "is required"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "not found",
			err:        trip.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "driver busy",
			err:        trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "already done",
			err:        trip.ErrAlreadyDone,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "idempotency conflict",
			err:        trip.ErrIdempotencyConflict,
			wantStatus: http.StatusConflict,
			wantCode:   "idempotency_conflict",
		},
		{
			name:       "unknown error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, body := domainerror.Problem(tt.err)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
		})
	}
}

package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

func TestFinish(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		status     trip.Status
		wantStatus trip.Status
		wantErr    error
	}{
		{
			name:       "active trip becomes completed",
			status:     trip.StatusActive,
			wantStatus: trip.StatusCompleted,
			wantErr:    nil,
		},
		{
			name:       "already completed trip returns error",
			status:     trip.StatusCompleted,
			wantStatus: trip.StatusCompleted,
			wantErr:    trip.ErrAlreadyDone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &trip.Trip{
				ID:     uuid.New(),
				Status: tt.status,
			}
			err := tr.Finish(now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Finish() err = %v, want %v", err, tt.wantErr)
			}
			if tr.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", tr.Status, tt.wantStatus)
			}
			if tt.wantErr == nil && tr.FinishedAt == nil {
				t.Errorf("FinishedAt = nil, want %v", now)
			}
			if tt.wantErr == nil && !tr.FinishedAt.Equal(now) {
				t.Errorf("FinishedAt = %v, want %v", tr.FinishedAt, now)
			}
		})
	}
}

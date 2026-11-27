package common_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/common"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func TestToAPITrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()

	tests := []struct {
		name       string
		in         model.Trip
		wantStatus api.TripStatus
	}{
		{
			name: "active trip",
			in: model.Trip{
				ID:             tripID,
				UserID:         userID,
				DriverID:       driverID,
				StartLatitude:  59.9398,
				StartLongitude: 30.3146,
				EndLatitude:    59.9290,
				EndLongitude:   30.3626,
				Price:          1450,
				Status:         model.StatusActive,
				StartedAt:      now,
			},
			wantStatus: api.Active,
		},
		{
			name: "completed trip",
			in: model.Trip{
				ID:             tripID,
				UserID:         userID,
				DriverID:       driverID,
				StartLatitude:  59.9398,
				StartLongitude: 30.3146,
				EndLatitude:    59.9290,
				EndLongitude:   30.3626,
				Price:          1450,
				Status:         model.StatusCompleted,
				StartedAt:      now,
				FinishedAt:     &now,
			},
			wantStatus: api.Completed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := common.ToAPITrip(tt.in)

			if got.Id != tt.in.ID {
				t.Errorf("Id = %v, want %v", got.Id, tt.in.ID)
			}
			if got.UserId != tt.in.UserID {
				t.Errorf("UserId = %v, want %v", got.UserId, tt.in.UserID)
			}
			if got.DriverId != tt.in.DriverID {
				t.Errorf("DriverId = %v, want %v", got.DriverId, tt.in.DriverID)
			}
			if got.StartPoint.Latitude != tt.in.StartLatitude {
				t.Errorf("StartPoint.Latitude = %v, want %v", got.StartPoint.Latitude, tt.in.StartLatitude)
			}
			if got.StartPoint.Longitude != tt.in.StartLongitude {
				t.Errorf("StartPoint.Longitude = %v, want %v", got.StartPoint.Longitude, tt.in.StartLongitude)
			}
			if got.EndPoint.Latitude != tt.in.EndLatitude {
				t.Errorf("EndPoint.Latitude = %v, want %v", got.EndPoint.Latitude, tt.in.EndLatitude)
			}
			if got.EndPoint.Longitude != tt.in.EndLongitude {
				t.Errorf("EndPoint.Longitude = %v, want %v", got.EndPoint.Longitude, tt.in.EndLongitude)
			}
			if got.Price != tt.in.Price {
				t.Errorf("Price = %v, want %v", got.Price, tt.in.Price)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}
			if !got.StartedAt.Equal(tt.in.StartedAt) {
				t.Errorf("StartedAt = %v, want %v", got.StartedAt, tt.in.StartedAt)
			}
			if tt.in.FinishedAt != nil {
				if got.FinishedAt == nil {
					t.Errorf("FinishedAt = nil, want %v", tt.in.FinishedAt)
				} else if !got.FinishedAt.Equal(*tt.in.FinishedAt) {
					t.Errorf("FinishedAt = %v, want %v", got.FinishedAt, *tt.in.FinishedAt)
				}
			} else if got.FinishedAt != nil {
				t.Errorf("FinishedAt = %v, want nil", got.FinishedAt)
			}
		})
	}
}

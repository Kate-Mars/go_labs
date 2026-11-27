package common

import (
	api "github.com/Kate-Mars/go_labs/internal/generated"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

func ToAPITrip(t model.Trip) api.Trip {
	var status api.TripStatus
	switch t.Status {
	case model.StatusActive:
		status = api.Active
	case model.StatusCompleted:
		status = api.Completed
	}

	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Coordinates{Latitude: t.StartLatitude, Longitude: t.StartLongitude},
		EndPoint:       api.Coordinates{Latitude: t.EndLatitude, Longitude: t.EndLongitude},
		Price:          t.Price,
		Status:         status,
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: nil, // для ЛР3
	}
}

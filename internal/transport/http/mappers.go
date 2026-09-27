package http

import (
	"errors"

	api "github.com/Kate-Mars/go_labs/internal/api"
	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

func validateTripData(d api.TripData) error {
	if d.UserId == (api.TripData{}).UserId {
		return errors.New("user_id is required")
	}
	if d.DriverId == (api.TripData{}).DriverId {
		return errors.New("driver_id is required")
	}
	if err := validateCoordinates(d.StartPoint); err != nil {
		return errors.New("start_point: " + err.Error())
	}
	if err := validateCoordinates(d.EndPoint); err != nil {
		return errors.New("end_point: " + err.Error())
	}
	if d.Price < 0 {
		return errors.New("price must be >= 0")
	}
	return nil
}

func validateCoordinates(c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.New("latitude out of range [-90, 90]")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("longitude out of range [-180, 180]")
	}
	return nil
}

func toAPITrip(t *trip.Trip) api.Trip {
	var status api.TripStatus
	switch t.Status {
	case trip.StatusActive:
		status = api.Active
	case trip.StatusCompleted:
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
		LastPositionAt: nil,
	}
}

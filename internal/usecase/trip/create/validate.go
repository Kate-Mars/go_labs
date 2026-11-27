package create

import (
	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

func validate(in Input) error {
	if in.UserID == zeroUUID {
		return &trip.ValidationError{Field: "user_id", Message: "is required"}
	}
	if in.DriverID == zeroUUID {
		return &trip.ValidationError{Field: "driver_id", Message: "is required"}
	}
	if err := validateCoords(in.StartLatitude, in.StartLongitude); err != nil {
		return &trip.ValidationError{Field: "start_point", Message: err.Error()}
	}
	if err := validateCoords(in.EndLatitude, in.EndLongitude); err != nil {
		return &trip.ValidationError{Field: "end_point", Message: err.Error()}
	}
	if in.Price < 0 {
		return &trip.ValidationError{Field: "price", Message: "must be >= 0"}
	}
	return nil
}

func validateCoords(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errLatRange
	}
	if lng < -180 || lng > 180 {
		return errLngRange
	}
	return nil
}

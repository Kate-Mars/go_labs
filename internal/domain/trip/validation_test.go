package trip_test

import (
	"testing"

	"github.com/Kate-Mars/go_labs/internal/domain/trip"
)

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *trip.ValidationError
		want string
	}{
		{
			name: "with field",
			err:  &trip.ValidationError{Field: "user_id", Message: "is required"},
			want: "user_id: is required",
		},
		{
			name: "without field",
			err:  &trip.ValidationError{Message: "something went wrong"},
			want: "something went wrong",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

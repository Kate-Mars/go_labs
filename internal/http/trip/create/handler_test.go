package create_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/Kate-Mars/go_labs/internal/generated"
	"github.com/Kate-Mars/go_labs/internal/http/trip/create"
	uc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
)

type fakeUsecase struct{}

func (f *fakeUsecase) Do(ctx context.Context, in uc.Input) (uc.Output, error) {
	return uc.Output{}, nil
}

func TestCreate_RejectsInvalidBodies(t *testing.T) {
	valid := `{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.9,"longitude":30.3},
		"end_point":{"latitude":59.9,"longitude":30.3},
		"price":100
	}`

	cases := []struct {
		name string
		body string
	}{
		{
			name: "missing price",
			body: `{"user_id":"u","driver_id":"d","start_point":{"latitude":1,"longitude":2},"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "null price",
			body: `{"user_id":"u","driver_id":"d","price":null,"start_point":{"latitude":1,"longitude":2},"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "missing start_point",
			body: `{"user_id":"u","driver_id":"d","price":1,"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "null start_point",
			body: `{"user_id":"u","driver_id":"d","price":1,"start_point":null,"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "missing end_point",
			body: `{"user_id":"u","driver_id":"d","price":1,"start_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "missing latitude",
			body: `{"user_id":"u","driver_id":"d","price":1,"start_point":{"longitude":2},"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "null longitude",
			body: `{"user_id":"u","driver_id":"d","price":1,"start_point":{"latitude":1,"longitude":null},"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "missing user_id",
			body: `{"driver_id":"d","price":1,"start_point":{"latitude":1,"longitude":2},"end_point":{"latitude":1,"longitude":2}}`,
		},
		{
			name: "trailing json object",
			body: valid + ` {"extra":1}`,
		},
		{
			name: "trailing garbage",
			body: valid + ` garbage`,
		},
		{
			name: "unknown field",
			body: `{"user_id":"u","driver_id":"d","price":1,"start_point":{"latitude":1,"longitude":2},"end_point":{"latitude":1,"longitude":2},"evil":1}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := create.NewHandler(slog.Default(), &fakeUsecase{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			h.Create(rec, req, api.CreateTripParams{})

			require.Equal(t, http.StatusBadRequest, rec.Code,
				"expected 400 for %q, got %d, body: %s", tc.name, rec.Code, rec.Body.String())
			require.Contains(t, rec.Header().Get("Content-Type"), "application/problem+json",
				"error must be problem+json")
		})
	}
}

package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type HTTPGateway struct {
	baseURL string
	client  *http.Client
}

func NewHTTPGateway(baseURL string, timeout time.Duration) *HTTPGateway {
	return &HTTPGateway{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

type pushRequest struct {
	RequestID   string            `json:"request_id"`
	RecipientID string            `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func (g *HTTPGateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	body := pushRequest{
		RequestID:   uuid.New().String(),
		RecipientID: driverID.String(),
		Kind:        "REQUEST_POSITION",
		Data: map[string]string{
			"trip_id": tripID.String(),
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal push request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/api/v1/push", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("do push request: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("drain push response: %w", err)
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return fmt.Errorf("push rejected: status %d", resp.StatusCode)
	default:
		return fmt.Errorf("push failed: status %d", resp.StatusCode)
	}
}

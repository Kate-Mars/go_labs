package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type BusinessMetrics struct {
	TripsCreated   metric.Int64Counter
	TripsCompleted metric.Int64Counter
}

func NewBusinessMetrics() (*BusinessMetrics, error) {
	meter := otel.Meter("trip-service")

	created, err := meter.Int64Counter(
		"tripgo.trips.created",
		metric.WithDescription("Number of trips created"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, err
	}

	completed, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metric.WithDescription("Number of trips completed"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		TripsCreated:   created,
		TripsCompleted: completed,
	}, nil
}

func (m *BusinessMetrics) IncCreated(ctx context.Context) {
	m.TripsCreated.Add(ctx, 1)
}

func (m *BusinessMetrics) IncCompleted(ctx context.Context) {
	m.TripsCompleted.Add(ctx, 1)
}

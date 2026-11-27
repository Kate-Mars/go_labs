package telemetry

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type BusinessMetrics struct {
	TripsCreated   metric.Int64Counter
	TripsCompleted metric.Int64Counter

	registration metric.Registration
}

func NewBusinessMetrics(pool *pgxpool.Pool) (*BusinessMetrics, error) {
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

	active, err := meter.Int64ObservableGauge(
		"tripgo.trips.active",
		metric.WithDescription("Current number of active trips"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, err
	}

	reg, err := meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			var n int64
			if err := pool.QueryRow(ctx,
				`SELECT count(*) FROM trips WHERE status = 'active'`,
			).Scan(&n); err != nil {
				return nil
			}
			o.ObserveInt64(active, n)
			return nil
		},
		active,
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		TripsCreated:   created,
		TripsCompleted: completed,
		registration:   reg,
	}, nil
}

func (m *BusinessMetrics) IncCreated(ctx context.Context) {
	m.TripsCreated.Add(ctx, 1)
}

func (m *BusinessMetrics) IncCompleted(ctx context.Context) {
	m.TripsCompleted.Add(ctx, 1)
}

func (m *BusinessMetrics) Shutdown() error {
	if m.registration == nil {
		return nil
	}
	return m.registration.Unregister()
}

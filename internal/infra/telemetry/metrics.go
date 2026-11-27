package telemetry

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

type BusinessMetrics struct {
	TripsCreated   metric.Int64Counter
	TripsCompleted metric.Int64Counter
	PositionsSaved metric.Int64Counter

	StaleChecks      metric.Int64Counter
	PositionRequests metric.Int64Counter
	PositionDuration metric.Float64Histogram
	QueueSize        metric.Int64UpDownCounter
	WorkersActive    metric.Int64UpDownCounter

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

	positionsSaved, err := meter.Int64Counter(
		"tripgo.positions.saved",
		metric.WithDescription("Number of positions saved"),
		metric.WithUnit("{position}"),
	)
	if err != nil {
		return nil, err
	}

	staleChecks, err := meter.Int64Counter(
		"tripgo.stale_checks",
		metric.WithDescription("Stale trip checks by result"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		return nil, err
	}

	positionRequests, err := meter.Int64Counter(
		"tripgo.position_requests",
		metric.WithDescription("Position requests to push service by result"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	positionDuration, err := meter.Float64Histogram(
		"tripgo.position_requests.duration",
		metric.WithDescription("Push request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	queueSize, err := meter.Int64UpDownCounter(
		"tripgo.position_requests.queue.size",
		metric.WithDescription("Current size of position requests queue"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, err
	}

	workersActive, err := meter.Int64UpDownCounter(
		"tripgo.position_requests.workers.active",
		metric.WithDescription("Number of active position request workers"),
		metric.WithUnit("{worker}"),
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
		TripsCreated:     created,
		TripsCompleted:   completed,
		PositionsSaved:   positionsSaved,
		StaleChecks:      staleChecks,
		PositionRequests: positionRequests,
		PositionDuration: positionDuration,
		QueueSize:        queueSize,
		WorkersActive:    workersActive,
		registration:     reg,
	}, nil
}

func NewNoopBusinessMetrics() *BusinessMetrics {
	meter := noop.NewMeterProvider().Meter("noop")
	created, _ := meter.Int64Counter("tripgo.trips.created")
	completed, _ := meter.Int64Counter("tripgo.trips.completed")
	positionsSaved, _ := meter.Int64Counter("tripgo.positions.saved")
	staleChecks, _ := meter.Int64Counter("tripgo.stale_checks")
	positionRequests, _ := meter.Int64Counter("tripgo.position_requests")
	positionDuration, _ := meter.Float64Histogram("tripgo.position_requests.duration")
	queueSize, _ := meter.Int64UpDownCounter("tripgo.position_requests.queue.size")
	workersActive, _ := meter.Int64UpDownCounter("tripgo.position_requests.workers.active")

	return &BusinessMetrics{
		TripsCreated:     created,
		TripsCompleted:   completed,
		PositionsSaved:   positionsSaved,
		StaleChecks:      staleChecks,
		PositionRequests: positionRequests,
		PositionDuration: positionDuration,
		QueueSize:        queueSize,
		WorkersActive:    workersActive,
	}
}

func (m *BusinessMetrics) IncCreated(ctx context.Context) {
	if m == nil || m.TripsCreated == nil {
		return
	}
	m.TripsCreated.Add(ctx, 1)
}

func (m *BusinessMetrics) IncCompleted(ctx context.Context) {
	if m == nil || m.TripsCompleted == nil {
		return
	}
	m.TripsCompleted.Add(ctx, 1)
}

func (m *BusinessMetrics) IncPositionSaved(ctx context.Context) {
	if m == nil || m.PositionsSaved == nil {
		return
	}
	m.PositionsSaved.Add(ctx, 1)
}

func (m *BusinessMetrics) IncStaleCheck(ctx context.Context, result string) {
	if m == nil || m.StaleChecks == nil {
		return
	}
	m.StaleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *BusinessMetrics) IncPositionRequest(ctx context.Context, result string) {
	if m == nil || m.PositionRequests == nil {
		return
	}
	m.PositionRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *BusinessMetrics) RecordPositionDuration(ctx context.Context, seconds float64) {
	if m == nil || m.PositionDuration == nil {
		return
	}
	m.PositionDuration.Record(ctx, seconds)
}

func (m *BusinessMetrics) QueueGauge() metric.Int64UpDownCounter {
	if m == nil {
		return nil
	}
	return m.QueueSize
}

func (m *BusinessMetrics) WorkersGauge() metric.Int64UpDownCounter {
	if m == nil {
		return nil
	}
	return m.WorkersActive
}

func (m *BusinessMetrics) Shutdown() error {
	if m == nil || m.registration == nil {
		return nil
	}
	return m.registration.Unregister()
}

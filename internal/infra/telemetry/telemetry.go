package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/Kate-Mars/go_labs/internal/config"
)

type ShutdownFunc func(ctx context.Context) error

func SetupTracing(ctx context.Context, cfg *config.Config) (ShutdownFunc, error) {
	endpoint := strings.TrimPrefix(cfg.OTELEndpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTELServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func SetupMetrics(ctx context.Context, cfg *config.Config) (ShutdownFunc, error) {
	endpoint := strings.TrimPrefix(cfg.OTELEndpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	exp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTELServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exp,
			metric.WithInterval(5*time.Second),
		)),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

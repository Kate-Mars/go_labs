package telemetry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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
	exp, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(newSampler(cfg)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func SetupMetrics(ctx context.Context, cfg *config.Config) (ShutdownFunc, error) {
	exp, err := newMetricExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, err
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

// --- exporters ---

func newTraceExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
	endpoint := stripScheme(cfg.OTELEndpoint)

	switch cfg.OTELExporterProtocol {
	case "http/protobuf":
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
	case "grpc":
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol %q", cfg.OTELExporterProtocol)
	}
}

func newMetricExporter(ctx context.Context, cfg *config.Config) (metric.Exporter, error) {
	endpoint := stripScheme(cfg.OTELEndpoint)

	switch cfg.OTELExporterProtocol {
	case "http/protobuf":
		return otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithInsecure(),
		)
	case "grpc":
		return otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
		)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol %q", cfg.OTELExporterProtocol)
	}
}

func stripScheme(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

// --- resource ---

func buildResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	parsed, err := parseResourceAttributes(cfg.OTELResourceAttributes)
	if err != nil {
		return nil, err
	}

	attrs := make([]attribute.KeyValue, 0, 2+len(parsed))
	attrs = append(attrs,
		semconv.ServiceName(cfg.OTELServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
	)
	attrs = append(attrs, parsed...)

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
	)
}

// parseResourceAttributes разбирает OTEL_RESOURCE_ATTRIBUTES вида
// "service.namespace=tripgo,deployment.environment=local"
func parseResourceAttributes(raw string) ([]attribute.KeyValue, error) {
	if raw == "" {
		return nil, nil
	}
	var attrs []attribute.KeyValue
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid OTEL_RESOURCE_ATTRIBUTES entry %q: expected key=value", part)
		}
		attrs = append(attrs, attribute.String(strings.TrimSpace(k), strings.TrimSpace(v)))
	}
	return attrs, nil
}

// --- sampler ---

func newSampler(cfg *config.Config) sdktrace.Sampler {
	switch cfg.OTELTracesSampler {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(parseRatio(cfg.OTELTracesSamplerArg))
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(parseRatio(cfg.OTELTracesSamplerArg)))
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

func parseRatio(raw string) float64 {
	if raw == "" {
		return 1.0
	}
	r, err := strconv.ParseFloat(raw, 64)
	if err != nil || r < 0 || r > 1 {
		return 1.0
	}
	return r
}

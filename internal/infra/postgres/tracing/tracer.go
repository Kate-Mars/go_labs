package tracing

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName   = "trip-service/postgres"
	unknownLabel = "unknown"
)

type queryInfo struct {
	span  trace.Span
	op    string
	table string
	start time.Time
}

type QueryTracer struct {
	tracer   trace.Tracer
	duration metric.Float64Histogram
	mu       sync.Mutex
	spans    map[*pgx.Conn]queryInfo
}

func NewQueryTracer() *QueryTracer {
	meter := otel.Meter(tracerName)
	duration, _ := meter.Float64Histogram(
		"db.client.operation.duration",
		metric.WithDescription("Database operation duration in seconds"),
		metric.WithUnit("s"),
	)

	return &QueryTracer{
		tracer:   otel.Tracer(tracerName),
		duration: duration,
		spans:    make(map[*pgx.Conn]queryInfo),
	}
}

func (t *QueryTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	op, table := parseSQL(data.SQL)

	ctx, span := t.tracer.Start(ctx, "db.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", op),
			attribute.String("db.sql.table", table),
			attribute.String("db.statement", data.SQL),
		),
	)

	t.mu.Lock()
	t.spans[conn] = queryInfo{span: span, op: op, table: table, start: time.Now()}
	t.mu.Unlock()

	return ctx
}

func (t *QueryTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	t.mu.Lock()
	info, ok := t.spans[conn]
	if ok {
		delete(t.spans, conn)
	}
	t.mu.Unlock()

	if !ok {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("db.operation", info.op),
		attribute.String("db.sql.table", info.table),
		attribute.Bool("error", data.Err != nil),
	}

	if data.Err != nil {
		info.span.RecordError(data.Err)
		info.span.SetStatus(codes.Error, data.Err.Error())
	} else {
		info.span.SetStatus(codes.Ok, "")
	}
	info.span.End()

	t.duration.Record(ctx, time.Since(info.start).Seconds(), metric.WithAttributes(attrs...))
}

// parseSQL — очень грубый парсер: первое слово = операция,
// после FROM/INTO/UPDATE — таблица. Достаточно для наших запросов.
func parseSQL(sql string) (op, table string) {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(sql)))
	if len(fields) == 0 {
		return unknownLabel, unknownLabel
	}

	op = fields[0]
	switch op {
	case "SELECT":
		for i, f := range fields {
			if f == "FROM" && i+1 < len(fields) {
				return op, strings.Trim(fields[i+1], `"`)
			}
		}
	case "INSERT":
		if len(fields) >= 3 && fields[1] == "INTO" {
			return op, strings.Trim(fields[2], `"`)
		}
	case "UPDATE":
		if len(fields) >= 2 {
			return op, strings.Trim(fields[1], `"`)
		}
	case "DELETE":
		for i, f := range fields {
			if f == "FROM" && i+1 < len(fields) {
				return op, strings.Trim(fields[i+1], `"`)
			}
		}
	}
	return op, unknownLabel
}

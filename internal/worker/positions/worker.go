package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Kate-Mars/go_labs/internal/gateway/push"
	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type Worker struct {
	log     *slog.Logger
	queue   *Queue
	gateway push.Gateway
	metrics *telemetry.BusinessMetrics
	timeout time.Duration
}

func NewWorker(log *slog.Logger, q *Queue, gw push.Gateway, m *telemetry.BusinessMetrics, timeout time.Duration) *Worker {
	return &Worker{log: log, queue: q, gateway: gw, metrics: m, timeout: timeout}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		task, ok := w.queue.Take(ctx)
		if !ok {
			return
		}

		if !w.queue.IsInflight(task.TripID) {
			if w.queue.Done(task.TripID) && w.metrics != nil {
				if g := w.metrics.QueueGauge(); g != nil {
					g.Add(ctx, -1)
				}
			}
			continue
		}

		w.process(ctx, task)
		if w.queue.Done(task.TripID) && w.metrics != nil {
			if g := w.metrics.QueueGauge(); g != nil {
				g.Add(ctx, -1)
			}
		}
	}
}

func (w *Worker) process(ctx context.Context, t model.StaleTrip) {
	if w.metrics != nil {
		if g := w.metrics.WorkersGauge(); g != nil {
			g.Add(ctx, 1)
			defer g.Add(ctx, -1)
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	start := time.Now()
	err := w.gateway.RequestPosition(reqCtx, t.DriverID, t.TripID)
	dur := time.Since(start).Seconds()
	if w.metrics != nil {
		w.metrics.RecordPositionDuration(ctx, dur)
	}

	if err != nil {
		if w.metrics != nil {
			w.metrics.IncPositionRequest(ctx, "error")
		}
		w.log.Warn("push request failed",
			"err", err,
			"trip_id", t.TripID,
			"driver_id", t.DriverID,
		)
		return
	}

	if w.metrics != nil {
		w.metrics.IncPositionRequest(ctx, "ok")
	}
}

func Start(ctx context.Context, n int, w *Worker) func() {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			w.Run(ctx)
		}()
	}
	return wg.Wait
}

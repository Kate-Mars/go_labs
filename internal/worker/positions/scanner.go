package positions

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

type staleRepo interface {
	FindStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error)
}

type Scanner struct {
	log        *slog.Logger
	repo       staleRepo
	queue      *Queue
	metrics    *telemetry.BusinessMetrics
	interval   time.Duration
	staleAfter time.Duration
}

func NewScanner(log *slog.Logger, repo staleRepo, q *Queue, m *telemetry.BusinessMetrics, interval, staleAfter time.Duration) *Scanner {
	return &Scanner{
		log:        log,
		repo:       repo,
		queue:      q,
		metrics:    m,
		interval:   interval,
		staleAfter: staleAfter,
	}
}

func (s *Scanner) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	stale, err := s.repo.FindStale(ctx, s.staleAfter)
	if err != nil {
		s.log.Error("find stale trips failed", "err", err)
		return
	}

	for _, t := range stale {
		result := s.queue.TryEnqueue(t)

		if s.metrics != nil {
			s.metrics.IncStaleCheck(ctx, result)
		}

		switch result {
		case "queued":
			if s.metrics != nil {
				if g := s.metrics.QueueGauge(); g != nil {
					g.Add(ctx, 1)
				}
			}
		case "dropped":
			s.log.Warn("queue is full, dropping stale trip",
				"trip_id", t.TripID,
				"driver_id", t.DriverID,
			)
		}
	}
}

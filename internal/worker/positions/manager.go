package positions

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Kate-Mars/go_labs/internal/gateway/push"
	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
)

type Manager struct {
	log     *slog.Logger
	scanner *Scanner
	queue   *Queue
	worker  *Worker
	workers int32
	metrics *telemetry.BusinessMetrics

	scanCancel context.CancelFunc
	scanDone   chan struct{}
	workersWg  func()
}

func NewManager(
	log *slog.Logger,
	repo staleRepo,
	gw push.Gateway,
	metrics *telemetry.BusinessMetrics,
	checkInterval, staleAfter time.Duration,
	workers int32,
	queueSize int32,
	pushTimeout time.Duration,
) *Manager {
	q := NewQueue(int(queueSize))
	scanner := NewScanner(log, repo, q, metrics, checkInterval, staleAfter)
	worker := NewWorker(log, q, gw, metrics, pushTimeout)

	return &Manager{
		log:     log,
		scanner: scanner,
		queue:   q,
		worker:  worker,
		workers: workers,
		metrics: metrics,
	}
}

func (m *Manager) Start(parentCtx context.Context) {
	scanCtx, scanCancel := context.WithCancel(parentCtx)
	m.scanCancel = scanCancel
	m.scanDone = make(chan struct{})

	go func() {
		defer close(m.scanDone)
		m.scanner.Run(scanCtx)
	}()

	m.workersWg = Start(parentCtx, int(m.workers), m.worker)
}

func (m *Manager) Stop(ctx context.Context) error {
	if m.scanCancel != nil {
		m.scanCancel()
		select {
		case <-m.scanDone:
		case <-ctx.Done():
			m.log.Warn("scanner did not stop in time")
		}
	}

	m.queue.Close()

	done := make(chan struct{})
	go func() {
		if m.workersWg != nil {
			m.workersWg()
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) Cancel(tripID uuid.UUID) bool {
	if m.queue.Cancel(tripID) {
		if m.metrics != nil {
			if g := m.metrics.QueueGauge(); g != nil {
				g.Add(context.Background(), -1)
			}
		}
		return true
	}
	return false
}

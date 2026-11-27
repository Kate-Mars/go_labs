package container

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/config"
	"github.com/Kate-Mars/go_labs/internal/gateway/push"
	apphttp "github.com/Kate-Mars/go_labs/internal/http"
	"github.com/Kate-Mars/go_labs/internal/http/middleware"
	createh "github.com/Kate-Mars/go_labs/internal/http/trip/create"
	finishh "github.com/Kate-Mars/go_labs/internal/http/trip/finish"
	geth "github.com/Kate-Mars/go_labs/internal/http/trip/get"
	positionsh "github.com/Kate-Mars/go_labs/internal/http/trip/positions"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres/transaction"
	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
	idemrepo "github.com/Kate-Mars/go_labs/internal/repository/idempotency"
	triprepo "github.com/Kate-Mars/go_labs/internal/repository/trip"
	createuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
	finishuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/finish"
	getuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/get"
	listposuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/positions/list"
	saveposuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/positions/save"
	positionsworker "github.com/Kate-Mars/go_labs/internal/worker/positions"
)

type Container struct {
	Pool      *pgxpool.Pool
	Router    *apphttp.Server
	Metrics   *middleware.Metrics
	Positions *positionsworker.Manager
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	bizMetrics, err := telemetry.NewBusinessMetrics(pool)
	if err != nil {
		pool.Close()
		return nil, err
	}

	txManager := transaction.NewManager(pool)
	tripRepo := triprepo.New(pool)
	idemRepo := idemrepo.New(pool)

	pushGateway := push.NewHTTPGateway(cfg.PushHTTPURL, cfg.PushRequestTimeout)

	posManager := positionsworker.NewManager(
		log,
		tripRepo,
		pushGateway,
		bizMetrics,
		cfg.PositionCheckInterval,
		cfg.PositionStaleAfter,
		cfg.PositionWorkers,
		cfg.PositionQueueSize,
		cfg.PushRequestTimeout,
	)

	createUC := createuc.New(tripRepo, idemRepo, txManager, bizMetrics)
	getUC := getuc.New(tripRepo)
	finishUC := finishuc.New(tripRepo, txManager, bizMetrics)
	savePosUC := saveposuc.New(tripRepo, txManager, bizMetrics, nil)
	listPosUC := listposuc.New(tripRepo)

	createH := createh.NewHandler(log, createUC)
	getH := geth.NewHandler(log, getUC)
	finishH := finishh.NewHandler(log, finishUC)
	savePosH := positionsh.NewSaveHandler(log, savePosUC, posManager)
	listPosH := positionsh.NewListHandler(log, listPosUC)

	server := apphttp.NewServer(log, pool, createH, getH, finishH, savePosH, listPosH)

	metrics, err := middleware.NewMetrics()
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Container{
		Pool:      pool,
		Router:    server,
		Metrics:   metrics,
		Positions: posManager,
	}, nil
}

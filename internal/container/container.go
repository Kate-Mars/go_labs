package container

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/config"
	apphttp "github.com/Kate-Mars/go_labs/internal/http"
	"github.com/Kate-Mars/go_labs/internal/http/middleware"
	createh "github.com/Kate-Mars/go_labs/internal/http/trip/create"
	finishh "github.com/Kate-Mars/go_labs/internal/http/trip/finish"
	geth "github.com/Kate-Mars/go_labs/internal/http/trip/get"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres/transaction"
	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
	idemrepo "github.com/Kate-Mars/go_labs/internal/repository/idempotency"
	triprepo "github.com/Kate-Mars/go_labs/internal/repository/trip"
	createuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
	finishuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/finish"
	getuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/get"
)

type Container struct {
	Pool    *pgxpool.Pool
	Router  *apphttp.Server
	Metrics *middleware.Metrics
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	bizMetrics, err := telemetry.NewBusinessMetrics()
	if err != nil {
		pool.Close()
		return nil, err
	}

	txManager := transaction.NewManager(pool)
	tripRepo := triprepo.New(pool)
	idemRepo := idemrepo.New(pool)

	createUC := createuc.New(tripRepo, idemRepo, txManager, bizMetrics)
	getUC := getuc.New(tripRepo)
	finishUC := finishuc.New(tripRepo, txManager, bizMetrics)

	createH := createh.NewHandler(log, createUC)
	getH := geth.NewHandler(log, getUC)
	finishH := finishh.NewHandler(log, finishUC)

	server := apphttp.NewServer(log, createH, getH, finishH)

	metrics, err := middleware.NewMetrics()
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Container{
		Pool:    pool,
		Router:  server,
		Metrics: metrics,
	}, nil
}

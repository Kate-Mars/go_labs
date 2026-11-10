package container

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/config"
	httprouter "github.com/Kate-Mars/go_labs/internal/http"
	"github.com/Kate-Mars/go_labs/internal/http/trip/create"
	"github.com/Kate-Mars/go_labs/internal/http/trip/finish"
	"github.com/Kate-Mars/go_labs/internal/http/trip/get"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres"
	"github.com/Kate-Mars/go_labs/internal/infra/postgres/transaction"
	idemrepo "github.com/Kate-Mars/go_labs/internal/repository/idempotency"
	triprepo "github.com/Kate-Mars/go_labs/internal/repository/trip"
	createuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/create"
	finishuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/finish"
	getuc "github.com/Kate-Mars/go_labs/internal/usecase/trip/get"
)

type Container struct {
	Pool   *pgxpool.Pool
	Router *httprouter.Server
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	txManager := transaction.NewManager(pool)

	tripRepo := triprepo.New(pool)
	idemRepo := idemrepo.New(pool)

	createUC := createuc.New(tripRepo, idemRepo, txManager)
	getUC := getuc.New(tripRepo)
	finishUC := finishuc.New(tripRepo, txManager)

	createH := create.NewHandler(log, createUC)
	getH := get.NewHandler(log, getUC)
	finishH := finish.NewHandler(log, finishUC)

	server := httprouter.NewServer(log, createH, getH, finishH)

	return &Container{Pool: pool, Router: server}, nil
}

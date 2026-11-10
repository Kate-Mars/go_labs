package trip

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/infra/postgres/txctx"
)

const (
	tableTrips         = "trips"
	tableStatusHistory = "trip_status_history"
)

const (
	pgUniqueViolation     = "23505"
	driverActiveUniqIndex = "trips_one_active_per_driver_uniq"
)

type Repo struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *Repo) exec(ctx context.Context) txctx.DBTX {
	if tx, ok := txctx.FromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == constraint
}

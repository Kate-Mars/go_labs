package idempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/infra/postgres/txctx"
	model "github.com/Kate-Mars/go_labs/internal/model/trip"
)

const tableIdempotency = "idempotency_keys"

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

func (r *Repo) InsertIfAbsent(ctx context.Context, rec model.IdempotencyRecord) (bool, error) {
	q, args, err := r.sb.
		Insert(tableIdempotency).
		Columns("key", "trip_id", "request_hash").
		Values(rec.Key, rec.TripID, rec.RequestHash).
		Suffix("ON CONFLICT (key) DO NOTHING").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build insert idempotency: %w", err)
	}

	tag, err := r.exec(ctx).Exec(ctx, q, args...)
	if err != nil {
		return false, fmt.Errorf("insert idempotency: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) Get(ctx context.Context, key uuid.UUID) (*model.IdempotencyRecord, error) {
	q, args, err := r.sb.
		Select("key", "trip_id", "request_hash").
		From(tableIdempotency).
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select idempotency: %w", err)
	}

	row := r.exec(ctx).QueryRow(ctx, q, args...)
	var rec model.IdempotencyRecord
	if err := row.Scan(&rec.Key, &rec.TripID, &rec.RequestHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select idempotency: %w", err)
	}
	return &rec, nil
}

func (r *Repo) SetTripID(ctx context.Context, key, tripID uuid.UUID) error {
	q, args, err := r.sb.
		Update(tableIdempotency).
		Set("trip_id", tripID).
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update idempotency: %w", err)
	}
	if _, err := r.exec(ctx).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("update idempotency: %w", err)
	}
	return nil
}

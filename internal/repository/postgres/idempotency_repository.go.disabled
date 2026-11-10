package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const idempotencyTable = "idempotency_keys"

type IdempotencyRecord struct {
	Key         uuid.UUID
	TripID      *uuid.UUID
	RequestHash []byte
}

type IdempotencyRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

// NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// (r *IdempotencyRepository) exec(ctx context.Context) DBTX

func (r *IdempotencyRepository) exec(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

// (r *IdempotencyRepository) Get(ctx context.Context, key uuid.UUID) (*IdempotencyRecord, error)

func (r *IdempotencyRepository) Get(ctx context.Context, key uuid.UUID) (*IdempotencyRecord, error) {
	q, args, err := r.sb.
		Select("key", "trip_id", "request_hash").
		From(idempotencyTable).
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select idempotency: %w", err)
	}

	row := r.exec(ctx).QueryRow(ctx, q, args...)
	var rec IdempotencyRecord
	if err := row.Scan(&rec.Key, &rec.TripID, &rec.RequestHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select idempotency: %w", err)
	}
	return &rec, nil
}

// (r *IdempotencyRepository) InsertIfAbsent(ctx context.Context, rec IdempotencyRecord) (bool, error)

func (r *IdempotencyRepository) InsertIfAbsent(ctx context.Context, rec IdempotencyRecord) (bool, error) {
	q, args, err := r.sb.
		Insert(idempotencyTable).
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

// (r *IdempotencyRepository) SetTripID(ctx context.Context, key, tripID uuid.UUID) error

func (r *IdempotencyRepository) SetTripID(ctx context.Context, key, tripID uuid.UUID) error {
	q, args, err := r.sb.
		Update(idempotencyTable).
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

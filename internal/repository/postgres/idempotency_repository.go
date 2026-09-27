package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const idempotencyTable = "idempotency_keys"

var ErrIdempotencyKeyExists = errors.New("idempotency key already exists")

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

// (r *IdempotencyRepository) Insert(ctx context.Context, rec IdempotencyRecord) error

func (r *IdempotencyRepository) Insert(ctx context.Context, rec IdempotencyRecord) error {
	q, args, err := r.sb.
		Insert(idempotencyTable).
		Columns("key", "trip_id", "request_hash").
		Values(rec.Key, rec.TripID, rec.RequestHash).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert idempotency: %w", err)
	}

	_, err = r.exec(ctx).Exec(ctx, q, args...)
	if err != nil {
		if isUniqueViolation(err, "idempotency_keys_pkey") {
			return ErrIdempotencyKeyExists
		}
		return fmt.Errorf("insert idempotency: %w", err)
	}
	return nil
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

// (r *IdempotencyRepository) Savepoint(ctx context.Context, name string) error

func (r *IdempotencyRepository) Savepoint(ctx context.Context, name string) error {
	if _, err := r.exec(ctx).Exec(ctx, "SAVEPOINT "+name); err != nil {
		return fmt.Errorf("savepoint %s: %w", name, err)
	}
	return nil
}

//  (r *IdempotencyRepository) RollbackToSavepoint(ctx context.Context, name string) error

func (r *IdempotencyRepository) RollbackToSavepoint(ctx context.Context, name string) error {
	if _, err := r.exec(ctx).Exec(ctx, "ROLLBACK TO SAVEPOINT "+name); err != nil {
		return fmt.Errorf("rollback to savepoint %s: %w", name, err)
	}
	return nil
}

var _ pgconn.CommandTag

package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kate-Mars/go_labs/internal/infra/postgres/txctx"
)

const rollbackTimeout = 5 * time.Second

type Manager struct {
	pool *pgxpool.Pool
}

func NewManager(pool *pgxpool.Pool) *Manager {
	return &Manager{pool: pool}
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := txctx.FromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if p := recover(); p != nil {
			if rbErr := rollback(tx); rbErr != nil {
				err = errors.Join(fmt.Errorf("panic: %v", p), rbErr)
			}
			panic(p)
		}
		if !committed {
			if rbErr := rollback(tx); rbErr != nil {
				err = errors.Join(err, rbErr)
			}
		}
	}()

	txCtx := txctx.WithTx(ctx, tx)

	if err = fn(txCtx); err != nil {
		return err
	}

	if cErr := tx.Commit(ctx); cErr != nil {
		return fmt.Errorf("commit tx: %w", cErr)
	}
	committed = true
	return nil
}

func rollback(tx pgx.Tx) error {
	ctx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
	defer cancel()
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback tx: %w", err)
	}
	return nil
}

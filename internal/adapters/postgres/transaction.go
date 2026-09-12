package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/adapters/postgres/generated"
)

// WithinTransaction owns the database transaction lifecycle. Callers must
// perform every related write through tx and return an error to roll back.
func WithinTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
			return fmt.Errorf("rollback transaction after %v: %w", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// WithinTransactionWithQueries exposes only generated query methods to an
// adapter repository while keeping transaction ownership in one place.
func WithinTransactionWithQueries(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, *generated.Queries) error) error {
	return WithinTransaction(ctx, pool, func(transactionContext context.Context, tx pgx.Tx) error {
		return fn(transactionContext, generated.New(tx))
	})
}

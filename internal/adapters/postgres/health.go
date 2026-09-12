package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}

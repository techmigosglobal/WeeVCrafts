package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	redisclient "github.com/redis/go-redis/v9"

	"github.com/wecratfs/commerce/internal/adapters/meilisearch"
	"github.com/wecratfs/commerce/internal/adapters/postgres"
	redisadapter "github.com/wecratfs/commerce/internal/adapters/redis"
	"github.com/wecratfs/commerce/internal/config"
	"github.com/wecratfs/commerce/internal/ports"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.FromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("create PostgreSQL pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	if err := postgres.ApplyMigrations(ctx, pool); err != nil {
		logger.Error("apply PostgreSQL migrations", "error", err)
		os.Exit(1)
	}
	documents, err := postgres.NewProductSearchRepository(pool).ListApprovedForIndex(ctx)
	if err != nil {
		logger.Error("read approved catalogue", "error", err)
		os.Exit(1)
	}
	var indexer ports.SearchIndexer = meilisearch.NewSearchEngineWithKey(cfg.MeiliAddress, cfg.MeiliAPIKey)
	if err := indexer.Rebuild(ctx, documents); err != nil {
		logger.Error("rebuild Meilisearch index", "error", err)
		os.Exit(1)
	}
	redisClient := redisclient.NewClient(&redisclient.Options{Addr: cfg.RedisAddress})
	defer redisClient.Close()
	if err := redisadapter.NewCache(redisClient).DeletePrefix(ctx, "catalog:search:"); err != nil {
		logger.Warn("invalidate search cache after rebuild", "error", err)
	}
	fmt.Printf("rebuilt products index with %d approved documents\n", len(documents))
}

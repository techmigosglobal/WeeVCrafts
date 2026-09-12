package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisclient "github.com/redis/go-redis/v9"

	"github.com/wecratfs/commerce/internal/ports"
)

type Cache struct {
	client *redisclient.Client
}

func NewCache(client *redisclient.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redisclient.Nil) {
		return nil, ports.ErrCacheMiss
	}
	return value, err
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	if ttlSeconds <= 0 {
		return nil
	}
	return c.client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) DeletePrefix(ctx context.Context, prefix string) error {
	if prefix == "" {
		return errors.New("cache invalidation prefix is required")
	}
	const maxKeys = 1000
	var cursor uint64
	deleted := 0
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 128).Result()
		if err != nil {
			return fmt.Errorf("scan cache keys: %w", err)
		}
		if len(keys) > maxKeys-deleted {
			keys = keys[:maxKeys-deleted]
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete cache keys: %w", err)
			}
			deleted += len(keys)
		}
		if deleted >= maxKeys {
			return fmt.Errorf("cache invalidation exceeded %d keys", maxKeys)
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

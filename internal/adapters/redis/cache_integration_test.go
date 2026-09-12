package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/wecratfs/commerce/internal/ports"
)

func TestCacheDeletePrefixInvalidatesSearchEntries(t *testing.T) {
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		t.Skip("REDIS_ADDR is required for Redis integration tests")
	}
	client := redisclient.NewClient(&redisclient.Options{Addr: address})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}

	prefix := fmt.Sprintf("catalog:search:integration:%d:", time.Now().UnixNano())
	cache := NewCache(client)
	if err := cache.Set(ctx, prefix+"one", []byte(`{"total":1}`), 30); err != nil {
		t.Fatalf("set first cache entry: %v", err)
	}
	if err := cache.Set(ctx, prefix+"two", []byte(`{"total":2}`), 30); err != nil {
		t.Fatalf("set second cache entry: %v", err)
	}
	if err := cache.DeletePrefix(ctx, prefix); err != nil {
		t.Fatalf("delete cache prefix: %v", err)
	}
	if _, err := cache.Get(ctx, prefix+"one"); err != ports.ErrCacheMiss {
		t.Fatalf("expected first cache entry to be invalidated, got %v", err)
	}
	if _, err := cache.Get(ctx, prefix+"two"); err != ports.ErrCacheMiss {
		t.Fatalf("expected second cache entry to be invalidated, got %v", err)
	}
}

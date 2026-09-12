package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

func TestRateLimiterBoundsAuthenticationAttempts(t *testing.T) {
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

	limiter := NewRateLimiter(client)
	key := fmt.Sprintf("integration-auth-%d", time.Now().UnixNano())
	for attempt := 1; attempt <= 2; attempt++ {
		allowed, err := limiter.Allow(ctx, key, 2, 30*time.Second)
		if err != nil || !allowed {
			t.Fatalf("attempt %d unexpectedly denied: allowed=%v err=%v", attempt, allowed, err)
		}
	}
	allowed, err := limiter.Allow(ctx, key, 2, 30*time.Second)
	if err != nil {
		t.Fatalf("third attempt: %v", err)
	}
	if allowed {
		t.Fatal("third authentication attempt was not rate limited")
	}
}

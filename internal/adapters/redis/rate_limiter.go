package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

type RateLimiter struct{ client *redisclient.Client }

func NewRateLimiter(client *redisclient.Client) *RateLimiter { return &RateLimiter{client: client} }

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return false, nil
	}
	redisKey := "wecratfs:rate:" + rateHash(key) + ":" + strconv.FormatInt(time.Now().UTC().UnixNano()/int64(window), 10)
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = r.client.Expire(ctx, redisKey, window)
	}
	return count <= int64(limit), nil
}

func rateHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

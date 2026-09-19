package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/ports"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionStore keeps server-side sessions in PostgreSQL so Netlify function
// instances can safely share login state. The browser token is hashed for
// lookups; its original value is retained for the account's active-session UI.
type SessionStore struct{ pool *pgxpool.Pool }

func NewSessionStore(pool *pgxpool.Pool) *SessionStore { return &SessionStore{pool: pool} }

func (s *SessionStore) Create(ctx context.Context, session ports.SessionRecord) error {
	if session.ID == "" || len(session.ID) > 256 || session.UserID <= 0 || !session.ExpiresAt.After(time.Now()) {
		return errors.New("invalid session record")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = session.CreatedAt
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO runtime_web_sessions
			(session_hash, session_id, user_id, role_slugs, csrf_token, user_agent, ip_address, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		sessionHash(session.ID), session.ID, session.UserID, session.RoleSlugs, session.CSRFToken,
		session.UserAgent, session.IPAddress, session.CreatedAt, session.LastSeenAt, session.ExpiresAt)
	return err
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (ports.SessionRecord, error) {
	if sessionID == "" {
		return ports.SessionRecord{}, ErrSessionNotFound
	}
	var session ports.SessionRecord
	err := s.pool.QueryRow(ctx, `
		SELECT session_id, user_id, role_slugs, csrf_token, user_agent, ip_address, created_at, last_seen_at, expires_at
		FROM runtime_web_sessions WHERE session_hash = $1 AND expires_at > NOW()`, sessionHash(sessionID)).
		Scan(&session.ID, &session.UserID, &session.RoleSlugs, &session.CSRFToken, &session.UserAgent,
			&session.IPAddress, &session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.SessionRecord{}, ErrSessionNotFound
	}
	return session, err
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM runtime_web_sessions WHERE session_hash = $1`, sessionHash(sessionID))
	return err
}

func (s *SessionStore) DeleteAll(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM runtime_web_sessions WHERE user_id = $1`, userID)
	return err
}

func (s *SessionStore) List(ctx context.Context, userID int64) ([]ports.SessionRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT session_id, user_id, role_slugs, csrf_token, user_agent, ip_address, created_at, last_seen_at, expires_at
		FROM runtime_web_sessions WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY last_seen_at DESC, created_at DESC LIMIT 128`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := make([]ports.SessionRecord, 0, 8)
	for rows.Next() {
		var session ports.SessionRecord
		if err := rows.Scan(&session.ID, &session.UserID, &session.RoleSlugs, &session.CSRFToken,
			&session.UserAgent, &session.IPAddress, &session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func sessionHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// RateLimiter uses one atomic upsert per fixed window. This keeps auth limits
// consistent across cold and warm serverless invocations.
type RateLimiter struct{ pool *pgxpool.Pool }

func NewRateLimiter(pool *pgxpool.Pool) *RateLimiter { return &RateLimiter{pool: pool} }

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return false, nil
	}
	if key == "" || len(key) > 4096 || window > 24*time.Hour {
		return false, fmt.Errorf("invalid rate-limit request")
	}
	now := time.Now().UTC()
	nanos := window.Nanoseconds()
	slot := now.UnixNano() / nanos
	expiresAt := time.Unix(0, (slot+1)*nanos).UTC()
	bucketDigest := sha256.Sum256([]byte(key + ":" + fmt.Sprint(slot)))
	bucket := hex.EncodeToString(bucketDigest[:])
	var hits int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO runtime_rate_limits (bucket_key, hits, expires_at)
		VALUES ($1, 1, $2)
		ON CONFLICT (bucket_key) DO UPDATE SET hits = runtime_rate_limits.hits + 1
		RETURNING hits`, bucket, expiresAt).Scan(&hits)
	return hits <= limit, err
}

// Cache contains only disposable search data; PostgreSQL failures are surfaced
// so the search service can follow its existing cache-failure policy.
type Cache struct{ pool *pgxpool.Pool }

func NewCache(pool *pgxpool.Pool) *Cache { return &Cache{pool: pool} }

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := c.pool.QueryRow(ctx, `SELECT value FROM runtime_cache WHERE cache_key = $1 AND expires_at > NOW()`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrCacheMiss
	}
	return value, err
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	if key == "" || len(key) > 2048 || len(value) > 1<<20 {
		return errors.New("invalid cache entry")
	}
	if ttlSeconds <= 0 {
		return nil
	}
	_, err := c.pool.Exec(ctx, `
		INSERT INTO runtime_cache (cache_key, value, expires_at)
		VALUES ($1, $2, NOW() + ($3 * INTERVAL '1 second'))
		ON CONFLICT (cache_key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at`, key, value, ttlSeconds)
	return err
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	_, err := c.pool.Exec(ctx, `DELETE FROM runtime_cache WHERE cache_key = $1`, key)
	return err
}

func (c *Cache) DeletePrefix(ctx context.Context, prefix string) error {
	if prefix == "" || len(prefix) > 2048 {
		return errors.New("cache invalidation prefix is required")
	}
	const maxKeys = 1000
	deleted := 0
	for {
		if deleted >= maxKeys {
			return fmt.Errorf("cache invalidation exceeded %d keys", maxKeys)
		}
		batchSize := min(128, maxKeys-deleted)
		var batchDeleted int
		err := c.pool.QueryRow(ctx, `
			WITH candidates AS (
				SELECT cache_key FROM runtime_cache
				WHERE left(cache_key, char_length($1)) = $1
				LIMIT $2
			), removed AS (
				DELETE FROM runtime_cache c USING candidates x
				WHERE c.cache_key = x.cache_key
				RETURNING 1
			)
			SELECT count(*) FROM removed`, prefix, batchSize).Scan(&batchDeleted)
		if err != nil {
			return err
		}
		deleted += batchDeleted
		if batchDeleted < batchSize {
			return nil
		}
	}
}

// CleanupRuntimeState removes expired ephemeral rows in bounded batches so
// Netlify's scheduled function cannot perform unbounded table maintenance.
func CleanupRuntimeState(ctx context.Context, pool *pgxpool.Pool, limit int) error {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	queries := []string{
		`WITH expired AS (SELECT session_hash FROM runtime_web_sessions WHERE expires_at <= NOW() ORDER BY expires_at LIMIT $1) DELETE FROM runtime_web_sessions s USING expired e WHERE s.session_hash = e.session_hash`,
		`WITH expired AS (SELECT bucket_key FROM runtime_rate_limits WHERE expires_at <= NOW() ORDER BY expires_at LIMIT $1) DELETE FROM runtime_rate_limits r USING expired e WHERE r.bucket_key = e.bucket_key`,
		`WITH expired AS (SELECT cache_key FROM runtime_cache WHERE expires_at <= NOW() ORDER BY expires_at LIMIT $1) DELETE FROM runtime_cache c USING expired e WHERE c.cache_key = e.cache_key`,
	}
	for _, query := range queries {
		if _, err := pool.Exec(ctx, query, limit); err != nil {
			return err
		}
	}
	return nil
}

package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestPostgresSessionStorePersistsAndRevokesSessions(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx := context.Background()
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name) VALUES ($1, 'Session fixture') RETURNING id`, fmt.Sprintf("session-%d@example.invalid", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	store := NewSessionStore(pool)
	want := ports.SessionRecord{
		ID: "opaque-session-token", UserID: userID, RoleSlugs: []string{"customer"}, CSRFToken: "csrf-token",
		UserAgent: "integration test", IPAddress: "192.0.2.1", CreatedAt: time.Now().UTC(),
		LastSeenAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC(),
	}
	if err := store.Create(ctx, want); err != nil {
		t.Fatalf("create session: %v", err)
	}
	got, err := store.Get(ctx, want.ID)
	if err != nil || got.UserID != want.UserID || got.CSRFToken != want.CSRFToken || len(got.RoleSlugs) != 1 || got.RoleSlugs[0] != "customer" {
		t.Fatalf("session did not persist: got=%#v err=%v", got, err)
	}
	listed, err := store.List(ctx, userID)
	if err != nil || len(listed) != 1 || listed[0].ID != want.ID {
		t.Fatalf("session list mismatch: got=%#v err=%v", listed, err)
	}
	if err := store.DeleteAll(ctx, userID); err != nil {
		t.Fatalf("revoke sessions: %v", err)
	}
	if _, err := store.Get(ctx, want.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("revoked session remained available: %v", err)
	}
}

func TestPostgresRateLimiterEnforcesConcurrentLimit(t *testing.T) {
	pool := openIntegrationPool(t)
	limiter := NewRateLimiter(pool)
	key := fmt.Sprintf("netlify-rate-%d", time.Now().UnixNano())
	const callers, limit = 20, 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := limiter.Allow(context.Background(), key, limit, time.Minute)
			if err != nil {
				t.Errorf("allow request: %v", err)
				return
			}
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowedCount != limit {
		t.Fatalf("allowed %d of %d concurrent requests, want %d", allowedCount, callers, limit)
	}
}

func TestPostgresCacheStoresAndInvalidatesOnlyMatchingPrefix(t *testing.T) {
	pool := openIntegrationPool(t)
	cache := NewCache(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	firstKey, secondKey := "catalog:"+suffix+":first", "product:"+suffix
	if err := cache.Set(context.Background(), firstKey, []byte("cached"), 60); err != nil {
		t.Fatal(err)
	}
	if err := cache.Set(context.Background(), secondKey, []byte("keep"), 60); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Get(context.Background(), firstKey)
	if err != nil || !bytes.Equal(got, []byte("cached")) {
		t.Fatalf("cache read mismatch: got=%q err=%v", got, err)
	}
	if err := cache.DeletePrefix(context.Background(), "catalog:"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get(context.Background(), firstKey); !errors.Is(err, ports.ErrCacheMiss) {
		t.Fatalf("matching cache key was not removed: %v", err)
	}
	if _, err := cache.Get(context.Background(), secondKey); err != nil {
		t.Fatalf("unrelated cache key was removed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM runtime_cache WHERE cache_key IN ($1, $2)`, firstKey, secondKey)
	})
}

func TestPostgresMediaStorageAcceptsOnlyOwnerBoundSmallUploads(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 0, 1)
	storage := NewDatabaseMediaStorage(pool)
	repository := NewMediaRepository(pool)
	payload := []byte("small image fixture")
	objectKey := fmt.Sprintf("products/%d/%d", fixture.ownerID, time.Now().UnixNano())
	mediaID, err := repository.PrepareUpload(context.Background(), fixture.ownerID, fixture.productID, objectKey, "image/png", int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveObject(context.Background(), fixture.ownerID+1, objectKey, "image/png", payload); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("another seller could write this upload: %v", err)
	}
	if err := storage.SaveObject(context.Background(), fixture.ownerID, objectKey, "image/png", payload); err != nil {
		t.Fatalf("owner upload failed: %v", err)
	}
	info, err := storage.InspectObject(context.Background(), objectKey)
	if err != nil || info.ContentType != "image/png" || info.ByteSize != int64(len(payload)) {
		t.Fatalf("stored object metadata mismatch: info=%#v err=%v", info, err)
	}
	read, readInfo, err := storage.ReadObject(context.Background(), objectKey)
	if err != nil || !bytes.Equal(read, payload) || readInfo.ContentType != "image/png" {
		t.Fatalf("stored object did not round-trip: bytes=%q info=%#v err=%v", read, readInfo, err)
	}
	if _, err := storage.CreateUploadURL(context.Background(), ports.UploadRequest{ObjectKey: objectKey, ContentType: "image/png", MaxBytes: 4 << 20}); err != nil {
		t.Fatalf("create same-origin upload URL: %v", err)
	}
	if err := storage.DeleteObject(context.Background(), objectKey); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.InspectObject(context.Background(), objectKey); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("deleted object remained available (media id %d): %v", mediaID, err)
	}
}

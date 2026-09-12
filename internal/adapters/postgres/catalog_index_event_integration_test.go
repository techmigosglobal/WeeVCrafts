package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestApprovedProductPublishesTransactionalIndexEvent(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, `UPDATE products SET status = 'pending_review' WHERE id = $1`, fixture.productID); err != nil {
		t.Fatalf("prepare product review: %v", err)
	}
	adminEmail := fmt.Sprintf("index-admin-%d@example.invalid", time.Now().UnixNano())
	var adminID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Index Admin', 'active') RETURNING id`, adminEmail).Scan(&adminID); err != nil {
		t.Fatalf("insert index admin: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM outbox_events WHERE event_type = 'ProductIndexRequested' AND aggregate_id = $1`, fmt.Sprint(fixture.productID))
		_, _ = pool.Exec(cleanupContext, `DELETE FROM users WHERE id = $1`, adminID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, 'marketplace_admin')`, adminID); err != nil {
		t.Fatalf("grant index admin role: %v", err)
	}
	if err := NewCommerceRepository(pool).ApproveProduct(ctx, adminID, fixture.productID, true, "ready for search"); err != nil {
		t.Fatalf("approve product: %v", err)
	}
	var eventCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'ProductIndexRequested' AND aggregate_id = $1 AND payload->>'status' = 'approved'`, fmt.Sprint(fixture.productID)).Scan(&eventCount); err != nil {
		t.Fatalf("count product index events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one product index event, got %d", eventCount)
	}
}

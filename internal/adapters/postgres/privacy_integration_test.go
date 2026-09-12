package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestPrivacyDeletionAnonymizesEligibleDataAndRetainsOrderHistory(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "privacy-delete-" + suffix + "@example.invalid"
	var userID, requestID, orderID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Privacy Customer', 'active') RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("insert privacy user: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM audit_logs WHERE resource_type = 'user' AND resource_id = $1`, fmt.Sprint(userID))
		_, _ = pool.Exec(cleanupContext, `DELETE FROM privacy_requests WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM orders WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM users WHERE id = $1`, userID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO credentials (user_id, password_hash) VALUES ($1, 'redacted-test-hash')`, userID); err != nil {
		t.Fatalf("insert privacy credentials: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO marketing_preferences (user_id, email_marketing) VALUES ($1, true)`, userID); err != nil {
		t.Fatalf("insert marketing preference: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO addresses (user_id, recipient_name, line1, city, state, postal_code) VALUES ($1, 'Privacy Customer', '1 Private Lane', 'Pune', 'MH', '411001')`, userID); err != nil {
		t.Fatalf("insert address: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders (order_number, user_id, status, subtotal_cents, total_cents, address_snapshot, idempotency_scope, idempotency_key) VALUES ($1, $2, 'delivered', 1000, 1000, '{"recipient_name":"Privacy Customer","line1":"1 Private Lane"}', $3, $4) RETURNING id`, "PRIVACY-"+suffix, userID, "privacy-test", suffix).Scan(&orderID); err != nil {
		t.Fatalf("insert retained order: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO privacy_requests (user_id, request_type, details, due_at) VALUES ($1, 'deletion', '{"message":"delete my account"}', NOW() + INTERVAL '30 days') RETURNING id`, userID).Scan(&requestID); err != nil {
		t.Fatalf("insert deletion request: %v", err)
	}

	repository := NewPrivacyRepository(pool)
	if err := repository.ExecuteDeletion(ctx, userID, requestID); err != nil {
		t.Fatalf("execute privacy deletion: %v", err)
	}

	var status, deletedEmail, displayName string
	if err := pool.QueryRow(ctx, `SELECT status, email, display_name FROM users WHERE id = $1`, userID).Scan(&status, &deletedEmail, &displayName); err != nil {
		t.Fatalf("read anonymized user: %v", err)
	}
	if status != "deleted" || deletedEmail != fmt.Sprintf("deleted+%d@invalid.wecratfs", userID) || displayName != "Deleted account" {
		t.Fatalf("unexpected anonymized user: status=%q email=%q display=%q", status, deletedEmail, displayName)
	}
	var credentialCount, preferenceCount, addressCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM credentials WHERE user_id = $1`, userID).Scan(&credentialCount); err != nil {
		t.Fatalf("count credentials: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM marketing_preferences WHERE user_id = $1`, userID).Scan(&preferenceCount); err != nil {
		t.Fatalf("count preferences: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM addresses WHERE user_id = $1`, userID).Scan(&addressCount); err != nil {
		t.Fatalf("count addresses: %v", err)
	}
	if credentialCount != 0 || preferenceCount != 0 || addressCount != 0 {
		t.Fatalf("eligible data remained: credentials=%d preferences=%d addresses=%d", credentialCount, preferenceCount, addressCount)
	}
	var redacted bool
	if err := pool.QueryRow(ctx, `SELECT address_snapshot = '{"redacted":true}'::jsonb FROM orders WHERE id = $1`, orderID).Scan(&redacted); err != nil {
		t.Fatalf("read retained order: %v", err)
	}
	if !redacted {
		t.Fatal("expected retained order address to be redacted")
	}
	var requestStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM privacy_requests WHERE id = $1`, requestID).Scan(&requestStatus); err != nil {
		t.Fatalf("read deletion request: %v", err)
	}
	if requestStatus != "completed" {
		t.Fatalf("expected completed deletion request, got %q", requestStatus)
	}
	if err := repository.ExecuteDeletion(ctx, userID, requestID); !errors.Is(err, ports.ErrPrivacyRequestState) {
		t.Fatalf("expected completed request rejection, got %v", err)
	}
}

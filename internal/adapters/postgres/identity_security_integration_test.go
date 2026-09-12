package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestFailedLoginUpdatesLockAndAuditAtomically(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Security Fixture', 'active') RETURNING id`, "failed-login-"+suffix+"@example.invalid").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE resource_type = 'user' AND resource_id = $1`, fmt.Sprint(userID))
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO credentials (user_id, password_hash) VALUES ($1, 'fixture-hash')`, userID); err != nil {
		t.Fatalf("insert credentials: %v", err)
	}

	repository := NewUserRepository(pool)
	for attempt := 0; attempt < 5; attempt++ {
		if err := repository.RecordFailedLogin(ctx, userID); err != nil {
			t.Fatalf("record failed login %d: %v", attempt+1, err)
		}
	}
	var failedAttempts int
	var lockedUntil *time.Time
	if err := pool.QueryRow(ctx, `SELECT failed_attempts, locked_until FROM credentials WHERE user_id = $1`, userID).Scan(&failedAttempts, &lockedUntil); err != nil {
		t.Fatalf("read lock state: %v", err)
	}
	if failedAttempts != 5 || lockedUntil == nil {
		t.Fatalf("expected five failures and a lock, attempts=%d locked=%v", failedAttempts, lockedUntil)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE actor_id = $1 AND action = 'auth.login_failed'`, fmt.Sprint(userID)).Scan(&auditCount); err != nil {
		t.Fatalf("read login audit: %v", err)
	}
	if auditCount != 5 {
		t.Fatalf("expected one audit record per failed login, got %d", auditCount)
	}
}

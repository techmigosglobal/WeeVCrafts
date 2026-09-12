package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestAuthActionTokensAreSingleUseAndResetPasswordAtomically(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	suffix := fmt.Sprint(time.Now().UnixNano())
	email := "auth-action-" + suffix + "@example.invalid"
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Action Fixture', 'active') RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO credentials (user_id, password_hash, failed_attempts) VALUES ($1, 'old-hash', 3)`, userID); err != nil {
		t.Fatalf("insert credentials: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE resource_type = 'user' AND resource_id = $1`, fmt.Sprint(userID))
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	repository := NewAuthActionRepository(pool)
	recipient, found, err := repository.IssuePasswordReset(ctx, email, "password-token-hash", time.Now().Add(time.Minute))
	if err != nil || !found || recipient.UserID != userID {
		t.Fatalf("issue password reset: recipient=%+v found=%v err=%v", recipient, found, err)
	}
	resetID, err := repository.ResetPassword(ctx, "password-token-hash", "new-hash")
	if err != nil || resetID != userID {
		t.Fatalf("reset password: user=%d err=%v", resetID, err)
	}
	var passwordHash string
	var failedAttempts int
	if err := pool.QueryRow(ctx, `SELECT password_hash, failed_attempts FROM credentials WHERE user_id = $1`, userID).Scan(&passwordHash, &failedAttempts); err != nil {
		t.Fatalf("read reset credentials: %v", err)
	}
	if passwordHash != "new-hash" || failedAttempts != 0 {
		t.Fatalf("reset did not replace credentials atomically: hash=%q failures=%d", passwordHash, failedAttempts)
	}
	if _, err := repository.ResetPassword(ctx, "password-token-hash", "another-hash"); !errors.Is(err, ports.ErrAuthTokenInvalid) {
		t.Fatalf("expected reset replay rejection, got %v", err)
	}

	if _, err := repository.IssueEmailVerification(ctx, userID, "email-token-hash", time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("issue email verification: %v", err)
	}
	verifiedID, err := repository.ConsumeEmailVerification(ctx, "email-token-hash")
	if err != nil || verifiedID != userID {
		t.Fatalf("consume email verification: user=%d err=%v", verifiedID, err)
	}
	var verifiedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT email_verified_at FROM users WHERE id = $1`, userID).Scan(&verifiedAt); err != nil {
		t.Fatalf("read email verification: %v", err)
	}
	if verifiedAt == nil {
		t.Fatal("email verification timestamp was not recorded")
	}
	if _, err := repository.ConsumeEmailVerification(ctx, "email-token-hash"); !errors.Is(err, ports.ErrAuthTokenInvalid) {
		t.Fatalf("expected verification replay rejection, got %v", err)
	}

	if _, _, err := repository.IssuePasswordReset(ctx, "missing-"+email, "missing-token", time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("unknown reset email leaked repository error: %v", err)
	}
	if _, err := repository.ConsumeEmailVerification(ctx, "missing-token"); !errors.Is(err, ports.ErrAuthTokenInvalid) {
		t.Fatalf("expected missing token rejection, got %v", err)
	}
}

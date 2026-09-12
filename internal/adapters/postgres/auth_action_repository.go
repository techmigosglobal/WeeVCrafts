package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/ports"
)

type AuthActionRepository struct{ pool *pgxpool.Pool }

func NewAuthActionRepository(pool *pgxpool.Pool) *AuthActionRepository {
	return &AuthActionRepository{pool: pool}
}

func (r *AuthActionRepository) IssueEmailVerification(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (ports.EmailRecipient, error) {
	var recipient ports.EmailRecipient
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(transactionContext, `
			SELECT id, email, display_name
			FROM users
			WHERE id = $1 AND status = 'active'`, userID).Scan(&recipient.UserID, &recipient.Email, &recipient.DisplayName); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrAuthTokenInvalid
			}
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE auth_action_tokens SET consumed_at = NOW() WHERE user_id = $1 AND purpose = 'email_verification' AND consumed_at IS NULL`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO auth_action_tokens (user_id, purpose, token_hash, expires_at) VALUES ($1, 'email_verification', $2, $3)`, userID, tokenHash, expiresAt)
		return err
	})
	return recipient, err
}

func (r *AuthActionRepository) IssuePasswordReset(ctx context.Context, email, tokenHash string, expiresAt time.Time) (ports.EmailRecipient, bool, error) {
	var recipient ports.EmailRecipient
	found := false
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(transactionContext, `
			SELECT u.id, u.email, u.display_name
			FROM users u
			JOIN credentials c ON c.user_id = u.id
			WHERE LOWER(u.email) = LOWER($1) AND u.status = 'active'`, email).Scan(&recipient.UserID, &recipient.Email, &recipient.DisplayName)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		if _, err := tx.Exec(transactionContext, `UPDATE auth_action_tokens SET consumed_at = NOW() WHERE user_id = $1 AND purpose = 'password_reset' AND consumed_at IS NULL`, recipient.UserID); err != nil {
			return err
		}
		_, err = tx.Exec(transactionContext, `INSERT INTO auth_action_tokens (user_id, purpose, token_hash, expires_at) VALUES ($1, 'password_reset', $2, $3)`, recipient.UserID, tokenHash, expiresAt)
		return err
	})
	return recipient, found, err
}

func (r *AuthActionRepository) ConsumeEmailVerification(ctx context.Context, tokenHash string) (int64, error) {
	var userID int64
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(transactionContext, `
			SELECT user_id
			FROM auth_action_tokens
			WHERE purpose = 'email_verification' AND token_hash = $1
			  AND consumed_at IS NULL AND expires_at > NOW()
			FOR UPDATE`, tokenHash).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrAuthTokenInvalid
			}
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE auth_action_tokens SET consumed_at = NOW() WHERE token_hash = $1`, tokenHash); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata) VALUES ($1, 'auth.email_verified', 'user', $2, '{}'::jsonb)`, fmt.Sprint(userID), fmt.Sprint(userID))
		return err
	})
	return userID, err
}

func (r *AuthActionRepository) ResetPassword(ctx context.Context, tokenHash, passwordHash string) (int64, error) {
	var userID int64
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(transactionContext, `
			SELECT user_id
			FROM auth_action_tokens
			WHERE purpose = 'password_reset' AND token_hash = $1
			  AND consumed_at IS NULL AND expires_at > NOW()
			FOR UPDATE`, tokenHash).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrAuthTokenInvalid
			}
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE credentials SET password_hash = $2, password_changed_at = NOW(), failed_attempts = 0, locked_until = NULL, updated_at = NOW() WHERE user_id = $1`, userID, passwordHash); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE auth_action_tokens SET consumed_at = NOW() WHERE token_hash = $1`, tokenHash); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata) VALUES ($1, 'auth.password_reset', 'user', $2, '{}'::jsonb)`, fmt.Sprint(userID), fmt.Sprint(userID))
		return err
	})
	return userID, err
}

func (r *AuthActionRepository) RevokeAuthToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE auth_action_tokens SET consumed_at = NOW() WHERE token_hash = $1 AND consumed_at IS NULL`, tokenHash)
	return err
}

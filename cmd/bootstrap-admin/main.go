// Command bootstrap-admin creates the first Super Admin once, after schema
// migrations. It accepts secrets only from the process environment and never
// seeds demo identities during deployment.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	"github.com/wecratfs/commerce/internal/config"
)

func main() {
	cfg := config.FromEnv()
	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_SUPER_ADMIN_EMAIL")))
	name := strings.TrimSpace(os.Getenv("BOOTSTRAP_SUPER_ADMIN_NAME"))
	password := os.Getenv("BOOTSTRAP_SUPER_ADMIN_PASSWORD")
	if cfg.DatabaseURL == "" || !validEmail(email) || name == "" || len(password) < 16 || len(password) > 1024 {
		slog.Error("set DATABASE_URL, BOOTSTRAP_SUPER_ADMIN_EMAIL, BOOTSTRAP_SUPER_ADMIN_NAME, and a 16+ character BOOTSTRAP_SUPER_ADMIN_PASSWORD")
		os.Exit(2)
	}
	hash, err := applicationauth.NewArgon2idHasher().Hash(password)
	if err != nil {
		slog.Error("hash bootstrap password", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	var newUserID int64
	err = func() error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(761923540271)`); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE role_slug = 'super_admin')`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return errors.New("a Super Admin already exists; bootstrap is one-time only")
		}
		if err := tx.QueryRow(ctx, `INSERT INTO users (email, display_name) VALUES ($1, $2) RETURNING id`, email, name).Scan(&newUserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO credentials (user_id, password_hash) VALUES ($1, $2)`, newUserID, hash); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, 'super_admin')`, newUserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata) VALUES ($1, 'identity.super_admin_bootstrapped', 'user', $1, jsonb_build_object('email', $2))`, fmt.Sprint(newUserID), email); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}()
	if err != nil {
		slog.Error("bootstrap Super Admin", "error", err)
		os.Exit(1)
	}
	slog.Info("first Super Admin created", "email", email, "user_id", newUserID)
}

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value && strings.Contains(value, "@")
}

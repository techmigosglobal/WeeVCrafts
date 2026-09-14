package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type SellerStaffRepository struct {
	pool *pgxpool.Pool
}

func NewSellerStaffRepository(pool *pgxpool.Pool) *SellerStaffRepository {
	return &SellerStaffRepository{pool: pool}
}

func (r *SellerStaffRepository) ListStaff(ctx context.Context, ownerUserID int64) ([]domaincommerce.SellerStaffMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT su.user_id, u.email, u.display_name, su.status, su.permissions, su.created_at
		FROM seller_users su
		JOIN sellers s ON s.id = su.seller_id AND s.owner_user_id = $1 AND s.status = 'active'
		JOIN users u ON u.id = su.user_id
		WHERE su.status = 'active'
		ORDER BY su.created_at, su.user_id`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	staff := make([]domaincommerce.SellerStaffMember, 0)
	for rows.Next() {
		var member domaincommerce.SellerStaffMember
		if err := rows.Scan(&member.UserID, &member.Email, &member.DisplayName, &member.Status, &member.Permissions, &member.AddedAt); err != nil {
			return nil, err
		}
		staff = append(staff, member)
	}
	return staff, rows.Err()
}

func (r *SellerStaffRepository) AddStaff(ctx context.Context, ownerUserID int64, email string, permissions []string) (domaincommerce.SellerStaffMember, error) {
	var member domaincommerce.SellerStaffMember
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var sellerID int64
		if err := tx.QueryRow(transactionContext, `SELECT id FROM sellers WHERE owner_user_id = $1 AND status = 'active'`, ownerUserID).Scan(&sellerID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrForbidden
			}
			return err
		}
		var userID int64
		if err := tx.QueryRow(transactionContext, `SELECT id FROM users WHERE lower(email) = lower($1) AND status = 'active'`, email).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrNotFound
			}
			return err
		}
		if userID == ownerUserID {
			return ports.ErrForbidden
		}
		_, err := tx.Exec(transactionContext, `
			INSERT INTO seller_users (seller_id, user_id, status, permissions, added_by)
			VALUES ($1, $2, 'active', $3, $4)
			ON CONFLICT (seller_id, user_id) DO UPDATE
			SET status = 'active', permissions = EXCLUDED.permissions, added_by = EXCLUDED.added_by, updated_at = NOW()`, sellerID, userID, permissions, ownerUserID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, string(domainidentity.RoleSellerStaff)); err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"permissions": permissions, "seller_id": sellerID})
		if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'seller.staff_added', 'seller_user', $2, $3, $4)`, fmt.Sprint(ownerUserID), fmt.Sprint(userID), ports.RequestID(transactionContext), metadata); err != nil {
			return err
		}
		return tx.QueryRow(transactionContext, `
			SELECT u.id, u.email, u.display_name, su.status, su.permissions, su.created_at
			FROM seller_users su JOIN users u ON u.id = su.user_id
			WHERE su.seller_id = $1 AND su.user_id = $2`, sellerID, userID).Scan(&member.UserID, &member.Email, &member.DisplayName, &member.Status, &member.Permissions, &member.AddedAt)
	})
	return member, err
}

func (r *SellerStaffRepository) RemoveStaff(ctx context.Context, ownerUserID, staffUserID int64) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var sellerID int64
		if err := tx.QueryRow(transactionContext, `SELECT id FROM sellers WHERE owner_user_id = $1 AND status = 'active'`, ownerUserID).Scan(&sellerID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrForbidden
			}
			return err
		}
		result, err := tx.Exec(transactionContext, `UPDATE seller_users SET status = 'revoked', updated_at = NOW() WHERE seller_id = $1 AND user_id = $2 AND status = 'active'`, sellerID, staffUserID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ports.ErrNotFound
		}
		if _, err := tx.Exec(transactionContext, `DELETE FROM user_roles WHERE user_id = $1 AND role_slug = $2 AND NOT EXISTS (SELECT 1 FROM seller_users WHERE user_id = $1 AND status = 'active')`, staffUserID, string(domainidentity.RoleSellerStaff)); err != nil {
			return err
		}
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'seller.staff_removed', 'seller_user', $2, $3, jsonb_build_object('seller_id', $4))`, fmt.Sprint(ownerUserID), fmt.Sprint(staffUserID), ports.RequestID(transactionContext), sellerID)
		return err
	})
}

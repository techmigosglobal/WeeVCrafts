package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) ListSellerApplications(ctx context.Context, actorID int64) ([]domaincommerce.SellerAdminEntry, error) {
	if err := r.requireMarketplaceAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.owner_user_id, s.display_name, u.email, u.display_name,
		       s.status, s.created_at, s.updated_at
		FROM sellers s
		JOIN users u ON u.id = s.owner_user_id
		ORDER BY CASE s.status WHEN 'pending' THEN 0 WHEN 'active' THEN 1 WHEN 'suspended' THEN 2 ELSE 3 END,
		         s.updated_at DESC, s.id DESC
		LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domaincommerce.SellerAdminEntry, 0)
	for rows.Next() {
		var entry domaincommerce.SellerAdminEntry
		if err := rows.Scan(&entry.ID, &entry.OwnerUserID, &entry.DisplayName, &entry.OwnerEmail, &entry.OwnerName, &entry.Status, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *CommerceRepository) UpdateSellerStatus(ctx context.Context, actorID, sellerID int64, status, reason string) error {
	if err := r.requireMarketplaceAdmin(ctx, actorID); err != nil {
		return err
	}
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var previous string
		if err := tx.QueryRow(transactionContext, `SELECT status FROM sellers WHERE id = $1 FOR UPDATE`, sellerID).Scan(&previous); errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrSellerNotFound
		} else if err != nil {
			return err
		}
		if previous == "closed" || previous == status {
			return ports.ErrSellerState
		}
		result, err := tx.Exec(transactionContext, `UPDATE sellers SET status = $1, updated_at = NOW() WHERE id = $2 AND status <> 'closed'`, status, sellerID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ports.ErrSellerState
		}
		metadata, _ := json.Marshal(map[string]string{"previous_status": previous, "status": status, "reason": reason})
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'seller.status_updated', 'seller', $2, $3, $4)`, fmt.Sprint(actorID), fmt.Sprint(sellerID), ports.RequestID(transactionContext), metadata)
		return err
	})
}

func (r *CommerceRepository) requireMarketplaceAdmin(ctx context.Context, actorID int64) error {
	var allowed bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug IN ($2, $3))`, actorID, string(domainidentity.RoleMarketplaceAdmin), string(domainidentity.RoleSuperAdmin)).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ports.ErrForbidden
	}
	return nil
}

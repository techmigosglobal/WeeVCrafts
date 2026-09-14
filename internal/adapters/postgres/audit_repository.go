package postgres

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) ListAuditEntries(ctx context.Context, actorID int64, limit int) ([]domaincommerce.AuditEntry, error) {
	if err := r.requireMarketplaceAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(actor_id, ''), action, resource_type, resource_id,
		       COALESCE(request_id, ''), COALESCE(metadata::text, '{}'), created_at
		FROM audit_logs
		ORDER BY created_at DESC, id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domaincommerce.AuditEntry, 0)
	for rows.Next() {
		var entry domaincommerce.AuditEntry
		if err := rows.Scan(&entry.ID, &entry.ActorID, &entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.RequestID, &entry.Metadata, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *CommerceRepository) ListSellerAuditEntries(ctx context.Context, ownerUserID int64, limit int) ([]domaincommerce.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT al.id, COALESCE(al.actor_id, ''), al.action, al.resource_type, al.resource_id,
		       COALESCE(al.request_id, ''), COALESCE(al.metadata::text, '{}'), al.created_at
		FROM audit_logs al
		WHERE EXISTS (
			SELECT 1 FROM user_roles ur
			WHERE ur.user_id = $1 AND ur.role_slug = 'seller_owner'
		)
		AND (
			al.actor_id = $1::text
			OR EXISTS (
				SELECT 1 FROM sellers s
				WHERE s.owner_user_id = $1 AND s.id::text = al.resource_id
				  AND al.resource_type = 'seller'
			)
			OR EXISTS (
				SELECT 1 FROM products p
				JOIN sellers s ON s.id = p.seller_id
				WHERE s.owner_user_id = $1 AND p.id::text = al.resource_id
				  AND al.resource_type = 'product'
			)
			OR EXISTS (
				SELECT 1 FROM product_variants pv
				JOIN products p ON p.id = pv.product_id
				JOIN sellers s ON s.id = p.seller_id
				WHERE s.owner_user_id = $1 AND pv.id::text = al.resource_id
				  AND al.resource_type = 'inventory_variant'
			)
			OR EXISTS (
				SELECT 1 FROM sellers s
				WHERE s.owner_user_id = $1
				  AND al.resource_type = 'seller_fulfillment'
				  AND al.resource_id LIKE s.id::text || ':%'
			)
			OR EXISTS (
				SELECT 1 FROM sellers s
				WHERE s.owner_user_id = $1
				  AND al.resource_type = 'seller_user'
				  AND al.metadata->>'seller_id' = s.id::text
			)
		)
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT $2`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domaincommerce.AuditEntry, 0)
	for rows.Next() {
		var entry domaincommerce.AuditEntry
		if err := rows.Scan(&entry.ID, &entry.ActorID, &entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.RequestID, &entry.Metadata, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

var _ ports.AuditRepository = (*CommerceRepository)(nil)

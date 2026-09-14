package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) ListSellerInventory(ctx context.Context, userID int64) ([]domaincommerce.InventoryItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pv.id, p.id, s.id, p.name, pv.sku, p.status,
		       COALESCE(i.available_quantity, 0), COALESCE(i.reserved_quantity, 0),
		       COALESCE(i.updated_at, pv.updated_at)
		FROM product_variants pv
		JOIN products p ON p.id = pv.product_id
		JOIN sellers s ON s.id = p.seller_id AND s.status = 'active'
		LEFT JOIN inventory_stock i ON i.variant_id = pv.id
		WHERE (s.owner_user_id = $1 OR EXISTS (
			SELECT 1 FROM seller_users su
			WHERE su.seller_id = s.id AND su.user_id = $1 AND su.status = 'active'
		)) AND pv.status = 'active'
		ORDER BY p.updated_at DESC, p.id DESC, pv.id
		LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domaincommerce.InventoryItem, 0)
	for rows.Next() {
		var item domaincommerce.InventoryItem
		if err := rows.Scan(&item.VariantID, &item.ProductID, &item.SellerID, &item.ProductName, &item.SKU, &item.Status, &item.AvailableQuantity, &item.ReservedQuantity, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CommerceRepository) AdjustSellerInventory(ctx context.Context, userID int64, adjustment domaincommerce.InventoryAdjustment) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var sellerID int64
		var before, reserved int
		err := tx.QueryRow(transactionContext, `
			SELECT s.id, i.available_quantity, i.reserved_quantity
			FROM inventory_stock i
			JOIN product_variants pv ON pv.id = i.variant_id AND pv.status = 'active'
			JOIN products p ON p.id = pv.product_id
			JOIN sellers s ON s.id = p.seller_id AND s.status = 'active'
			WHERE i.variant_id = $1 AND (s.owner_user_id = $2 OR EXISTS (
				SELECT 1 FROM seller_users su
				WHERE su.seller_id = s.id AND su.user_id = $2 AND su.status = 'active'
			))
			FOR UPDATE OF i`, adjustment.VariantID, userID).Scan(&sellerID, &before, &reserved)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrForbidden
		}
		if err != nil {
			return err
		}
		after := before + adjustment.Delta
		if after < 0 {
			return ports.ErrInsufficientStock
		}
		if _, err := tx.Exec(transactionContext, `UPDATE inventory_stock SET available_quantity = $1, updated_at = NOW() WHERE variant_id = $2`, after, adjustment.VariantID); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO inventory_transactions (seller_id, variant_id, actor_id, delta, before_quantity, after_quantity, reason) VALUES ($1, $2, $3, $4, $5, $6, $7)`, sellerID, adjustment.VariantID, userID, adjustment.Delta, before, after, adjustment.Reason); err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{
			"variant_id": adjustment.VariantID, "delta": adjustment.Delta,
			"before_quantity": before, "after_quantity": after,
			"reserved_quantity": reserved, "reason": adjustment.Reason,
		})
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'seller.inventory_adjusted', 'inventory_variant', $2, $3, $4)`, fmt.Sprint(userID), fmt.Sprint(adjustment.VariantID), ports.RequestID(transactionContext), metadata)
		return err
	})
}

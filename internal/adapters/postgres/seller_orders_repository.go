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

func (r *CommerceRepository) ListSellerOrders(ctx context.Context, userID int64) ([]domaincommerce.SellerOrder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, s.id, o.order_number, o.status,
		       CASE WHEN o.payment_method = 'cash_on_delivery' THEN 'due_on_delivery'
		            ELSE COALESCE((SELECT pa.status FROM payment_attempts pa WHERE pa.order_id = o.id ORDER BY pa.id DESC LIMIT 1), 'unrecorded') END,
		       COALESCE(sf.status, 'pending'), o.currency,
		       COALESCE(sf.carrier, ''), COALESCE(sf.tracking_number, ''), COALESCE(sf.last_note, ''),
		       o.created_at, oi.product_name, oi.sku, oi.quantity, oi.unit_price_cents, oi.line_total_cents
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		JOIN product_variants pv ON pv.id = oi.variant_id
		JOIN products p ON p.id = pv.product_id
		JOIN sellers s ON s.id = p.seller_id AND s.status = 'active'
		LEFT JOIN seller_fulfillments sf ON sf.order_id = o.id AND sf.seller_id = s.id
		WHERE o.status IN ('paid', 'processing', 'shipped', 'delivered')
		  AND (s.owner_user_id = $1 OR EXISTS (
			SELECT 1 FROM seller_users su
			WHERE su.seller_id = s.id AND su.user_id = $1 AND su.status = 'active'
		  ))
		ORDER BY o.created_at DESC, o.id DESC, s.id, oi.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type orderKey struct{ orderID, sellerID int64 }
	orders := make([]domaincommerce.SellerOrder, 0)
	indexes := make(map[orderKey]int)
	for rows.Next() {
		var order domaincommerce.SellerOrder
		var item domaincommerce.SellerOrderItem
		if err := rows.Scan(&order.ID, &order.SellerID, &order.OrderNumber, &order.OverallStatus, &order.PaymentStatus, &order.FulfillmentStatus, &order.Currency, &order.Carrier, &order.TrackingNumber, &order.LastNote, &order.CreatedAt, &item.ProductName, &item.SKU, &item.Quantity, &item.UnitPriceCents, &item.LineTotalCents); err != nil {
			return nil, err
		}
		key := orderKey{orderID: order.ID, sellerID: order.SellerID}
		index, ok := indexes[key]
		if !ok {
			order.Items = make([]domaincommerce.SellerOrderItem, 0, 2)
			orders = append(orders, order)
			index = len(orders) - 1
			indexes[key] = index
		}
		orders[index].Items = append(orders[index].Items, item)
		orders[index].SellerSubtotalCents += item.LineTotalCents
	}
	return orders, rows.Err()
}

func (r *CommerceRepository) UpdateSellerFulfillment(ctx context.Context, userID, sellerID int64, orderNumber, status, carrier, trackingNumber, note string) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID int64
		var currentStatus, overallStatus string
		err := tx.QueryRow(transactionContext, `
			SELECT o.id, COALESCE(sf.status, 'pending'), o.status
			FROM orders o
			JOIN sellers s ON s.id = $2 AND s.status = 'active'
			LEFT JOIN seller_fulfillments sf ON sf.order_id = o.id AND sf.seller_id = s.id
			WHERE o.order_number = $1 AND s.id = $2
			  AND EXISTS (
				SELECT 1 FROM order_items oi
				JOIN product_variants pv ON pv.id = oi.variant_id
				JOIN products p ON p.id = pv.product_id
				WHERE oi.order_id = o.id AND p.seller_id = s.id
			  )
			  AND (s.owner_user_id = $3 OR EXISTS (
				SELECT 1 FROM seller_users su
				WHERE su.seller_id = s.id AND su.user_id = $3 AND su.status = 'active'
			  ))
			FOR UPDATE OF o`, orderNumber, sellerID, userID).Scan(&orderID, &currentStatus, &overallStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrOrderNotFound
		}
		if err != nil {
			return err
		}
		if overallStatus != "paid" && overallStatus != "processing" && overallStatus != "shipped" && overallStatus != "delivered" {
			return ports.ErrInvalidState
		}
		if currentStatus != status {
			valid := (currentStatus == "pending" && status == "processing") || (currentStatus == "processing" && status == "shipped") || (currentStatus == "shipped" && status == "delivered")
			if !valid {
				return ports.ErrInvalidState
			}
		}
		if _, err := tx.Exec(transactionContext, `
			INSERT INTO seller_fulfillments (seller_id, order_id, status, carrier, tracking_number, last_note, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (seller_id, order_id) DO UPDATE SET status = EXCLUDED.status, carrier = EXCLUDED.carrier, tracking_number = EXCLUDED.tracking_number, last_note = EXCLUDED.last_note, updated_by = EXCLUDED.updated_by, updated_at = NOW()`, sellerID, orderID, status, carrier, trackingNumber, note, userID); err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{
			"seller_id": sellerID, "order_id": orderID, "order_number": orderNumber,
			"from_status": currentStatus, "to_status": status, "carrier": carrier,
			"tracking_number": trackingNumber, "note": note,
		})
		if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'seller.order_fulfillment_updated', 'seller_fulfillment', $2, $3, $4)`, fmt.Sprint(userID), fmt.Sprintf("%d:%d", sellerID, orderID), ports.RequestID(transactionContext), metadata); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"seller_id": sellerID, "order_id": orderID, "order_number": orderNumber, "status": status})
		_, err = tx.Exec(transactionContext, `INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload) VALUES ('SellerFulfillmentUpdated', 'SellerFulfillment', $1, $2)`, fmt.Sprintf("%d:%d", sellerID, orderID), payload)
		return err
	})
}

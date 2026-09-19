package postgres

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) ListAdminOrders(ctx context.Context, actorID int64, limit int) ([]domaincommerce.AdminOrder, error) {
	var allowed bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug IN ('marketplace_admin', 'operations_returns', 'super_admin'))`, actorID).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ports.ErrForbidden
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.status, u.display_name,
		       CASE WHEN POSITION('@' IN u.email) > 1
		            THEN LEFT(u.email, 1) || '***@' || SPLIT_PART(u.email, '@', 2)
		            ELSE '***' END,
		       CASE WHEN o.payment_method = 'cash_on_delivery' AND o.status = 'cancelled' THEN 'cancelled_unpaid'
		            WHEN o.payment_method = 'cash_on_delivery' THEN 'due_on_delivery' ELSE COALESCE(pa.status, 'unrecorded') END,
		       COALESCE(sf.status, 'pending'),
		       o.total_cents, o.currency, o.created_at
		FROM orders o
		JOIN users u ON u.id = o.user_id
		LEFT JOIN LATERAL (
			SELECT status FROM payment_attempts WHERE order_id = o.id ORDER BY id DESC LIMIT 1
		) pa ON TRUE
		LEFT JOIN LATERAL (
			SELECT status FROM seller_fulfillments WHERE order_id = o.id ORDER BY updated_at DESC, id DESC LIMIT 1
		) sf ON TRUE
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]domaincommerce.AdminOrder, 0)
	for rows.Next() {
		var order domaincommerce.AdminOrder
		if err := rows.Scan(&order.ID, &order.OrderNumber, &order.Status, &order.CustomerLabel, &order.CustomerEmailMasked, &order.PaymentStatus, &order.FulfillmentStatus, &order.TotalCents, &order.Currency, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

var _ ports.AdminOperationsRepository = (*CommerceRepository)(nil)

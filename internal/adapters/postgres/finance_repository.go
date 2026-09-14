package postgres

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
)

func (r *CommerceRepository) ListFinanceEntries(ctx context.Context, limit int) ([]domaincommerce.FinanceEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.status,
		       COALESCE(pa.status, 'unrecorded'), COALESCE(pa.provider, ''),
		       COALESCE(pa.provider_payment_id, ''), COALESCE(pa.amount_cents, o.total_cents),
		       COALESCE(pa.currency, o.currency), COALESCE(pr.status, ''),
		       COALESCE(pr.amount_cents, 0), o.created_at
		FROM orders o
		LEFT JOIN LATERAL (
			SELECT status, provider, provider_payment_id, amount_cents, currency
			FROM payment_attempts WHERE order_id = o.id ORDER BY id DESC LIMIT 1
		) pa ON TRUE
		LEFT JOIN payment_refunds pr ON pr.order_id = o.id
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domaincommerce.FinanceEntry, 0)
	for rows.Next() {
		var entry domaincommerce.FinanceEntry
		if err := rows.Scan(&entry.OrderID, &entry.OrderNumber, &entry.OrderStatus, &entry.PaymentStatus, &entry.Provider, &entry.ProviderPaymentID, &entry.AmountCents, &entry.Currency, &entry.RefundStatus, &entry.RefundedAmountCents, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

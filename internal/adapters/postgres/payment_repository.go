package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/ports"
)

type PaymentRepository struct{ pool *pgxpool.Pool }

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) GetPaymentOrder(ctx context.Context, userID int64, orderNumber string) (ports.PaymentOrderRecord, error) {
	var record ports.PaymentOrderRecord
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.status, pa.provider,
		       COALESCE(pa.provider_order_id, ''), pa.status,
		       pa.amount_cents, pa.currency
		FROM orders o
		JOIN payment_attempts pa ON pa.order_id = o.id
		WHERE o.user_id = $1 AND o.order_number = $2
		ORDER BY pa.id DESC
		LIMIT 1`, userID, orderNumber).Scan(
		&record.OrderID,
		&record.OrderNumber,
		&record.OrderStatus,
		&record.Provider,
		&record.ProviderOrderID,
		&record.PaymentStatus,
		&record.AmountCents,
		&record.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.PaymentOrderRecord{}, ports.ErrPaymentNotFound
	}
	return record, err
}

func (r *PaymentRepository) AttachProviderOrder(ctx context.Context, userID int64, orderNumber string, providerOrder ports.PaymentOrder) (ports.PaymentOrderRecord, error) {
	returnRecord := ports.PaymentOrderRecord{}
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID int64
		var orderStatus string
		if err := tx.QueryRow(transactionContext, `SELECT id, status FROM orders WHERE user_id = $1 AND order_number = $2 FOR UPDATE`, userID, orderNumber).Scan(&orderID, &orderStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrPaymentNotFound
			}
			return err
		}
		if orderStatus != "pending_payment" {
			return ports.ErrPaymentState
		}
		result, err := tx.Exec(transactionContext, `
			UPDATE payment_attempts
			SET provider_order_id = $1, status = 'pending', updated_at = NOW()
			WHERE order_id = $2 AND provider = 'razorpay' AND provider_order_id IS NULL
			  AND status = 'created' AND amount_cents = $3 AND currency = $4`,
			providerOrder.ProviderOrderID, orderID, providerOrder.AmountCents, providerOrder.Currency)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			// A retry may have attached the provider order already. Return the
			// existing authoritative value rather than overwriting it.
			return tx.QueryRow(transactionContext, `
				SELECT o.id, o.order_number, o.status, pa.provider,
				       COALESCE(pa.provider_order_id, ''), pa.status,
				       pa.amount_cents, pa.currency
				FROM orders o JOIN payment_attempts pa ON pa.order_id = o.id
				WHERE o.id = $1 ORDER BY pa.id DESC LIMIT 1`, orderID).Scan(
				&returnRecord.OrderID,
				&returnRecord.OrderNumber,
				&returnRecord.OrderStatus,
				&returnRecord.Provider,
				&returnRecord.ProviderOrderID,
				&returnRecord.PaymentStatus,
				&returnRecord.AmountCents,
				&returnRecord.Currency,
			)
		}
		return tx.QueryRow(transactionContext, `
			SELECT o.id, o.order_number, o.status, pa.provider,
			       COALESCE(pa.provider_order_id, ''), pa.status,
			       pa.amount_cents, pa.currency
			FROM orders o JOIN payment_attempts pa ON pa.order_id = o.id
			WHERE o.id = $1 ORDER BY pa.id DESC LIMIT 1`, orderID).Scan(
			&returnRecord.OrderID,
			&returnRecord.OrderNumber,
			&returnRecord.OrderStatus,
			&returnRecord.Provider,
			&returnRecord.ProviderOrderID,
			&returnRecord.PaymentStatus,
			&returnRecord.AmountCents,
			&returnRecord.Currency,
		)
	})
	return returnRecord, err
}

func (r *PaymentRepository) ProcessWebhook(ctx context.Context, webhook ports.PaymentWebhook, payload []byte) (bool, error) {
	returnValue := false
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var inserted bool
		if err := tx.QueryRow(transactionContext, `INSERT INTO payment_webhook_events (event_id, event_type, payload) VALUES ($1, $2, $3) ON CONFLICT (event_id) DO NOTHING RETURNING TRUE`, webhook.EventID, webhook.EventType, payload).Scan(&inserted); errors.Is(err, pgx.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		returnValue = inserted
		if webhook.ProviderOrderID == "" {
			_, err := tx.Exec(transactionContext, `UPDATE payment_webhook_events SET processed_at = NOW() WHERE event_id = $1`, webhook.EventID)
			return err
		}
		var orderID, expectedAmount int64
		var expectedCurrency string
		if err := tx.QueryRow(transactionContext, `SELECT order_id, amount_cents, currency FROM payment_attempts WHERE provider_order_id = $1 FOR UPDATE`, webhook.ProviderOrderID).Scan(&orderID, &expectedAmount, &expectedCurrency); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				_, updateErr := tx.Exec(transactionContext, `UPDATE payment_webhook_events SET processed_at = NOW() WHERE event_id = $1`, webhook.EventID)
				return updateErr
			}
			return err
		}
		if webhook.AmountCents != expectedAmount || webhook.Currency != expectedCurrency {
			metadata, _ := json.Marshal(map[string]any{
				"event_type": webhook.EventType, "provider_order_id": webhook.ProviderOrderID,
				"expected_amount_cents": expectedAmount, "received_amount_cents": webhook.AmountCents,
				"expected_currency": expectedCurrency, "received_currency": webhook.Currency,
			})
			if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (action, resource_type, resource_id, request_id, metadata) VALUES ('payment.webhook_rejected_mismatch', 'order', $1, $2, $3)`, fmt.Sprint(orderID), ports.RequestID(transactionContext), metadata); err != nil {
				return err
			}
			_, err := tx.Exec(transactionContext, `UPDATE payment_webhook_events SET processed_at = NOW() WHERE event_id = $1`, webhook.EventID)
			return err
		}
		if webhook.EventType == "payment.captured" || webhook.EventType == "order.paid" {
			if _, err := tx.Exec(transactionContext, `UPDATE payment_attempts SET provider_payment_id = $1, status = 'captured', updated_at = NOW() WHERE order_id = $2 AND (provider_payment_id IS NULL OR provider_payment_id = $1)`, webhook.ProviderPaymentID, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE orders SET status = 'paid', updated_at = NOW() WHERE id = $1 AND status = 'pending_payment'`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, from_status, to_status, reason) SELECT $1, 'pending_payment', 'paid', 'payment webhook captured' WHERE NOT EXISTS (SELECT 1 FROM order_status_history WHERE order_id = $1 AND to_status = 'paid')`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_stock i SET reserved_quantity = reserved_quantity - ri.quantity, updated_at = NOW() FROM inventory_reservation_items ri JOIN inventory_reservations r ON r.id = ri.reservation_id WHERE r.order_id = $1 AND r.status = 'active' AND i.variant_id = ri.variant_id`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_reservations SET status = 'committed', updated_at = NOW() WHERE order_id = $1 AND status = 'active'`, orderID); err != nil {
				return err
			}
		} else if webhook.EventType == "payment.failed" {
			if _, err := tx.Exec(transactionContext, `UPDATE payment_attempts SET provider_payment_id = $1, status = 'failed', updated_at = NOW() WHERE order_id = $2`, webhook.ProviderPaymentID, orderID); err != nil {
				return err
			}
			if _, err := releaseReservation(transactionContext, tx, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE orders SET status = 'payment_failed', updated_at = NOW() WHERE id = $1 AND status = 'pending_payment'`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, from_status, to_status, reason) SELECT $1, 'pending_payment', 'payment_failed', 'payment webhook failed' WHERE NOT EXISTS (SELECT 1 FROM order_status_history WHERE order_id = $1 AND to_status = 'payment_failed')`, orderID); err != nil {
				return err
			}
		} else if webhook.EventType == "payment.authorized" || webhook.EventType == "payment.pending" {
			status := "pending"
			if webhook.EventType == "payment.authorized" {
				status = "authorized"
			}
			if _, err := tx.Exec(transactionContext, `UPDATE payment_attempts SET provider_payment_id = $1, status = $2, updated_at = NOW() WHERE order_id = $3 AND (provider_payment_id IS NULL OR provider_payment_id = $1)`, webhook.ProviderPaymentID, status, orderID); err != nil {
				return err
			}
		}
		metadata, _ := json.Marshal(map[string]string{"event_type": webhook.EventType, "provider_order_id": webhook.ProviderOrderID})
		if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (action, resource_type, resource_id, request_id, metadata) VALUES ('payment.webhook_processed', 'order', $1, $2, $3)`, fmt.Sprint(orderID), ports.RequestID(transactionContext), metadata); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `UPDATE payment_webhook_events SET processed_at = NOW() WHERE event_id = $1`, webhook.EventID)
		return err
	})
	return returnValue, err
}

func releaseReservation(ctx context.Context, tx pgx.Tx, orderID int64) (bool, error) {
	var reservationID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM inventory_reservations WHERE order_id = $1 AND status = 'active' FOR UPDATE`, orderID).Scan(&reservationID); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return releaseReservationWithStatus(ctx, tx, reservationID, "released")
}

func releaseReservationWithStatus(ctx context.Context, tx pgx.Tx, reservationID int64, finalStatus string) (bool, error) {
	rows, err := tx.Query(ctx, `SELECT variant_id, quantity FROM inventory_reservation_items WHERE reservation_id = $1`, reservationID)
	if err != nil {
		return false, err
	}
	type reservationItem struct {
		variantID int64
		quantity  int
	}
	items := make([]reservationItem, 0, 4)
	for rows.Next() {
		var variantID int64
		var quantity int
		if err := rows.Scan(&variantID, &quantity); err != nil {
			rows.Close()
			return false, err
		}
		items = append(items, reservationItem{variantID: variantID, quantity: quantity})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	rows.Close()
	for _, item := range items {
		if _, err := tx.Exec(ctx, `UPDATE inventory_stock SET available_quantity = available_quantity + $2, reserved_quantity = reserved_quantity - $2, updated_at = NOW() WHERE variant_id = $1`, item.variantID, item.quantity); err != nil {
			return false, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE inventory_reservations SET status = $2, updated_at = NOW() WHERE id = $1 AND status = 'active'`, reservationID, finalStatus)
	return true, err
}

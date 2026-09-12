package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/wecratfs/commerce/internal/ports"
)

const refundRecordSelect = `
	SELECT pr.id, pr.order_id, o.order_number, o.status, pa.status,
	       COALESCE(pa.provider_payment_id, ''), COALESCE(pr.provider_refund_id, ''),
	       pr.status, pr.amount_cents, pr.currency, pr.reason, pr.request_key
	FROM payment_refunds pr
	JOIN orders o ON o.id = pr.order_id
	JOIN payment_attempts pa ON pa.order_id = o.id
	WHERE pr.id = $1
	ORDER BY pa.id DESC
	LIMIT 1`

// BeginRefund creates or reopens the one full-refund record for an order. The
// amount and currency are read while the paid order/payment rows are locked,
// so a client cannot refund an amount different from PostgreSQL's total.
func (r *PaymentRepository) BeginRefund(ctx context.Context, userID int64, orderNumber, reason, requestKey string) (ports.PaymentRefundRecord, error) {
	var record ports.PaymentRefundRecord
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID, amountCents int64
		var orderStatus, paymentStatus, providerPaymentID, currency string
		err := tx.QueryRow(transactionContext, `
			SELECT o.id, o.status, pa.status, COALESCE(pa.provider_payment_id, ''), o.total_cents, o.currency
			FROM orders o
			JOIN payment_attempts pa ON pa.order_id = o.id
			WHERE o.user_id = $1 AND o.order_number = $2
			ORDER BY pa.id DESC
			LIMIT 1
			FOR UPDATE OF o, pa`, userID, orderNumber).Scan(&orderID, &orderStatus, &paymentStatus, &providerPaymentID, &amountCents, &currency)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrRefundNotFound
		}
		if err != nil {
			return err
		}
		var refundID int64
		err = tx.QueryRow(transactionContext, `SELECT id FROM payment_refunds WHERE order_id = $1 FOR UPDATE`, orderID).Scan(&refundID)
		if errors.Is(err, pgx.ErrNoRows) {
			if orderStatus != "paid" || paymentStatus != "captured" || providerPaymentID == "" || amountCents <= 0 || currency == "" {
				return ports.ErrRefundState
			}
			err = tx.QueryRow(transactionContext, `
				INSERT INTO payment_refunds (order_id, request_key, amount_cents, currency, reason)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`, orderID, requestKey, amountCents, currency, reason).Scan(&refundID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			var existingRequestKey, existingStatus string
			if err := tx.QueryRow(transactionContext, `SELECT request_key, status FROM payment_refunds WHERE id = $1`, refundID).Scan(&existingRequestKey, &existingStatus); err != nil {
				return err
			}
			if existingRequestKey != requestKey {
				return ports.ErrRefundState
			}
			if existingStatus == "processed" {
				return tx.QueryRow(transactionContext, refundRecordSelect, refundID).Scan(
					&record.ID, &record.OrderID, &record.OrderNumber, &record.OrderStatus, &record.PaymentStatus,
					&record.ProviderPaymentID, &record.ProviderRefundID, &record.Status, &record.AmountCents,
					&record.Currency, &record.Reason, &record.RequestKey,
				)
			}
			if existingStatus == "pending" {
				var existingProviderRefundID string
				if err := tx.QueryRow(transactionContext, `SELECT COALESCE(provider_refund_id, '') FROM payment_refunds WHERE id = $1`, refundID).Scan(&existingProviderRefundID); err != nil {
					return err
				}
				if existingProviderRefundID != "" {
					return tx.QueryRow(transactionContext, refundRecordSelect, refundID).Scan(
						&record.ID, &record.OrderID, &record.OrderNumber, &record.OrderStatus, &record.PaymentStatus,
						&record.ProviderPaymentID, &record.ProviderRefundID, &record.Status, &record.AmountCents,
						&record.Currency, &record.Reason, &record.RequestKey,
					)
				}
			}
			if orderStatus != "paid" || paymentStatus != "captured" || providerPaymentID == "" || amountCents <= 0 || currency == "" {
				return ports.ErrRefundState
			}
			if existingStatus == "failed" {
				if _, err := tx.Exec(transactionContext, `UPDATE payment_refunds SET status = 'pending', reason = $2, last_error = '', updated_at = NOW() WHERE id = $1`, refundID, reason); err != nil {
					return err
				}
			}
		}
		return tx.QueryRow(transactionContext, refundRecordSelect, refundID).Scan(
			&record.ID, &record.OrderID, &record.OrderNumber, &record.OrderStatus, &record.PaymentStatus,
			&record.ProviderPaymentID, &record.ProviderRefundID, &record.Status, &record.AmountCents,
			&record.Currency, &record.Reason, &record.RequestKey,
		)
	})
	return record, err
}

// CompleteRefund persists the provider result and performs the paid ->
// refunded transition atomically for processed refunds. A provider-pending
// result remains pending until a later reconciliation capability is added.
func (r *PaymentRepository) CompleteRefund(ctx context.Context, refundID int64, refund ports.PaymentRefund) (ports.PaymentRefundRecord, error) {
	var record ports.PaymentRefundRecord
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID, amountCents int64
		var orderNumber, orderStatus, paymentStatus, providerPaymentID, currentStatus, currency, reason, requestKey string
		err := tx.QueryRow(transactionContext, `
			SELECT pr.order_id, o.order_number, o.status, pa.status,
			       COALESCE(pa.provider_payment_id, ''), pr.status, pr.amount_cents,
			       pr.currency, pr.reason, pr.request_key
			FROM payment_refunds pr
			JOIN orders o ON o.id = pr.order_id
			JOIN payment_attempts pa ON pa.order_id = o.id
			WHERE pr.id = $1
			ORDER BY pa.id DESC
			LIMIT 1
			FOR UPDATE OF pr, o, pa`, refundID).Scan(
			&orderID, &orderNumber, &orderStatus, &paymentStatus, &providerPaymentID, &currentStatus,
			&amountCents, &currency, &reason, &requestKey,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrRefundNotFound
		}
		if err != nil {
			return err
		}
		if currentStatus == "processed" {
			return tx.QueryRow(transactionContext, refundRecordSelect, refundID).Scan(
				&record.ID, &record.OrderID, &record.OrderNumber, &record.OrderStatus, &record.PaymentStatus,
				&record.ProviderPaymentID, &record.ProviderRefundID, &record.Status, &record.AmountCents,
				&record.Currency, &record.Reason, &record.RequestKey,
			)
		}
		if currentStatus != "pending" || refund.ProviderRefundID == "" || refund.AmountCents != amountCents ||
			(refund.ProviderPaymentID != "" && refund.ProviderPaymentID != providerPaymentID) ||
			(refund.Currency != "" && refund.Currency != currency) ||
			(refund.Status != "pending" && refund.Status != "processed") {
			return ports.ErrRefundState
		}
		if _, err := tx.Exec(transactionContext, `
			UPDATE payment_refunds
			SET provider_refund_id = $2, status = $3, last_error = '', updated_at = NOW()
			WHERE id = $1`, refundID, refund.ProviderRefundID, refund.Status); err != nil {
			return err
		}
		if refund.Status == "processed" {
			if orderStatus != "paid" || paymentStatus != "captured" {
				return ports.ErrRefundState
			}
			if _, err := tx.Exec(transactionContext, `UPDATE payment_attempts SET status = 'refunded', updated_at = NOW() WHERE order_id = $1 AND status = 'captured'`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE orders SET status = 'refunded', updated_at = NOW() WHERE id = $1 AND status = 'paid'`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `
				INSERT INTO order_status_history (order_id, from_status, to_status, reason)
				SELECT $1, 'paid', 'refunded', 'payment refund processed'
				WHERE NOT EXISTS (SELECT 1 FROM order_status_history WHERE order_id = $1 AND to_status = 'refunded')`, orderID); err != nil {
				return err
			}
		}
		metadata, _ := json.Marshal(map[string]string{
			"provider_refund_id":  refund.ProviderRefundID,
			"provider_payment_id": providerPaymentID,
			"status":              refund.Status,
		})
		if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (action, resource_type, resource_id, metadata) VALUES ($1, 'order', $2, $3)`, "payment.refund_"+refund.Status, fmt.Sprint(orderID), metadata); err != nil {
			return err
		}
		return tx.QueryRow(transactionContext, refundRecordSelect, refundID).Scan(
			&record.ID, &record.OrderID, &record.OrderNumber, &record.OrderStatus, &record.PaymentStatus,
			&record.ProviderPaymentID, &record.ProviderRefundID, &record.Status, &record.AmountCents,
			&record.Currency, &record.Reason, &record.RequestKey,
		)
	})
	return record, err
}

func (r *PaymentRepository) FailRefund(ctx context.Context, refundID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	_, err := r.pool.Exec(ctx, `UPDATE payment_refunds SET status = 'failed', last_error = $2, updated_at = NOW() WHERE id = $1 AND status = 'pending'`, refundID, reason)
	return err
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) CreateReturnRequest(ctx context.Context, userID int64, orderNumber, reason string) (domaincommerce.ReturnRequest, error) {
	var request domaincommerce.ReturnRequest
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID int64
		if err := tx.QueryRow(transactionContext, `
			SELECT id, total_cents, currency
			FROM orders
			WHERE user_id = $1 AND order_number = $2 AND status IN ('paid', 'processing', 'shipped', 'delivered')
			FOR UPDATE`, userID, orderNumber).Scan(&orderID, &request.AmountCents, &request.Currency); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrReturnNotFound
			}
			return err
		}
		request.UserID = userID
		request.OrderNumber = orderNumber
		request.Reason = reason
		if err := tx.QueryRow(transactionContext, `
			INSERT INTO return_requests (order_id, user_id, reason)
			VALUES ($1, $2, $3)
			RETURNING id, status, created_at, updated_at`, orderID, userID, reason).Scan(&request.ID, &request.Status, &request.CreatedAt, &request.UpdatedAt); err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23505" {
				return ports.ErrReturnState
			}
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'customer.return_requested', 'return_request', $2, $3, jsonb_build_object('order_number', $4, 'reason', $5))`, fmt.Sprint(userID), fmt.Sprint(request.ID), ports.RequestID(transactionContext), orderNumber, reason)
		return err
	})
	return request, err
}

func (r *CommerceRepository) GetReturnRequest(ctx context.Context, userID int64, orderNumber string) (domaincommerce.ReturnRequest, error) {
	return r.getReturnRequest(ctx, `WHERE rr.user_id = $1 AND o.order_number = $2`, userID, orderNumber)
}

func (r *CommerceRepository) ListReturnRequests(ctx context.Context, actorID int64) ([]domaincommerce.ReturnRequest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rr.id, o.order_number, rr.user_id, u.email, o.total_cents, o.currency,
		       rr.status, rr.reason, rr.created_at, rr.updated_at
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		JOIN users u ON u.id = rr.user_id
		WHERE EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = $1 AND ur.role_slug IN ('operations_returns', 'marketplace_admin', 'super_admin'))
		ORDER BY CASE rr.status WHEN 'pending' THEN 0 WHEN 'approved' THEN 1 ELSE 2 END, rr.created_at ASC, rr.id ASC
		LIMIT 200`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]domaincommerce.ReturnRequest, 0)
	for rows.Next() {
		var request domaincommerce.ReturnRequest
		if err := rows.Scan(&request.ID, &request.OrderNumber, &request.UserID, &request.CustomerEmail, &request.AmountCents, &request.Currency, &request.Status, &request.Reason, &request.CreatedAt, &request.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (r *CommerceRepository) UpdateReturnRequest(ctx context.Context, actorID, requestID int64, status, reason string) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID int64
		err := tx.QueryRow(transactionContext, `
			UPDATE return_requests rr
			SET status = $1, reason = CASE WHEN $3 <> '' THEN $3 ELSE rr.reason END, updated_at = NOW()
			WHERE rr.id = $2
			  AND EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = $4 AND ur.role_slug IN ('operations_returns', 'marketplace_admin', 'super_admin'))
			  AND ((rr.status = 'pending' AND $1 IN ('approved', 'rejected')) OR (rr.status = 'approved' AND $1 = 'received'))
			RETURNING rr.order_id`, status, requestID, reason, actorID).Scan(&orderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrReturnState
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'operations.return_status_changed', 'return_request', $2, $3, jsonb_build_object('status', $4, 'reason', $5, 'order_id', $6))`, fmt.Sprint(actorID), fmt.Sprint(requestID), ports.RequestID(transactionContext), status, reason, orderID)
		return err
	})
}

func (r *CommerceRepository) getReturnRequest(ctx context.Context, filter string, args ...any) (domaincommerce.ReturnRequest, error) {
	var request domaincommerce.ReturnRequest
	err := r.pool.QueryRow(ctx, `
		SELECT rr.id, o.order_number, rr.user_id, u.email, o.total_cents, o.currency,
		       rr.status, rr.reason, rr.created_at, rr.updated_at
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		JOIN users u ON u.id = rr.user_id `+filter, args...).Scan(&request.ID, &request.OrderNumber, &request.UserID, &request.CustomerEmail, &request.AmountCents, &request.Currency, &request.Status, &request.Reason, &request.CreatedAt, &request.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.ReturnRequest{}, ports.ErrReturnNotFound
	}
	return request, err
}

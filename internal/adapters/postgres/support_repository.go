package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) CreateSupportTicket(ctx context.Context, userID int64, orderNumber, subject, message string) (domaincommerce.SupportTicket, error) {
	var ticket domaincommerce.SupportTicket
	orderNumber = strings.TrimSpace(orderNumber)
	ticketNumber := fmt.Sprintf("SUP-%d-%d", time.Now().UTC().UnixNano(), userID)
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID *int64
		if orderNumber != "" {
			var value int64
			if err := tx.QueryRow(transactionContext, `SELECT id FROM orders WHERE user_id = $1 AND order_number = $2`, userID, orderNumber).Scan(&value); errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrOrderNotFound
			} else if err != nil {
				return err
			} else {
				orderID = &value
			}
		}
		if err := tx.QueryRow(transactionContext, `
			INSERT INTO support_tickets (ticket_number, user_id, order_id, subject, message)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, ticket_number, COALESCE($6, ''), subject, message, status, priority, created_at, updated_at`, ticketNumber, userID, orderID, subject, message, orderNumber).Scan(&ticket.ID, &ticket.TicketNumber, &ticket.OrderNumber, &ticket.Subject, &ticket.Message, &ticket.Status, &ticket.Priority, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"ticket_number": ticket.TicketNumber, "order_number": ticket.OrderNumber})
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'support.ticket_created', 'support_ticket', $2, $3, $4)`, fmt.Sprint(userID), fmt.Sprint(ticket.ID), ports.RequestID(transactionContext), metadata)
		return err
	})
	return ticket, err
}

func (r *CommerceRepository) ListCustomerSupportTickets(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error) {
	return r.listSupportTickets(ctx, `WHERE st.user_id = $1`, userID)
}

func (r *CommerceRepository) ListSellerSupportTickets(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error) {
	// Seller support requests are owned by the authenticated seller account,
	// just like customer requests are owned by the customer account. Keeping
	// this query user-scoped prevents another seller's correspondence from
	// entering the workspace while allowing the shared support queue to triage
	// both kinds of requester.
	return r.listSupportTickets(ctx, `WHERE st.user_id = $1`, userID)
}

func (r *CommerceRepository) ListSupportTickets(ctx context.Context, limit int) ([]domaincommerce.SupportTicket, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	return r.listSupportTickets(ctx, `ORDER BY st.updated_at DESC, st.id DESC LIMIT $1`, limit)
}

func (r *CommerceRepository) listSupportTickets(ctx context.Context, suffix string, arg any) ([]domaincommerce.SupportTicket, error) {
	query := `
		SELECT st.id, st.ticket_number, COALESCE(o.order_number, ''), st.subject, st.message, st.status, st.priority,
		       CASE WHEN st.user_id IS NOT NULL THEN 'Customer ' || LEFT(COALESCE(u.display_name, 'account'), 1) || '***' ELSE 'Customer' END,
		       CASE WHEN POSITION('@' IN u.email) > 1 THEN LEFT(u.email, 1) || '***' || SUBSTRING(u.email FROM POSITION('@' IN u.email)) ELSE '***' END,
		       st.agent_note, st.created_at, st.updated_at
		FROM support_tickets st
		JOIN users u ON u.id = st.user_id
		LEFT JOIN orders o ON o.id = st.order_id
		` + suffix
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tickets := make([]domaincommerce.SupportTicket, 0)
	for rows.Next() {
		var ticket domaincommerce.SupportTicket
		if err := rows.Scan(&ticket.ID, &ticket.TicketNumber, &ticket.OrderNumber, &ticket.Subject, &ticket.Message, &ticket.Status, &ticket.Priority, &ticket.CustomerLabel, &ticket.CustomerEmailMasked, &ticket.AgentNote, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	return tickets, rows.Err()
}

func (r *CommerceRepository) UpdateSupportTicket(ctx context.Context, actorID, ticketID int64, status, note string) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var current string
		if err := tx.QueryRow(transactionContext, `SELECT status FROM support_tickets WHERE id = $1 FOR UPDATE`, ticketID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrSupportNotFound
		} else if err != nil {
			return err
		}
		if current == "closed" && status != "closed" {
			return ports.ErrSupportState
		}
		valid := current == status || (current == "open" && (status == "in_progress" || status == "waiting_customer" || status == "resolved")) || (current == "in_progress" && (status == "waiting_customer" || status == "resolved")) || (current == "waiting_customer" && (status == "in_progress" || status == "resolved")) || (current == "resolved" && status == "closed")
		if !valid {
			return ports.ErrSupportState
		}
		if _, err := tx.Exec(transactionContext, `UPDATE support_tickets SET status = $1, agent_note = $2, updated_by = $3, updated_at = NOW() WHERE id = $4`, status, note, actorID, ticketID); err != nil {
			return err
		}
		metadata, _ := json.Marshal(map[string]any{"from_status": current, "to_status": status, "note": note})
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'support.ticket_updated', 'support_ticket', $2, $3, $4)`, fmt.Sprint(actorID), fmt.Sprint(ticketID), ports.RequestID(transactionContext), metadata)
		return err
	})
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainprivacy "github.com/wecratfs/commerce/internal/domain/privacy"
	"github.com/wecratfs/commerce/internal/ports"
)

type PrivacyRepository struct{ pool *pgxpool.Pool }

func NewPrivacyRepository(pool *pgxpool.Pool) *PrivacyRepository {
	return &PrivacyRepository{pool: pool}
}

func (r *PrivacyRepository) GetCenter(ctx context.Context, userID int64) (domainprivacy.Center, error) {
	center := domainprivacy.Center{Consents: []domainprivacy.Consent{}, Requests: []domainprivacy.Request{}}
	if err := r.pool.QueryRow(ctx, `SELECT email_marketing FROM marketing_preferences WHERE user_id = $1`, userID).Scan(&center.EmailMarketing); errors.Is(err, pgx.ErrNoRows) {
		center.EmailMarketing = false
	} else if err != nil {
		return center, err
	}
	consentRows, err := r.pool.Query(ctx, `SELECT purpose, policy_version, granted, source, created_at FROM consent_records WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return center, err
	}
	for consentRows.Next() {
		var consent domainprivacy.Consent
		if err := consentRows.Scan(&consent.Purpose, &consent.PolicyVersion, &consent.Granted, &consent.Source, &consent.CreatedAt); err != nil {
			consentRows.Close()
			return center, err
		}
		center.Consents = append(center.Consents, consent)
	}
	if err := consentRows.Err(); err != nil {
		consentRows.Close()
		return center, err
	}
	consentRows.Close()
	requestRows, err := r.pool.Query(ctx, `SELECT id, request_type, status, details, due_at, created_at FROM privacy_requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return center, err
	}
	for requestRows.Next() {
		var request domainprivacy.Request
		var details []byte
		if err := requestRows.Scan(&request.ID, &request.Type, &request.Status, &details, &request.DueAt, &request.CreatedAt); err != nil {
			requestRows.Close()
			return center, err
		}
		request.Details = string(details)
		center.Requests = append(center.Requests, request)
	}
	if err := requestRows.Err(); err != nil {
		requestRows.Close()
		return center, err
	}
	requestRows.Close()
	return center, nil
}

func (r *PrivacyRepository) RecordConsent(ctx context.Context, userID int64, purpose, policyVersion string, granted bool, source string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO consent_records (user_id, purpose, policy_version, granted, source) VALUES ($1, $2, $3, $4, $5)`, userID, purpose, policyVersion, granted, source)
	return err
}

func (r *PrivacyRepository) SetEmailMarketing(ctx context.Context, userID int64, enabled bool) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO marketing_preferences (user_id, email_marketing) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET email_marketing = EXCLUDED.email_marketing, updated_at = NOW()`, userID, enabled)
	return err
}

func (r *PrivacyRepository) CreatePrivacyRequest(ctx context.Context, userID int64, requestType, details string) (domainprivacy.Request, error) {
	detailsJSON, err := json.Marshal(map[string]string{"message": details})
	if err != nil {
		return domainprivacy.Request{}, err
	}
	var request domainprivacy.Request
	err = r.pool.QueryRow(ctx, `INSERT INTO privacy_requests (user_id, request_type, details, due_at) VALUES ($1, $2, $3, $4) RETURNING id, request_type, status, details, due_at, created_at`, userID, requestType, detailsJSON, time.Now().UTC().Add(30*24*time.Hour)).Scan(&request.ID, &request.Type, &request.Status, &detailsJSON, &request.DueAt, &request.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainprivacy.Request{}, ports.ErrNotFound
	}
	request.Details = string(detailsJSON)
	return request, err
}

func (r *PrivacyRepository) ExecuteDeletion(ctx context.Context, userID, requestID int64) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var requestType, status string
		if err := tx.QueryRow(transactionContext, `SELECT request_type, status FROM privacy_requests WHERE id = $1 AND user_id = $2 FOR UPDATE`, requestID, userID).Scan(&requestType, &status); errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrPrivacyRequestNotFound
		} else if err != nil {
			return err
		}
		if requestType != "deletion" || (status != "open" && status != "in_review") {
			return ports.ErrPrivacyRequestState
		}

		var userStatus string
		if err := tx.QueryRow(transactionContext, `SELECT status FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&userStatus); errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrPrivacyRequestNotFound
		} else if err != nil {
			return err
		}
		if userStatus == "deleted" {
			return ports.ErrPrivacyRequestState
		}

		// Orders, payment attempts, and consent history remain available for
		// accounting, dispute, and compliance retention. Direct account data
		// and delivery details are removed or anonymized in the same transaction.
		if _, err := tx.Exec(transactionContext, `UPDATE orders SET address_snapshot = '{"redacted":true}'::jsonb, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE sellers SET display_name = 'Deleted seller', status = 'closed', updated_at = NOW() WHERE owner_user_id = $1`, userID); err != nil {
			return err
		}
		for _, statement := range []string{
			`DELETE FROM addresses WHERE user_id = $1`,
			`DELETE FROM wishlist_items WHERE user_id = $1`,
			`DELETE FROM carts WHERE user_id = $1`,
			`DELETE FROM marketing_preferences WHERE user_id = $1`,
			`DELETE FROM credentials WHERE user_id = $1`,
			`DELETE FROM sessions_metadata WHERE user_id = $1`,
		} {
			if _, err := tx.Exec(transactionContext, statement, userID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(transactionContext, `UPDATE users SET email = $2, display_name = 'Deleted account', status = 'deleted', updated_at = NOW() WHERE id = $1`, userID, fmt.Sprintf("deleted+%d@invalid.wecratfs", userID)); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE privacy_requests SET status = 'completed', details = details || jsonb_build_object('completed_at', NOW(), 'retained_records', jsonb_build_array('orders', 'payment_attempts', 'consent_history')), updated_at = NOW() WHERE id = $1 AND user_id = $2`, requestID, userID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'privacy.deletion_completed', 'user', $2, $3, $4)`, fmt.Sprint(userID), fmt.Sprint(userID), ports.RequestID(transactionContext), []byte(`{"retained_records":["orders","payment_attempts","consent_history"]}`))
		return err
	})
}

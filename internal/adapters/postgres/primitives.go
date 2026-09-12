package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/adapters/postgres/generated"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrSensitiveAuditMetadata = errors.New("audit metadata contains a sensitive field")

type IdempotencyRepository struct {
	queries *generated.Queries
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{queries: generated.New(pool)}
}

func (r *IdempotencyRepository) Reserve(ctx context.Context, scope, key, requestHash string, responseJSON []byte) (ports.IdempotencyRecord, bool, error) {
	record, err := r.queries.InsertIdempotencyKey(ctx, generated.InsertIdempotencyKeyParams{
		Scope:        scope,
		Key:          key,
		RequestHash:  requestHash,
		ResponseJson: responseJSON,
	})
	if err == nil {
		return idempotencyRecord(record), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ports.IdempotencyRecord{}, false, fmt.Errorf("reserve idempotency key: %w", err)
	}

	existing, err := r.queries.GetIdempotencyKey(ctx, generated.GetIdempotencyKeyParams{Scope: scope, Key: key})
	if err != nil {
		return ports.IdempotencyRecord{}, false, fmt.Errorf("read idempotency key: %w", err)
	}
	if existing.RequestHash != requestHash {
		return ports.IdempotencyRecord{}, false, ports.ErrIdempotencyConflict
	}
	return idempotencyRecord(existing), false, nil
}

type OutboxRepository struct {
	queries *generated.Queries
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{queries: generated.New(pool)}
}

func (r *OutboxRepository) Publish(ctx context.Context, event ports.OutboxEventInput) (ports.OutboxEvent, error) {
	published, err := r.queries.InsertOutboxEvent(ctx, generated.InsertOutboxEventParams{
		EventType:     event.EventType,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		Payload:       event.Payload,
		AvailableAt:   nullableTime(event.AvailableAt),
	})
	if err != nil {
		return ports.OutboxEvent{}, fmt.Errorf("publish outbox event: %w", err)
	}
	return outboxEvent(published), nil
}

func (r *OutboxRepository) Claim(ctx context.Context, limit int) ([]ports.OutboxEvent, error) {
	if limit < 1 {
		return []ports.OutboxEvent{}, nil
	}
	claimed, err := r.queries.ClaimOutboxEvents(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	events := make([]ports.OutboxEvent, 0, len(claimed))
	for _, event := range claimed {
		events = append(events, outboxEvent(event))
	}
	return events, nil
}

func (r *OutboxRepository) ClaimByType(ctx context.Context, eventType string, limit int) ([]ports.OutboxEvent, error) {
	if strings.TrimSpace(eventType) == "" || limit < 1 {
		return []ports.OutboxEvent{}, nil
	}
	claimed, err := r.queries.ClaimOutboxEventsByType(ctx, generated.ClaimOutboxEventsByTypeParams{EventType: eventType, BatchSize: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events by type: %w", err)
	}
	events := make([]ports.OutboxEvent, 0, len(claimed))
	for _, event := range claimed {
		events = append(events, outboxEvent(event))
	}
	return events, nil
}

func (r *OutboxRepository) Ack(ctx context.Context, id int64) error {
	if err := r.queries.AckOutboxEvent(ctx, id); err != nil {
		return fmt.Errorf("ack outbox event: %w", err)
	}
	return nil
}

func (r *OutboxRepository) Fail(ctx context.Context, id int64, retryAfter time.Duration, message string) error {
	if err := r.queries.FailOutboxEvent(ctx, generated.FailOutboxEventParams{
		RetryAfter: pgtype.Interval{Microseconds: retryAfter.Microseconds(), Valid: true},
		LastError:  pgtype.Text{String: message, Valid: message != ""},
		ID:         id,
	}); err != nil {
		return fmt.Errorf("fail outbox event: %w", err)
	}
	return nil
}

type AuditRepository struct {
	queries *generated.Queries
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{queries: generated.New(pool)}
}

func (r *AuditRepository) Record(ctx context.Context, record ports.AuditRecord) error {
	metadata := record.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	if err := validateAuditMetadata(metadata); err != nil {
		return err
	}
	if err := r.queries.InsertAuditLog(ctx, generated.InsertAuditLogParams{
		ActorID:      pgtype.Text{String: record.ActorID, Valid: record.ActorID != ""},
		Action:       record.Action,
		ResourceType: record.ResourceType,
		ResourceID:   record.ResourceID,
		RequestID:    pgtype.Text{String: record.RequestID, Valid: record.RequestID != ""},
		Metadata:     metadata,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func validateAuditMetadata(metadata []byte) error {
	var value any
	if err := json.Unmarshal(metadata, &value); err != nil {
		return errors.New("audit metadata must be valid JSON")
	}
	return validateAuditValue(value)
}

func validateAuditValue(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			keyLower := strings.ToLower(key)
			for _, forbidden := range []string{"password", "token", "secret", "card", "cvv", "kyc", "credential", "authorization"} {
				if strings.Contains(keyLower, forbidden) {
					return fmt.Errorf("%w: %s", ErrSensitiveAuditMetadata, key)
				}
			}
			if err := validateAuditValue(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateAuditValue(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func idempotencyRecord(record generated.IdempotencyKey) ports.IdempotencyRecord {
	return ports.IdempotencyRecord{
		Scope:        record.Scope,
		Key:          record.Key,
		RequestHash:  record.RequestHash,
		ResponseJSON: record.ResponseJson,
		CreatedAt:    record.CreatedAt.Time,
	}
}

func outboxEvent(event generated.OutboxEvent) ports.OutboxEvent {
	return ports.OutboxEvent{
		ID:            event.ID,
		EventType:     event.EventType,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		Payload:       event.Payload,
		AvailableAt:   event.AvailableAt.Time,
		ClaimedAt:     optionalTime(event.ClaimedAt),
		ProcessedAt:   optionalTime(event.ProcessedAt),
		Attempts:      int(event.Attempts),
		LastError:     event.LastError.String,
		CreatedAt:     event.CreatedAt.Time,
	}
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")

type IdempotencyRecord struct {
	Scope        string
	Key          string
	RequestHash  string
	ResponseJSON []byte
	CreatedAt    time.Time
}

type IdempotencyStore interface {
	Reserve(ctx context.Context, scope, key, requestHash string, responseJSON []byte) (record IdempotencyRecord, created bool, err error)
}

type OutboxEventInput struct {
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	AvailableAt   time.Time
}

type OutboxEvent struct {
	ID            int64
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	AvailableAt   time.Time
	ClaimedAt     *time.Time
	ProcessedAt   *time.Time
	Attempts      int
	LastError     string
	CreatedAt     time.Time
}

type OutboxStore interface {
	Publish(ctx context.Context, event OutboxEventInput) (OutboxEvent, error)
	Claim(ctx context.Context, limit int) ([]OutboxEvent, error)
	ClaimByType(ctx context.Context, eventType string, limit int) ([]OutboxEvent, error)
	Ack(ctx context.Context, id int64) error
	Fail(ctx context.Context, id int64, retryAfter time.Duration, message string) error
}

type AuditRecord struct {
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	RequestID    string
	Metadata     json.RawMessage
}

type AuditWriter interface {
	Record(ctx context.Context, record AuditRecord) error
}

-- name: GetIdempotencyKey :one
SELECT scope, key, request_hash, response_json, created_at
FROM idempotency_keys
WHERE scope = @scope AND key = @key;

-- name: InsertIdempotencyKey :one
INSERT INTO idempotency_keys (scope, key, request_hash, response_json)
VALUES (@scope, @key, @request_hash, @response_json)
ON CONFLICT (scope, key) DO NOTHING
RETURNING scope, key, request_hash, response_json, created_at;

-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload, available_at)
VALUES (@event_type, @aggregate_type, @aggregate_id, @payload, COALESCE(@available_at::timestamptz, NOW()))
RETURNING id, event_type, aggregate_type, aggregate_id, payload, available_at,
          claimed_at, processed_at, attempts, last_error, created_at;

-- name: ClaimOutboxEvents :many
WITH candidates AS (
    SELECT id
    FROM outbox_events
    WHERE processed_at IS NULL
      AND available_at <= NOW()
      AND (claimed_at IS NULL OR claimed_at < NOW() - INTERVAL '5 minutes')
    ORDER BY id
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events AS events
SET claimed_at = NOW(), attempts = events.attempts + 1
FROM candidates
WHERE events.id = candidates.id
RETURNING events.id, events.event_type, events.aggregate_type, events.aggregate_id,
          events.payload, events.available_at, events.claimed_at, events.processed_at,
          events.attempts, events.last_error, events.created_at;

-- name: ClaimOutboxEventsByType :many
WITH candidates AS (
    SELECT outbox_events.id
    FROM outbox_events
    WHERE outbox_events.event_type = @event_type
      AND outbox_events.processed_at IS NULL
      AND outbox_events.available_at <= NOW()
      AND (outbox_events.claimed_at IS NULL OR outbox_events.claimed_at < NOW() - INTERVAL '5 minutes')
    ORDER BY outbox_events.id
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events AS events
SET claimed_at = NOW(), attempts = events.attempts + 1
FROM candidates
WHERE events.id = candidates.id
RETURNING events.id, events.event_type, events.aggregate_type, events.aggregate_id,
          events.payload, events.available_at, events.claimed_at, events.processed_at,
          events.attempts, events.last_error, events.created_at;

-- name: AckOutboxEvent :exec
UPDATE outbox_events
SET processed_at = NOW(), claimed_at = NULL, last_error = NULL
WHERE id = @id AND processed_at IS NULL;

-- name: FailOutboxEvent :exec
UPDATE outbox_events
SET claimed_at = NULL, available_at = NOW() + @retry_after::interval, last_error = @last_error
WHERE id = @id AND processed_at IS NULL;

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata)
VALUES (@actor_id, @action, @resource_type, @resource_id, @request_id, @metadata);

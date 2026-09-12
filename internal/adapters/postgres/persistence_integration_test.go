package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/adapters/postgres/generated"
	"github.com/wecratfs/commerce/internal/ports"
)

func openIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create integration pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("apply integration migrations: %v", err)
	}
	return pool
}

func TestWithinTransactionCommitsRollsBackAndHonorsCancellation(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS persistence_integration_probe (
			marker TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`)
	if err != nil {
		t.Fatalf("create integration probe: %v", err)
	}
	marker := fmt.Sprintf("%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM persistence_integration_probe WHERE marker = $1", marker)
		_, _ = pool.Exec(context.Background(), "DROP TABLE IF EXISTS persistence_integration_probe")
	})

	wantRollback := errors.New("force rollback")
	err = WithinTransaction(ctx, pool, func(transactionContext context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(transactionContext, "INSERT INTO persistence_integration_probe (marker, value) VALUES ($1, $2)", marker, "rolled-back"); err != nil {
			return err
		}
		return wantRollback
	})
	if !errors.Is(err, wantRollback) {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM persistence_integration_probe WHERE marker = $1", marker).Scan(&count); err != nil {
		t.Fatalf("count rolled-back probe: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback left %d rows", count)
	}

	if err := WithinTransaction(ctx, pool, func(transactionContext context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(transactionContext, "INSERT INTO persistence_integration_probe (marker, value) VALUES ($1, $2)", marker, "committed")
		return err
	}); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM persistence_integration_probe WHERE marker = $1", marker).Scan(&count); err != nil {
		t.Fatalf("count committed probe: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one committed row, got %d", count)
	}

	canceled, cancelContext := context.WithCancel(ctx)
	cancelContext()
	if err := WithinTransaction(canceled, pool, func(context.Context, pgx.Tx) error { return nil }); err == nil {
		t.Fatal("expected canceled transaction to fail")
	}
}

func TestOutboxWriteRollsBackWithTransaction(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aggregateID := fmt.Sprintf("atomic-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE aggregate_id = $1", aggregateID)
	})

	wantRollback := errors.New("rollback outbox probe")
	err := WithinTransactionWithQueries(ctx, pool, func(transactionContext context.Context, queries *generated.Queries) error {
		_, err := queries.InsertOutboxEvent(transactionContext, generated.InsertOutboxEventParams{
			EventType:     "AtomicProbe",
			AggregateType: "Probe",
			AggregateID:   aggregateID,
			Payload:       []byte(`{"atomic":true}`),
		})
		if err != nil {
			return err
		}
		return wantRollback
	})
	if !errors.Is(err, wantRollback) {
		t.Fatalf("expected outbox transaction rollback, got %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1", aggregateID).Scan(&count); err != nil {
		t.Fatalf("count rolled-back outbox event: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback left %d outbox events", count)
	}
}

func TestBusinessMutationAndOutboxCommitAtomically(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	email := fmt.Sprintf("atomic-%d@example.invalid", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
		_, _ = pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE payload->>'probe' = $1", email)
	})

	wantRollback := errors.New("rollback business mutation probe")
	err := WithinTransactionWithQueries(ctx, pool, func(transactionContext context.Context, queries *generated.Queries) error {
		user, err := queries.InsertUser(transactionContext, generated.InsertUserParams{
			Email:       email,
			DisplayName: "Atomic Probe",
			Status:      "active",
		})
		if err != nil {
			return err
		}
		if _, err := queries.InsertOutboxEvent(transactionContext, generated.InsertOutboxEventParams{
			EventType:     "UserCreated",
			AggregateType: "User",
			AggregateID:   fmt.Sprint(user.ID),
			Payload:       []byte(fmt.Sprintf(`{"probe":%q}`, email)),
		}); err != nil {
			return err
		}
		return wantRollback
	})
	if !errors.Is(err, wantRollback) {
		t.Fatalf("expected atomic rollback, got %v", err)
	}
	var userCount, eventCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&userCount); err != nil {
		t.Fatalf("count rolled-back user: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE aggregate_type = 'User' AND payload->>'probe' = $1", email).Scan(&eventCount); err != nil {
		t.Fatalf("count rolled-back outbox event: %v", err)
	}
	if userCount != 0 || eventCount != 0 {
		t.Fatalf("atomic rollback left user=%d events=%d", userCount, eventCount)
	}

	if err := WithinTransactionWithQueries(ctx, pool, func(transactionContext context.Context, queries *generated.Queries) error {
		user, err := queries.InsertUser(transactionContext, generated.InsertUserParams{
			Email:       email,
			DisplayName: "Atomic Probe",
			Status:      "active",
		})
		if err != nil {
			return err
		}
		_, err = queries.InsertOutboxEvent(transactionContext, generated.InsertOutboxEventParams{
			EventType:     "UserCreated",
			AggregateType: "User",
			AggregateID:   fmt.Sprint(user.ID),
			Payload:       []byte(fmt.Sprintf(`{"probe":%q}`, email)),
		})
		return err
	}); err != nil {
		t.Fatalf("commit atomic user/outbox mutation: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&userCount); err != nil {
		t.Fatalf("count committed user: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected one committed user, got %d", userCount)
	}
}

func TestPersistencePrimitivesAreReplaySafeAndAuditable(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suffix := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	scope := "integration"
	key := suffix
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM idempotency_keys WHERE scope = $1 AND key = $2", scope, key)
		_, _ = pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE aggregate_id = $1", suffix)
		_, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE resource_id = $1", suffix)
	})

	idempotency := NewIdempotencyRepository(pool)
	first, created, err := idempotency.Reserve(ctx, scope, key, "hash-a", []byte(`{"result":"pending"}`))
	if err != nil {
		t.Fatalf("reserve first idempotency key: %v", err)
	}
	if !created || !sameJSON(first.ResponseJSON, []byte(`{"result":"pending"}`)) {
		t.Fatalf("unexpected first idempotency result: created=%t record=%+v", created, first)
	}
	second, created, err := idempotency.Reserve(ctx, scope, key, "hash-a", []byte(`{"different":"ignored"}`))
	if err != nil {
		t.Fatalf("reserve repeated idempotency key: %v", err)
	}
	if created || !sameJSON(second.ResponseJSON, []byte(`{"result":"pending"}`)) {
		t.Fatalf("repeated idempotency key was not replayed: created=%t record=%+v", created, second)
	}
	if _, _, err := idempotency.Reserve(ctx, scope, key, "hash-b", nil); !errors.Is(err, ports.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}

	outbox := NewOutboxRepository(pool)
	published, err := outbox.Publish(ctx, ports.OutboxEventInput{
		EventType:     "IntegrationProbe",
		AggregateType: "Probe",
		AggregateID:   suffix,
		Payload:       json.RawMessage(`{"safe":true}`),
		AvailableAt:   time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("publish outbox event: %v", err)
	}
	claimed, err := outbox.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("claim outbox event: %v", err)
	}
	var found bool
	for _, event := range claimed {
		if event.ID == published.ID {
			found = true
			if event.Attempts != 1 {
				t.Fatalf("expected first claim attempt, got %d", event.Attempts)
			}
		}
	}
	if !found {
		t.Fatalf("published event %d was not claimed", published.ID)
	}
	if err := outbox.Fail(ctx, published.ID, 0, "integration retry"); err != nil {
		t.Fatalf("fail outbox event: %v", err)
	}
	if err := outbox.Ack(ctx, published.ID); err != nil {
		t.Fatalf("ack outbox event: %v", err)
	}

	audit := NewAuditRepository(pool)
	if err := audit.Record(ctx, ports.AuditRecord{
		ActorID:      "integration-actor",
		Action:       "integration_probe",
		ResourceType: "probe",
		ResourceID:   suffix,
		RequestID:    "integration-request",
		Metadata:     json.RawMessage(`{"reason":"test"}`),
	}); err != nil {
		t.Fatalf("write audit record: %v", err)
	}
	if err := audit.Record(ctx, ports.AuditRecord{Metadata: json.RawMessage(`not-json`)}); err == nil {
		t.Fatal("expected invalid audit metadata to fail")
	}
	if err := audit.Record(ctx, ports.AuditRecord{Metadata: json.RawMessage(`{"password":"must-not-be-stored"}`)}); !errors.Is(err, ErrSensitiveAuditMetadata) {
		t.Fatalf("expected sensitive audit metadata to fail, got %v", err)
	}
}

func TestQueryCancellationAndPoolTimeoutAreBounded(t *testing.T) {
	pool := openIntegrationPool(t)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pool.Exec(canceled, "SELECT pg_sleep(1)"); err == nil {
		t.Fatal("expected canceled query to fail")
	}

	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	config.MaxConns = 1
	limitedPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("create limited pool: %v", err)
	}
	t.Cleanup(limitedPool.Close)

	connection, err := limitedPool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire only pool connection: %v", err)
	}
	defer connection.Release()
	poolContext, cancelPoolWait := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelPoolWait()
	if _, err := limitedPool.Exec(poolContext, "SELECT 1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected pool wait deadline, got %v", err)
	}
}

func sameJSON(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

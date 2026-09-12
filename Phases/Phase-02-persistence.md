# Phase 02 — Persistence foundation

**Status:** COMPLETE  
**PRD source:** Section 30, Phase 2, Sections 38–41, 58  
**Depends on:** Phase 01  
**Unlocks:** Phase 03

## Outcome

Establish PostgreSQL as the only authoritative business-data store with versioned migrations, generated sqlc access, bounded pooling, cancellable queries, and reusable transaction primitives.

## In scope

- migrations that build an empty database reproducibly;
- sqlc query definitions and generated output produced only by the generator;
- pgxpool configuration and transaction wrapper;
- context-aware repository methods;
- foundational tables for identity references, idempotency keys, outbox events, and audit logs;
- uniqueness, foreign keys, timestamps, and indexes required by current access patterns;
- outbox claim/ack/failure primitives with bounded worker integration seam.

## Out of scope

Feature-complete catalog, checkout, payment provider behavior, search indexing, or manual edits to generated sqlc files.

## Deliverables

- [x] migrations are version controlled and run through the startup migrator;
- [x] sqlc generation command is documented and reproducible;
- [x] repository methods accept and propagate `context.Context`;
- [x] transaction helper rolls back on error/cancellation;
- [x] idempotency records support replay-safe critical mutations;
- [x] outbox records are atomically created with business mutations;
- [x] audit records can capture actor/action/resource with JSON validation.

## Preview

The local system can create and inspect its persistence status from an empty database. The preview shows migration version and connectivity, not fabricated commerce records.

## Verification

- [x] destroy/recreate a disposable local database and run all migrations;
- [x] run sqlc generation and prove generated files are unchanged afterward;
- [x] integration test commit and rollback paths;
- [x] integration test query cancellation and pool exhaustion behavior;
- [x] test duplicate idempotency key returns the original result safely;
- [x] test outbox and business mutation commit atomically;
- [x] test audit writes reject passwords, tokens, card data, or raw KYC documents;
- [x] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G02

**PASS when:** a clean database reaches the expected schema, generated data access is reproducible, and transaction/idempotency/outbox/audit primitives are integration-tested.

**BLOCK when:** migrations depend on hand-edited state, generated sqlc output is manually patched, or a critical mutation can commit without its required outbox/idempotency record.

## Evidence

Record migration transcript, generator command/version, integration test output, schema checksum, and database engine version.

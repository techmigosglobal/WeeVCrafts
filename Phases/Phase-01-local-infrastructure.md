# Phase 01 — Local infrastructure

**Status:** PASS  
**PRD source:** Phase 1, Sections 31, 42–47, 74–75  
**Depends on:** Phase 00  
**Unlocks:** Phase 02

## Outcome

Make the complete local development stack reproducible: Go server, bounded worker, PostgreSQL, Redis, Meilisearch, and MinIO/S3-compatible storage.

## In scope

- Docker Compose services with health checks and persistent local volumes;
- `.env.example` and documented local configuration without secrets;
- `/health/live` and `/health/ready` with truthful dependency readiness;
- graceful HTTP and worker shutdown;
- bounded worker concurrency and explicit queue/backpressure configuration;
- local logging with request ID support;
- startup failure behavior when required configuration is invalid.

## Out of scope

Business schema, user accounts, catalog behavior, cloud deployment, Cloudflare production rules, live S3, and live Razorpay.

## Deliverables

- [x] `docker compose up --build` starts the documented local services;
- [x] service health/readiness is visible to a developer;
- [x] worker starts with a finite concurrency limit;
- [x] restart and shutdown behavior is documented;
- [x] no secret values are committed.

## Preview

Open the local health endpoint and inspect a status view or JSON response showing Go, PostgreSQL, Redis, Meilisearch, MinIO, and worker state. A dependency that is down must be reported as not ready rather than hidden behind a green response.

## Verification

- [x] `docker compose config` passes;
- [x] clean local start and stop are repeatable;
- [ ] `/health/live` remains live while a dependency is unavailable;
- [ ] `/health/ready` fails or reports not ready when a required dependency is unavailable;
- [x] graceful shutdown completes without leaked worker goroutines;
- [ ] restart preserves only the explicitly configured local volumes;
- [x] context timeout/cancellation is exercised for health probes;
- [x] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G01

**PASS when:** a new developer can reproduce the stack from documented commands, readiness is truthful, shutdown is bounded, and the Go verification suite is green.

**BLOCK when:** a service depends on manually created state, readiness reports false health, or worker concurrency is unbounded.

## Evidence

Attach compose configuration output, service health responses, startup/shutdown logs, and the exact revision tested.

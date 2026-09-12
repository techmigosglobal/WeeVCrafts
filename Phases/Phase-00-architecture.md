# Phase 00 — Architecture foundation

**Status:** PASS  
**PRD source:** Sections 1–4, 88–89  
**Depends on:** None  
**Unlocks:** Phase 01

## Outcome

Create the Go modular-monolith skeleton and freeze the boundaries that keep the Commerce Core independent from delivery mechanisms and providers.

## In scope

- repository layout for domain, application/use cases, ports, adapters, web transport, `/api/v1`, workers, migrations, templates, and static assets;
- domain/application contracts for `ProductRepository`, `PaymentGateway`, `SearchEngine`, `ObjectStorage`, `Cache`, session, and transaction boundaries;
- context propagation from request to application service to adapter;
- explicit dependency direction: adapters depend inward, business code does not depend on provider SDKs;
- configuration boundary with no committed secrets;
- baseline health route and application bootstrap seam;
- test seams for in-memory/fake adapters without putting business fixtures in production runtime.

## Out of scope

Business features, database schema, authentication behavior, live integrations, UI screens, mobile clients, and production deployment.

## Deliverables

- [x] documented package/module dependency graph;
- [x] interfaces owned by the application/domain boundary;
- [x] PostgreSQL catalog and health adapters registered at composition root;
- [x] reserved `/api/v1` transport package that returns JSON only;
- [x] web transport package that owns templates/HTMX concerns;
- [x] context and error-handling conventions;
- [x] architecture notes record the provider-neutral contract choices.

## Preview

The foundation preview is a startable Go application with a live endpoint and a test composition using fake ports. The preview must make clear that no catalog, payment, or production data exists yet.

## Verification

- [x] `gofmt -w` has been run on changed Go files;
- [x] `go vet ./...` passes;
- [x] `go test ./...` passes;
- [x] `go test -race ./...` passes;
- [x] import scan proves domain/application packages do not import HTMX, PostgreSQL implementation packages, Redis implementation packages, Meilisearch SDK, S3 SDK, or Razorpay SDK;
- [x] context cancellation reaches a fake adapter in a unit test;
- [x] web and API handlers are tested through their public seams;
- [x] no global mutable business state or unbounded worker exists.

## Gate G00

**PASS when:** the application builds from a clean checkout, the dependency direction is documented and mechanically checked, ports are independently testable, and all required Go checks pass.

**BLOCK when:** business code imports an adapter/provider, contracts are hidden in handlers, or the app requires a real external service to run unit tests.

## Evidence

Record the code revision, package graph/import scan, command output, and the architecture decision record in the phase handoff.

# Phase 05 — Inventory and pricing

**Status:** IN PROGRESS  
**PRD source:** Sections 30, 38, 76–81, Phase 5  
**Depends on:** Phase 04  
**Unlocks:** Phase 06 and Phase 09

## Outcome

Make price calculation and stock reservation authoritative, transactional, and safe under concurrency.

## In scope

- price lists, active prices, promotion inputs, and money/rounding rules;
- inventory locations, quantities, transactions, and reservations;
- reserve, release, and commit reservation lifecycle;
- PostgreSQL row locks or verified conditional updates;
- expiration/recovery for abandoned reservations;
- inventory movement audit trail;
- read models that may be cached but never decide availability.

## Out of scope

Payment capture, shipping carrier integration, search, and production load tuning beyond the concurrency checkpoint.

## Deliverables

- [ ] server-side final price calculation;
- [ ] stock cannot become negative through supported mutations;
- [ ] reservation has explicit owner, expiry, and state;
- [ ] release/commit operations are idempotent;
- [ ] stock changes are represented as auditable transactions;
- [ ] product pages show availability from authoritative application reads.

## Preview

Reviewer can see current price and availability, reserve stock through the checkout path, and observe that a second request receives a stock error when no quantity remains. Any cache outage leaves correctness unchanged.

## Verification

- [ ] unit tests cover money arithmetic, rounding, price selection, and promotion boundaries;
- [ ] integration tests cover reservation, release, commit, expiry, and retries;
- [ ] run the 100-concurrent-attempt checkpoint repeatedly for stock 10;
- [ ] result is exactly 10 successful reservations, 90 stock errors, and zero overselling;
- [ ] test concurrent reservation and cancellation/release interactions;
- [ ] test database rollback leaves stock and reservations consistent;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G05

**PASS when:** prices are recalculated server-side, reservations are transactional/idempotent, and repeated concurrency runs show zero overselling.

**BLOCK when:** Redis or browser state decides stock, reservations can be double-committed, or any run creates negative/oversold inventory.

## Evidence

Attach money tests, reservation lifecycle output, database transaction traces, and repeated concurrency results with configuration recorded.

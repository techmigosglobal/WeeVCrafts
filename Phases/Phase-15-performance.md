# Phase 15 — Performance and capacity

**Status:** PLANNED  
**PRD source:** Sections 31–37, 42–47, 74–81, Phase 15  
**Depends on:** Phases 07, 09, 13, and 14  
**Unlocks:** Phase 16 and Phase 17

## Outcome

Measure the real bottlenecks and prove the agreed performance, concurrency, and resource-boundary targets under realistic load.

## In scope

- k6 or equivalent scenarios for browse, search, shopping, checkout, and mixed traffic;
- staged load at 10, 50, 100, 250, 500, and 1000 users before increasing further;
- P95/P99 and error-rate measurements against PRD targets;
- PostgreSQL query/lock/connection review;
- pgxpool, Redis, Meilisearch, worker, Go runtime, and pprof measurements;
- bounded worker/goroutine verification and backpressure;
- repeated concurrency and idempotency checkpoints;
- optimization only after a captured baseline.

## Out of scope

Premature microservices, read replicas, dedicated clusters, or tuning based only on intuition.

## Deliverables

- [ ] versioned load scenarios and repeatable runner configuration;
- [ ] baseline and final reports with endpoint/resource metrics;
- [ ] SLO comparison and approved exception record where needed;
- [ ] pool, query, cache, search, worker, and goroutine evidence;
- [ ] concurrency, idempotency, and race-test results.

## Targets

| Operation | Initial target |
| --- | ---: |
| Cached GET P95 | `<150 ms` |
| Product GET P95 | `<250 ms` |
| Search P95 | `<300 ms` |
| Cart mutation P95 | `<350 ms` |
| Checkout internal P95 | `<500 ms` |
| Error rate | `<1%` |

External payment latency is measured separately.

## Preview

Reviewer receives a performance report/dashboard showing load level, endpoint latency, error rate, resource use, cache/search/worker metrics, and known bottlenecks. It is evidence, not a visual storefront feature.

## Verification

- [ ] each load scenario is versioned and repeatable;
- [ ] baseline and post-change reports use the same workload/configuration;
- [ ] P95/P99 and error rate are compared with targets;
- [ ] database pool remains bounded and slow queries/locks are reviewed;
- [ ] Redis hit ratio/eviction/latency and Meilisearch indexing/search latency are recorded;
- [ ] worker queue depth, oldest-job age, failure rate, and concurrency are recorded;
- [ ] no unbounded goroutines or retries are found;
- [ ] 100-way inventory and 20-way identical-checkout tests pass under race detection;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G15

**PASS when:** the agreed target load has a current report showing SLO compliance or an explicitly approved exception, with bounded resource behavior and no unresolved race.

**BLOCK when:** measurements are missing, optimizations lack a baseline, or unbounded resource growth appears under target load.

## Evidence

Attach scripts/configuration, baseline and final reports, dashboards/metrics, profiling notes, and approved exceptions if any.

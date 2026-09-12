# Phase 07 — Search and cache

**Status:** IN PROGRESS  
**PRD source:** Sections 35–37, 48, Phase 7  
**Depends on:** Phase 06 and Phase 02  
**Unlocks:** Phase 08

## Outcome

Add disposable Meilisearch product discovery and Redis cache-aside behavior without moving business truth out of PostgreSQL.

## In scope

- product change events through the outbox;
- bounded worker synchronization into Meilisearch;
- typo-tolerant search, filters, sorting, facets, and pagination;
- complete index rebuild from PostgreSQL;
- Redis cache-aside for measured read paths;
- cache keys, TTLs, invalidation, and stampede protection;
- honest empty, stale, unavailable-index, and cache-down behavior;
- shareable URL query state for search and filters.

## Out of scope

Recommendations, a dedicated search cluster, cache-driven writes, or treating search results as authoritative order/catalog data.

## Deliverables

- [ ] public search route and query contract;
- [ ] product synchronization and retry/failure handling;
- [ ] rebuild command that can repopulate an empty index;
- [ ] cache adapter and singleflight/lock strategy for hot reads;
- [ ] metrics for search latency, indexing backlog, cache hits, misses, and errors;
- [ ] fallback behavior documented for Meilisearch/Redis outages.

## Preview

Reviewer can search with a typo, filter and sort results, paginate, copy the URL, and see an explicit degraded/empty state when the index is unavailable. Cache loss slows reads but does not alter stock, price, orders, or other business state.

## Verification

- [ ] index a product change and verify eventual search visibility;
- [ ] clear the index and rebuild it completely from PostgreSQL;
- [ ] test typo search, filters, sorting, facets, pagination, and no results;
- [ ] test stale/out-of-order indexing events;
- [ ] test cache hit, miss, expiration, invalidation, backend outage, and stampede protection;
- [ ] confirm no business mutation succeeds only because Redis/Meilisearch is available;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G07

**PASS when:** search is rebuildable, cache is disposable, URL state is shareable, and outages degrade safely without changing PostgreSQL-backed business correctness.

**BLOCK when:** an index or cache is the only copy of a business fact, unbounded sync workers exist, or stale search data can authorize a purchase.

## Evidence

Attach search/API previews, rebuild transcript, cache test output, queue metrics, and outage behavior notes.

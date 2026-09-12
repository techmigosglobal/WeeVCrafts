# Local performance checkpoint

The repeatable local smoke scenario is `scripts/load-smoke.js`, executed by
`scripts/run-load-smoke.sh`. It exercises public browse, PostgreSQL-backed
search, and the live probe without creating business records or requiring
credentials.

Run it against a disposable local stack:

```sh
docker compose up -d --build
BASE_URL=http://localhost:8080 scripts/run-load-smoke.sh
```

The scenario stages 10 virtual users, then 50, and records request failure rate
and P95/P99 latency. It is a baseline smoke workload, not production approval.
The remaining performance gate requires repeated 100-way inventory and
20-way identical-checkout runs, PostgreSQL/Redis/Meilisearch/worker metrics,
and comparison with the PRD SLOs.

The latest local result is recorded in
`docs/performance-load-2026-09-06.md`.

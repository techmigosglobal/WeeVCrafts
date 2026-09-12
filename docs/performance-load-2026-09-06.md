# Local k6 baseline — 2026-09-06

```text
environment: local Docker Compose stack
scenario: scripts/load-smoke.js
workload: 10 → 50 virtual users over 30s/60s/30s stages
business mutations: none
requests: 7,947
checks: 7,947 passed, 0 failed
http failures: 0.00%
http duration: avg 6.70ms, p95 17.88ms, p99 36.26ms, max 124.23ms
throughput: 66.05 requests/second
iterations: 2,649
result: PASS for the current local read-only smoke baseline
```

The workload covered PostgreSQL-backed product browse, search, and the live
probe. It did not cover authenticated checkout, payment-provider latency,
backup/restore, production networking, or a multi-node deployment, so this
artifact does not close the production performance gate by itself.

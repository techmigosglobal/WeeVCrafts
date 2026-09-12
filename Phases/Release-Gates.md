# RES2 release gates

These gates apply in addition to the detailed checks in every phase file. A gate is `PASS` only when its evidence is current, repeatable, and attached to the implementation or release record.

## Gate rules

- A failed or missing required check blocks phase completion.
- A test result from an older code state is not current evidence.
- A passing unit test cannot substitute for integration, browser, database, payment, restore, load, or security evidence where that layer is required.
- Any contract change must record the reason, affected consumers, compatibility decision, and updated verification before the phase can pass.
- Development seed data must be explicit and isolated; it must never silently masquerade as production data.

## Phase exit gates

| Gate | Required evidence | Blocks progression when |
| --- | --- | --- |
| G00 Architecture | Import-boundary check, port contracts, `gofmt`, `go vet`, `go test ./...`, `go test -race ./...` | Business code imports an adapter or the core cannot be tested independently |
| G01 Infrastructure | `docker compose config`, healthy PostgreSQL/Redis/Meilisearch/MinIO, live/ready checks, shutdown evidence | A required local dependency is non-reproducible or readiness lies |
| G02 Persistence | Empty-database migration run, generated sqlc check, transaction/cancellation/idempotency/outbox/audit tests | Data cannot be created from zero or critical mutation primitives are unproven |
| G03 Auth | Registration/login/logout/session/CSRF/RBAC/rate-limit tests, cookie inspection | Authentication or role isolation can be bypassed |
| G04 Catalog | Seller ownership tests, admin approval tests, cursor pagination tests, catalog preview | A seller can read or mutate another seller's data |
| G05 Inventory | Repeated 100-concurrent-attempt test, reservation lifecycle tests, zero overselling evidence | Inventory is cache-authoritative or oversells |
| G06 Storefront | Browser/manual preview at supported widths, non-HTMX fallback, keyboard and reduced-motion checks | The storefront requires JavaScript/HTMX or fails accessibility basics |
| G07 Search/cache | Index sync/rebuild evidence, typo/filter/sort tests, cache hit/miss/invalidation/stampede tests | Search becomes the source of truth or cache loss changes business state |
| G08 Cart | Guest-to-user merge tests, quantity/retry tests, HTMX and normal-form paths | Duplicate clicks or merge behavior loses items or creates invalid state |
| G09 Orders | Atomic order/reservation/outbox test, concurrent idempotency test, order preview | Retries can create duplicate orders or partial commits |
| G10 Payments | Razorpay test-mode signature/webhook/replay/failure/refund evidence | Browser-only payment success can mark an order paid |
| G11 Operations | Role matrix tests and audit evidence for seller/admin/finance/support actions | Privileged users can cross tenant or finance/security boundaries |
| G12 Storage | Presigned upload, private object denial, public media, variant and cleanup tests | Private seller/customer files are guessable or publicly accessible |
| G13 API | JSON contract tests, error envelope, auth/pagination/OpenAPI evidence, shared application-service test | API duplicates or bypasses commerce rules |
| G14 Privacy | Notice/consent/withdrawal/access/correction/erasure/retention evidence and legal review | Privacy operations are only documented, not executable, or legal review is absent |
| G15 Performance | k6/load report, p95/p99 SLOs, pool/query/cache/worker metrics, race tests | Resource use is unbounded or target load misses agreed SLOs |
| G16 Security | Negative tests, dependency/secrets/log review, upload/webhook/replay checks | Critical/high release-blocking findings remain unresolved |
| G17 Staging | Clean deploy, HTTPS/Cloudflare, migration, backup restore, Razorpay test webhook, full E2E | Staging cannot reproduce the release or restore a working system |
| G18 Production | All critical flows, monitoring, rollback, backup/restore, load, security, privacy, and operations evidence | Any production blocker from the PRD remains open |

## Final production gate

Production is blocked if any of these conditions is true:

- inventory can oversell;
- payment retries can create duplicate orders or payment state;
- seller isolation or admin authorization can be bypassed;
- Razorpay webhook signatures are not verified and replay-safe;
- passwords, tokens, payment credentials, or KYC documents appear in logs;
- clean migrations or backup restoration fail;
- `go test -race ./...` reports an unresolved race;
- checkout performance/error targets fail under agreed load;
- privacy/legal review is incomplete;
- a required rollback or incident procedure has not been rehearsed.

## Evidence record

Each gate record should include:

```text
gate: Gxx
code revision:
date/time:
environment:
commands or manual steps:
artifacts/URLs:
result: PASS | FAIL | PARTIAL
known issues:
owner:
```


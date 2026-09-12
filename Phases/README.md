# RES2 execution phases

This folder turns [`../Prd.md`](../Prd.md) into an execution plan for the RES2 multi-vendor marketplace. The PRD remains the product and architecture source of truth; these files define sequencing, observable previews, verification evidence, and phase gates.

## How to use this plan

1. Work on one phase at a time.
2. Do not start a phase until the previous phase gate is marked `PASS` with evidence.
3. Keep PostgreSQL authoritative. Redis, Meilisearch, object storage, and payment providers are adapters or disposable infrastructure.
4. Keep domain and application code independent of HTTP templates, HTMX, PostgreSQL implementation packages, Redis SDKs, Meilisearch SDKs, S3 SDKs, and Razorpay SDKs.
5. Mark a checkbox only after the linked verification has been run against the current code.
6. A preview is evidence of what a user can see; it is not automatically production readiness.

## Status legend

- `PLANNED`: not started.
- `IN PROGRESS`: implementation is active.
- `VERIFYING`: implementation exists and evidence is being collected.
- `PASS`: exit gate passed; phase may be closed.
- `BLOCKED`: a named dependency or failed gate prevents progression.

## Execution map

| Phase | Focus | First visible preview | Exit gate |
| --- | --- | --- | --- |
| 00 | Architecture contracts | Health of the empty application shell | Boundaries, ports, build, and tests are proven |
| 01 | Local infrastructure | Local service health | Compose stack is reproducible and ready |
| 02 | Persistence | Database-backed system status | Clean migrations and transactional primitives pass |
| 03 | Security and auth | Register, login, logout, session controls | Auth and role-isolation tests pass |
| 04 | Catalog | Seller/admin catalog workspace | Seller ownership and moderation are enforced |
| 05 | Inventory and pricing | Stock and reservation status | 100-way checkout concurrency has zero oversell |
| 06 | Storefront | Responsive Home → Category → Product pages | Web preview works with and without HTMX |
| 07 | Search and cache | Search, filters, sorting, URL state | Index rebuild and cache failure behavior pass |
| 08 | Cart and wishlist | Guest/auth cart and wishlist interactions | Merge and retry behavior is correct |
| 09 | Checkout and order | Review → pending order preview | Atomic, idempotent order creation passes |
| 10 | Razorpay test payments | Test payment success/failure/webhook flows | Signatures and duplicate events are safe |
| 11 | Seller and admin operations | Role-specific dashboards | Tenant isolation and audit evidence pass |
| 12 | Storage and media | Product media upload and rendering | Public/private object access is enforced |
| 13 | `/api/v1` readiness | JSON API and OpenAPI preview | API uses the same application services |
| 14 | Privacy | Privacy center and consent controls | Operational privacy flows and legal review pass |
| 15 | Performance | SLO/load report | Target load and bounded resources pass |
| 16 | Security testing | Security verification report | No unresolved release-blocking findings |
| 17 | Staging | Deployed end-to-end staging URL | Staging restore and purchase rehearsal pass |
| 18 | Production readiness | Release candidate preview | Production release gate is fully green |

## Preview ladder

The application should become visible progressively rather than waiting for the end:

| Preview | Available after | What must be visible | What it does not prove |
| --- | --- | --- | --- |
| Foundation preview | Phase 01 | Health endpoints and service status | Business behavior |
| Identity preview | Phase 03 | Register/login/logout/session controls | Catalog or checkout |
| Storefront prototype | Phase 06 | Real PostgreSQL-backed catalog pages, responsive layout, accessible interactions | Search, payment, or production readiness |
| Commerce prototype | Phase 09 | Browse → product → cart → checkout review → pending order | Razorpay confirmation, staging, or production readiness |
| V1 release candidate | Phase 18 | Browse → payment test/live configuration → order → operations, with API, privacy, performance, and security evidence | None of the release gates may be assumed without evidence |

Detailed preview rules are in [`V1-Prototype-Preview.md`](V1-Prototype-Preview.md).

## Phase files

- [Phase 00 — Architecture](Phase-00-architecture.md)
- [Phase 01 — Local infrastructure](Phase-01-local-infrastructure.md)
- [Phase 02 — Persistence](Phase-02-persistence.md)
- [Phase 03 — Security and authentication](Phase-03-security-auth.md)
- [Phase 04 — Catalog](Phase-04-catalog.md)
- [Phase 05 — Inventory and pricing](Phase-05-inventory-pricing.md)
- [Phase 06 — Storefront](Phase-06-storefront.md)
- [Phase 07 — Search and cache](Phase-07-search-cache.md)
- [Phase 08 — Cart and wishlist](Phase-08-cart-wishlist.md)
- [Phase 09 — Checkout and orders](Phase-09-checkout-orders.md)
- [Phase 10 — Razorpay test payments](Phase-10-razorpay.md)
- [Phase 11 — Seller and admin operations](Phase-11-seller-admin.md)
- [Phase 12 — Storage and media](Phase-12-storage.md)
- [Phase 13 — API/mobile readiness](Phase-13-api-v1.md)
- [Phase 14 — Privacy](Phase-14-privacy.md)
- [Phase 15 — Performance](Phase-15-performance.md)
- [Phase 16 — Security testing](Phase-16-security-testing.md)
- [Phase 17 — Staging](Phase-17-staging.md)
- [Phase 18 — Production readiness](Phase-18-production-readiness.md)

Cross-phase controls are consolidated in [`Release-Gates.md`](Release-Gates.md), and PRD coverage is mapped in [`Traceability.md`](Traceability.md).

## Working status

The current implementation/evidence status is maintained in
[`../docs/phase-status.md`](../docs/phase-status.md). The local worktree has
completed the architecture, infrastructure, and persistence foundations;
Phases 03–14 are in active implementation/verification; and Phases 15–18
remain external performance, security, staging, legal, and production gates.
The phase files describe requirements and do not claim a gate is `PASS` unless
the linked evidence is current.

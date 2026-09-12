# V1 prototype and preview contract

This document defines what a reviewer must be able to see while RES2 is being built. It separates a useful working prototype from a production release claim.

## Preview milestones

### Foundation preview — after Phase 01

Reviewer can start the local stack and see:

- Go application live status;
- readiness status for PostgreSQL, Redis, Meilisearch, and MinIO;
- worker status and graceful shutdown behavior.

Required evidence: reproducible compose command, health endpoint responses, and service logs. This preview contains no business fixture fallback.

### Storefront prototype — after Phase 06

Reviewer can open a local URL and navigate:

```text
Home → category/product listing → product detail
```

The preview must show real records from the local PostgreSQL-backed catalog. Development seed data must be created by an explicit seed/migration path and visibly identified as development data. Empty states must be shown when no data exists; the runtime must not silently inject products.

The preview must include:

- responsive desktop and mobile layouts;
- product images or an explicit empty-media state;
- accessible keyboard navigation and focus states;
- reduced-motion behavior;
- normal HTML form navigation when HTMX is disabled;
- HTMX partial updates where available;
- Alpine only for local visual state.

### Commerce prototype — after Phase 09

Reviewer can complete:

```text
Browse → product → add to cart → address → price review → create pending order
```

The prototype must visibly distinguish `pending payment` from `paid`. It must not label an order paid merely because a browser callback or development fixture says so. Inventory reservation, order idempotency, and outbox behavior must be backed by PostgreSQL tests.

### V1 release candidate — after Phase 18

Reviewer can validate the complete controlled flow:

```text
Browse → search → cart → checkout → Razorpay test/live configuration
       → verified webhook → order → seller/admin operations → refund/return path
```

The candidate also exposes the `/api/v1` contract, privacy center, operational metrics, backup/restore evidence, and security/load reports. It is not a release until the final production gate passes.

## Preview checklist

- [ ] preview URL or local command is recorded;
- [ ] code revision is recorded;
- [ ] data source is stated as PostgreSQL, with seed mode explicitly identified;
- [ ] empty states have been tested;
- [ ] screenshots or a short screen recording cover the primary flow;
- [ ] browser widths tested: mobile, tablet, desktop;
- [ ] keyboard-only and reduced-motion checks recorded;
- [ ] HTMX disabled fallback tested for the same flow;
- [ ] no password, token, card data, or private document appears in evidence;
- [ ] known limitations are written next to the preview rather than hidden.

## Prototype data policy

- PostgreSQL is authoritative in every environment.
- Redis and Meilisearch may be empty and rebuilt without losing business data.
- Development seed data is explicit, repeatable, and isolated from production.
- No fake successful payment, fake order, or fake stock is used as a runtime fallback.
- Razorpay test mode is allowed only in the payment test phase; live mode requires the production gate.

## Reviewer handoff

Every preview handoff should include:

```text
preview:
revision:
environment:
start command:
entry URLs:
test account/role: supplied out-of-band, never committed
primary flow:
evidence links/paths:
known limitations:
next gate:
```

## Current local handoff evidence

```text
preview: http://localhost:8080/
revision: working-tree snapshot; no commit is assumed
environment: Docker Compose development stack
start command: docker compose up -d --build
entry URLs: /, /products, /search, /cart, /checkout, /orders, /wishlist, /seller, /admin/products
data source: PostgreSQL records created by explicit local smoke/rehearsal actions; no startup fixture fallback
primary flow: register → wishlist/cart → checkout → pending order → payment-order attempt → cancel/release; privacy center → deletion request → authenticated eligible-data deletion
evidence: /tmp/wecratfs-v1-final-desktop.png, /tmp/wecratfs-v1-final-tablet.png, /tmp/wecratfs-v1-final-mobile.png, /tmp/wecratfs-v1-final-no-js-2.png
known limitations: Razorpay credentials are unset locally; payment-order attempt is expected to return 502 until configured; image variants/private document workflows, keyboard manual session, and legal/production gates remain open
next gate: G03–G14 dedicated verification, then G15–G18 external release evidence
```

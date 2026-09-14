# WeCratfs phase status

**Current status:** The locally verifiable V1 frontend is complete for the customer, seller, marketplace-admin, super-admin, and support role previews. It runs with mock data, is responsive, and is interactive without a backend; live integrations, staging, legal, and production gates remain separate follow-up work.

**Current gate:** Local frontend V1 — complete; full production-release gates remain open
**Completion estimate:** 100% of the requested locally verifiable frontend scope; approximately 45% of the broader production-release gate set.
**Source PRD:** [`../Prd.md`](../Prd.md)  
**Execution plan:** [`../Phases/README.md`](../Phases/README.md)

## Files changed

- Added Go module/bootstrap code under `cmd/` and `internal/`.
- Added PostgreSQL migrations and catalog repository under
  `internal/adapters/postgres/`.
- Added migration `0011_checkout_idempotency_hash.sql`, generated outbox query
  access, and identity JSON contract tags.
- Added migrations `0012_product_media_lifecycle.sql` and
  `0013_product_media_deletion_claim.sql` for verified media references and
  race-safe orphan cleanup claims.
- Added migration `0014_payment_refunds.sql` for one idempotent full-refund
  record per order, provider refund state, and bounded failure details.
- Added migration `0015_auth_action_tokens.sql` for hashed, one-time email
  verification and password-reset tokens plus the authoritative verification
  timestamp on users.
- Added migration `0016_seller_staff_permissions.sql`, seller-scoped staff
  membership management, granular permission storage, owner-only team routes,
  and catalog permission enforcement for seller staff.
- Added live seller onboarding state, marketplace seller approval/suspension,
  role-scoped audit-log reads, and Super Admin operational role management;
  privileged changes require a reason and receive request-aware audit records.
- Added domain, application, and provider-neutral port contracts under
  `internal/domain/`, `internal/application/`, and `internal/ports/`.
- Added JSON API, web template, health transport, responsive compiled stylesheet,
  and embedded storefront assets under `internal/transport/`.
- Added local stack files: `Dockerfile`, `compose.yaml`, `.env.example`, and
  `.dockerignore`.
- Added `scripts/check-architecture.sh`, `README.md`,
  `docs/architecture.md`, and `docs/design/fidelity-ledger.md`.
- Current pass files include `internal/adapters/email/`, migration
  `0015_auth_action_tokens.sql`, `internal/application/auth/recovery.go`,
  `internal/adapters/postgres/auth_action_repository.go`, search facet/metric
  ports and adapters, recovery API/web routes and templates, SMTP configuration
  in `internal/config/`, and the generated sqlc identity files.
- Added unit tests for the catalog service, migration version parsing, health
  transport, API JSON separation, and web empty/product/404 states.
- Added bounded worker runner and cancellation/shutdown tests under
  `internal/worker/`.
- Added `sqlc.yaml`, query inputs, generated PostgreSQL access, transaction
  helper, idempotency/outbox/audit adapters, identity foundation migration,
  identity role migration, identity repository, and PostgreSQL integration
  tests.
- Added provider-neutral identity and password-auth application contracts with
  focused unit tests, Redis-backed opaque sessions, CSRF-protected HTML auth
  routes, API auth, session revocation, and authentication rate limiting.
- Added seller/catalog ownership, moderation, variants/SKUs, inventory,
  reservations, guest/auth carts, wishlist storage, checkout, pending orders,
  payment intents, order history, and transactional outbox events.
- Added PostgreSQL-backed search fallback with Redis cache-aside, a
  Meilisearch HTTP adapter plus rebuild command, Razorpay HTTP/HMAC adapter,
  server-side payment-order preparation and webhook processor, S3/MinIO
  presigned media adapter, privacy center records, and OpenAPI documentation at
  `docs/api/openapi.yaml`.
- Added an additive search-options contract for category filtering, price/newest
  sorting, bounded page navigation, shareable web query state, PostgreSQL
  fallback ordering, and Meilisearch sortable attributes.
- Added responsive commerce templates for search, auth, cart, checkout,
  orders, cancellation, payment preparation, wishlist, seller workspace,
  moderation, privacy, and session controls, with embedded `commerce.css`
  additions and normal HTML fallbacks.
- Added bounded expired-reservation cleanup through the worker scheduler and
  PostgreSQL-backed cancellation/release paths.
- Added checkout request hashing so an idempotency key cannot be reused with a
  different cart/address request, plus concurrent identical-retry evidence.
- Added typed product-index outbox events, a bounded rebuild processor, and
  PostgreSQL hydration after Meilisearch discovery with safe fallback and
  bounded search-cache invalidation after rebuild.
- Added a provider-neutral application role-checker boundary, PostgreSQL-backed
  role authorization for catalog mutations, and transactional failed-login
  auditing with lockout state updates.
- Added an allowlisted, provider-neutral payment-webhook application boundary
  and PostgreSQL integration proof for signed capture, replay safety, atomic
  order settlement, reservation commitment, and unsupported-event rejection.
- Added an additive `PaymentRefundGateway` and refund repository contract,
  Razorpay normal-refund adapter, server-authoritative full-refund service,
  authenticated HTML/API routes, atomic `paid` → `refunded` settlement, and
  replay-safe refund audit records. Existing `PaymentGateway` callers and
  fakes remain unchanged.
- Added media lifecycle verification: PostgreSQL-backed pending/ready/deleting
  states, object metadata verification, stable public media paths, signed
  download resolution, seller ownership checks, and bounded orphan cleanup.
- Added provider-neutral password recovery and email-verification contracts,
  anonymous non-enumerating recovery routes, CSRF-protected web forms, JSON
  API routes, referrer suppression on token pages, password-reset session
  revocation, and a bounded standard-library SMTP adapter.
- Added category search facets from Meilisearch/PostgreSQL, bounded in-process
  search observations, `/health/metrics`, and stale-event tests proving the
  authoritative PostgreSQL record is rehydrated before disposable indexing.
- Added authenticated privacy deletion confirmation with transactional
  account-data removal/anonymization and retained-order evidence.
- Added `docs/privacy-retention.md`, `docs/processor-inventory.md`, and
  `docs/breach-response.md` as versioned operational baselines.
- Added repeatable PostgreSQL concurrency fixtures for 100 checkout attempts,
  reservation expiry, cancellation, and zero-overselling verification.
- Added the directional visual reference
  `docs/design/wecratfs-storefront-concept.png`.
- The master `Prd.md` remains unchanged.

## Architecture changes

- The public product name and runtime wordmark are exactly `WeCratfs`.
- PostgreSQL is the only authoritative source for identity, catalog, stock,
  carts, orders, payments, privacy records, and media metadata. The local
  database contains only records created by smoke rehearsals, never runtime
  demo fixtures.
- Redis, Meilisearch, S3-compatible storage, sessions, and payment providers
  are provider-neutral ports; their SDK/HTTP implementations stay in adapter
  packages and no provider type crosses domain/application boundaries.
- Versioned migrations 0001–0020 create the current commerce, privacy,
  webhook, media, refund, and account-action metadata model. Checkout, payment, media
  lifecycle, and processed-refund transitions use
  PostgreSQL transactions and row locks.
- `/api/v1` is JSON-only and template-free. HTML is server-rendered first;
  HTMX/Alpine remain progressive enhancements and are never business state.
- API checkout accepts the documented `Idempotency-Key` header (with a
  backwards-compatible JSON fallback), and structured API errors include the
  propagated request ID when the HTTP middleware provides one.
- This implementation pass adds the provider-neutral payment-order contract,
  reservation-reaper contract, wishlist read contract, typed index-event
  contract, and privacy deletion contract because the existing
  database writes had no complete V1 transport path; existing callers remain
  source-compatible through additive constructors and interfaces.
- Checkout now persists a request hash beside the idempotency key and returns
  a conflict for a different cart/address reuse. Search discovery may use
  Meilisearch, but product details are always rehydrated from PostgreSQL and
  index failures fall back to the authoritative repository.
- The moderation-list repository contract now accepts the requesting actor ID;
  this is a necessary security correction because the previous no-argument
  listing path could expose pending seller records to any authenticated user.
- Search sort/filter/pagination state is validated in the application service;
  Meilisearch is used only for discovery and PostgreSQL remains the hydrated
  source of truth, including when the index is unavailable.
- Payment webhooks accept only supported Razorpay-shaped event types after raw
  body verification; capture settlement, payment status, order status,
  reservation commitment, audit, and processed-event recording remain in one
  PostgreSQL transaction.
- Refund requests derive amount and currency from locked PostgreSQL order and
  payment rows. V1 supports one full refund per order, stores provider
  pending/processed/failed state, and marks the order refunded only after a
  validated provider response; provider-pending results remain for a future
  reconciliation worker.
- Browser media uploads are verified against object-store size and content
  type before PostgreSQL marks them ready. Public catalog references resolve
  through a short-lived signed download URL; the Go server does not proxy file
  bytes.
- Password-reset and email-verification tokens are generated in the application,
  stored only as SHA-256 hashes in PostgreSQL, consumed once inside a
  transaction, and never logged. SMTP delivery is optional by configuration and
  is disabled explicitly in the local Compose environment when unset.
- Search facets are disposable read data and search observations are bounded
  atomic counters; neither changes PostgreSQL business correctness.
- Rebuild commands and the bounded product-index worker invalidate at most
  1,000 disposable search-cache entries per run; business writes do not depend
  on cache availability.
- HTTP, PostgreSQL, and health-probe contexts are bounded and propagated.
  The HTTP server and background worker use finite timeouts and graceful
  shutdown.
- No runtime fake product seed or business fixture fallback is present.
- The storefront CSS is a checked-in static asset embedded by Go; Node.js/npm
  is not required for runtime or local startup.
- PostgreSQL catalog reads now use generated sqlc access; generated files are
  marked by sqlc and are not manually edited.
- API identity responses now use stable lower-snake-case JSON field names rather
  than leaking Go struct casing.

## Tests added

- Catalog application validation and context propagation tests.
- PostgreSQL migration filename/version tests.
- Health live/readiness behavior tests with cancellable probes.
- API JSON-only and method-contract tests.
- Web rendering tests for the exact brand name, truthful empty state, product
  record, unknown product response, policy route, and static cache headers.
- PostgreSQL integration tests for transaction rollback/commit/cancellation,
  atomic outbox rollback and business-mutation commit, idempotency
  replay/conflict, outbox lifecycle, and audit metadata validation/redaction.
- Argon2id, session expiry/revocation, commerce validation, Razorpay raw-body
  HMAC/order creation, storage key validation, privacy service, API
  auth/CSRF, wishlist validation, search rebuild, reservation cleanup, and
  responsive transport tests.
- PostgreSQL integration tests for 20 identical checkout retries, idempotency
  conflict rejection, guest-cart merge/clamping/unavailable-product behavior,
  transactional product-index events, and executable privacy deletion.
- Application tests for Meilisearch hydration/fallback/cache behavior,
  concurrent cache-miss deduplication, and the bounded product-index outbox
  processor.
- Application and transport tests for catalog role allow/deny policy and
  authentication rate-limit responses; PostgreSQL role/failed-login audit and
  Redis rate-limit integration tests pass against the local stack.
- Search application, Meilisearch adapter, PostgreSQL integration, and HTML
  transport tests cover sort forwarding, stable price ordering, and preserved
  category/query/page URL state.
- PostgreSQL payment integration covers signed `payment.captured` settlement,
  one-time replay handling, reservation commitment, audit cardinality, and
  unsupported-event rejection.
- Refund application tests cover authoritative amount propagation, provider
  response mismatch rejection, failure recording, and no-provider-call replay.
- Razorpay adapter tests cover normal-refund method/path/auth/body mapping and
  response consistency checks. PostgreSQL refund integration covers atomic
  paid-to-refunded state, one order-history/audit transition, same-key replay,
  and different-key rejection.
- Media unit, PostgreSQL lifecycle, MinIO adapter, and opt-in PostgreSQL plus
  MinIO end-to-end tests cover presigned upload, object inspection, metadata
  mismatch rejection, stable public references, signed readback, ownership
  denial, and cleanup deletion.
- Recovery application, PostgreSQL token-consumption, SMTP validation, and web
  transport tests cover non-enumerating reset requests, hashed one-time tokens,
  expired/replayed-token denial, email verification, referrer suppression, and
  session revocation after reset.
- Search facet/metrics adapter tests and stale index-event tests cover
  Meilisearch facet mapping, PostgreSQL facet fallback, bounded observations,
  and authoritative rehydration.
- Mechanical forbidden-import scan in `scripts/check-architecture.sh`.

## Verification results

The following checks passed in the current workspace:

- `bash scripts/check-architecture.sh`
- `SQLC_BIN="$(go env GOPATH)/bin/sqlc" bash scripts/check-sqlc-generated.sh`
- `gofmt` on all Go files
- `go vet ./...`
- `go test ./...`
- `go test -race ./...`
- `sqlc generate -f sqlc.yaml`
- `docker compose config`
- `docker compose up -d --build` with migrations 0001–0015 and local MinIO
  bucket creation attempted through the adapter
- `GET /health/live` → HTTP 200, `{"status":"live"}`
- `GET /health/ready` → HTTP 200 with PostgreSQL, Redis, Meilisearch, and S3
  checks ready
- Final rebuilt container smoke passed for health, PostgreSQL-backed search,
  request-ID API errors, immutable static CSS caching, migration version 15,
  and eight role records.
- Latest rebuilt container smoke passed with migration version 15, all four
  dependency readiness checks, indexed search hydration, and the unauthenticated
  privacy-delete denial path.
- Auth HTTP smoke passed for registration, HttpOnly/SameSite cookies, session
  page, CSRF rejection, and logout path.
- Phase 03 hardening tests passed for application-level seller/admin role
  policy, rate-limit middleware allow/deny/error behavior, authoritative
  PostgreSQL role assignments, failed-login lockout, and one audit record per
  failed credential attempt.
- Seller → draft → submit → moderation → public product → cart → checkout
  smoke passed with stock 10→8 and one `pending_payment` order.
- JSON/API/privacy/webhook smoke passed: JSON contains no HTML, consent is
  recorded, and an unsigned Razorpay webhook returns 401.
- Explicit PostgreSQL integration run passed for transaction, idempotency,
  outbox, audit, and business-mutation atomicity against a disposable
  database. The running stack reached migrations 1–14 and eight role seeds.
- PostgreSQL commerce integration passed the stock-10/100-attempt checkpoint
  (10 successful reservations, 90 stock errors, zero overselling), expired
  reservation release, and customer cancellation release.
- PostgreSQL commerce integration passed 20 identical checkout retries as one
  order, one reservation, one payment attempt, and one outbox event; reused
  keys with a different address are rejected.
- PostgreSQL integration passed guest-cart duplicate clamping/unavailable-item
  handling, transactional product-index event publication, and privacy
  deletion with retained-order redaction.
- Search ordering integration passed for category-filtered price-descending
  results; the HTML search flow preserved query, category, sort, and page state.
- The rebuilt runtime returned the sorted search result after cache
  invalidation, and the fresh 50-VU read-only smoke passed with 7,947 requests,
  0.00% failures, P95 17.88ms, and P99 36.26ms.
- Verified payment-webhook integration passed against the disposable local
  PostgreSQL stack: one signed capture changed the pending order to `paid`,
  marked the payment captured, committed the reservation, and a replay caused
  no second transition or audit record.
- `go run ./cmd/rebuild-search` rebuilt the disposable Meilisearch index from
  one approved PostgreSQL catalogue document, including filterable category
  settings; the live search path hydrated that indexed hit from PostgreSQL.
- Opt-in MinIO media E2E passed: a presigned image upload was written to
  MinIO, verified by byte size/content type, finalized in PostgreSQL, resolved
  through a signed read URL, and read back byte-for-byte.
- Authenticated API privacy E2E passed registration with lower-snake-case JSON,
  deletion-request creation, transactional eligible-data deletion, session
  revocation, and the `deleted` account state.
- Final rebuilt-container smoke passed with readiness for PostgreSQL, Redis,
  Meilisearch, and S3; indexed `/api/v1/search` hydration returned the
  PostgreSQL-backed product, unauthenticated privacy deletion returned 403,
  unauthenticated refund returned 403, active reservations were 0, and
  pending product-index events were 0.
- Integration-enabled `go test ./...`, `go test -race ./...`, `go vet ./...`,
  the architecture scan, sqlc generated-output check, Compose config
  validation, and OpenAPI parsing passed after the refund implementation.
- Final post-rebuild verification also passed `DATABASE_URL=... go test ./...`
  and `DATABASE_URL=... go test -race ./...`; the boundary scan, sqlc check,
  and vet were rerun after the SMTP/search changes.
- Latest local k6 read-only baseline passed at 50 virtual users: 7,947 requests,
  0.00% failures, P95 17.88ms, P99 36.26ms; report:
  `docs/performance-load-2026-09-06.md`.
- Headless Chrome captures at 1440×1100 and 390×844 completed successfully.
- Additional headless captures at 768×1024 and JavaScript-disabled 390×844
  completed successfully.
- The captured desktop and mobile views were visually inspected for responsive
  layout, readable controls, truthful empty states, and absence of horizontal
  overflow.

## Known issues and remaining gates

### Frontend issue-list checkpoint (2026-09-14)

- The isolated HTMX/Alpine UI preview covers the customer P0–P3 issue list:
  real route links, query-backed filters and sorting, wishlist/cart mutations,
  checkout/payment states, product interactions, responsive mobile sheets and
  purchase actions, focus/keyboard affordances, readable controls, loading and
  empty states, and local image/font assets. Customer, marketplace-admin,
  seller-admin, and support route matrices are covered by HTTP/template tests
  across 96 named customer/seller/admin routes. Playwright verification also
  passes the representative customer/admin/seller/support pages at 320, 768,
  1024, and 1440px, with the customer interaction paths and fluid mobile
  checkout/returns controls exercised. A 21-route WCAG A/AA axe scan reports
  zero violations.
- The live `cmd/res2` frontend now has customer commerce, real-data deals and
  maker discovery, payment outcome pages, customer shipment tracking, seller catalog draft
  editing, owner-managed seller staff membership, catalog product permission
  enforcement, a seller inventory read/adjust workflow, and customer return
  requests with an operations returns queue. Inventory reads and adjustments
  are seller-scoped, permission-checked, transaction-safe, and audited;
  catalogue editing no longer changes stock. Return requests are customer-owned
  and status transitions are restricted and audited; payment refunds remain a
  separate provider-confirmed workflow. These are
  source/unit-test verified in this checkout. The migrations have not been
  applied to a running PostgreSQL instance because the Docker daemon is
  unavailable here.
- Live seller order fulfilment is now available through a seller-scoped queue
  with sequential, audited transitions and `ORDER_READ`/`ORDER_FULFILL`
  separation. Finance has a read-only payment/refund reconciliation view
  restricted to finance operators, marketplace administrators, and super administrators. Customer support
  requests and a masked support-agent queue are also live, with audited status
  notes and customer-owned order references. HTTP request IDs are propagated
  into the audit records for these privileged flows. The preview role portals are
  intentionally not being presented as live backend integrations.
- Live marketplace order inspection now masks customer email and exposes order,
  payment, and seller fulfilment status to marketplace/operations/Super Admin
  roles. Audit history is readable by marketplace/Super Admin roles, while
  operational role assignments are Super Admin-only and reason-required.
- Seller onboarding now creates a pending seller application; marketplace and
  Super Admins can approve, suspend, or close it, with seller tools gated by
  the authoritative seller status. The live audit screen exposes privileged
  records to marketplace/Super Admin roles, while role assignment is restricted
  to Super Admin and excludes bootstrap-controlled Super Admin membership.
- Customer payment success/failure/pending-verification states, a real-data
  deals page, maker directory/detail pages, and customer-safe shipment tracking
  are now available through the live HTMX frontend. Tracking reads only the
  seller fulfilment projection attached to the authenticated customer's order.
- The live customer header now provides debounced, PostgreSQL-backed HTMX
  search suggestions with a normal search fallback. Authenticated cart rows
  can move an item to the customer's wishlist through a CSRF-protected,
  transactional route that refreshes the cart and badge with focused HTMX/OOB
  fragments.
- The preview customer summary is now data-driven for category counts, payment
  activity, order totals, and filtered tab counts; Returns tabs render their
  own refund/closed datasets and truthful empty states. Safe helpers keep empty
  orders, returns, support queues, and role tables renderable when records are
  absent. A seller workspace five-column table regression at the tablet
  breakpoint was corrected, and fresh headless Chrome captures at 1440, 768,
  and 390px were visually inspected after the fix.
- Product cards, saved products, and maker profiles now render rating-derived
  star counts with accessible labels; checkout payment tabs are entirely
  Alpine-controlled so the selected method and visible fields stay in sync.
  Handler/component regression coverage and a fresh headless Chrome smoke pass
  cover these local interactions.
- Preview CSS sources remain readable for maintenance, while `make css-build`
  now produces and serves minified font, customer, admin, seller, and support
  stylesheets. The build contract checks every generated preview stylesheet;
  no Node.js process is required at runtime.
- Generated Chromium QA profile directories were moved out of the checkout
  (the visual PNG evidence remains under the role QA directories), reducing
  the working-tree footprint by roughly 1 GB and leaving no zero-byte files in
  the project tree.
- Seller owners now have live, seller-scoped analytics, support requests, and
  audit activity pages. Analytics aggregates the authenticated seller's order
  projection without inventing settlement or commission values; support is
  owner-scoped and the audit view exposes only seller-owned resources. These
  pages are role-checked, empty-state aware, and covered by application,
  handler, template, and focused route tests.
- Marketplace and Super Admins also have role-checked, read-only live views for
  approved brands, masked customers, payments, failed payments, refunds, and
  finance reconciliation. Unsupported mutations are deliberately absent from
  these views.
- Live settlement/commission calculation, provider reconciliation automation,
  MFA/break-glass security policy controls, and the remaining admin operational
  queues remain open; no settlement total or manual override is implied by the
  finance view.

- G00/G01/G02 are complete in local evidence, with dependency outage and
  restart-volume evidence retained as follow-up checks.
- G02 is complete: generated persistence, migrations, transaction behavior,
  replay-safe idempotency, audit validation, and business-mutation/outbox
  atomicity are covered by integration tests.
- G03–G14 are in progress rather than falsely marked PASS: full
  rate-limit/role matrices, SMTP delivery in a real relay, search
  outage/rebuild rehearsal, image variants/private
  document workflows, live/test
  Razorpay credentials and refund reconciliation, external API contract
  validation, and production deletion/retention review still need dedicated
  verification. Product-index event publication and processor unit coverage
  now exist locally, but a live worker/index outage rehearsal remains open.
- Refund support is currently full-refund-only. Provider-pending refunds are
  persisted safely but do not yet have a reconciliation worker or live
  Razorpay test-mode evidence; no live payment claim is made.
- Local recovery routes are fully wired but email delivery stays unavailable
  until `WECRATFS_SMTP_ADDR` and `WECRATFS_SMTP_FROM` are configured. The
  checked-in adapter uses bounded STARTTLS-capable SMTP and does not provide a
  fake local token or recovery success.
- Image thumbnail/small/medium/large generation and private KYC/document
  access workflows remain open; current media support is public product media
  with signed readback, verified uploads, ownership checks, and cleanup.
- G15–G18 remain open because the available load report is only a local
  read-only baseline; no independent security report, staging deployment/
  restore, current legal review, or production approval exists in this local
  workspace.
- Local catalogue rows used for smoke tests are explicit operator-created
  records and can be removed from the disposable PostgreSQL volume; the app
  has no startup seed or fake fallback.
- The browser connector was unavailable in this environment; responsive
  evidence used installed headless Chrome. Chrome emitted non-blocking local
  telemetry messages while still producing the screenshots.
- The current rebuilt runtime also returned migration version 15, rendered
  `/forgot-password`, `/reset-password`, and `/verify-email`, suppressed the
  reset/verification page referrer, returned category facets from
  `/api/v1/search`, and exposed bounded counters at `/health/metrics`.

## Next phase entry

Close the remaining G03–G14 verification gaps, then repeat load testing at
production-like scale and enter the independent security, staging, legal, and
production-release gates; no live-payment or production-readiness claim is
made by local smoke tests alone.

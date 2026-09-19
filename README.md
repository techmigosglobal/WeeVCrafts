# WeCratfs

WeeVCrafts is a handmade and organic product marketplace focused on honest
materials, useful rituals, and products that stay close to nature. Its Go
application persists customer identity, seller/catalog moderation, inventory,
carts, orders, wishlist, returns, support, roles, and audit history in
PostgreSQL, with Redis sessions/rate limits and S3-compatible product media.
Online payment collection is deliberately disabled in this MVP; customers can
place an unpaid cash-on-delivery order and the UI clearly discloses that payment
is settled outside WeeVCrafts.

## Run locally

Requirements: Go 1.26+ and Docker Compose.

```sh
cp .env.example .env
go test ./...
docker compose up -d --build
```

Open <http://localhost:8080/>. The service reports:

```sh
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/metrics
curl http://localhost:8080/api/v1/products
curl 'http://localhost:8080/api/v1/search?q=soap'
```

To rebuild the disposable Meilisearch index from authoritative PostgreSQL
catalogue records:

```sh
go run ./cmd/rebuild-search
```

The database does not contain startup seed data. The storefront never invents
products, accounts, or orders. After migrations, create the first real
Super Admin once using the environment-only `cmd/bootstrap-admin` command
described below. Customers then register normally; sellers submit an
application, and an administrator approves it before product drafts can be
submitted. No credentials are committed. Seller owners can manage active
accounts from `/seller/team`; administrators can review sellers at
`/admin/sellers` and inspect privileged changes at `/admin/audit`.
Staff membership and permission changes are persisted and audited, and product
read/write permissions are enforced by the live catalog workflow. Inventory,
order, return, finance, support, seller-approval, audit, and role-management
workflows are exposed through server-rendered HTMX-compatible routes with role
checks at the application and repository boundaries. Settlement/commission
reporting and provider reconciliation remain intentionally unimplemented until
their business rules and provider contract are finalized.

Customers can browse real-data `/deals` and `/brands` directories, inspect a
maker's approved products, place offline-payment orders, and open
`/orders/{orderNumber}/tracking` for seller-published fulfilment updates. The
order stores `payment_method=cash_on_delivery`; payment is not marked received
or verified by the system.
Marketplace and Super Admins can inspect approved brands and masked customers;
finance views are also available at `/admin/payments`,
`/admin/failed-payments`, `/admin/refunds`, and
`/admin/finance-reconciliation`.

The runtime is a compiled Go binary. HTMX and Alpine are progressive
enhancements, while CSS is embedded/check-in based; Node.js/npm is not started
in production. When a stylesheet source changes, run `npm ci` followed by
`make css-build`; the build generates the minified live stylesheet at
`internal/transport/web/static/app.css` and minified preview styles beside
their readable sources under `web/assets/css/*.min.css`. The local Compose dependencies
(PostgreSQL, Redis, Meilisearch, and MinIO) are disposable infrastructure
behind ports.

Password recovery and email verification are wired through hashed one-time
tokens. Configure `WECRATFS_SMTP_ADDR`, `WECRATFS_SMTP_USERNAME`,
`WECRATFS_SMTP_PASSWORD`, and `WECRATFS_SMTP_FROM` to enable bounded SMTP
delivery; without those values, local recovery pages remain available but the
runtime returns an explicit delivery-unavailable response instead of exposing a
fake token.

Manual-payment checkout consumes inventory in the same transaction that
creates the order, and the seller queue can move it through fulfilment. There
is no card/gateway checkout, automated payment confirmation, settlement, or
online refund in this deployment. Return requests and support cases remain
persisted operational workflows; any offline refund/payment collection must be
handled directly by the business and must not be represented as provider-verified.

## Netlify real-backend deployment

Netlify builds two Go Functions: `web` serves the server-rendered storefront,
API, and role workspaces; `maintenance` runs bounded cleanup and search-index
work every five minutes. Neither function uses the local `cmd/web` preview,
in-memory business state, or seeded demo users. The production PostgreSQL
database is the authoritative store for accounts, catalog, inventory, carts,
orders, support, audit history, sessions, rate limits, and disposable cache
entries. Product images are stored as PostgreSQL bytea for this MVP, with a
4 MiB per-image limit. No Redis, S3 bucket, Netlify Blobs, or payment provider
is required by the Netlify runtime. The VPS/Compose runtime continues to use
Redis and S3-compatible storage.

The Netlify Database integration applies the SQL files in
`netlify/database/migrations/` before publishing a deploy. Those files mirror
the Go-embedded migrations under `internal/adapters/postgres/migrations/`; a
test fails if the copies diverge. Do not run `cmd/migrate` against the Netlify
managed database. The schedule is activated on a published deploy; for an
unpublished preview, invoke `maintenance` manually from the Netlify Functions
UI when testing cleanup or search-index work.

Configure these Netlify environment variables:

| Variable | Required | Purpose |
|---|---:|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string with `sslmode=require` or stricter; use the provider's pooled URL when available. |
| `DATABASE_MAX_CONNS` | Optional | Per-function pool cap; defaults to `1` to limit serverless fan-out. |
| `WECRATFS_PUBLIC_URL` | Yes | Canonical HTTPS site URL. |
| `WECRATFS_SECURE_COOKIES` | Optional | Defaults to `true` in the Netlify runtime. |
| `MEILI_ADDR`, `MEILI_API_KEY` | Optional | Search index acceleration; PostgreSQL search remains the fallback. |
| `WECRATFS_SMTP_ADDR`, `WECRATFS_SMTP_USERNAME`, `WECRATFS_SMTP_PASSWORD`, `WECRATFS_SMTP_FROM` | Optional | Real account-recovery and verification email delivery. |

Ensure the database connection variable is available to both the build-time
migration integration and Functions runtime for the production deploy context.
The app rejects local PostgreSQL URLs, connections without TLS, and non-HTTPS
public URLs. Keep production and deploy-preview data separated by Netlify's
database branches. The application does not seed sample products, accounts,
or orders; customers register, vendors apply, and the first privileged
administrator is bootstrapped explicitly.

Bootstrap the first real Super Admin once, after the first successful schema
deploy, from a trusted environment with the production database URL and a
unique password (16+ characters):

```sh
DATABASE_URL='postgres://…?sslmode=require' go run ./cmd/migrate
DATABASE_URL='postgres://…?sslmode=require' \\
  BOOTSTRAP_SUPER_ADMIN_EMAIL='owner@example.com' \\
  BOOTSTRAP_SUPER_ADMIN_NAME='Marketplace Owner' \\
  BOOTSTRAP_SUPER_ADMIN_PASSWORD='use-a-unique-secret-of-16-or-more-characters' \\
  go run ./cmd/bootstrap-admin
```

The bootstrap command refuses to run once any Super Admin exists. Do not put
the connection URL or administrator credentials in this repository, build
output, or chat logs. Customers can register directly; vendors submit an
application and an administrator must approve it before the vendor can publish
products. Without configured SMTP, password-reset/verification email delivery
is explicitly unavailable rather than simulated.

The per-runtime PostgreSQL pool defaults to one connection. This limits
per-instance fan-out; it is not a 20-user capacity guarantee. The linked
Netlify Free database is constrained to one compute unit, 48 database compute
units per billing period, and 5 GB each of writes, outbound bandwidth, and
storage. It sleeps after inactivity and has a hard monthly credit cap, so it
is suitable for short MVP validation, not a continuously available or
production-scale promise. Check current account usage and plan limits before
running extended tests or inviting users. For a short staging rehearsal with a
test-only customer account and k6 installed, run:

```sh
BASE_URL='https://your-staging-site.netlify.app' \\
CUSTOMER_EMAIL='load-test-customer@example.com' \\
CUSTOMER_PASSWORD='staging-account-password' \\
k6 run scripts/netlify-load-test.js
```

The gate ramps to 20 concurrent virtual users, exercises customer login,
storefront/search/catalogue/API/account/cart/orders/readiness, and checks for
under 1% request failures and a 2.5-second p95. It is a staging test, not a
production claim; the login rate limit may also require coordinating a shared
load-generator IP with normal test traffic.

## Verification

```sh
bash scripts/check-architecture.sh
find . -name '*.go' -not -path './vendor/*' -not -path './internal/adapters/postgres/generated/*' -print0 | xargs -0 gofmt -w
go vet ./...
go test ./...
go test -race ./...
docker compose config
```

The storefront stylesheet is checked in at
`internal/transport/web/static/app.css` and embedded into the Go binary. No
Node.js process is required to run the server; npm is only a build-time tool
for regenerating that checked-in artifact.

`docs/phase-status.md` records the current implementation evidence and the
remaining gates. `docs/architecture.md` describes dependency direction and
 provider boundaries. `docs/api/openapi.yaml` records the JSON contract. `docs/design/fidelity-ledger.md` records the visual
prototype decisions.

## Regenerate PostgreSQL access

Generated sqlc files are never hand-edited. When query or schema inputs change:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
sqlc generate -f sqlc.yaml
bash scripts/check-sqlc-generated.sh
```

The generated output lives under `internal/adapters/postgres/generated/`.
PostgreSQL integration coverage can be run against the local Compose database:

```sh
DATABASE_URL='postgres://wecratfs:wecratfs@localhost:5432/wecratfs?sslmode=disable' \
  go test -v ./internal/adapters/postgres -run 'Integration|WithinTransaction|PersistencePrimitives'
```

## Isolated visual prototype (not deployed)

The older `cmd/web` and `internal/web/handlers` packages remain as an isolated
local design/test harness. They are not imported by the Netlify or `cmd/res2`
runtime and must not be used for real customer accounts or orders:

```sh
make templ-generate
make web-mock                 # http://localhost:8090
```

It contains sample interface fixtures for visual development only. No preview
credentials are documented here, and the preview binary is not built by the
Netlify deployment script.

# WeCratfs

WeCratfs is a handmade and organic product marketplace focused on honest
materials, useful rituals, and products that stay close to nature. The local
V1 prototype now covers PostgreSQL-backed identity, seller/catalog moderation,
inventory reservations, carts, wishlist, pending-payment checkout, server-side
payment-order preparation, indexed-search hydration/rebuild, privacy deletion controls, JSON APIs,
and provider adapters.

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
products or pretends that sample data is real. Create an account, open the
seller workspace, create a draft, and use an explicitly assigned
`marketplace_admin` role for local moderation. No credentials are committed.

The runtime is a compiled Go binary. HTMX and Alpine are progressive
enhancements, while CSS is embedded/check-in based; Node.js/npm is not started
in production. The local Compose dependencies (PostgreSQL, Redis, Meilisearch,
and MinIO) are disposable infrastructure behind ports.

Password recovery and email verification are wired through hashed one-time
tokens. Configure `WECRATFS_SMTP_ADDR`, `WECRATFS_SMTP_USERNAME`,
`WECRATFS_SMTP_PASSWORD`, and `WECRATFS_SMTP_FROM` to enable bounded SMTP
delivery; without those values, local recovery pages remain available but the
runtime returns an explicit delivery-unavailable response instead of exposing a
fake token.

After checkout, the authenticated payment endpoint creates or reuses the
server-side provider order from the PostgreSQL total:
`POST /api/v1/orders/{orderNumber}/payment`. A verified Razorpay webhook is the
only path that changes an order to paid; expired reservations are released by
the bounded background reaper. Paid orders expose an authenticated, idempotent
full-refund endpoint at `POST /api/v1/orders/{orderNumber}/refund`; its amount
is always derived from PostgreSQL, while provider-pending refunds remain open
for a future reconciliation worker.

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
Node.js process, npm dependency, or CSS build step is required to run the
server.

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

## WeeVCrafts UI-first preview

The customer storefront milestone is available as an isolated templated Go
preview. It does not connect to PostgreSQL, payments, shipping, or account
services:

```sh
make templ-generate
make web-mock                 # http://localhost:8090
```

`cmd/web` serves the responsive customer routes using a per-browser fixture
session. HTMX mutation targets live under `/ui/*` (with stable
`/ui/fragments/*` aliases), and Alpine is limited to local menus, galleries,
tabs, and selectors. The existing `cmd/res2` backend composition and
`Dockerfile` remain available for the later live integration milestone;
`Dockerfile.web` packages this UI preview independently.

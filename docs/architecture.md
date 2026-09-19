# WeCratfs architecture boundary

WeCratfs is a Go modular monolith. PostgreSQL owns authoritative business
state. On Netlify it also stores shared sessions, rate limits, disposable cache
data, and small product images; on VPS/Compose, Redis stores ephemeral state
and S3-compatible storage holds product media. Meilisearch is optional and
rebuildable. The current MVP does not wire an online payment provider.

## Composition roots

```text
cmd/res2 ───────────────────────┐
Netlify Functions: web/maintenance ─┤
                                  ▼
                         internal/apphost
                                  │
          ┌───────────────────────┼──────────────────────┐
          ▼                       ▼                      ▼
   transport/web            transport/api          transport/health
          └───────────────────────┼──────────────────────┘
                                  ▼
                       internal/application
                                  ▼
                         internal/domain + ports
                                  ▲
          ┌───────────────────────┼──────────────────────┐
          ▼                       ▼                      ▼
       PostgreSQL                Redis          S3 / Meilisearch
```

Both the VPS process and Netlify functions use `internal/apphost`; there is no
second, fixture-backed application in the deployed request path. The existing
`cmd/web` mock is a separate local-only visual harness and is not imported by
the Netlify build. Netlify applies migrations from
`netlify/database/migrations/` immediately before publishing; the synchronized
copies of Go-embedded migration files are tested for equality. VPS/Compose
migrations run through `cmd/migrate`. `cmd/bootstrap-admin` creates the first
Super Admin once from environment-only credentials after the schema is ready.

## Transport and data rules

- `internal/transport/web` owns server-rendered HTML, HTMX enhancements, and
  role-checked workspaces. `/seller-admin`, `/admin`, and `/support-portal`
  remain aliases for the corresponding persisted-role workspaces.
- `internal/transport/api` owns JSON-only `/api/v1` responses; both transports
  call the same application services.
- New customer registration creates real PostgreSQL identities. Role access is
  granted by persisted role assignments; there are no runtime demo accounts or
  business fixture records.
- Checkout places an idempotent unpaid cash-on-delivery order. PostgreSQL
  atomically validates stock, commits its inventory allocation, records the
  payment method, and writes order history. Customer cancellation restores
  stock only before seller fulfilment begins. No payment is described as
  collected or verified by WeeVCrafts.
- Netlify sessions and authentication rate limiting use atomic PostgreSQL
  operations, so requests remain coherent across warm and cold function
  instances. Netlify requires remote TLS PostgreSQL and HTTPS secure cookies;
  its per-runtime pool defaults to one connection. VPS/Compose uses Redis for
  sessions, rate limiting, and disposable search cache.
- Netlify product media is stored in PostgreSQL and capped at 4 MiB per image
  to respect buffered function request limits. VPS/Compose uses presigned
  S3-compatible uploads and does not proxy image bytes.
- Optional Meilisearch is disposable discovery acceleration. Product records
  remain PostgreSQL-authoritative and search falls back to PostgreSQL.
- Netlify's scheduled function runs bounded outbox, reservation, and media
  cleanup batches. The VPS process uses the same maintenance method from its
  periodic runner; neither holds business state in package globals.
- The legacy payment/provider adapters remain testable for a later integration,
  but are not composed by the current Netlify or `cmd/res2` runtime.

## Verification

Run `bash scripts/check-architecture.sh`, `go test ./...`, and `go test -race
./...` from the repository root. PostgreSQL integration tests require a
migrated disposable database. Staging browser, provider, and 20-user load
evidence must be reported separately from source and unit-test results. Netlify
Free's fixed one-compute-unit database and monthly caps are a short validation
target, not a production capacity guarantee.

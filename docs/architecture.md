# WeCratfs architecture boundary

WeCratfs is a Go modular monolith. PostgreSQL owns authoritative business
state; Redis stores shared sessions, rate limits, and disposable cache data;
S3-compatible storage holds product media. Meilisearch is optional and
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
the Netlify build. Migrations run through `cmd/migrate` as an explicit
deployment task, never on a function invocation. `cmd/bootstrap-admin` creates
the first Super Admin once from environment-only credentials after migrations.

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
- Sessions and authentication rate limiting use Redis so requests remain
  coherent across warm and cold Netlify function instances. The Netlify config
  rejects local DB/Redis/object-store defaults, requires TLS, and defaults to a
  two-connection pool per function runtime; use a provider pooler for serverless
  fan-out.
- Product media uses presigned S3-compatible uploads; the function does not
  proxy image bytes. Provision the bucket and browser CORS policy in advance.
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
evidence must be reported separately from source and unit-test results.

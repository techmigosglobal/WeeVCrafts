# WeCratfs architecture boundary

WeCratfs is a Go modular monolith. PostgreSQL owns authoritative business
state; Redis, Meilisearch, S3-compatible storage, and payment providers are
replaceable infrastructure behind ports.

## Dependency direction

```text
cmd/res2 (composition root)
├── internal/adapters/postgres ─────┐
├── internal/transport/api          │
├── internal/transport/web          │
└── internal/transport/health       │
                                    ▼
                         internal/application
                                    ▼
                          internal/domain + ports
```

Adapters and transports depend inward. Domain, application, and port packages
do not import HTTP/template concerns, PostgreSQL implementation packages,
Redis or Meilisearch SDKs, S3 SDKs, or payment-provider SDKs.

The composition root registers PostgreSQL, Redis sessions/cache/rate limiting,
the Meilisearch HTTP adapter seam and rebuild command, S3/MinIO media,
Razorpay payment/webhook services, and health probes. Provider SDKs remain in
adapter packages; the application layer receives only the provider-neutral
ports.

## Transport separation

- `internal/transport/web` owns HTML templates, Tailwind output, HTMX
  progressive enhancement, and Alpine-compatible markup.
- `internal/transport/api` owns JSON-only `/api/v1` responses.
- Both transports call the same application service and receive request
  contexts from the HTTP layer.

## Data and runtime rules

- The storefront reads only approved records from PostgreSQL.
- An empty database renders an explicit empty state; no runtime product
  fixtures or fake seed command are present.
- Migrations are embedded, versioned, transactional, and applied at startup.
- Cache/search/storage/payment ports are disposable or external and cannot
  become an alternate business-data source.
- Checkout totals and payment-order references are read from PostgreSQL;
  verified gateway webhooks alone transition payment state. A bounded worker
  scheduler releases expired inventory reservations without making Redis or
  Meilisearch authoritative.
- HTTP and health probe timeouts are bounded, and shutdown uses a finite
  context. No business state is held in mutable package globals.

## Mechanical check

Run `bash scripts/check-architecture.sh` from the repository root. It scans
the business-boundary packages for forbidden provider imports and fails on a
match.

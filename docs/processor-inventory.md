# WeCratfs processor and data-location inventory

Version: `privacy-2026-09`  
Status: local architecture inventory; production vendor/legal confirmation is open.

| System | Data handled | Authority/classification | Local configuration |
| --- | --- | --- | --- |
| PostgreSQL | Identity, catalog, stock, carts, orders, payments metadata, privacy records, Netlify sessions/rate limits/cache, Netlify product-image bytes | Authoritative business data plus bounded ephemeral state/media on Netlify | `DATABASE_URL`; local Compose volume or managed Netlify Database |
| Redis | Opaque sessions, CSRF/session metadata, rate limits, search cache for VPS/Compose runtime | Disposable infrastructure; no business authority | `REDIS_ADDR`; local Compose service; not required on Netlify |
| Meilisearch | Approved product search documents | Disposable/rebuildable index | `MEILI_ADDR` and optional `MEILI_API_KEY` |
| S3-compatible storage/MinIO | Product media and future private documents for VPS/Compose runtime | File-byte owner; metadata remains in PostgreSQL | `S3_*`; local MinIO bucket; not required on Netlify where product images are capped at 4 MiB and stored in PostgreSQL |
| Razorpay adapter | Gateway contract and provider adapter remain available for a future integration | Not composed by the Netlify/VPS MVP; no payment credentials are configured or required | Disabled in the current runtime |

Before staging/production, the owner must confirm provider region, transfer
safeguards, subprocessors, access roles, deletion/retention commitments,
incident contacts, and the current privacy notice wording.

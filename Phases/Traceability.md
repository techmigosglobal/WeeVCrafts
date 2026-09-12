# RES2 PRD traceability

This map prevents the phase split from dropping requirements from the master PRD. The phase file is the execution owner; `Prd.md` remains the normative source.

| PRD sections | Covered by | Evidence expected |
| --- | --- | --- |
| 1–4 Product vision, locked architecture, responsibilities, ports | Phase 00 | Architecture contract, import checks, port tests |
| 5–13 Marketplace actors and role definitions | Phases 03, 04, 11 | Auth/RBAC matrix, seller isolation, admin audit tests |
| 14–19 Screen inventory | Phases 03, 06, 08, 09, 10, 11, 13, 14 | Route inventory, browser previews, API contract coverage |
| 20–29 UX, Living Storefront, HTMX, Alpine, accessibility | Phase 06 and Phase 08 | Responsive/browser evidence, non-HTMX fallback, keyboard/reduced-motion checks |
| 30 Schema families | Phase 02 plus Phases 04–14 | Migrations, queries, integration tests, retention controls |
| 31–47 Performance architecture, transactions, async, context, backpressure | Phases 00, 02, 05, 07, 09, 15 | Context tests, transaction tests, bounded-worker/load evidence |
| 48–51 Search, storage, image variants, Cloudflare cache policy | Phases 07 and 12 | Index/rebuild, object access, image-variant, cache-policy evidence |
| 52–58 Security, sessions, passwords, MFA, payment/webhook, audit | Phases 03, 10, 11, 16 | Negative tests, signature/replay evidence, audit logs |
| 59–71 Privacy, consent, minimization, retention, breach, processor inventory | Phase 14 | Executable privacy workflows, retention matrix, legal review |
| 72–75 API, error format, observability, metrics | Phases 13 and 15 | Contract/OpenAPI tests, request IDs, dashboards/metrics |
| 76–81 SLOs, load stages, scenarios, concurrency, idempotency, race testing | Phases 05, 09, 15, 16 | k6 report, repeated concurrency runs, race result |
| 82–83 Backups and disaster recovery | Phases 17 and 18 | Restore transcript and post-restore business checks |
| 84–85 Development phases and production gate | All phase files and Release-Gates | Current gate records and final release checklist |
| 86–87 V2 mobile and scaling | Phase 13 for readiness; deferred after Phase 18 | Shared application-service/API evidence; no premature mobile implementation |
| 88–89 Definition of success and AI instructions | All phases | Final architecture audit and phase-gate discipline |

## Deliberately deferred

- Flutter Android/iOS implementation is outside this v1 execution plan; Phase 13 only preserves the `/api/v1` seam.
- Microservices, event brokers, read replicas, and dedicated clusters are future scaling responses, not v1 prerequisites.
- Social login and passkeys are future enhancements unless a later approved phase adds them.
- Live Razorpay credentials and production deployment are not used in local prototype work.


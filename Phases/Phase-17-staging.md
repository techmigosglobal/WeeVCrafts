# Phase 17 — Staging release rehearsal

**Status:** PLANNED  
**PRD source:** Phase 17, Sections 49, 51, 82–83  
**Depends on:** Phases 11, 12, 14, 15, and 16  
**Unlocks:** Phase 18

## Outcome

Deploy the same application shape to a controlled staging environment and prove that migrations, configuration, webhooks, backups, restore, and the complete purchase workflow work together.

## In scope

- clean deployment from versioned artifacts/configuration;
- staging domain, HTTPS, and Cloudflare/DNS/WAF configuration as approved;
- migrations from a clean database and safe repeat deployment;
- explicit staging seed data;
- Razorpay Test webhook integration;
- backup creation and off-server restore;
- full E2E browse → search → cart → checkout → payment test → order → seller/admin flow;
- monitoring, logs, request IDs, worker/search health, and rollback rehearsal.

## Out of scope

Live customer traffic, live Razorpay credentials, irreversible production migrations, and unapproved DNS/domain changes.

## Deliverables

- [ ] deployment runbook and environment manifest;
- [ ] staging health/readiness evidence;
- [ ] migration/rollback decision documented;
- [ ] test webhook endpoint and signature configuration;
- [ ] backup/restore transcript and post-restore checks;
- [ ] E2E test report with browser evidence;
- [ ] rollback and incident contacts/runbook.

## Preview

Reviewer receives a staging URL and can complete the controlled test purchase flow with explicit test data and Razorpay Test mode. The environment banner makes staging/test payment status obvious.

## Verification

- [ ] clean deploy starts without manual database edits;
- [ ] `/health/live` and `/health/ready` are truthful;
- [ ] TLS/security headers and private route cache behavior are checked;
- [ ] test webhook signature and duplicate event behavior work over staging HTTPS;
- [ ] backup is restored into a disposable database/environment;
- [ ] post-restore checks find customers, products, inventory, orders, and payments;
- [ ] full E2E path passes, including a failure/pending payment case;
- [ ] rollback rehearsal is recorded;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass against the release revision.

## Gate G17

**PASS when:** staging reproduces the release, the controlled E2E flow and test webhook pass, and a restore produces a working system.

**BLOCK when:** staging requires undocumented manual changes, restore is untested, or test/live payment configuration is ambiguous.

## Evidence

Attach staging URL/handoff, release manifest, health output, E2E report, test webhook results, restore transcript, and rollback notes.


# Phase 18 — Production readiness

**Status:** PLANNED  
**PRD source:** Sections 76–89, Phase 18, production release gate  
**Depends on:** Phase 17  
**Unlocks:** Controlled v1 production release only

## Outcome

Make a release decision using current evidence for the complete customer, seller, admin, payment, privacy, security, performance, backup, and operational flows.

## In scope

- production configuration and secret-management review;
- final browse → search → product → cart → checkout → payment → order flow;
- seller, admin, finance, returns, refund, privacy-center, and support flows;
- monitoring, alerting, request IDs, worker/search health, and incident ownership;
- backup schedule, off-server destination, restoration rehearsal, and retention;
- load/SLO, race, security, privacy/legal, and API evidence review;
- migration and rollback plan;
- release notes, known issues, support runbook, and launch approval.

## Out of scope

New product features, unmeasured scaling architecture, mobile implementation, microservices extraction, and live rollout before every blocking gate is addressed.

## Deliverables

- [ ] final release evidence index and go/no-go record;
- [ ] production configuration, migration, rollback, and monitoring runbooks;
- [ ] current known-issues and approved-risk register;
- [ ] production candidate preview with environment/payment-mode labeling.

## V1 release checklist

- [ ] G00–G17 are `PASS` or have documented approved exceptions;
- [ ] production migrations run from a clean database rehearsal;
- [ ] inventory concurrency produces zero overselling;
- [ ] duplicate checkout/payment/webhook retries are safe;
- [ ] seller/admin/finance/support isolation is proven;
- [ ] webhook signatures and replay protection are verified;
- [ ] no secrets, payment credentials, passwords, tokens, or raw KYC appear in logs/artifacts;
- [ ] backup has been restored successfully and business records verified;
- [ ] SLO/load targets and error budget decision are recorded;
- [ ] current security findings are triaged with no release-blocking issue;
- [ ] privacy/legal review and operational rights flows are complete;
- [ ] `/api/v1` contract and compatibility policy are documented;
- [ ] rollback, incident, support, and monitoring runbooks are available;
- [ ] release owner and approver are recorded.

## Preview

The v1 release candidate must be previewable in staging or a controlled production-like environment. It must show the full purchase and operations flow with an explicit environment/payment-mode label. Production readiness is a gate decision, not a visual claim based on a screenshot.

## Verification

- [ ] final browser/API E2E report is current for the release revision;
- [ ] all required Go checks pass: `gofmt`, `go vet ./...`, `go test ./...`, `go test -race ./...`;
- [ ] load, security, privacy, restore, and payment evidence are linked;
- [ ] release artifact/configuration manifest is complete;
- [ ] canary/rollback decision and owner are documented;
- [ ] known issues are classified as blocker, accepted risk, or deferred scope;
- [ ] final go/no-go review is signed by the responsible owner.

## Gate G18

**PASS only when:** the production release checklist is complete and every blocking condition in `Release-Gates.md` is false.

**BLOCK when:** any critical business, payment, authorization, backup/restore, race, security, privacy, or SLO gate is missing or failed.

## Evidence

Attach the final release checklist, evidence index, deployment/rollback plan, runbooks, current known-issues list, approver, and release revision.

# Phase 16 — Security testing

**Status:** PLANNED  
**PRD source:** Sections 52–58, 65, 69–71, Phase 16  
**Depends on:** Phases 03, 10, 11, 12, 13, and 14  
**Unlocks:** Phase 17

## Outcome

Independently test the highest-risk abuse paths before staging: identity, authorization, payments, uploads, web security, secrets, logs, and data isolation.

## In scope

- authentication and authorization negative tests;
- seller/customer/admin/finance/support isolation;
- CSRF, XSS/output escaping, SQL injection, and request validation testing;
- rate limits and brute-force behavior;
- file upload/content-type/path validation;
- Razorpay webhook signature/replay and idempotency testing;
- sensitive log and secret scanning;
- dependency/vulnerability review;
- audit-log completeness and tamper/access review;
- incident/breach response tabletop.

## Out of scope

Accepting unresolved critical/high findings, production penetration-test claims without a real scope/report, or changing architecture solely to satisfy a scanner without risk analysis.

## Deliverables

- [ ] threat/abuse-path inventory;
- [ ] automated negative security suite;
- [ ] dependency and secrets scan report;
- [ ] sanitized logging review;
- [ ] upload and webhook abuse report;
- [ ] findings triage with severity, owner, remediation, and retest evidence;
- [ ] incident response tabletop notes.

## Preview

Reviewer receives a security verification report with executed attacks/checks, expected denial behavior, open findings, and retest results. No secrets or exploitable payloads are committed as fixtures.

## Verification

- [ ] all auth/RBAC/CSRF/XSS/SQLi/upload/webhook checks pass or have approved non-blocking findings;
- [ ] seller isolation is retested after all operational/API changes;
- [ ] logs are scanned for passwords, tokens, OTPs, card data, and raw KYC;
- [ ] secret scanning covers Git history/current tree/configuration inputs;
- [ ] dependencies are reviewed and actionable vulnerabilities triaged;
- [ ] critical and high findings are fixed and retested;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G16

**PASS when:** release-blocking findings are closed or explicitly rejected with documented risk acceptance, and security tests demonstrate allowed and denied behavior.

**BLOCK when:** a critical/high issue is open, a secret is exposed, or a security boundary is only asserted without a negative test.

## Evidence

Attach scan versions, test reports, finding register, remediation commits, retest output, and tabletop record without exposing sensitive exploit material.


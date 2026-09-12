# Phase 14 — Privacy operations

**Status:** IN PROGRESS  
**PRD source:** Sections 59–71, Phase 14  
**Depends on:** Phases 03, 10, 11, and 12  
**Unlocks:** Phase 15 and Phase 17

## Outcome

Turn privacy commitments into executable product and operational workflows, aligned with the DPDP 2023 baseline and reviewed against current legal requirements before launch.

## In scope

- plain-language privacy notice and policy versioning;
- purpose-specific consent records;
- marketing preferences and withdrawal;
- privacy center for access/summary, correction, consent, grievance, and deletion requests;
- deletion/anonymization workflow with retention exceptions;
- data minimization and support masking;
- retention matrix for profiles, carts, payments, orders, KYC, logs, and consent history;
- processor/data-location inventory and breach response runbook;
- private KYC handling and access audit.

## Out of scope

Legal advice, automatic deletion of legally required records, collection of unnecessary child data, and claiming compliance without counsel/current-rule review.

## Deliverables

- [ ] privacy notice is shown before relevant collection;
- [ ] consent purpose, policy version, source, and timestamp are stored separately;
- [ ] withdrawal is as easy as grant where applicable;
- [ ] access/correction/deletion requests have states, owner, due date, and audit trail;
- [ ] deletion flow verifies identity, checks retention, anonymizes/erases eligible data, and records completion;
- [ ] retention schedule and processor inventory are versioned;
- [ ] breach procedure identifies detection, containment, evidence, assessment, notification, and remediation steps;
- [ ] latest applicable legal review is recorded before production.

## Preview

Customer can open a privacy center, change marketing preferences, view a consent history, request data access/correction/deletion, and see a clear status. Support views show masked data and no secrets.

## Verification

- [ ] notice/consent/version tests pass;
- [ ] withdrawal stops the corresponding optional communication path;
- [ ] access and correction requests are scoped to the requesting data principal;
- [ ] deletion workflow preserves legally required order/accounting records while erasing eligible data;
- [ ] retention job behavior is tested in a disposable environment;
- [ ] KYC access and audit tests pass;
- [ ] logs/analytics contain minimized or pseudonymous data;
- [ ] legal review checklist for current DPDP requirements is signed before release;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G14

**PASS when:** privacy actions are executable and tested, retention/processor/breach artifacts exist, and current legal review is complete.

**BLOCK when:** privacy is only a document, optional consent is bundled with necessary processing, or deletion/retention behavior is undefined.

## Evidence

Attach privacy-center preview, workflow records, retention matrix, processor inventory, redacted audit examples, and legal review outcome.

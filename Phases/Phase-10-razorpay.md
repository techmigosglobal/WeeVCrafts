# Phase 10 — Razorpay test payments

**Status:** IN PROGRESS  
**PRD source:** Sections 11, 16, 30, 56–57, Phase 10  
**Depends on:** Phase 09  
**Unlocks:** Phase 11

## Outcome

Integrate Razorpay only through `PaymentGateway`, using test mode first, server-side order creation, verified webhooks, and idempotent payment state transitions.

## In scope

- Razorpay adapter behind the payment port;
- server-side payment-order creation with amount/currency checks;
- raw-body webhook signature verification;
- unique event IDs, replay resistance, and idempotent event handling;
- success, failure, pending, duplicate webhook, and invalid-signature states;
- refund initiation/status and reconciliation records;
- payment audit trail without card number, CVV, or credential storage;
- external request timeout, bounded retry, and no blind retry of non-idempotent operations.

## Out of scope

Live payment credentials, production release, alternate gateway integration, and client-owned payment logic.

## Deliverables

- [ ] payment order is created only by the server;
- [ ] browser callback is treated as a signal, not proof of payment;
- [ ] verified webhook is the authoritative payment transition input;
- [ ] duplicate webhook delivery is harmless;
- [ ] invalid signatures and replayed events are rejected/audited;
- [ ] refund/reconciliation states are explicit;
- [ ] secrets are environment-provided and absent from source/evidence.

## Preview

Using Razorpay test mode, reviewer can create a test payment, observe pending/success/failure states, deliver a webhook, and see the order transition only after verification. No live customer/card data is used.

## Verification

- [ ] adapter contract tests use a fake gateway and do not require Razorpay SDK in core packages;
- [ ] test-mode integration checks server order creation and amount consistency;
- [ ] valid webhook signature is accepted;
- [ ] invalid signature, altered raw body, unknown event, and replay are rejected safely;
- [ ] duplicate valid webhook produces one state transition and one audit result;
- [ ] failed/pending payment leaves correct order/reservation state;
- [ ] refund/reconciliation retry behavior is idempotent;
- [ ] no payment credential appears in database fixtures, logs, or tests;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G10

**PASS when:** test-mode end-to-end payment and webhook evidence proves signature verification, replay safety, idempotent transitions, and no card-data storage.

**BLOCK when:** client callbacks can mark orders paid, signatures are not checked over the raw body, or payment retries can duplicate state.

## Evidence

Attach adapter contract results, test webhook payload/signature evidence with secrets redacted, payment state timeline, and reconciliation output.

# Phase 09 — Checkout and orders

**Status:** IN PROGRESS  
**PRD source:** Sections 16–17, 30, 38–41, 72–73, Phase 9  
**Depends on:** Phases 05 and 08  
**Unlocks:** Phase 10 and the Commerce Prototype gate

## Outcome

Create an atomic, idempotent checkout/order flow that recalculates price, reserves inventory, creates order data, and emits durable outbox work.

## In scope

- customer addresses and address selection;
- server-side price/promotion/shipping recalculation;
- final inventory validation and reservation;
- order and order-item creation with immutable purchase values;
- idempotency-key handling for checkout/order creation;
- order status/history and pending-payment state;
- transactional outbox event for downstream work;
- failure, retry, cancellation-before-payment, and reservation-release paths;
- customer order history/detail read paths.

## Out of scope

Razorpay provider calls and webhook verification (Phase 10), seller operations (Phase 11), live fulfillment, and production refunds.

## Deliverables

- [ ] checkout review shows authoritative recalculated totals;
- [ ] order creation, reservation, and outbox commit atomically;
- [ ] order item stores the price/name/variant values needed for historical truth;
- [ ] repeated identical idempotency keys return one stable result;
- [ ] order state visibly distinguishes pending payment from paid;
- [ ] abandoned/failed paths release reservations safely;
- [ ] customer can view only their own orders.

## Preview

This is the Commerce Prototype:

```text
Browse → product → cart → address → final review → pending order
```

The preview must not simulate a successful payment. It must show the order as pending or awaiting payment until Phase 10 verifies a real payment event through the gateway adapter.

## Verification

- [ ] integration test rolls back all business writes on a failed checkout;
- [ ] test stale price, unavailable stock, invalid address, and expired cart cases;
- [ ] run 20 concurrent identical checkout requests with one idempotency key;
- [ ] result is one order, one reservation, and one payment-preparation intent at most;
- [ ] outbox event exists exactly with the committed order state;
- [ ] test reservation release after failed/pending paths;
- [ ] test customer/order authorization and keyset order pagination;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G09

**PASS when:** the browse-to-pending-order prototype works, transaction boundaries are integration-tested, and concurrent retries cannot duplicate orders or reservations.

**BLOCK when:** payment is marked successful by the browser, order writes can partially commit, or idempotency is optional for checkout.

## Evidence

Attach checkout preview evidence, transaction test output, concurrent idempotency results, order-state examples, and outbox records with sensitive data redacted.

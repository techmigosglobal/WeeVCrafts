# Phase 08 — Cart and wishlist

**Status:** IN PROGRESS  
**PRD source:** Sections 7, 16–17, 23–29, Phase 8  
**Depends on:** Phase 07 and Phase 03  
**Unlocks:** Phase 09

## Outcome

Support guest and authenticated carts, safe guest-to-user merge, wishlist operations, and progressive HTMX interactions while keeping server state authoritative.

## In scope

- guest cart identity and authenticated cart ownership;
- add, remove, quantity, and wishlist operations;
- merge policy for guest cart at login;
- server-side product/variant validation and current-price display;
- HTMX out-of-band updates for cart count/toasts where useful;
- normal HTML form fallback for every mutation;
- idempotency/double-click protection and clear validation errors;
- cart/session expiry policy.

## Out of scope

Final checkout reservation, payment creation, shipping selection, and production recommendation logic.

## Deliverables

- [ ] cart and wishlist repository/application ports;
- [ ] guest and authenticated flows;
- [ ] deterministic merge behavior for duplicates and unavailable items;
- [ ] cart count and item totals update without page reload when HTMX is available;
- [ ] full-page fallback works when HTMX is not loaded;
- [ ] no Alpine state is treated as the cart source of truth.

## Preview

Reviewer can add an item from the storefront, see the cart count change, open the cart, update quantity, remove an item, and merge a guest cart after login. The same operations work through normal HTML requests.

## Verification

- [ ] unit tests cover merge rules, duplicate variants, unavailable products, and quantity limits;
- [ ] integration tests cover guest cart, authenticated cart, login merge, expiry, and wishlist;
- [ ] repeated/double-click requests do not create duplicate cart lines or invalid quantities;
- [ ] HTMX and non-HTMX responses are tested;
- [ ] authorization tests prevent cart/wishlist cross-user access;
- [ ] cache/index outages do not corrupt cart state;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G08

**PASS when:** guest/auth carts and wishlist behavior are persistent, retry-safe, role-scoped, and usable with or without HTMX.

**BLOCK when:** browser state is authoritative, guest merge loses items, or repeated clicks create duplicate/invalid cart state.

## Evidence

Attach a browser flow recording, cart merge test output, request/response examples for HTMX and normal HTML, and known limitations before checkout.

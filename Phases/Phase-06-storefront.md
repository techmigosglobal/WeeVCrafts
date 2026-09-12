# Phase 06 — Storefront prototype

**Status:** IN PROGRESS  
**PRD source:** Sections 14–29, 51, Phase 6, V1 prototype contract  
**Depends on:** Phase 05  
**Unlocks:** Phase 07 and Phase 08

## Outcome

Deliver the first visible, responsive RES2 storefront using Go HTML templates, Tailwind CSS, HTMX progressive enhancement, and tightly scoped Alpine.js.

## In scope

- shared layout, navigation, footer, public policy links, and empty/error states;
- home, category/listing, product detail, seller summary, and deals entry points;
- product cards with essential information and restrained microinteraction;
- responsive mobile/tablet/desktop layouts;
- server-rendered HTML first; HTMX only for progressive partial updates;
- Alpine only for local dropdown, modal, gallery, tabs, drawer, and selection state;
- keyboard navigation, focus states, semantic markup, reduced-motion handling, and no layout-moving effects;
- public cache headers that do not expose private/account/order/checkout content.

## Out of scope

Search behavior, cart mutation, wishlist mutation, checkout, payment, seller/admin workspaces, and a production CDN rollout.

## Deliverables

- [x] storefront routes render server-side without requiring HTMX;
- [x] product data is read from PostgreSQL-backed application services;
- [x] no runtime demo fallback exists;
- [x] mobile and desktop browse entry points are visible without excessive animation;
- [x] templates do not contain provider calls;
- [x] browser error/empty states are intentional.

## Preview

This is the first user-visible prototype:

```text
Home → category → product detail
```

The preview must use explicit local catalog data, show a truthful empty state when data is absent, and provide screenshots or a recording at mobile, tablet, and desktop widths.

## Verification

- [ ] browser/manual route checklist covers home, category, product, seller summary, policy links, 404, and empty states;
- [ ] the same routes work with HTMX disabled;
- [ ] keyboard-only navigation and visible focus are checked;
- [ ] `prefers-reduced-motion` is checked;
- [ ] HTML output is escaped and invalid product identifiers do not leak internals;
- [ ] responsive preview is captured at agreed widths;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass;
- [ ] browser evidence records revision, data source, and known limitations.

## Gate G06

**PASS when:** the storefront is previewable from real local application data, works without HTMX, meets the baseline accessibility checks, and does not claim search/cart/payment readiness.

**BLOCK when:** the page requires JavaScript to render core content, uses runtime demo fallbacks, leaks private data through caching, or fails at supported widths.

## Evidence

Store the preview URL/command, screenshots or recording, route checklist, accessibility notes, and browser console/network findings.

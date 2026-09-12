# Phase 04 — Catalog foundation

**Status:** IN PROGRESS  
**PRD source:** Sections 4–5, 18–19, 30, Phase 4  
**Depends on:** Phase 03  
**Unlocks:** Phase 05

## Outcome

Create the multi-vendor catalog model and workflows for sellers, categories, brands, products, variants, statuses, ownership, approval, and cursor pagination.

## In scope

- seller profile and lifecycle state;
- categories and brands;
- products, variants, SKU, attributes, media references, and product status;
- seller-owned create/update/read behavior;
- marketplace-admin approval/rejection/moderation;
- seller isolation in every repository and application-service path;
- cursor/keyset pagination for large lists;
- audit events for approval, rejection, suspension, and material catalog changes.

## Out of scope

Inventory reservation, checkout, search synchronization, image upload implementation, seller settlements, and full admin analytics.

## Deliverables

- [ ] catalog migrations and queries;
- [ ] seller create/update and approval workflows;
- [ ] variant/SKU uniqueness rules;
- [ ] product status transitions are explicit and authorized;
- [ ] seller list/detail views are scoped to the current seller;
- [ ] admin moderation view is separate from seller editing;
- [ ] cursor pagination contract is documented for web and future API use.

## Preview

An authorized seller can create a catalog draft, while an admin can approve it. A different seller receives a denial or empty result and cannot infer or mutate the first seller's records.

## Verification

- [ ] integration tests cover seller ownership and cross-seller denial;
- [ ] status transition tests cover invalid transitions;
- [ ] SKU/slug uniqueness and validation tests pass;
- [ ] admin approval and audit tests pass;
- [ ] cursor pagination tests cover stable ordering and no duplicate/missing rows;
- [ ] empty catalog states are rendered honestly;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G04

**PASS when:** approved catalog records can be created and reviewed through role-scoped workflows, pagination is stable, and seller isolation is proven by negative tests.

**BLOCK when:** draft/unapproved products leak into public reads, seller ownership is inferred only from request input, or pagination uses unbounded offsets for large lists.

## Evidence

Record schema/migration checks, role-scoped HTTP/API tests, pagination fixture results, and a short seller/admin preview.

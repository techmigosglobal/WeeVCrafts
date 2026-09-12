# Phase 11 — Seller and admin operations

**Status:** IN PROGRESS  
**PRD source:** Sections 8–13, 18–19, 58, Phase 11  
**Depends on:** Phases 04 and 10  
**Unlocks:** Phase 12 and Phase 17

## Outcome

Provide role-specific seller, support, finance, operations, marketplace-admin, and super-admin workspaces with least privilege, seller isolation, and complete auditability for privileged actions.

## In scope

- seller dashboard, profile, staff, permissions, catalog, inventory, orders, fulfillment, returns, reports, commission, and settlement views;
- marketplace-admin seller/product moderation, customer/order/payment/refund/return/review/promotion views;
- finance payment/refund/commission/settlement/reconciliation views;
- support views with masked/minimized customer data;
- operations/returns workflows;
- super-admin role/security configuration with controlled break-glass audit;
- permission matrix rather than broad admin bypass;
- audit events for role changes, approvals, suspensions, refunds, manual order changes, and settlements.

## Out of scope

Unapproved finance policy changes, automatic settlement calculation overrides, broad cross-tenant data exports, and mobile admin clients.

## Deliverables

- [ ] each role has explicit allowed actions and data scope;
- [ ] seller owner can manage staff permissions;
- [ ] finance can access payment/settlement data but not catalog/security policy;
- [ ] support sees only necessary masked data;
- [ ] every privileged mutation records actor, action, resource, timestamp, request ID, and state change where appropriate;
- [ ] manual overrides require explicit reason and authorization.

## Preview

Reviewer can switch between controlled role accounts and see distinct dashboards, menus, records, and denied actions. A seller cannot see another seller's product/order/settlement data; support cannot see secrets or full payment credentials.

## Verification

- [ ] role/permission matrix is tested for allow and deny paths;
- [ ] seller isolation is tested at repository, application, and HTTP/API layers;
- [ ] finance/support/admin/super-admin boundaries have negative tests;
- [ ] audit log completeness is checked for each privileged mutation;
- [ ] masked data output is inspected in browser and logs;
- [ ] refund/order/manual-change workflows are idempotent and transactional;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G11

**PASS when:** each privileged workflow is role-scoped, seller isolation is proven end-to-end, and audit records are complete without sensitive leakage.

**BLOCK when:** an admin flag bypasses application authorization, seller scoping depends on client input, or privileged changes cannot be attributed.

## Evidence

Attach role matrix, role-based browser/API previews, negative test results, and redacted audit examples.

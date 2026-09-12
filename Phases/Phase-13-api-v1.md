# Phase 13 — `/api/v1` and mobile readiness

**Status:** IN PROGRESS  
**PRD source:** Sections 2, 4, 72–73, 86–87, Phase 13  
**Depends on:** Phases 09, 10, and 12  
**Unlocks:** Phase 14 and future Flutter work

## Outcome

Expose stable JSON APIs over the same application services used by the web, preserving a clean seam for future Android/iOS clients without implementing mobile clients in v1.

## In scope

- `/api/v1` routing and versioning;
- product/search/detail, cart, checkout/order, auth, account, and role-scoped API contracts needed by v1;
- JSON-only transport with no template/HTMX concerns;
- authentication and authorization for API clients;
- cursor pagination and standardized error envelope;
- request IDs and safe error handling without stack traces;
- OpenAPI/API documentation generated or maintained from the actual contract;
- reuse of application services rather than duplicated business logic.

## Out of scope

Flutter implementation, `/api/v2`, provider-specific mobile logic, and broad public API exposure without an approved contract.

## Deliverables

- [ ] API route inventory and versioning rules;
- [ ] JSON request/response schemas;
- [ ] stable error shape with code, message, and request ID;
- [ ] authentication, session/token, CSRF or API-specific mutation protection decision;
- [ ] pagination and filtering contracts;
- [ ] OpenAPI documentation matches implemented behavior;
- [ ] web and API tests prove both invoke the same application use cases.

## Preview

Reviewer can call documented JSON endpoints for catalog, cart, checkout/order, and account flows, inspect request IDs/errors, and compare results with the web flow. No HTML fragment or template concern appears in `/api/v1` responses.

## Verification

- [ ] contract tests cover success, validation, auth, authorization, pagination, and errors;
- [ ] API responses contain no templates, HTMX headers, or internal stack traces;
- [ ] API and web behavior share application-service tests or an integration fixture;
- [ ] old-version compatibility rules are documented before changes;
- [ ] OpenAPI validation passes;
- [ ] API rate limiting and idempotency behavior is tested for mutations;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G13

**PASS when:** `/api/v1` is documented, JSON-only, authorized, contract-tested, and demonstrably reuses the same commerce rules as the web.

**BLOCK when:** API logic is duplicated, web-template concerns leak into JSON, or mobile clients would need provider-specific business behavior.

## Evidence

Attach OpenAPI validation, contract-test output, sample JSON/error responses, request-ID trace, and web/API parity evidence.

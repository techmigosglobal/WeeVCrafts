# Phase 03 — Security and authentication

**Status:** IN PROGRESS  
**PRD source:** Sections 5–13, 52–55, 59–64, Phase 3  
**Depends on:** Phase 02  
**Unlocks:** Phase 04

## Outcome

Provide secure registration, login, logout, session management, CSRF protection, rate limits, and server-side role authorization for the defined marketplace actors.

## In scope

- customer, seller, staff, admin, support, finance, operations, and super-admin role model;
- Argon2id password hashing and credential handling;
- opaque Redis-backed web sessions with secure, HttpOnly, SameSite cookies;
- session rotation after login/privilege changes and logout-current/all-session controls;
- registration, login, logout, password reset/recovery, email verification seams;
- request validation, CSRF protection for state-changing web requests, and authentication rate limiting;
- server-side authorization and role/permission checks;
- masked support views and audit logging of security-sensitive actions.

## Out of scope

Social login, passkeys/WebAuthn implementation, MFA enforcement rollout beyond the required seam, mobile client UI, and payment authorization.

## Deliverables

- [ ] auth routes work as normal HTML and progressive HTMX enhancements;
- [ ] protected state-changing routes require authentication and CSRF validation;
- [ ] role checks are enforced in application services, not only hidden in templates;
- [ ] seller/customer/admin records cannot cross role or tenant boundaries;
- [ ] session expiry, revocation, and logout-all behavior are observable;
- [ ] brute-force limits and security events are measured/audited.

## Preview

Reviewer can register, log in, log out, view a session/device list, and see a denied response when attempting an unauthorized action. The preview uses an explicit local account setup and never displays credentials in committed files.

## Verification

- [ ] unit tests cover password hashing and credential failure behavior;
- [ ] integration tests cover registration, login, logout, expiry, rotation, and revocation;
- [ ] negative tests cover CSRF omission, invalid session, privilege escalation, and cross-role access;
- [ ] rate-limit tests cover repeated failed authentication;
- [ ] secure cookie attributes are inspected in HTTP responses;
- [ ] logs contain request IDs but no passwords, tokens, OTPs, or payment credentials;
- [ ] normal HTML and HTMX-enhanced auth flows both work;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G03

**PASS when:** all protected mutations are authenticated, CSRF-protected, authorized server-side, rate-limited where appropriate, and role-isolation tests pass.

**BLOCK when:** authorization exists only in the UI, sessions are bearer data in the browser, CSRF can be bypassed, or sensitive values enter logs.

## Evidence

Attach role matrix, HTTP test output, cookie/header inspection, denied-access examples, and audit-log examples with sensitive data redacted.

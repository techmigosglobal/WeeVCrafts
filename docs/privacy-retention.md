# WeCratfs privacy retention matrix

Version: `privacy-2026-09`  
Status: implementation baseline; legal review is required before release.

This matrix describes the current product behavior. It is not a substitute for
current legal advice or a signed retention decision.

| Record family | Local behavior | Retention rationale/owner |
| --- | --- | --- |
| Account profile and credentials | Deletion anonymizes the email/display name, marks the account `deleted`, and removes credentials | Privacy/identity owner; retain only the minimum account state needed to prevent reactivation and preserve references |
| Addresses, carts, wishlist, marketing preference, sessions | Removed in the same deletion transaction | Privacy/identity owner |
| Orders and order items | Order rows and financial values remain; address snapshot is replaced with `{"redacted":true}` | Finance/support owner; exact period requires accounting/legal approval |
| Payment attempts and webhook events | Remain linked to the retained order; no card number/CVV is stored | Finance/security owner; provider and accounting retention review required |
| Consent history | Remains versioned and timestamped for the consent audit trail | Privacy owner; legal review required |
| Audit logs | Deletion completion and privileged changes are recorded without secrets | Security/operations owner; retention period requires legal approval |
| Seller/catalog records | Seller display is anonymized/closed; catalog history remains subject to marketplace policy | Marketplace operations owner; legal review required |

Deletion requires an authenticated customer, a CSRF-protected confirmation, and
a customer-owned open deletion request. The database transaction locks the
request and user, performs eligible deletion/anonymization, updates the
request to `completed`, and writes an audit record. A completed request cannot
be executed again.

Open decisions before production:

- approve exact retention periods and lawful basis for every record family;
- define a verified support/finance exception workflow;
- add a scheduled retention job only after the legal retention matrix is signed;
- record processor locations and contractual safeguards for each external system.

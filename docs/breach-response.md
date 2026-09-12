# WeCratfs breach response runbook

This runbook is an operational baseline and requires owner/security/legal review
before production use.

1. Detect and record the alert, request ID, affected system, time window, and
   suspected data classes without copying secrets into tickets or logs.
2. Contain access by revoking sessions/credentials, disabling compromised
   provider keys, restricting the affected object prefix, or isolating the
   workload as appropriate.
3. Preserve evidence: immutable application/audit logs, database timestamps,
   provider event IDs, deployment revision, and a timeline of actions.
4. Assess scope across PostgreSQL, Redis, search, object storage, payment
   metadata, seller data, and customer data. Treat Redis/Meilisearch as
   disposable copies but still investigate exposure.
5. Remediate the root cause, rotate secrets, deploy the smallest reviewed fix,
   and run the relevant negative and replay tests.
6. Notify the responsible owner, processors, affected people, and authorities
   when the current legal assessment requires notification.
7. Close with a post-incident report containing impact, timeline, evidence,
   corrective actions, owners, due dates, and a retest result.

No payment credential, password, token, CVV, or raw KYC document belongs in an
incident ticket or ordinary application log.

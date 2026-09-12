Yes. We should now freeze a **Master PRD for res2** and use it as the source of truth for development, QA, security, UI/UX, backend architecture, and future mobile work.

One compliance note: live web lookup is disabled in this chat right now, so I can’t verify any DPDP Rules, notifications, or amendments issued after my available legal knowledge. The privacy section below is therefore based on the **Digital Personal Data Protection Act, 2023 baseline** and should be reviewed against the latest rules/counsel before production launch.

# RES2 E-Commerce Platform — Master PRD v1.0

## 1. Product vision

Build a modern multi-vendor e-commerce platform that is:

* extremely fast;
* highly interactive without depending on a heavy SPA;
* affordable to operate initially;
* capable of scaling horizontally later;
* portable between hosting providers;
* independent of any single database/cloud vendor;
* ready for Android/iOS later;
* privacy-first;
* secure around payments, sellers and customer data;
* modular enough that individual services can be extracted later without rewriting the whole application.

The architectural goal is not:

> “Build Amazon infrastructure on day one.”

It is:

> **“Build a clean architecture today that can evolve toward Amazon/Flipkart-like scaling patterns when traffic actually requires it.”**

---

# 2. Locked technology architecture — `res2`

```text
                         USERS
                           │
                           ▼
                    CLOUDFLARE
            DNS + CDN + SSL + DDoS + Cache
                  + Turnstile
                           │
             ┌─────────────┴─────────────┐
             │                           │
       Cached public pages         Dynamic requests
             │                           │
             └─────────────┬─────────────┘
                           ▼
                  HOSTINGER / VPS
                           │
                         Caddy
                           │
                 ┌─────────┴─────────┐
                 │                   │
              Go App              Go Worker
       HTMX + templ + REST API
                 │                   │
                 └─────────┬─────────┘
                           │
                ┌──────────┼──────────┐
                ▼          ▼          ▼
            PostgreSQL   Valkey    Cloudflare R2
                                      │
                              Images / Documents
                           Public + Private Buckets

                External integrations
                ├── Razorpay / Route
                ├── PayPal
                └── Shiprocket
```

---

# 3. Architectural responsibilities

| Layer                 | Responsibility                                |
| --------------------- | --------------------------------------------- |
| Cloudflare            | DNS, TLS, CDN, WAF, bot/rate controls         |
| Go                    | Business execution and HTTP/API server        |
| HTML Templates        | Server-rendered web UI                        |
| HTMX                  | Partial updates and server interactions       |
| Tailwind CSS          | Design system and responsive styling          |
| Alpine.js             | Small client-side UI state                    |
| Commerce Core         | Business rules                                |
| PostgreSQL            | Permanent source of truth                     |
| Redis                 | Cache, sessions, temporary state, rate limits |
| Meilisearch           | Search/indexing/filtering                     |
| S3-compatible storage | Product media, uploads, documents             |
| Razorpay              | Payment processing                            |
| `/api/v1`           | Stable API for Flutter/future frontends       |

---

# 4. Core architectural principle

Business logic must never depend directly on:

* HTMX;
* Razorpay SDK;
* Redis;
* Meilisearch;
* AWS;
* Hostinger;
* Cloudflare;
* a particular frontend.

Instead:

```text
Domain / Commerce Core
         │
         ▼
Interfaces / Ports
         │
   ┌─────┼────────┬─────────┐
   ▼     ▼        ▼         ▼
Postgres Redis   Search   Payment
Adapter  Adapter Adapter  Adapter
```

Examples:

```go
type PaymentGateway interface {}
type SearchEngine interface {}
type ObjectStorage interface {}
type Cache interface {}
type ProductRepository interface {}
```

Today:

```text
PaymentGateway → Razorpay
```

Tomorrow:

```text
PaymentGateway → Cashfree
```

without rewriting Checkout.

---

# 5. Initial product model

The product is a **multi-vendor marketplace**.

Customer may purchase products sold by different sellers through the same marketplace.

Initial supported actors:

1. Guest
2. Customer
3. Seller Owner
4. Seller Staff
5. Marketplace Admin
6. Super Admin
7. Support Agent
8. Finance/Settlement Operator
9. Operations/Returns Operator

We should not give everyone broad “admin” access.

---

# 6. Role definitions

## Guest

Can:

* browse;
* search;
* filter;
* view product;
* view seller;
* create guest cart;
* add/remove items;
* register/login;
* begin checkout.

Cannot:

* place final order without required customer information;
* review products;
* access protected customer data.

---

# 7. Customer

Can:

* manage profile;
* manage addresses;
* browse/search;
* use cart;
* wishlist;
* purchase;
* view orders;
* cancel eligible orders;
* initiate return/refund;
* review verified purchases;
* manage privacy preferences;
* request account/data actions;
* log out all sessions.

---

# 8. Seller Owner

Can:

* manage seller profile;
* invite/manage seller staff;
* create products;
* manage variants;
* upload product media;
* manage pricing;
* manage inventory;
* view own orders;
* process fulfillment;
* handle return workflows where permitted;
* see reports;
* view settlement/commission records.

Cannot:

* see other sellers' records;
* directly alter customer payment state;
* directly modify settlement calculations.

---

# 9. Seller Staff

Permissions should be granular.

Example permission set:

```text
PRODUCT_READ
PRODUCT_WRITE

INVENTORY_READ
INVENTORY_WRITE

ORDER_READ
ORDER_FULFILL

RETURN_READ
RETURN_PROCESS

REPORT_READ
```

Seller owner chooses access.

---

# 10. Marketplace Admin

Can:

* approve sellers;
* approve/reject products;
* manage categories;
* manage promotions;
* view marketplace orders;
* moderate reviews;
* manage returns/disputes;
* manage customer issues;
* manage seller suspension;
* inspect audit logs.

Cannot automatically override Super Admin/security-sensitive configuration.

---

# 11. Finance Operator

Can access:

* payments;
* refunds;
* commissions;
* seller settlements;
* Razorpay reconciliation;
* finance reports.

Cannot modify:

* product/catalog information;
* authentication/security policy.

---

# 12. Support Agent

Can see only what is necessary for customer support:

* order summary;
* shipment status;
* support history;
* masked customer information where possible.

Should not see:

* passwords;
* security tokens;
* full payment credentials;
* unrelated customer data.

---

# 13. Super Admin

Highest privileged role.

Can:

* manage administrator roles;
* manage system settings;
* modify security-critical configuration;
* suspend accounts;
* inspect complete audit trails;
* initiate controlled break-glass operations.

Every Super Admin operation should be heavily audited.

---

# 14. Customer-facing screen list

## Public

1. Home
2. Search
3. Category listing
4. Brand page
5. Product listing
6. Product detail
7. Seller storefront
8. Deals/promotions
9. Search suggestions
10. About
11. Contact
12. FAQ
13. Privacy Policy
14. Terms
15. Refund/return policy
16. Shipping policy

---

# 15. Authentication screens

17. Register
18. Login
19. Forgot password
20. Reset password
21. Email verification
22. Session/device management
23. Account recovery

Future:

* Google login;
* Apple login;
* passkeys.

---

# 16. Shopping screens

24. Cart
25. Wishlist
26. Address selection
27. Add address
28. Shipping method
29. Coupon/promotion
30. Order review
31. Razorpay checkout
32. Payment processing
33. Payment success
34. Payment failed
35. Pending payment reconciliation

---

# 17. Customer account screens

36. Dashboard
37. Profile
38. Addresses
39. Orders
40. Order detail
41. Shipment tracking
42. Cancellation
43. Returns
44. Return detail
45. Refund status
46. Reviews
47. Wishlist
48. Notifications
49. Privacy center
50. Download/request data
51. Delete account/request erasure
52. Consent preferences

---

# 18. Seller screens

1. Seller onboarding
2. Verification/document submission
3. Seller dashboard
4. Product list
5. Add product
6. Edit product
7. Variants
8. Product media
9. Pricing
10. Inventory
11. Inventory transactions
12. Orders
13. Order details
14. Fulfillment
15. Returns
16. Refund-related status
17. Promotions
18. Seller analytics
19. Commission reports
20. Settlement reports
21. Seller profile
22. Staff
23. Staff permissions
24. Support
25. Audit activity

---

# 19. Admin screens

1. Admin dashboard
2. Seller approvals
3. Seller details
4. Seller suspensions
5. Product moderation
6. Product details
7. Category management
8. Brands
9. Customers
10. Orders
11. Payments
12. Failed payments
13. Refunds
14. Returns
15. Disputes
16. Promotions
17. Coupons
18. Reviews moderation
19. Commission rules
20. Settlements
21. Finance reconciliation
22. Support tickets
23. Audit logs
24. System health
25. Search indexing health
26. Worker queue health
27. Security events
28. Settings
29. Role management

---

# 20. UI/UX vision

We should avoid making it look like:

> another Bootstrap/React admin template.

The storefront should feel distinct.

Design principles:

* fast;
* tactile;
* visual;
* clean;
* minimal cognitive load;
* product-first;
* playful where safe;
* mobile-first;
* accessible.

---

# 21. Creative design concept

I recommend a design direction called:

# **Living Storefront**

The storefront subtly reacts to user intent without becoming distracting.

Examples:

### Product cards

Hover/focus:

```text
Image gently enlarges
      +
secondary image crossfades
      +
quick actions appear
```

No excessive movement.

---

# 22. Intelligent product card

Instead of:

```text
Image
Name
Price
Add
```

use:

```text
Image gallery preview

Brand
Product

₹29,999
↓ ₹33,999

★ 4.6 (2.3k)

● Blue
● Black
● Silver

Delivery tomorrow

[Quick View]
[Add]
```

Only essential information appears initially; secondary information expands contextually.

---

# 23. Cart microinteraction

When customer adds product:

Don't open a blocking modal.

Do:

```text
product image
     ↓
small animated movement toward cart

Cart badge:
2 → 3

mini toast:
"Added to cart"
```

HTMX can update:

* cart button;
* header count;
* toast

from one response using out-of-band swaps.

---

# 24. Search experience

Search box should feel like a command center.

Example:

```text
Search products...

iphone

Suggestions
────────────
iPhone 17
iPhone 17 Pro
iPhone cases

Categories
────────────
Phones

Brands
────────────
Apple

Recent
────────────
iPhone charger
```

Meilisearch powers results.

Use approximately:

```text
300ms debounce
```

and cancel obsolete requests.

---

# 25. Visual filtering

Desktop:

```text
[Brand ▼] [Price ▼] [Rating ▼] [Delivery ▼]
```

Mobile:

```text
Filter
  ↓
bottom sheet
```

HTMX updates product grid without full page reload.

URL still becomes:

```text
/products?brand=apple&price_max=80000
```

so search/filter states remain shareable.

---

# 26. Product detail experience

Sections:

* image/gallery;
* essential purchase information;
* variants;
* price;
* offers;
* delivery estimate;
* seller information;
* return policy;
* highlights;
* specifications;
* description;
* reviews;
* related products.

Sticky purchase area on desktop.

Sticky bottom purchase controls on mobile.

---

# 27. Effects rules

Allowed:

* subtle scale;
* fade;
* slide;
* skeleton placeholders;
* animated cart count;
* toast;
* smooth drawer;
* image transition;
* micro-feedback.

Avoid:

* constant parallax;
* huge animations;
* forced scroll effects;
* autoplay audio;
* layout-moving animations;
* effects that delay shopping.

Respect:

```css
prefers-reduced-motion
```

---

# 28. HTMX responsibilities

HTMX should handle:

* cart mutation;
* wishlist;
* filters;
* search;
* pagination;
* seller forms;
* inventory editing;
* admin moderation;
* checkout step transitions;
* inline validation;
* inline address management.

HTMX should not become application state storage.

---

# 29. Alpine.js responsibilities

Alpine is limited to:

* dropdown;
* modal;
* accordion;
* image gallery;
* tabs;
* mobile drawer;
* local selection state;
* visual toggles.

Never:

```text
Alpine = authoritative cart

Alpine = authoritative inventory

Alpine = payment state
```

---

# 30. PostgreSQL schema families

Core tables:

### Identity

```text
users
credentials
roles
user_roles
sessions_metadata
```

### Customer

```text
customers
customer_addresses
wishlists
wishlist_items
```

### Seller

```text
sellers
seller_users
seller_documents
seller_addresses
```

### Catalog

```text
categories
brands
products
product_variants
product_images
product_attributes
```

### Pricing

```text
prices
price_lists
promotions
promotion_rules
promotion_usage
```

### Inventory

```text
inventory_locations
inventory_items
inventory_transactions
inventory_reservations
```

### Cart

```text
carts
cart_items
```

### Orders

```text
orders
order_items
order_status_history
```

### Payment

```text
payment_attempts
payment_events
refunds
```

### Fulfillment

```text
shipments
shipment_items
tracking_events
```

### Returns

```text
returns
return_items
```

### Seller finance

```text
commission_rules
seller_ledger
seller_settlements
```

### Platform

```text
outbox_events
idempotency_keys
audit_logs
```

---

# 31. Backend performance architecture

Performance is achieved through several layers rather than one technique.

```text
Cloudflare
    ↓
Go
    ↓
Redis
    ↓
PostgreSQL
```

and independently:

```text
Search
 ↓
Meilisearch
```

---

# 32. Database connection pooling

Use:

```text
pgxpool
```

Go should maintain a bounded pool.

Do not map:

```text
one user
=
one DB connection
```

Example starting point:

```text
20–30 application DB connections
```

and tune using measurements.

---

# 33. PostgreSQL indexing

Indexes should be created based on real access patterns.

Examples:

```text
products.slug

product_variants.sku

products.category_id

inventory_items.variant_id

orders.customer_id

orders.created_at

orders.status

order_items.order_id

payments.order_id
```

Avoid blindly indexing every column because indexes increase write overhead.

---

# 34. Keyset pagination

Large tables should not rely on:

```sql
OFFSET 100000 LIMIT 20
```

Use:

```sql
WHERE (created_at, id) < ($1,$2)
ORDER BY created_at DESC, id DESC
LIMIT 25
```

This applies to:

* orders;
* seller orders;
* customers;
* transactions;
* reviews;
* audit logs.

---

# 35. Meilisearch pagination

Product discovery uses Meilisearch's search-oriented pagination.

Customer-facing URLs can remain human-readable:

```text
/search?q=iphone&page=3
```

---

# 36. Redis cache-aside

Example:

```text
Request
   ↓
Redis
   │
   ├── HIT
   │     ↓
   │ response
   │
   └── MISS
         ↓
     PostgreSQL
         ↓
       Redis
         ↓
      response
```

Use for:

* product summary;
* categories;
* navigation;
* seller summary;
* feature flags;
* sessions.

---

# 37. Cache stampede protection

If a popular product cache expires:

Bad:

```text
500 requests
   ↓
500 DB queries
```

Use:

```text
singleflight/lock
      ↓
1 query
      ↓
refresh cache
      ↓
all waiters reuse result
```

---

# 38. Inventory concurrency

Inventory cannot depend on caching.

Example:

```text
Stock = 1
```

Two users checkout.

Use PostgreSQL transaction:

```text
BEGIN

SELECT inventory
FOR UPDATE

validate stock

reserve

COMMIT
```

or verified atomic conditional updates.

Overselling is never acceptable.

---

# 39. Idempotency

Critical endpoints require idempotency.

Especially:

```text
checkout
order creation
payment initiation
refund
seller settlement
```

Example:

```text
POST /checkout

Idempotency-Key:
abc123
```

Twenty retries with the same key:

```text
1 order
```

not twenty.

---

# 40. Async processing

Customer HTTP request should not wait for:

* email;
* invoice generation;
* analytics;
* notification;
* search indexing;
* recommendation updates.

Use:

```text
Postgres transaction
      ↓
outbox_events
      ↓
Go worker
```

---

# 41. Outbox pattern

Inside same DB transaction:

```text
Create order
+
Insert outbox event
+
COMMIT
```

Worker later sends:

```text
OrderCreated
```

This avoids the situation:

```text
Order saved
but
notification/search event lost
```

---

# 42. Worker concurrency

Use bounded concurrency.

Example:

```text
4 workers
```

initially.

Not:

```text
1000 goroutines
```

Workers can claim jobs using:

```sql
FOR UPDATE SKIP LOCKED
```

---

# 43. Goroutine rules

Use concurrency for independent work.

Good:

```text
Product page

product
reviews summary
seller summary
delivery estimate
```

can potentially be fetched concurrently.

Not:

```text
inventory update
order creation
payment state update
```

inside uncontrolled goroutines.

Those require ordered transactions.

---

# 44. Context propagation

Every request uses:

```go
context.Context
```

through:

```text
HTTP
 ↓
application service
 ↓
repository
 ↓
database
```

If request is cancelled:

```text
DB operation can stop
```

preventing wasted work.

---

# 45. Request timeouts

Use time limits.

Example categories:

```text
normal HTTP
search
external Razorpay
storage
worker
```

No network request should wait indefinitely.

---

# 46. External dependency protection

External APIs should use:

* timeout;
* bounded retry;
* exponential backoff;
* jitter;
* circuit-breaker-style protection where appropriate.

Do not blindly retry:

```text
payment creation
refund
```

without idempotency.

---

# 47. Backpressure

If worker queue becomes huge:

Do not spawn unlimited workers.

Limit processing and expose:

```text
queue_depth
oldest_job_age
processing_rate
```

through monitoring.

---

# 48. Search architecture

```text
PostgreSQL
     │
     │ ProductChanged
     ▼
Outbox
     ↓
Worker
     ↓
Meilisearch
```

PostgreSQL remains truth.

If Meilisearch dies:

```text
rebuild index
from PostgreSQL
```

---

# 49. Storage architecture

Browser upload:

```text
Browser
  │
  │ request signed URL
  ▼
Go
  │
  ▼
signed URL
  │
Browser ──────────► S3
```

File bytes avoid Go.

Benefits:

* lower RAM use;
* lower CPU;
* less server bandwidth;
* fewer upload timeouts.

---

# 50. Image variants

Generate:

```text
thumbnail
small
medium
large
original
```

where appropriate.

Prefer optimized formats where supported.

Never load a 6MB original image for a 200px product card.

---

# 51. Cloudflare caching policy

Cache:

* CSS;
* JS;
* fonts;
* public images;
* marketing assets;
* selected anonymous HTML.

Do not cache private:

* account;
* order;
* checkout;
* seller;
* admin.

---

# 52. Security architecture

Mandatory:

* HTTPS everywhere;
* secure cookies;
* HttpOnly cookies;
* SameSite strategy;
* CSRF protection;
* authentication rate limiting;
* password hashing using Argon2id;
* authorization server-side;
* input validation;
* output escaping;
* secure headers;
* secret management;
* dependency updates;
* audit logs.

---

# 53. Session security

Web sessions:

```text
browser
 ↓
opaque session ID
 ↓
Redis
```

Cookie:

```text
HttpOnly
Secure
SameSite
```

Rotate session after login/privilege change.

Support:

```text
logout current session
logout all sessions
device/session list
```

---

# 54. Password policy

Do not force arbitrary complexity such as:

```text
1 uppercase
1 special
1 number
```

as the sole security mechanism.

Prefer:

* sufficient length;
* compromised-password checks where possible;
* rate limiting;
* secure hashing;
* optional MFA for privileged accounts.

---

# 55. Admin MFA

Admin, Super Admin and Finance should eventually require MFA.

Prefer phishing-resistant methods later:

```text
passkeys / WebAuthn
```

or secure TOTP.

---

# 56. Payment security

Never store:

* card number;
* CVV;
* card credentials.

Razorpay handles payment credentials.

Our DB stores:

```text
provider
razorpay_order_id
payment_id
status
amount
timestamps
```

---

# 57. Razorpay webhook security

Requirements:

* HTTPS;
* signature verification;
* raw body verification;
* event ID uniqueness;
* replay resistance;
* idempotent handling;
* audit trail.

---

# 58. Audit logging

Audit:

```text
admin login
role modification
seller suspension
product approval
refund
manual order change
settlement
security configuration change
```

Record:

```text
actor
action
resource
timestamp
request_id
old state where appropriate
new state where appropriate
```

Avoid dumping sensitive personal data unnecessarily into logs.

---

# 59. DPDP/privacy baseline

The platform should follow principles consistent with India's Digital Personal Data Protection framework.

Core principles:

* clear notice;
* lawful processing;
* specific purpose;
* data minimization;
* accuracy where relevant;
* security safeguards;
* retention limitation;
* data principal rights;
* grievance handling;
* controlled processor access;
* breach response.

Again, current rules should be legally verified before launch.

---

# 60. Privacy Notice

Before collecting relevant personal data, clearly state:

* what is collected;
* why it is collected;
* how it is used;
* who processes it;
* retention period/category;
* rights available;
* grievance/contact route;
* how consent can be withdrawn where processing relies on consent.

Use simple language.

Avoid giant unreadable legal-only forms.

---

# 61. Consent records

Where consent is applicable, store:

```text
customer_id
purpose
policy_version
consent_state
timestamp
source
```

Example:

```text
marketing_email
=
true
```

must remain distinct from:

```text
transactional_order_email
```

Don't bundle optional marketing consent with necessary order processing.

---

# 62. Consent withdrawal

Customer should have a simple privacy/preferences screen.

Example:

```text
Marketing email        ON/OFF
SMS offers             ON/OFF
Personalized offers    ON/OFF
```

Withdrawal should be as easy as granting consent where legally required.

---

# 63. Data minimization

Don't collect:

```text
date of birth
gender
government ID
location history
```

unless genuinely required.

For ordinary customer checkout we generally need:

```text
name
contact
shipping address
billing information as necessary
```

and nothing more.

---

# 64. Children's data

The DPDP Act treats children as persons under 18 unless changed by applicable rules/exemptions.

If the service knowingly processes children's personal data, design for:

* verifiable parental consent where applicable;
* no harmful processing;
* restrictions on tracking/behavioural advertising to children as required.

Simplest marketplace policy may be:

```text
accounts intended for adults
```

unless the business specifically needs child accounts.

Legal review required.

---

# 65. Seller KYC/privacy

Seller documents may be highly sensitive.

Use:

* private S3 bucket;
* signed temporary access;
* least privilege;
* encryption;
* audit access;
* retention/deletion policy.

Seller documents should never be publicly accessible via predictable URLs.

---

# 66. Data subject/account controls

Privacy center should support processes for:

* access/request summary;
* correction;
* updating data;
* erasure/deletion where applicable;
* consent withdrawal;
* grievance submission.

Some business records may need to be retained where legally required.

Deletion does not necessarily mean deleting legally required invoice/accounting records.

---

# 67. Account deletion architecture

Customer clicks:

```text
Delete account
```

Do not immediately cascade-delete orders.

Instead:

```text
request
 ↓
verification
 ↓
eligibility/retention check
 ↓
erase/anonymize eligible data
 ↓
retain legally required records
 ↓
audit completion
```

---

# 68. Data retention

Create a formal retention table.

Example concept:

| Data                      | Retention                           |
| ------------------------- | ----------------------------------- |
| Active profile            | Account lifetime                    |
| Session                   | Short TTL                           |
| Abandoned cart            | Defined limited period              |
| Payment logs              | Finance/legal policy                |
| Orders/invoices           | Statutory/accounting period         |
| Failed login logs         | Security retention window           |
| Seller KYC                | Required business/compliance period |
| Marketing consent history | Consent/legal evidence period       |
| Uploaded temporary files  | Short cleanup period                |

Exact legal periods must be decided with accounting/legal requirements.

---

# 69. Data breach response

Have an operational procedure:

```text
Detect
 ↓
Contain
 ↓
Investigate
 ↓
Preserve evidence
 ↓
Assess affected data/users
 ↓
Notify responsible internal personnel
 ↓
Perform legally required notifications
 ↓
Remediate
 ↓
Post-incident review
```

DPDP breach reporting requirements should be aligned with the latest applicable Rules.

---

# 70. Cross-border processing

Design an inventory of where data lives:

```text
Database location
Object storage location
Email provider
Razorpay
Monitoring
Backups
```

If cross-border restrictions are notified under applicable DPDP provisions, infrastructure can then be adjusted.

This is another reason for res2's portability.

---

# 71. Privacy by architecture

Privacy should not be just a policy document.

Examples:

### Support screen

Show:

```text
V***y
+91 ******5219
```

where full detail isn't needed.

### Logs

Don't log:

```text
password
OTP
full tokens
full KYC documents
```

### Analytics

Use pseudonymous IDs wherever possible.

---

# 72. API design

All reusable APIs:

```text
/api/v1
```

Example:

```text
GET /api/v1/products
GET /api/v1/products/{id}

POST /api/v1/cart/items

POST /api/v1/checkout

GET /api/v1/orders
```

Future breaking API:

```text
/api/v2
```

Old mobile versions may continue using `/api/v1`.

---

# 73. API error format

Standard response:

```json
{
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Requested quantity is unavailable.",
    "request_id": "..."
  }
}
```

Never return internal stack traces.

---

# 74. Observability

Every request should receive:

```text
request_id
```

Important flows include:

```text
user_id
seller_id
cart_id
order_id
payment_id
```

where appropriate.

---

# 75. Metrics

Track:

### HTTP

```text
request count
latency
error rate
status codes
```

### Go

```text
CPU
memory
goroutines
GC
```

### PostgreSQL

```text
connections
query latency
slow queries
locks
deadlocks
transactions/sec
```

### Redis

```text
hit rate
memory
evictions
latency
```

### Meilisearch

```text
search latency
index size
indexing queue
```

### Worker

```text
pending jobs
processing rate
oldest job age
failed jobs
```

---

# 76. Performance SLOs

Initial targets:

| Operation             | Target |
| --------------------- | -----: |
| Cached GET P95        | <150ms |
| Product GET P95       | <250ms |
| Search P95            | <300ms |
| Cart mutation P95     | <350ms |
| Checkout internal P95 | <500ms |
| Error rate            |    <1% |

External payment latency is measured separately.

---

# 77. Load testing stages

Test:

```text
10 users
 ↓
50
 ↓
100
 ↓
250
 ↓
500
 ↓
1000
```

Then increase further if stable.

Do not immediately hammer with 50,000 simulated customers.

---

# 78. Load scenarios

Separate scenarios:

### Browse

```text
homepage
category
product
```

### Search

```text
query
filter
sort
pagination
```

### Shopping

```text
add
quantity
wishlist
```

### Checkout

```text
inventory
order
payment preparation
```

### Mixed

Realistic ratios.

---

# 79. Concurrency checkpoint

Example:

```text
Stock = 10

100 concurrent purchase attempts
```

Expected:

```text
10 successes
90 stock errors
0 overselling
```

Run repeatedly.

---

# 80. Idempotency checkpoint

Same checkout:

```text
20 concurrent identical requests
```

Expected:

```text
1 order
1 reservation
1 payment preparation
```

---

# 81. Race testing

Run:

```bash
go test -race ./...
```

after every phase involving concurrency.

No known race should be accepted into production.

---

# 82. Backup architecture

At minimum:

```text
PostgreSQL scheduled backup
      │
      ▼
off-server backup destination
```

Do not keep your only backup on the same KVM.

Also back up:

* object metadata;
* deployment configuration;
* migrations;
* secrets securely;
* Meilisearch config, though index itself can be regenerated.

---

# 83. Disaster recovery test

A backup isn't valid until restoration has been tested.

Checkpoint:

```text
destroy test DB

restore backup

start app

verify:
customers
products
inventory
orders
payments
```

PASS.

---

# 84. Development phases and gates

Now the master checkpoints.

---

## Phase 0 — architecture

Must pass:

* [ ] repository structure created;
* [ ] domain/application/adapter boundaries documented;
* [ ] `/api/v1` location reserved;
* [ ] PaymentGateway interface;
* [ ] ObjectStorage interface;
* [ ] Search interface;
* [ ] Cache interface;
* [ ] build passes;
* [ ] tests pass.

---

# Phase 1 — local infrastructure

* [ ] PostgreSQL healthy;
* [ ] Redis healthy;
* [ ] Meilisearch healthy;
* [ ] MinIO healthy;
* [ ] Go server healthy;
* [ ] Go worker healthy;
* [ ] `/health/live`;
* [ ] `/health/ready`;
* [ ] graceful shutdown;
* [ ] `.env.example`;
* [ ] Docker Compose reproducible.

---

# Phase 2 — persistence

* [ ] migrations from empty DB;
* [ ] sqlc generation;
* [ ] pgxpool;
* [ ] transaction wrapper;
* [ ] query cancellation;
* [ ] idempotency table;
* [ ] outbox;
* [ ] audit table;
* [ ] integration tests.

---

# Phase 3 — security/auth

* [ ] registration;
* [ ] login;
* [ ] logout;
* [ ] Argon2id;
* [ ] Redis session;
* [ ] secure cookie;
* [ ] CSRF;
* [ ] role checks;
* [ ] brute-force limits;
* [ ] session expiry;
* [ ] role-isolation tests.

---

# Phase 4 — catalog

* [ ] seller;
* [ ] category;
* [ ] brand;
* [ ] product;
* [ ] variant;
* [ ] product statuses;
* [ ] seller isolation;
* [ ] admin approval;
* [ ] cursor pagination.

---

# Phase 5 — inventory/pricing

* [ ] price;
* [ ] stock;
* [ ] inventory location;
* [ ] reservation;
* [ ] release;
* [ ] commit reservation;
* [ ] transaction safety;
* [ ] 100-way concurrency test;
* [ ] zero overselling.

---

# Phase 6 — storefront

* [ ] layout;
* [ ] homepage;
* [ ] category;
* [ ] product;
* [ ] responsive design;
* [ ] HTMX progressive enhancement;
* [ ] Tailwind;
* [ ] Alpine limited to local state;
* [ ] reduced motion;
* [ ] keyboard accessibility.

---

# Phase 7 — search/cache

* [ ] Meilisearch indexing;
* [ ] outbox sync;
* [ ] typo search;
* [ ] filtering;
* [ ] sorting;
* [ ] search pagination;
* [ ] Redis cache-aside;
* [ ] cache invalidation;
* [ ] stampede protection;
* [ ] search rebuild test.

---

# Phase 8 — cart/wishlist

* [ ] guest cart;
* [ ] logged-in cart;
* [ ] guest merge;
* [ ] add;
* [ ] remove;
* [ ] quantity;
* [ ] wishlist;
* [ ] HTMX OOB cart counter;
* [ ] double-click protection;
* [ ] authoritative price validation later.

---

# Phase 9 — checkout/order

* [ ] addresses;
* [ ] final price recalculation;
* [ ] inventory lock;
* [ ] reservation;
* [ ] order;
* [ ] order items;
* [ ] idempotency;
* [ ] outbox event;
* [ ] transactional integrity;
* [ ] duplicate checkout test.

---

# Phase 10 — Razorpay

* [ ] test mode;
* [ ] server order creation;
* [ ] signature verification;
* [ ] webhook;
* [ ] duplicate webhook;
* [ ] invalid signature;
* [ ] failed payment;
* [ ] pending payment;
* [ ] refund;
* [ ] payment reconciliation;
* [ ] no card data storage.

---

# Phase 11 — Seller/Admin

* [ ] seller dashboard;
* [ ] inventory;
* [ ] orders;
* [ ] returns;
* [ ] product management;
* [ ] admin moderation;
* [ ] payments;
* [ ] refunds;
* [ ] audit trail;
* [ ] seller isolation.

---

# Phase 12 — storage

* [ ] MinIO development;
* [ ] S3 adapter;
* [ ] presigned upload;
* [ ] private documents;
* [ ] public media;
* [ ] access control;
* [ ] image optimization;
* [ ] file cleanup.

---

# Phase 13 — API/mobile readiness

* [ ] `/api/v1`;
* [ ] JSON contracts;
* [ ] API authentication;
* [ ] pagination;
* [ ] standardized errors;
* [ ] same application services as web;
* [ ] OpenAPI/API documentation;
* [ ] no duplicate business logic.

---

# Phase 14 — privacy

* [ ] privacy notice;
* [ ] consent records;
* [ ] marketing preferences;
* [ ] withdrawal flow;
* [ ] privacy center;
* [ ] correction flow;
* [ ] deletion/erasure process;
* [ ] grievance process;
* [ ] data retention policy;
* [ ] KYC access restrictions;
* [ ] processor inventory;
* [ ] breach procedure;
* [ ] legal review of latest DPDP requirements.

---

# Phase 15 — performance

* [ ] k6 scenarios;
* [ ] P95/P99 metrics;
* [ ] PostgreSQL slow-query review;
* [ ] indexes reviewed;
* [ ] DB pool tuned;
* [ ] Redis hit ratio;
* [ ] worker concurrency;
* [ ] Meilisearch memory limits;
* [ ] Go pprof;
* [ ] race tests;
* [ ] no unbounded goroutines.

---

# Phase 16 — security testing

* [ ] authentication tests;
* [ ] authorization tests;
* [ ] seller isolation;
* [ ] admin isolation;
* [ ] CSRF;
* [ ] XSS escaping;
* [ ] SQL injection testing;
* [ ] rate limiting;
* [ ] upload validation;
* [ ] webhook replay;
* [ ] sensitive logs review;
* [ ] secrets review.

---

# Phase 17 — staging

* [ ] clean deployment;
* [ ] Cloudflare;
* [ ] HTTPS;
* [ ] staging domain;
* [ ] Razorpay Test webhook;
* [ ] migrations;
* [ ] seed data;
* [ ] backup;
* [ ] restore;
* [ ] full E2E flow.

---

# Phase 18 — production readiness

Launch only when:

```text
Browse
✓

Search
✓

Product
✓

Cart
✓

Inventory
✓

Checkout
✓

Razorpay
✓

Order
✓

Seller
✓

Admin
✓

Refund
✓

Backup/Restore
✓

Load
✓

Security
✓

Privacy
✓
```

---

# 85. Production release gate

The production release is blocked if any of these are true:

* inventory can oversell;
* payment can create duplicate orders;
* seller can access another seller;
* admin authorization can be bypassed;
* backup hasn't been restored successfully;
* Razorpay webhook signature isn't verified;
* passwords/tokens appear in logs;
* database migration can't run from a clean database;
* `go test -race` finds unresolved races;
* checkout P95/error rate exceeds agreed threshold under target load;
* privacy/legal checklist isn't completed.

---

# 86. V2 — Android/iOS

Flutter will not introduce new commerce logic.

Architecture:

```text
                      Commerce Core
                           ▲
                 ┌─────────┴─────────┐
                 │                   │
            HTMX Web              /api/v1
                                     │
                              ┌──────┴──────┐
                              ▼             ▼
                           Android         iOS
                           Flutter        Flutter
```

Reuse:

* customers;
* auth;
* products;
* inventory;
* cart;
* orders;
* payments;
* search;
* storage;
* seller data.

---

# 87. Future scaling checkpoints

## Stage A

```text
KVM4

Go
PostgreSQL
Redis
Meili
```

---

## Stage B

First bottleneck appears.

Move that component.

Example:

```text
KVM Go
   ↓
Managed Postgres
```

---

## Stage C

Multiple Go servers:

```text
Load Balancer

Go1
Go2
Go3

   ↓

PostgreSQL
Redis
Search
```

---

## Stage D

Database read replicas.

---

## Stage E

Separate workers.

---

## Stage F

Dedicated search cluster.

---

## Stage G

Event broker:

```text
NATS
or
RabbitMQ
```

---

## Stage H

Only then consider extracting:

```text
Catalog Service
Order Service
Inventory Service
Payment Service
Notification Service
```

---

# 88. Final definition of success

The project succeeds architecturally if:

1. We can redesign the complete website without touching order/payment logic.
2. We can add Flutter without rewriting commerce.
3. We can move PostgreSQL to another provider by changing infrastructure, not rewriting the application.
4. Redis can disappear without losing permanent data.
5. Meilisearch can be rebuilt completely from PostgreSQL.
6. Storage provider can be replaced through an adapter.
7. Razorpay can be replaced through a payment adapter.
8. One Go instance can become five without session/data problems.
9. Database requests remain bounded through pooling.
10. Concurrent checkout cannot oversell inventory.
11. retries cannot create duplicate orders/payments.
12. heavy non-critical work is asynchronous.
13. public assets do not unnecessarily consume origin server CPU/bandwidth.
14. privacy rights can be operationally fulfilled.
15. every privileged action is auditable.

---

# 89. Development instruction for all AI agents

This rule should be added to every implementation prompt:

> **Do not optimize by adding complexity. First measure the bottleneck. Keep PostgreSQL authoritative, Redis disposable, Meilisearch rebuildable, Go stateless, business logic transport-independent, all external providers behind interfaces, and all critical mutations transactional/idempotent. Every phase must pass its checkpoint before work begins on the next phase.**

That should become the **Master PRD + Architecture Contract + QA checklist for res2**.

The immediate next step is **Phase 0**, not UI development: repository structure, architecture contracts, Docker strategy, interfaces and engineering rules. Once that gate passes, we begin Phase 1 infrastructure and then proceed sequentially.

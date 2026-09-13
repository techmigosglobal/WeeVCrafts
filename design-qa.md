# Super Admin visual QA

## Source of visual truth

Reference screens reviewed from `Weevcrafts_Screens/super admin/`:

- Dashboard
- Sellers
- Products and categories
- Own inventory
- Orders and shipping
- Returns and refunds
- Finance
- Customers and support
- Marketing and content
- Analytics and settings

## Implementation evidence

- Desktop captures: `admin-qa/*-clean.png` at 1440px wide.
- Mobile capture: `admin-qa/dashboard-mobile.png` at 390px wide.
- Final dashboard capture: `admin-qa/dashboard-postfix.png`.
- Preview URL: `http://localhost:8090/admin`.
- Verified routes: `/admin`, `/admin/sellers`, `/admin/products`, `/admin/inventory`, `/admin/orders`, `/admin/returns`, `/admin/finance`, `/admin/support`, `/admin/marketing`, `/admin/analytics`.

## Review results

| Area | Result | Evidence |
|---|---|---|
| Shared admin shell, brand, sidebar, header and footer | Pass | All ten desktop captures |
| Dashboard KPIs, sales chart, category mix and operational queues | Pass | `admin-qa/dashboard-postfix.png` |
| Responsive navigation and two-column mobile cards | Pass | `admin-qa/dashboard-mobile.png` |
| Seller applications, KYC verification and seller profile | Pass | `admin-qa/sellers-clean.png` |
| Product moderation, category management and compliance queue | Pass | `admin-qa/products-clean.png` |
| Own inventory, stock alerts and warehouse coverage | Pass | `admin-qa/inventory-clean.png` |
| Marketplace orders, fulfilment and shipment detail | Pass | `admin-qa/orders-clean.png` |
| Return queue, evidence, refund decision and replacement actions | Pass | `admin-qa/returns-clean.png` |
| Payments, settlements, transactions and finance actions | Pass | `admin-qa/finance-clean.png` |
| Customer support cases, satisfaction and care standards | Pass | `admin-qa/support-clean.png` |
| Campaigns, coupons/content tabs and homepage content | Pass | `admin-qa/marketing-clean.png` |
| Analytics, export destinations, audit log, settings and system health | Pass | `admin-qa/analytics-clean.png` |
| Admin navigation and notice actions | Pass | HTTP smoke check: all discovered `/admin*` links returned 200 |

## Review notes

- The in-app browser was unavailable in this environment, so screenshots were captured with the installed headless Chrome fallback.
- The preview uses the repository's local mock images and fixture data. It demonstrates the screen responsibilities and interaction targets; it does not claim that live persistence, authorization, payment settlement, or audit storage are connected to these preview routes.
- No P0, P1 or P2 visual issues remain for this fixture-backed preview.

## Seller Admin review

### Reference and scope

- Accepted visual reference set: `Weevcrafts_Screens/seller admin/ChatGPT Image Sep 11, 2026, 07_15_44 AM (1).png` through `(9).png`.
- Reviewed all 9 seller-admin references; each reference is 1055 x 1491.
- PRD scope reviewed in `Prd.md` and `Phases/Phase-11-seller-admin.md`: seller onboarding, dashboard, products, inventory, order fulfilment, returns, earnings/settlements, analytics, store/account settings, seller isolation, and audit-friendly actions.

### Implemented routes

- `/seller-admin/onboarding`
- `/seller-admin`
- `/seller-admin/products`
- `/seller-admin/inventory`
- `/seller-admin/orders`
- `/seller-admin/returns`
- `/seller-admin/earnings`
- `/seller-admin/analytics`
- `/seller-admin/settings`

### Review results

| Area | Result | Evidence |
|---|---|---|
| Shared WeeVCrafts utility bar, public header, seller identity and portal navigation | Pass | `seller-admin-qa-final2/*.png` |
| Seller-scoped sidebar, active states, store card, seller support and logout affordances | Pass | All 9 final seller captures |
| Seller onboarding steps, KYC/business/bank/store setup, review status and support | Pass | `seller-admin-qa-final2/onboarding.png` |
| Dashboard KPIs, sales trend, quick actions, best sellers, recent orders and approval alerts | Pass | `seller-admin-qa-final2/dashboard.png` |
| Product catalog filters, product rows, add/edit panel, variants, compliance and approval state | Pass | `seller-admin-qa-final2/products.png` |
| Inventory KPIs, low-stock alerts, stock adjustment form, movement history and inventory table | Pass | `seller-admin-qa-final2/inventory.png` |
| Orders and shipping filters, expanded order, customer details, fulfilment timeline and Shiprocket action | Pass | `seller-admin-qa-final2/orders.png` |
| Returns evidence, response actions, platform decision boundary, status timeline and support notes | Pass | `seller-admin-qa-final2/returns.png` |
| Earnings ledger, commission/adjustment values, settlement timeline, payout account and help | Pass | `seller-admin-qa-final2/earnings.png` |
| Analytics filters, KPI cards, revenue/order trends, category/destination charts, top products and insights | Pass | `seller-admin-qa-final2/analytics.png` |
| Store profile/banner/description, public preview, pickup locations, business/bank/export settings and notifications | Pass | `seller-admin-qa-final2/settings.png` |

### Comparison notes

- Layout: the seller portal keeps the reference's 1055 x 1491 composition, left navigation, page intro band, multi-column content panels, trust strip and dark footer.
- Typography and color: matched the cream paper surface, maroon and terracotta actions, forest footer/sidebar accents, Playfair Display headings, DM Sans utility text, thin borders and rounded panels.
- Functionality: every primary seller responsibility has a dedicated route and fixture-backed action affordance; action links return visible status notices for product, inventory, order, return, support and settings workflows.
- Imagery: reused the repository's local craft and maker images for product, seller, onboarding and store settings surfaces; no generated imagery was needed.
- Responsive behavior: the seller sidebar has a mobile drawer state, content grids collapse at the existing responsive breakpoints, and forms/tables retain usable overflow behavior.

### Intentional remaining deviations

This is the repository's mock preview, so authentication/authorization, persistence, live inventory, Shiprocket, payment settlement, file storage, and seller staff permissions are not connected to a backend. Charts are deterministic fixture SVG/CSS visuals, and notification toggles are local Alpine preview state. These are presentation-layer placeholders for the PRD workflows, not production integrations.

Final captures: `seller-admin-qa-final2/*.png` at the native 1055 x 1491 viewport. The in-app browser was unavailable (`Browser is not available: iab`), so captures were made with the installed headless Chrome fallback and inspected with `view_image`.

# Customer visual QA

## Source of visual truth

All eleven reference screens reviewed from `Weevcrafts_Screens/customer/`:

- Home
- Categories
- Search/listing
- Product detail
- Maker store
- Wishlist
- Cart
- Checkout
- Orders
- Returns and refunds
- Account

## Implementation evidence

- Final captures: `customer-qa-final/*.png` at the native 1055 x 1491 reference viewport.
- Preview URL: `http://localhost:8090/`.
- Verified routes: `/`, `/categories`, `/search?q=handloom+sarees`, `/products/chanderi-royal`, `/makers/mithila-arts`, `/wishlist`, `/cart`, `/checkout`, `/orders`, `/returns`, `/account`.
- Interaction smoke checks: cart add, wishlist toggle, and customer image asset retrieval returned HTTP 200.

## Review results

| Area | Result | Evidence |
|---|---|---|
| Shared customer shell, utility bar, header, navigation and footer | Pass | All final customer captures |
| Home hero, categories, offers, trending products, makers and collections | Pass | `customer-qa-final/home.png` |
| Category browse sidebar, editorial feature, category cards and subcategories | Pass | `customer-qa-final/categories.png` |
| Search filters, handloom banner, four-column product grid and pagination | Pass | `customer-qa-final/listing.png` |
| Product gallery, seller, variants, quantity, delivery benefits, tabs and related products | Pass | `customer-qa-final/product.png` |
| Maker profile, follow/share actions, stats, products, story and why-shop section | Pass | `customer-qa-final/seller.png` |
| Six detailed wishlist cards, remove/move-to-cart controls and recommendations | Pass | `customer-qa-final/wishlist.png` |
| Seller-grouped cart lines, quantity controls, order summary, coupon and recommendations | Pass | `customer-qa-final/cart.png` |
| Address selection, seller delivery details, payment methods and order summary | Pass | `customer-qa-final/checkout.png` |
| Account navigation, order progress, seller order rows and past order history | Pass | `customer-qa-final/orders.png` |
| Return progress, pickup/refund status, active requests and support cards | Pass | `customer-qa-final/returns.png` |
| Account hero, stats, profile, addresses, payments, preferences, security and support | Pass | `customer-qa-final/account.png` |

## Comparison notes

- Layout: reviewed the full page flow of each reference against the 1055 x 1491 capture, including the first viewport and the visible lower-page sections.
- Typography and color: matched the paper background, maroon/terracotta actions, forest navigation and serif/sans hierarchy across all customer screens.
- Components: matched the reference card patterns for products, sellers, filters, order rows, return progress, summary panels and account cards.
- Copy: removed the mojibake currency/icon text from customer fixtures and retained the intended product, maker, delivery, payment and return responsibilities in plain readable copy.
- Imagery: replaced composite screenshot fragments with individual local craft and artisan assets so every product/gallery/card image is a valid standalone image.

Remaining deviation: this is still the repository's fixture-backed preview, so account persistence, real inventory, payment settlement, shipping, authorization and customer-service integrations are not connected to these routes.

The in-app browser was unavailable in this environment, so final screenshots were captured with the installed headless Chrome fallback and inspected with `view_image`. No P0, P1 or P2 visual issues remain for this customer preview.

# Supporter Portal visual QA

## Source of visual truth

Reviewed all four Supporter Portal references from `Weevcrafts_Screens/supporter portal/`, each at the native 1055 x 1491 viewport:

- Support Dashboard
- Cases
- Customers & Sellers
- Order / Transaction View

The implementation follows the Support Agent responsibilities in `Prd.md` and the least-privilege, masked-data and auditability boundaries in `Phases/Phase-11-seller-admin.md`.

## Implementation evidence

- Preview URL: `http://localhost:8090/support-portal`.
- Implemented routes: `/support-portal`, `/support-portal/cases`, `/support-portal/customers`, and `/support-portal/orders`.
- Support-specific templates and styling: `web/supportportal/` and `web/assets/css/support-portal.css`.
- Fixture data and route handling: `internal/web/handlers/support_portal_data.go` and `internal/web/handlers/mock_handler.go`.
- Final captures: `support-portal-qa-final/dashboard.png`, `cases.png`, `customers.png`, and `orders.png` at 1055 x 1491; `dashboard-mobile.png` was also checked at 390 x 844.

## Review results

| Area | Result | Evidence |
|---|---|---|
| Shared WeeVCrafts support shell, portal identity, search, notification and Aditi Rao profile | Pass | All final Supporter Portal captures |
| Role-specific sidebar and active navigation states | Pass | All four portal routes |
| Dashboard KPI cards, case queue, live conversations, issue breakdown, SLA, escalations, workload and macros | Pass | `support-portal-qa-final/dashboard.png` |
| Case queue, filters, customer/seller conversation, masked contact context, reply/internal-note composer and quick macros | Pass | `support-portal-qa-final/cases.png` |
| Customer and seller directory, case context, customer details, seller details, case timeline and support history | Pass | `support-portal-qa-final/customers.png` |
| Order, payment, shipping/tracking, return/refund, internal support, timeline and quick actions | Pass | `support-portal-qa-final/orders.png` |
| Support data minimization: masked customer contact and no passwords, CVV or full payment credentials | Pass | Route smoke check for `/support-portal/customers` |
| Cream/maroon/forest palette, serif/sans hierarchy, panel spacing, local craft imagery and responsive collapse | Pass | Reference comparison and desktop/mobile captures |

## Interaction and safety checks

- Support route smoke checks returned HTTP 200 for all four routes.
- Case reply notice is visible after the fixture action, and unsupported POST requests return HTTP 405.
- Customer contact is masked as `p***a.sharma@gmail.com` and `+91 ******5219` in the support context.
- Navigation and action links provide preview notices for case replies, escalation, notes, macros, invoice, tracking, refund and contact workflows.

## Intentional remaining deviations

This is the repository's fixture-backed mock preview. Live authentication and role enforcement, persistence, real ticket assignment, audit-log storage, customer/seller search, payment/refund execution, shipping-provider integrations and production notification delivery are not connected. Charts are deterministic CSS visuals, and the visible support actions are safe preview links that return notices rather than mutating production data. Support-facing customer information is intentionally masked and limited to the order/support context required by the PRD.

The in-app browser was unavailable (`Browser is not available: iab`), so final screenshots were captured with the installed headless Chrome fallback and inspected with `view_image`. No P0, P1 or P2 visual issues remain for this Supporter Portal preview.

final result: passed

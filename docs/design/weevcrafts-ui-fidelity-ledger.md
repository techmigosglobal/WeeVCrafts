# WeeVCrafts customer UI fidelity ledger

The customer reference screens in `Weevcrafts_Screens/customer/` are the visual
source of truth for this UI-first milestone. The approved mobile continuation
is `/home/vinay/.codex/generated_images/01a0940b-33fc-7a53-9511-acf421440e5d/exec-36873e1d-cf37-4a8e-9d1b-ccc834c4a041.png`.
The implementation keeps their editorial hierarchy while making the content
responsive and keyboard reachable.

| Area | Implemented direction |
| --- | --- |
| Copy and identity | WeeVCrafts wordmark, “Crafted by India. Cherished Everywhere.”, artisan-first copy, and mock-only preview notice. |
| Palette | Ivory paper, forest green, terracotta accents, warm tan surfaces, and muted ink text. |
| Typography | Serif display headings with compact uppercase sans labels and readable body copy. |
| Layout | Shared shipping strip/header/search/nav, responsive cards/grids, trust strip, and dark footer. |
| Imagery | Local WebP crops derived from supplied customer screens; source screenshots remain untouched. |
| Responsive behavior | Desktop multi-column layouts, tablet collapsed rails, mobile two-column products and horizontally scrollable rails/tabs. |
| Interactions | HTMX endpoints for cart/wishlist/checkout mutations; Alpine state only for menu, gallery, and product tabs. |

This milestone is intentionally fixture-backed. PostgreSQL, payments, shipping,
authentication, and seller/admin portals are deferred to the backend milestone.

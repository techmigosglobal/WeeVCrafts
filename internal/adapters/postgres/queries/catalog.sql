-- name: ListApprovedProducts :many
SELECT
    p.id,
    p.slug,
    b.name AS brand,
    p.name,
    p.description,
    c.slug AS category_slug,
    c.name AS category_name,
    p.price_cents,
    COALESCE(p.compare_at_cents, 0) AS compare_at_cents,
    (p.compare_at_cents IS NOT NULL)::boolean AS has_compare_at,
    p.rating::float8 AS rating,
    p.review_count,
    p.delivery_label,
    COALESCE(NULLIF((SELECT pm.public_url FROM product_media pm WHERE pm.product_id = p.id AND pm.status = 'ready' AND pm.visibility = 'public' ORDER BY pm.id LIMIT 1), ''), p.image_url) AS image_url,
    p.swatches
    ,pv.id AS variant_id
    ,COALESCE(i.available_quantity, 0) AS available_quantity
FROM products p
JOIN categories c ON c.slug = p.category_slug
JOIN brands b ON b.slug = p.brand_slug
JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active'
LEFT JOIN inventory_stock i ON i.variant_id = pv.id
WHERE p.status = 'approved'
  AND (@category_slug::text = '' OR p.category_slug = @category_slug)
ORDER BY p.featured_rank ASC, p.id ASC
LIMIT @result_limit;

-- name: GetApprovedProductBySlug :one
SELECT
    p.id,
    p.slug,
    b.name AS brand,
    p.name,
    p.description,
    c.slug AS category_slug,
    c.name AS category_name,
    p.price_cents,
    COALESCE(p.compare_at_cents, 0) AS compare_at_cents,
    (p.compare_at_cents IS NOT NULL)::boolean AS has_compare_at,
    p.rating::float8 AS rating,
    p.review_count,
    p.delivery_label,
    COALESCE(NULLIF((SELECT pm.public_url FROM product_media pm WHERE pm.product_id = p.id AND pm.status = 'ready' AND pm.visibility = 'public' ORDER BY pm.id LIMIT 1), ''), p.image_url) AS image_url,
    p.swatches
    ,pv.id AS variant_id
    ,COALESCE(i.available_quantity, 0) AS available_quantity
FROM products p
JOIN categories c ON c.slug = p.category_slug
JOIN brands b ON b.slug = p.brand_slug
JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active'
LEFT JOIN inventory_stock i ON i.variant_id = pv.id
WHERE p.status = 'approved' AND p.slug = @slug;

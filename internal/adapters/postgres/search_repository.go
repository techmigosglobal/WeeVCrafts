package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	domaincatalog "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

type ProductSearchRepository struct {
	pool *pgxpool.Pool
}

func NewProductSearchRepository(pool *pgxpool.Pool) *ProductSearchRepository {
	return &ProductSearchRepository{pool: pool}
}

func (r *ProductSearchRepository) Search(ctx context.Context, query, categorySlug string, limit, offset int) ([]domaincatalog.Product, int, error) {
	return r.SearchWithOptions(ctx, ports.ProductSearchOptions{Query: query, CategorySlug: categorySlug, Limit: limit, Offset: offset})
}

func (r *ProductSearchRepository) SearchWithOptions(ctx context.Context, options ports.ProductSearchOptions) ([]domaincatalog.Product, int, error) {
	query := options.Query
	categorySlug := options.CategorySlug
	limit := options.Limit
	offset := options.Offset
	if limit <= 0 || limit > 48 {
		limit = 24
	}
	if offset < 0 {
		offset = 0
	}
	orderBy := "p.featured_rank ASC, p.id ASC"
	switch options.Sort {
	case "price_asc":
		orderBy = "p.price_cents ASC, p.id ASC"
	case "price_desc":
		orderBy = "p.price_cents DESC, p.id DESC"
	case "newest":
		orderBy = "p.created_at DESC, p.id DESC"
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products p WHERE p.status = 'approved' AND ($1 = '' OR p.category_slug = $2) AND ($3 = '' OR p.name ILIKE '%' || $3 || '%' OR p.description ILIKE '%' || $3 || '%' OR p.slug ILIKE '%' || $3 || '%')`, categorySlug, categorySlug, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count product search: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.slug, b.name, p.name, p.description, p.category_slug, c.name,
		       p.price_cents, COALESCE(p.compare_at_cents, 0), (p.compare_at_cents IS NOT NULL),
		       p.rating::float8, p.review_count, p.delivery_label, COALESCE(NULLIF((SELECT pm.public_url FROM product_media pm WHERE pm.product_id = p.id AND pm.status = 'ready' AND pm.visibility = 'public' ORDER BY pm.id LIMIT 1), ''), p.image_url), p.swatches,
		       pv.id, COALESCE(i.available_quantity, 0)
		FROM products p
		JOIN categories c ON c.slug = p.category_slug
		JOIN brands b ON b.slug = p.brand_slug
		JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active'
		LEFT JOIN inventory_stock i ON i.variant_id = pv.id
		WHERE p.status = 'approved'
		  AND ($1 = '' OR p.category_slug = $1)
		  AND ($2 = '' OR p.name ILIKE '%' || $2 || '%' OR p.description ILIKE '%' || $2 || '%' OR p.slug ILIKE '%' || $2 || '%')
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4`, categorySlug, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()
	products := make([]domaincatalog.Product, 0)
	for rows.Next() {
		var product domaincatalog.Product
		if err := rows.Scan(&product.ID, &product.Slug, &product.Brand, &product.Name, &product.Description, &product.CategorySlug, &product.CategoryName, &product.PriceCents, &product.CompareAtCents, &product.HasCompareAt, &product.Rating, &product.ReviewCount, &product.DeliveryLabel, &product.ImageURL, &product.Swatches, &product.VariantID, &product.Available); err != nil {
			return nil, 0, fmt.Errorf("scan product search: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *ProductSearchRepository) SearchFacets(ctx context.Context, options ports.ProductSearchOptions) ([]ports.SearchFacet, error) {
	query := strings.TrimSpace(options.Query)
	category := strings.TrimSpace(options.CategorySlug)
	rows, err := r.pool.Query(ctx, `
		SELECT p.category_slug, c.name, COUNT(DISTINCT p.id)
		FROM products p
		JOIN categories c ON c.slug = p.category_slug
		WHERE p.status = 'approved'
		  AND ($1 = '' OR p.category_slug = $1)
		  AND ($2 = '' OR p.name ILIKE '%' || $2 || '%' OR p.description ILIKE '%' || $2 || '%' OR p.slug ILIKE '%' || $2 || '%')
		GROUP BY p.category_slug, c.name
		ORDER BY COUNT(DISTINCT p.id) DESC, p.category_slug
		LIMIT 24`, category, query)
	if err != nil {
		return nil, fmt.Errorf("search product facets: %w", err)
	}
	defer rows.Close()
	facets := make([]ports.SearchFacet, 0)
	for rows.Next() {
		var facet ports.SearchFacet
		if err := rows.Scan(&facet.Value, &facet.Label, &facet.Count); err != nil {
			return nil, fmt.Errorf("scan product facet: %w", err)
		}
		facets = append(facets, facet)
	}
	return facets, rows.Err()
}

func (r *ProductSearchRepository) SearchBySlugs(ctx context.Context, slugs []string) ([]domaincatalog.Product, error) {
	if len(slugs) == 0 {
		return []domaincatalog.Product{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.slug, b.name, p.name, p.description, p.category_slug, c.name,
		       p.price_cents, COALESCE(p.compare_at_cents, 0), (p.compare_at_cents IS NOT NULL),
		       p.rating::float8, p.review_count, p.delivery_label, COALESCE(NULLIF((SELECT pm.public_url FROM product_media pm WHERE pm.product_id = p.id AND pm.status = 'ready' AND pm.visibility = 'public' ORDER BY pm.id LIMIT 1), ''), p.image_url), p.swatches,
		       pv.id, COALESCE(i.available_quantity, 0)
		FROM products p
		JOIN categories c ON c.slug = p.category_slug
		JOIN brands b ON b.slug = p.brand_slug
		JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active'
		LEFT JOIN inventory_stock i ON i.variant_id = pv.id
		WHERE p.status = 'approved' AND p.slug = ANY($1::text[])
		ORDER BY array_position($1::text[], p.slug), pv.id`, slugs)
	if err != nil {
		return nil, fmt.Errorf("hydrate search products: %w", err)
	}
	defer rows.Close()
	products := make([]domaincatalog.Product, 0, len(slugs))
	for rows.Next() {
		var product domaincatalog.Product
		if err := rows.Scan(&product.ID, &product.Slug, &product.Brand, &product.Name, &product.Description, &product.CategorySlug, &product.CategoryName, &product.PriceCents, &product.CompareAtCents, &product.HasCompareAt, &product.Rating, &product.ReviewCount, &product.DeliveryLabel, &product.ImageURL, &product.Swatches, &product.VariantID, &product.Available); err != nil {
			return nil, fmt.Errorf("scan hydrated search product: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductSearchRepository) ListApprovedForIndex(ctx context.Context) ([]ports.SearchDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.slug, p.name, p.description, b.name, p.category_slug,
		       p.price_cents, COALESCE(i.available_quantity, 0)
		FROM products p
		JOIN brands b ON b.slug = p.brand_slug
		LEFT JOIN LATERAL (
			SELECT pv.id
			FROM product_variants pv
			WHERE pv.product_id = p.id AND pv.status = 'active'
			ORDER BY pv.id
			LIMIT 1
		) pv ON TRUE
		LEFT JOIN inventory_stock i ON i.variant_id = pv.id
		WHERE p.status = 'approved'
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	documents := make([]ports.SearchDocument, 0)
	for rows.Next() {
		var document ports.SearchDocument
		if err := rows.Scan(&document.ID, &document.Slug, &document.Name, &document.Description, &document.Brand, &document.CategorySlug, &document.PriceCents, &document.Available); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

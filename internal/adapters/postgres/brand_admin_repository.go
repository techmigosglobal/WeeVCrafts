package postgres

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (r *CommerceRepository) ListBrandDirectory(ctx context.Context, actorID int64) ([]domaincommerce.AdminBrand, error) {
	if err := r.requireMarketplaceAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT b.slug, b.name, COUNT(DISTINCT p.id), COUNT(DISTINCT p.category_slug)
		FROM brands b
		LEFT JOIN products p ON p.brand_slug = b.slug AND p.status = 'approved'
		GROUP BY b.slug, b.name
		ORDER BY b.name, b.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	brands := make([]domaincommerce.AdminBrand, 0)
	for rows.Next() {
		var brand domaincommerce.AdminBrand
		if err := rows.Scan(&brand.Slug, &brand.Name, &brand.ProductCount, &brand.CategoryCount); err != nil {
			return nil, err
		}
		brands = append(brands, brand)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return brands, nil
}

var _ ports.BrandDirectoryRepository = (*CommerceRepository)(nil)

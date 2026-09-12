package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/adapters/postgres/generated"
	"github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

type ProductRepository struct {
	queries *generated.Queries
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{queries: generated.New(pool)}
}

func NewProductRepositoryFromQueries(queries *generated.Queries) *ProductRepository {
	return &ProductRepository{queries: queries}
}

func (r *ProductRepository) ListApproved(ctx context.Context, categorySlug string, limit int) ([]catalog.Product, error) {
	rows, err := r.queries.ListApprovedProducts(ctx, generated.ListApprovedProductsParams{
		CategorySlug: categorySlug,
		ResultLimit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list approved products: %w", err)
	}

	products := make([]catalog.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, catalog.Product{
			ID:             row.ID,
			VariantID:      row.VariantID,
			Slug:           row.Slug,
			Brand:          row.Brand,
			Name:           row.Name,
			Description:    row.Description,
			CategorySlug:   row.CategorySlug,
			CategoryName:   row.CategoryName,
			PriceCents:     row.PriceCents,
			CompareAtCents: row.CompareAtCents,
			HasCompareAt:   row.HasCompareAt,
			Rating:         row.Rating,
			ReviewCount:    int(row.ReviewCount),
			DeliveryLabel:  row.DeliveryLabel,
			ImageURL:       row.ImageUrl,
			Swatches:       row.Swatches,
			Available:      int(row.AvailableQuantity),
		})
	}
	return products, nil
}

func (r *ProductRepository) GetApprovedBySlug(ctx context.Context, slug string) (catalog.Product, error) {
	row, err := r.queries.GetApprovedProductBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Product{}, fmt.Errorf("%w: %s", ports.ErrNotFound, slug)
	}
	if err != nil {
		return catalog.Product{}, fmt.Errorf("get approved product: %w", err)
	}
	return catalog.Product{
		ID:             row.ID,
		VariantID:      row.VariantID,
		Slug:           row.Slug,
		Brand:          row.Brand,
		Name:           row.Name,
		Description:    row.Description,
		CategorySlug:   row.CategorySlug,
		CategoryName:   row.CategoryName,
		PriceCents:     row.PriceCents,
		CompareAtCents: row.CompareAtCents,
		HasCompareAt:   row.HasCompareAt,
		Rating:         row.Rating,
		ReviewCount:    int(row.ReviewCount),
		DeliveryLabel:  row.DeliveryLabel,
		ImageURL:       row.ImageUrl,
		Swatches:       row.Swatches,
		Available:      int(row.AvailableQuantity),
	}, nil
}

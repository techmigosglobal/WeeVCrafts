package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestProductSearchOptionsUsesStablePriceOrdering(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	brandSlug := "search-brand-" + suffix
	categorySlug := "search-category-" + suffix
	if _, err := pool.Exec(ctx, `INSERT INTO brands (slug, name) VALUES ($1, 'Search Fixture Brand')`, brandSlug); err != nil {
		t.Fatalf("insert search brand: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO categories (slug, name) VALUES ($1, 'Search Fixture Category')`, categorySlug); err != nil {
		t.Fatalf("insert search category: %v", err)
	}
	var ownerID, sellerID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Search Fixture Owner', 'active') RETURNING id`, "search-owner-"+suffix+"@example.invalid").Scan(&ownerID); err != nil {
		t.Fatalf("insert search owner: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (owner_user_id, display_name, status) VALUES ($1, 'Search Fixture Seller', 'active') RETURNING id`, ownerID).Scan(&sellerID); err != nil {
		t.Fatalf("insert search seller: %v", err)
	}
	productIDs := make([]int64, 0, 2)
	for index, price := range []int{900, 1900} {
		var productID, variantID int64
		slug := fmt.Sprintf("search-product-%s-%d", suffix, index)
		if err := pool.QueryRow(ctx, `INSERT INTO products (slug, brand_slug, category_slug, name, description, price_cents, seller_id, status) VALUES ($1, $2, $3, $4, 'Search fixture', $5, $6, 'approved') RETURNING id`, slug, brandSlug, categorySlug, fmt.Sprintf("Search Product %d", index), price, sellerID).Scan(&productID); err != nil {
			t.Fatalf("insert search product: %v", err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO product_variants (product_id, sku, price_cents) VALUES ($1, $2, $3) RETURNING id`, productID, fmt.Sprintf("SEARCH-%s-%d", suffix, index), price).Scan(&variantID); err != nil {
			t.Fatalf("insert search variant: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO inventory_stock (variant_id, available_quantity) VALUES ($1, 5)`, variantID); err != nil {
			t.Fatalf("insert search stock: %v", err)
		}
		productIDs = append(productIDs, productID)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM products WHERE id = ANY($1)`, productIDs)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM sellers WHERE id = $1`, sellerID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM users WHERE id = $1`, ownerID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM brands WHERE slug = $1`, brandSlug)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM categories WHERE slug = $1`, categorySlug)
	})

	products, total, err := NewProductSearchRepository(pool).SearchWithOptions(ctx, ports.ProductSearchOptions{CategorySlug: categorySlug, Sort: "price_desc", Limit: 2})
	if err != nil {
		t.Fatalf("search with sort: %v", err)
	}
	if total != 2 || len(products) != 2 || products[0].PriceCents != 1900 || products[1].PriceCents != 900 {
		t.Fatalf("unexpected sorted products: total=%d products=%+v", total, products)
	}
	facets, err := NewProductSearchRepository(pool).SearchFacets(ctx, ports.ProductSearchOptions{Query: "Search fixture"})
	if err != nil || len(facets) == 0 || facets[0].Value != categorySlug || facets[0].Count != 2 {
		t.Fatalf("unexpected category facets: facets=%+v err=%v", facets, err)
	}
}

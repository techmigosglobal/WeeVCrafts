package ports

import (
	"context"
	"time"

	"github.com/wecratfs/commerce/internal/domain/catalog"
)

type ProductRepository interface {
	ListApproved(ctx context.Context, categorySlug string, limit int) ([]catalog.Product, error)
	GetApprovedBySlug(ctx context.Context, slug string) (catalog.Product, error)
}

type ProductSearchRepository interface {
	Search(ctx context.Context, query, categorySlug string, limit, offset int) ([]catalog.Product, int, error)
	SearchBySlugs(ctx context.Context, slugs []string) ([]catalog.Product, error)
}

// ProductSearchOptionsRepository is an additive capability for callers that
// need stable, shareable search ordering. ProductSearchRepository remains the
// compatibility seam for simple catalogue fixtures.
type ProductSearchOptionsRepository interface {
	SearchWithOptions(ctx context.Context, options ProductSearchOptions) ([]catalog.Product, int, error)
}

// ProductSearchFacetsRepository is an additive read capability. Facets are
// advisory navigation data and never replace PostgreSQL product hydration.
type ProductSearchFacetsRepository interface {
	SearchFacets(ctx context.Context, options ProductSearchOptions) ([]SearchFacet, error)
}

type ProductSearchOptions struct {
	Query        string
	CategorySlug string
	Sort         string
	Limit        int
	Offset       int
}

type SearchFacet struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type SearchObservation struct {
	Backend     string
	CacheHit    bool
	Error       bool
	Duration    time.Duration
	ResultCount int
}

type SearchObserver interface {
	ObserveSearch(ctx context.Context, observation SearchObservation)
}

type ApprovedIndexSource interface {
	ListApprovedForIndex(ctx context.Context) ([]SearchDocument, error)
}

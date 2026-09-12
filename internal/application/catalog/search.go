package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	domaincatalog "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

type SearchService struct {
	repository ports.ProductSearchRepository
	cache      ports.Cache
	engine     ports.SearchEngine
	observer   ports.SearchObserver
	flight     searchFlight
}

const (
	SearchSortRelevance = ""
	SearchSortPriceAsc  = "price_asc"
	SearchSortPriceDesc = "price_desc"
	SearchSortNewest    = "newest"
)

func NewSearchService(repository ports.ProductSearchRepository, cache ports.Cache) *SearchService {
	return NewSearchServiceWithEngine(repository, cache, nil)
}

func NewSearchServiceWithEngine(repository ports.ProductSearchRepository, cache ports.Cache, engine ports.SearchEngine) *SearchService {
	return &SearchService{repository: repository, cache: cache, engine: engine}
}

func NewSearchServiceWithEngineAndObserver(repository ports.ProductSearchRepository, cache ports.Cache, engine ports.SearchEngine, observer ports.SearchObserver) *SearchService {
	return &SearchService{repository: repository, cache: cache, engine: engine, observer: observer}
}

func (s *SearchService) Search(ctx context.Context, query, category string, limit, offset int) ([]domaincatalog.Product, int, error) {
	return s.SearchWithOptions(ctx, query, category, SearchSortRelevance, limit, offset)
}

func (s *SearchService) SearchWithOptions(ctx context.Context, query, category, sort string, limit, offset int) ([]domaincatalog.Product, int, error) {
	products, total, _, err := s.SearchWithOptionsAndFacets(ctx, query, category, sort, limit, offset)
	return products, total, err
}

func (s *SearchService) SearchWithOptionsAndFacets(ctx context.Context, query, category, sort string, limit, offset int) ([]domaincatalog.Product, int, []ports.SearchFacet, error) {
	started := time.Now()
	if limit <= 0 || limit > 48 {
		limit = 24
	}
	if offset < 0 || offset > 10000 {
		offset = 0
	}
	sort = normalizeSearchSort(sort)
	cacheKey := fmt.Sprintf("catalog:search:%s:%s:%s:%d:%d", query, category, sort, limit, offset)
	if s.cache != nil {
		if value, err := s.cache.Get(ctx, cacheKey); err == nil {
			var cached searchResult
			if json.Unmarshal(value, &cached) == nil {
				s.observe(ctx, ports.SearchObservation{Backend: "cache", CacheHit: true, Duration: time.Since(started), ResultCount: len(cached.Products)})
				return cached.Products, cached.Total, cached.Facets, nil
			}
		}
	}
	call, leader := s.flight.begin(cacheKey)
	if !leader {
		select {
		case <-call.done:
			return call.result.Products, call.result.Total, call.result.Facets, call.err
		case <-ctx.Done():
			return nil, 0, nil, ctx.Err()
		}
	}
	result, err := s.searchBackend(ctx, query, category, sort, limit, offset, cacheKey)
	s.flight.finish(cacheKey, call, result, err)
	s.observe(ctx, ports.SearchObservation{Backend: result.Backend, Error: err != nil, Duration: time.Since(started), ResultCount: len(result.Products)})
	return result.Products, result.Total, result.Facets, err
}

func (s *SearchService) searchBackend(ctx context.Context, query, category, sort string, limit, offset int, cacheKey string) (searchResult, error) {
	if s.engine != nil {
		indexed, engineErr := s.engine.Search(ctx, ports.SearchRequest{Query: query, CategorySlug: category, Sort: sort, Limit: limit, Offset: offset})
		if engineErr == nil {
			indexed.ProductSlugs = normalizeSearchSlugs(indexed.ProductSlugs)
			products, hydrateErr := s.repository.SearchBySlugs(ctx, indexed.ProductSlugs)
			if hydrateErr == nil {
				facets := indexed.Facets
				if len(facets) == 0 {
					facets = s.searchFacets(ctx, query, category, sort, limit, offset)
				}
				result := searchResult{Products: products, Total: indexed.Total, Facets: facets, Backend: "index"}
				if s.cache != nil {
					if value, marshalErr := json.Marshal(result); marshalErr == nil {
						_ = s.cache.Set(ctx, cacheKey, value, 30)
					}
				}
				return result, nil
			}
		}
	}
	var products []domaincatalog.Product
	var total int
	var err error
	if advanced, ok := s.repository.(ports.ProductSearchOptionsRepository); ok {
		products, total, err = advanced.SearchWithOptions(ctx, ports.ProductSearchOptions{Query: query, CategorySlug: category, Sort: sort, Limit: limit, Offset: offset})
	} else {
		products, total, err = s.repository.Search(ctx, query, category, limit, offset)
	}
	if err != nil {
		return searchResult{}, err
	}
	result := searchResult{Products: products, Total: total, Facets: s.searchFacets(ctx, query, category, sort, limit, offset), Backend: "postgresql"}
	if s.cache != nil {
		if value, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, value, 30)
		}
	}
	return result, nil
}

func (s *SearchService) searchFacets(ctx context.Context, query, category, sort string, limit, offset int) []ports.SearchFacet {
	provider, ok := s.repository.(ports.ProductSearchFacetsRepository)
	if !ok {
		return nil
	}
	facets, err := provider.SearchFacets(ctx, ports.ProductSearchOptions{Query: query, CategorySlug: category, Sort: sort, Limit: limit, Offset: offset})
	if err != nil {
		return nil
	}
	return facets
}

func (s *SearchService) observe(ctx context.Context, observation ports.SearchObservation) {
	if s.observer != nil {
		s.observer.ObserveSearch(ctx, observation)
	}
}

func normalizeSearchSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case SearchSortPriceAsc:
		return SearchSortPriceAsc
	case SearchSortPriceDesc:
		return SearchSortPriceDesc
	case SearchSortNewest:
		return SearchSortNewest
	default:
		return SearchSortRelevance
	}
}

type searchCall struct {
	done   chan struct{}
	result searchResult
	err    error
}

type searchFlight struct {
	mu    sync.Mutex
	calls map[string]*searchCall
}

func (f *searchFlight) begin(key string) (*searchCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls == nil {
		f.calls = make(map[string]*searchCall)
	}
	if call, exists := f.calls[key]; exists {
		return call, false
	}
	call := &searchCall{done: make(chan struct{})}
	f.calls[key] = call
	return call, true
}

func (f *searchFlight) finish(key string, call *searchCall, result searchResult, err error) {
	f.mu.Lock()
	call.result = result
	call.err = err
	delete(f.calls, key)
	close(call.done)
	f.mu.Unlock()
}

func normalizeSearchSlugs(slugs []string) []string {
	result := make([]string, 0, len(slugs))
	seen := make(map[string]struct{}, len(slugs))
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}
		result = append(result, slug)
	}
	return result
}

type searchResult struct {
	Products []domaincatalog.Product `json:"products"`
	Total    int                     `json:"total"`
	Facets   []ports.SearchFacet     `json:"facets,omitempty"`
	Backend  string                  `json:"-"`
}

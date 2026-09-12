package catalog

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

type searchRepository struct {
	searchProducts []domain.Product
	searchTotal    int
	hydrated       []domain.Product
	searchCalls    int
	hydrateCalls   int
	facets         []ports.SearchFacet
}

func (r *searchRepository) Search(context.Context, string, string, int, int) ([]domain.Product, int, error) {
	r.searchCalls++
	return r.searchProducts, r.searchTotal, nil
}

func (r *searchRepository) SearchBySlugs(_ context.Context, slugs []string) ([]domain.Product, error) {
	r.hydrateCalls++
	if len(slugs) == 0 {
		return []domain.Product{}, nil
	}
	return r.hydrated, nil
}

func (r *searchRepository) SearchFacets(context.Context, ports.ProductSearchOptions) ([]ports.SearchFacet, error) {
	return r.facets, nil
}

type searchObserver struct {
	observations []ports.SearchObservation
}

func (o *searchObserver) ObserveSearch(_ context.Context, observation ports.SearchObservation) {
	o.observations = append(o.observations, observation)
}

type searchEngine struct {
	result  ports.SearchResult
	err     error
	calls   int
	request ports.SearchRequest
}

func (e *searchEngine) Search(_ context.Context, request ports.SearchRequest) (ports.SearchResult, error) {
	e.calls++
	e.request = request
	return e.result, e.err
}

type searchCache struct {
	mu      sync.Mutex
	value   []byte
	getErr  error
	setCall int
}

func (c *searchCache) Get(context.Context, string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.value...), c.getErr
}
func (c *searchCache) Set(_ context.Context, _ string, value []byte, _ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = append([]byte(nil), value...)
	c.getErr = nil
	c.setCall++
	return nil
}
func (c *searchCache) Delete(context.Context, string) error { return nil }

func TestSearchServiceHydratesIndexedSlugsAndPreservesTotal(t *testing.T) {
	repository := &searchRepository{hydrated: []domain.Product{{Slug: "soap"}}}
	engine := &searchEngine{result: ports.SearchResult{ProductSlugs: []string{"soap", "soap", ""}, Total: 7}}
	service := NewSearchServiceWithEngine(repository, nil, engine)

	products, total, err := service.Search(context.Background(), "sope", "natural-care", 24, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 7 || len(products) != 1 || products[0].Slug != "soap" {
		t.Fatalf("unexpected hydrated search result: total=%d products=%+v", total, products)
	}
	if engine.calls != 1 || repository.hydrateCalls != 1 || repository.searchCalls != 0 {
		t.Fatalf("expected indexed path only, engine=%d hydrate=%d postgres=%d", engine.calls, repository.hydrateCalls, repository.searchCalls)
	}
}

func TestSearchServiceForwardsSortToIndexAndFallbackContract(t *testing.T) {
	repository := &searchRepository{hydrated: []domain.Product{{Slug: "soap"}}}
	engine := &searchEngine{result: ports.SearchResult{ProductSlugs: []string{"soap"}, Total: 1}}
	service := NewSearchServiceWithEngine(repository, nil, engine)

	products, total, err := service.SearchWithOptions(context.Background(), "soap", "natural-care", SearchSortPriceDesc, 12, 24)
	if err != nil || total != 1 || len(products) != 1 {
		t.Fatalf("search options failed: total=%d products=%+v err=%v", total, products, err)
	}
	if engine.request.Sort != SearchSortPriceDesc || engine.request.Limit != 12 || engine.request.Offset != 24 {
		t.Fatalf("search options were not forwarded: %+v", engine.request)
	}
}

func TestSearchServiceReturnsFacetsAndObservesBackend(t *testing.T) {
	repository := &searchRepository{hydrated: []domain.Product{{Slug: "soap"}}, facets: []ports.SearchFacet{{Value: "natural-care", Label: "Natural care", Count: 3}}}
	observer := &searchObserver{}
	engine := &searchEngine{result: ports.SearchResult{ProductSlugs: []string{"soap"}, Total: 3}}
	service := NewSearchServiceWithEngineAndObserver(repository, nil, engine, observer)

	_, total, facets, err := service.SearchWithOptionsAndFacets(context.Background(), "soap", "", "", 24, 0)
	if err != nil || total != 3 || len(facets) != 1 || facets[0].Value != "natural-care" {
		t.Fatalf("unexpected search facets: total=%d facets=%+v err=%v", total, facets, err)
	}
	if len(observer.observations) != 1 || observer.observations[0].Backend != "index" || observer.observations[0].Error {
		t.Fatalf("unexpected search observation: %+v", observer.observations)
	}
}

func TestSearchServiceFallsBackToPostgresWhenIndexUnavailable(t *testing.T) {
	repository := &searchRepository{searchProducts: []domain.Product{{Slug: "fallback"}}, searchTotal: 1}
	engine := &searchEngine{err: errors.New("index unavailable")}
	service := NewSearchServiceWithEngine(repository, nil, engine)

	products, total, err := service.Search(context.Background(), "soap", "", 24, 0)
	if err != nil {
		t.Fatalf("fallback search: %v", err)
	}
	if total != 1 || len(products) != 1 || products[0].Slug != "fallback" || repository.searchCalls != 1 {
		t.Fatalf("unexpected fallback result: total=%d products=%+v postgres=%d", total, products, repository.searchCalls)
	}
}

func TestSearchServiceUsesDisposableCacheBeforeBackends(t *testing.T) {
	cache := &searchCache{value: []byte(`{"products":[{"slug":"cached"}],"total":1}`)}
	repository := &searchRepository{}
	engine := &searchEngine{err: errors.New("should not be called")}
	service := NewSearchServiceWithEngine(repository, cache, engine)

	products, total, err := service.Search(context.Background(), "soap", "", 24, 0)
	if err != nil {
		t.Fatalf("cached search: %v", err)
	}
	if total != 1 || len(products) != 1 || products[0].Slug != "cached" || repository.searchCalls != 0 || engine.calls != 0 || cache.setCall != 0 {
		t.Fatalf("unexpected cached result: total=%d products=%+v postgres=%d engine=%d sets=%d", total, products, repository.searchCalls, engine.calls, cache.setCall)
	}
}

func TestSearchServiceDeduplicatesConcurrentCacheMisses(t *testing.T) {
	repository := &blockingSearchRepository{started: make(chan struct{}), release: make(chan struct{})}
	service := NewSearchService(repository, &searchCache{getErr: ports.ErrCacheMiss})
	const callers = 12
	results := make(chan error, callers)
	var group sync.WaitGroup
	for i := 0; i < callers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, _, err := service.Search(context.Background(), "soap", "", 24, 0)
			results <- err
		}()
	}
	<-repository.started
	close(repository.release)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("deduplicated search failed: %v", err)
		}
	}
	if calls := repository.calls.Load(); calls != 1 {
		t.Fatalf("expected one backend search on concurrent cache miss, got %d", calls)
	}
}

type blockingSearchRepository struct {
	started     chan struct{}
	startedOnce sync.Once
	release     chan struct{}
	calls       atomic.Int32
}

func (r *blockingSearchRepository) Search(context.Context, string, string, int, int) ([]domain.Product, int, error) {
	r.calls.Add(1)
	r.startedOnce.Do(func() { close(r.started) })
	<-r.release
	return []domain.Product{{Slug: "deduplicated"}}, 1, nil
}

func (r *blockingSearchRepository) SearchBySlugs(context.Context, []string) ([]domain.Product, error) {
	return []domain.Product{}, nil
}

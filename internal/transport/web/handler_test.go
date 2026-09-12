package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	domain "github.com/wecratfs/commerce/internal/domain/catalog"
)

type repository struct {
	products []domain.Product
}

func (r repository) Search(context.Context, string, string, int, int) ([]domain.Product, int, error) {
	return r.products, len(r.products), nil
}

func (r repository) SearchBySlugs(context.Context, []string) ([]domain.Product, error) {
	return r.products, nil
}

func (r repository) ListApproved(context.Context, string, int) ([]domain.Product, error) {
	return r.products, nil
}

func (r repository) GetApprovedBySlug(context.Context, string) (domain.Product, error) {
	if len(r.products) == 0 {
		return domain.Product{}, applicationcatalog.ErrProductNotFound
	}
	return r.products[0], nil
}

func TestHomeShowsTruthfulEmptyState(t *testing.T) {
	handler, err := NewHandler(applicationcatalog.NewService(repository{}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"WeCratfs", "Closer to nature", "No products published yet", "PostgreSQL"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in body", expected)
		}
	}
}

func TestProductPageRendersCatalogueRecord(t *testing.T) {
	product := domain.Product{
		Slug:          "handmade-bowl",
		Brand:         "A real maker",
		Name:          "Handmade Bowl",
		CategoryName:  "Slow living",
		PriceCents:    1200,
		Description:   "A product description from PostgreSQL.",
		DeliveryLabel: "Dispatches in 2–4 days",
	}
	handler, err := NewHandler(applicationcatalog.NewService(repository{products: []domain.Product{product}}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/products/handmade-bowl", nil))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "A product description from PostgreSQL.") {
		t.Fatalf("unexpected product response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestUnknownProductIsNotFound(t *testing.T) {
	handler, err := NewHandler(applicationcatalog.NewService(repository{}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/products/missing", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestPolicyRouteIsReachableFromFooterContract(t *testing.T) {
	handler, err := NewHandler(applicationcatalog.NewService(repository{}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/shipping", nil))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Shipping details") {
		t.Fatalf("unexpected policy response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestStaticAssetsArePubliclyCacheable(t *testing.T) {
	handler, err := NewHandler(applicationcatalog.NewService(repository{}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected static asset status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected cache header: %q", got)
	}
}

func TestSearchPreservesFilterSortAndPageStateInHTML(t *testing.T) {
	products := []domain.Product{{Slug: "handmade-soap", Name: "Handmade Soap"}}
	searchService := applicationcatalog.NewSearchService(repository{products: products}, nil)
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{products: products}), nil, nil, nil, nil, nil, searchService, nil, false)
	if err != nil {
		t.Fatalf("create full handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search?q=soap&category=natural-care&sort=price_desc&page=2", nil))
	body := recorder.Body.String()
	for _, expected := range []string{`name="q" value="soap"`, `name="category" value="natural-care"`, `value="price_desc" selected`, "Page 2", "Previous"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected search state %q in response: %s", expected, body)
		}
	}
}

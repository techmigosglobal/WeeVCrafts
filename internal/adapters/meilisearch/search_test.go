package meilisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestRebuildCreatesClearsAndPopulatesIndex(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/indexes":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && r.URL.Path == "/indexes/products/settings/filterable-attributes":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPut && r.URL.Path == "/indexes/products/settings/sortable-attributes":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodDelete && r.URL.Path == "/indexes/products/documents":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/products/documents":
			var documents []ports.SearchDocument
			if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
				t.Fatalf("decode documents: %v", err)
			}
			if len(documents) != 1 || documents[0].Slug != "soap" {
				t.Fatalf("unexpected documents: %+v", documents)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := NewSearchEngine(server.URL).Rebuild(context.Background(), []ports.SearchDocument{{Slug: "soap", Name: "Soap"}})
	if err != nil {
		t.Fatalf("rebuild index: %v", err)
	}
	if len(calls) != 5 || calls[0] != "POST /indexes" || calls[1] != "PUT /indexes/products/settings/filterable-attributes" || calls[2] != "PUT /indexes/products/settings/sortable-attributes" || calls[3] != "DELETE /indexes/products/documents" || calls[4] != "POST /indexes/products/documents" {
		t.Fatalf("unexpected rebuild calls: %v", calls)
	}
}

func TestSearchSendsValidatedFilterAndSort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Filter string   `json:"filter"`
			Sort   []string `json:"sort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode search payload: %v", err)
		}
		if payload.Filter != "" || len(payload.Sort) != 1 || payload.Sort[0] != "price_cents:desc" {
			t.Fatalf("unexpected search payload: %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"hits": []any{}, "estimatedTotalHits": 0})
	}))
	defer server.Close()

	result, err := NewSearchEngine(server.URL).Search(context.Background(), ports.SearchRequest{Query: "soap", CategorySlug: "", Sort: "price_desc", Limit: 24, Offset: 0})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 0 || len(result.ProductSlugs) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSearchMapsCategoryFacetDistribution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Facets []string `json:"facets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode facet payload: %v", err)
		}
		if len(payload.Facets) != 1 || payload.Facets[0] != "category_slug" {
			t.Fatalf("category facets were not requested: %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": []map[string]any{{"slug": "soap"}}, "estimatedTotalHits": 1,
			"facetDistribution": map[string]map[string]int{"category_slug": {"natural-care": 4}},
		})
	}))
	defer server.Close()

	result, err := NewSearchEngine(server.URL).Search(context.Background(), ports.SearchRequest{Query: "soap", Limit: 24})
	if err != nil || len(result.Facets) != 1 || result.Facets[0].Value != "natural-care" || result.Facets[0].Count != 4 {
		t.Fatalf("unexpected facet result: %+v err=%v", result, err)
	}
}

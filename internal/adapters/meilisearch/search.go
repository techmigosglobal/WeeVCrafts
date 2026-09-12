package meilisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

type SearchEngine struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewSearchEngine(address string) *SearchEngine {
	return NewSearchEngineWithKey(address, "")
}

func NewSearchEngineWithKey(address, apiKey string) *SearchEngine {
	baseURL := strings.TrimRight(address, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &SearchEngine{baseURL: baseURL, apiKey: apiKey, client: &http.Client{Timeout: 2 * time.Second}}
}

func (s *SearchEngine) Search(ctx context.Context, request ports.SearchRequest) (ports.SearchResult, error) {
	payloadValues := map[string]any{"q": request.Query, "limit": request.Limit, "offset": request.Offset, "filter": categoryFilter(request.CategorySlug), "facets": []string{"category_slug"}}
	if sort := meiliSort(request.Sort); sort != "" {
		payloadValues["sort"] = []string{sort}
	}
	payload, err := json.Marshal(payloadValues)
	if err != nil {
		return ports.SearchResult{}, err
	}
	endpoint := s.baseURL + "/indexes/products/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ports.SearchResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return ports.SearchResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ports.SearchResult{}, fmt.Errorf("meilisearch returned status %d", response.StatusCode)
	}
	var result struct {
		Hits []struct {
			Slug string `json:"slug"`
		} `json:"hits"`
		Total             int                       `json:"estimatedTotalHits"`
		FacetDistribution map[string]map[string]int `json:"facetDistribution"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return ports.SearchResult{}, err
	}
	slugs := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		slugs = append(slugs, hit.Slug)
	}
	facets := make([]ports.SearchFacet, 0)
	for value, count := range result.FacetDistribution["category_slug"] {
		facets = append(facets, ports.SearchFacet{Value: value, Label: value, Count: count})
	}
	sort.Slice(facets, func(i, j int) bool {
		if facets[i].Count == facets[j].Count {
			return facets[i].Value < facets[j].Value
		}
		return facets[i].Count > facets[j].Count
	})
	return ports.SearchResult{ProductSlugs: slugs, Total: result.Total, Facets: facets}, nil
}

func (s *SearchEngine) Rebuild(ctx context.Context, documents []ports.SearchDocument) error {
	if err := s.ensureIndex(ctx); err != nil {
		return err
	}
	if err := s.request(ctx, http.MethodPut, "/indexes/products/settings/filterable-attributes", []byte(`["category_slug"]`), nil); err != nil {
		return err
	}
	if err := s.request(ctx, http.MethodPut, "/indexes/products/settings/sortable-attributes", []byte(`["price_cents","id"]`), nil); err != nil {
		return err
	}
	if err := s.request(ctx, http.MethodDelete, "/indexes/products/documents", nil, nil); err != nil {
		return err
	}
	body, err := json.Marshal(documents)
	if err != nil {
		return err
	}
	return s.request(ctx, http.MethodPost, "/indexes/products/documents?primaryKey=slug", body, nil)
}

func (s *SearchEngine) ensureIndex(ctx context.Context) error {
	body := []byte(`{"uid":"products","primaryKey":"slug"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/indexes", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict || response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusAccepted {
		return nil
	}
	return fmt.Errorf("meilisearch index creation returned status %d", response.StatusCode)
}

func (s *SearchEngine) request(ctx context.Context, method, path string, body []byte, destination any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("meilisearch returned status %d", response.StatusCode)
	}
	if destination == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return nil
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(destination)
}

func categoryFilter(category string) string {
	if category == "" {
		return ""
	}
	return "category_slug = '" + strings.ReplaceAll(category, "'", "") + "'"
}

func meiliSort(sort string) string {
	switch sort {
	case "price_asc":
		return "price_cents:asc"
	case "price_desc":
		return "price_cents:desc"
	case "newest":
		return "id:desc"
	default:
		return ""
	}
}

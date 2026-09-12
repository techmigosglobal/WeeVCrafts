package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
)

type repository struct {
	products []domain.Product
}

func (r repository) ListApproved(context.Context, string, int) ([]domain.Product, error) {
	return r.products, nil
}

func (r repository) GetApprovedBySlug(context.Context, string) (domain.Product, error) {
	return r.products[0], nil
}

func TestProductsReturnsJSONWithoutTemplateConcerns(t *testing.T) {
	service := applicationcatalog.NewService(repository{products: []domain.Product{{Slug: "handmade-bowl", Name: "Handmade Bowl"}}})
	handler := NewHandler(service)
	recorder := httptest.NewRecorder()
	handler.Products(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/products", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"source":"postgresql"`) || strings.Contains(body, "<html") || strings.Contains(body, "hx-") {
		t.Fatalf("unexpected API body: %s", body)
	}
}

func TestProductsRejectsNonGet(t *testing.T) {
	service := applicationcatalog.NewService(repository{products: []domain.Product{{Slug: "handmade-bowl"}}})
	recorder := httptest.NewRecorder()
	NewHandler(service).Products(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/products", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestErrorsCarryRequestIDWhenProvidedByMiddleware(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("X-Request-ID", "req-test-123")
	NewHandler(&applicationcatalog.Service{}).Products(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/products", nil))

	if !strings.Contains(recorder.Body.String(), `"request_id":"req-test-123"`) {
		t.Fatalf("expected request ID in API error: %s", recorder.Body.String())
	}
}

func TestIdentityResponsesUseStableJSONNames(t *testing.T) {
	payload, err := json.Marshal(domainidentity.User{ID: 7, Email: "maker@example.invalid", DisplayName: "Maker", Status: "active"})
	if err != nil {
		t.Fatalf("marshal identity response: %v", err)
	}
	body := string(payload)
	if !strings.Contains(body, `"id":7`) || !strings.Contains(body, `"display_name":"Maker"`) || strings.Contains(body, `"ID"`) {
		t.Fatalf("identity JSON contract leaked Go casing: %s", body)
	}
}

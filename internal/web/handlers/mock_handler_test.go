package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockCustomerRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/", "/categories", "/products", "/search?q=saree", "/products/chanderi-royal", "/makers/mithila-arts", "/wishlist", "/cart", "/checkout", "/orders", "/orders/%23WC2504267819", "/returns", "/account"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			if !strings.Contains(res.Body.String(), "WeeVCrafts") {
				t.Fatal("response does not contain public brand")
			}
		})
	}
}

func TestMockSessionIsolationAndMutations(t *testing.T) {
	h := NewMockHandler()
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/cart", nil))
	cookie := first.Result().Cookies()[0]

	add := httptest.NewRequest(http.MethodPost, "/ui/cart/blue-tote", nil)
	add.Header.Set("HX-Request", "true")
	add.AddCookie(cookie)
	addResponse := httptest.NewRecorder()
	h.ServeHTTP(addResponse, add)
	if addResponse.Code != http.StatusOK || !strings.Contains(addResponse.Body.String(), "Ikat Handwoven Tote") {
		t.Fatalf("cart mutation response = %d, missing product", addResponse.Code)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/cart", nil))
	if strings.Contains(second.Body.String(), "Ikat Handwoven Tote") {
		t.Fatal("new browser session inherited first session cart mutation")
	}

	wish := httptest.NewRequest(http.MethodPost, "/ui/wishlist/blue-tote", nil)
	wish.Header.Set("HX-Request", "true")
	wish.AddCookie(cookie)
	wishResponse := httptest.NewRecorder()
	h.ServeHTTP(wishResponse, wish)
	if wishResponse.Code != http.StatusOK || !strings.Contains(wishResponse.Body.String(), "Ikat Handwoven Tote") {
		t.Fatalf("wishlist mutation response = %d, missing product", wishResponse.Code)
	}
}

func TestMockUnknownRoute(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}

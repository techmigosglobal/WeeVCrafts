package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockCustomerRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/", "/categories", "/products", "/search?q=saree", "/products/chanderi-royal", "/makers/mithila-arts", "/wishlist", "/cart", "/checkout", "/orders", "/orders/%23WC2504267819", "/returns", "/account", "/login", "/register", "/forgot-password", "/reset-password", "/verify-email", "/faq", "/contact", "/shipping", "/size-guide", "/terms", "/privacy", "/about", "/our-story", "/sustainability", "/press", "/careers", "/account/profile", "/account/addresses", "/account/notifications", "/account/security", "/account/privacy", "/account/reviews", "/seller"}
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

func TestMockCustomerResponsibilities(t *testing.T) {
	tests := []struct {
		route string
		want  []string
	}{
		{route: "/", want: []string{"Trending now", "Stories behind the craft", "/assets/images/customer/hero-studio.png"}},
		{route: "/categories", want: []string{"Browse by craft", "Shop by craft", "Gifts that feel personal"}},
		{route: "/search?q=handloom+sarees", want: []string{"Filter by", "THE HANDLOOM EDIT", "Add to Cart"}},
		{route: "/products/chanderi-royal", want: []string{"Add to cart", "Specifications", "Maker story", "More made-for-you finds"}},
		{route: "/wishlist", want: []string{"items in your wishlist", "Move to cart", "You may also love"}},
		{route: "/cart", want: []string{"Order summary", "Proceed to checkout", "supporting independent makers"}},
		{route: "/checkout", want: []string{"Delivery address", "Payment method", "Place order"}},
		{route: "/orders", want: []string{"Track order", "Past orders", "Orders"}},
		{route: "/returns", want: []string{"Active returns", "Pickup scheduled", "Refund completed"}},
		{route: "/account", want: []string{"Profile information", "Security &amp; sessions", "Need a hand?"}},
		{route: "/login", want: []string{"Sign in to WeeVCrafts", "Keep me signed in"}},
		{route: "/faq", want: []string{"Frequently asked questions", "How do I know a product is handmade?"}},
		{route: "/contact", want: []string{"Contact WeeVCrafts", "Send us a note"}},
		{route: "/account/security", want: []string{"Security and sessions", "Two-step verification"}},
		{route: "/account/privacy", want: []string{"Privacy preferences", "Data access request"}},
		{route: "/account/reviews", want: []string{"Reviews to write", "Publish review"}},
	}
	for _, test := range tests {
		t.Run(test.route, func(t *testing.T) {
			res := httptest.NewRecorder()
			NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, test.route, nil))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			for _, want := range test.want {
				if !strings.Contains(body, want) {
					t.Errorf("response is missing %q", want)
				}
			}
		})
	}
}

func TestMockAdminRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/admin", "/admin/sellers", "/admin/products", "/admin/inventory", "/admin/orders", "/admin/returns", "/admin/finance", "/admin/support", "/admin/marketing", "/admin/analytics", "/admin/security"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route+"?notice=Preview+action+completed", nil)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			if !strings.Contains(body, "Super Admin") || !strings.Contains(body, "Preview action completed") {
				t.Fatal("admin response is missing the portal identity or action notice")
			}
		})
	}
}

func TestMockSellerAdminRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/seller-admin/onboarding", "/seller-admin", "/seller-admin/products", "/seller-admin/inventory", "/seller-admin/orders", "/seller-admin/returns", "/seller-admin/earnings", "/seller-admin/analytics", "/seller-admin/settings", "/seller-admin/team"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, route+"?notice=Preview+action+completed", nil))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			if !strings.Contains(body, "Seller Portal") || !strings.Contains(body, "Weaver's Touch") || !strings.Contains(body, "Preview action completed") {
				t.Fatal("seller response is missing the portal identity, seller scope or action notice")
			}
		})
	}
}

func TestMockSellerAdminMethodBoundary(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/seller-admin/orders", nil))
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.Code)
	}
}

func TestMockSupportPortalRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/support-portal", "/support-portal/cases", "/support-portal/customers", "/support-portal/orders"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, route+"?notice=Preview+action+completed", nil))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			for _, want := range []string{"Support Portal", "Aditi Rao", "Preview action completed"} {
				if !strings.Contains(body, want) {
					t.Errorf("response is missing %q", want)
				}
			}
		})
	}
}

func TestMockSupportPortalDataBoundary(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/support-portal/customers", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{"p***a.sharma@gmail.com", "+91 ******5219", "Case Context", "Seller / Store Details"} {
		if !strings.Contains(body, want) {
			t.Errorf("support response is missing %q", want)
		}
	}
	for _, secret := range []string{"password", "CVV", "card number"} {
		if strings.Contains(strings.ToLower(body), secret) {
			t.Errorf("support response leaked restricted field %q", secret)
		}
	}
}

func TestMockSupportPortalMethodBoundary(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/support-portal/cases", nil))
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.Code)
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

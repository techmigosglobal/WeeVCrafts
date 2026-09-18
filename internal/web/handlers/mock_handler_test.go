package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func previewLogin(t *testing.T, h *MockHandler, role mockRole) *http.Cookie {
	t.Helper()
	account, ok := findMockAccountByRole(string(role))
	if !ok {
		t.Fatalf("missing preview account for %q", role)
	}
	form := url.Values{"email": {account.Identity.Email}, "password": {account.Password}, "role": {string(role)}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("preview login for %q = %d, want 303", role, response.Code)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "weevcrafts_ui" {
			return cookie
		}
	}
	t.Fatalf("preview login for %q did not set a session cookie", role)
	return nil
}

func authenticatedPreviewRequest(t *testing.T, h *MockHandler, method, path string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	var role mockRole
	switch {
	case strings.HasPrefix(path, "/admin"):
		role = roleSuperAdmin
	case strings.HasPrefix(path, "/seller-admin"):
		role = roleVendor
	case strings.HasPrefix(path, "/support-portal"):
		role = roleSupport
	default:
		return request
	}
	request.AddCookie(previewLogin(t, h, role))
	return request
}

func TestMockCustomerRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/", "/categories", "/brands", "/brands/mithila-arts", "/products", "/deals", "/search?q=saree", "/products/chanderi-royal", "/makers/mithila-arts", "/makers/weavers-touch", "/wishlist", "/cart", "/checkout", "/orders", "/tracking", "/payments", "/payments/success", "/payments/pending", "/payments/failed", "/orders/invoice", "/orders/%23WC2504267819", "/returns", "/returns/RET123456/label", "/account", "/login", "/register", "/forgot-password", "/reset-password", "/verify-email", "/faq", "/contact", "/shipping", "/size-guide", "/terms", "/privacy", "/about", "/our-story", "/sustainability", "/press", "/careers", "/account/profile", "/account/addresses", "/account/notifications", "/account/security", "/account/privacy", "/account/reviews", "/seller"}
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

func TestMockSellerStoreUsesRequestedSeller(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/makers/weavers-touch", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Products by Weaver&#39;s Touch") {
		t.Fatalf("seller store = %d, expected requested seller content", res.Code)
	}
	if strings.Contains(res.Body.String(), "Products by Mithila Arts") {
		t.Fatal("seller store leaked the Mithila Arts fixture into another seller route")
	}
}

func TestMockProductDetailUsesProductAndMakerData(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/products/madhubani-tree", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{"Madhubani Painting - Tree of Life", "By Mithila Arts", "/makers/mithila-arts", "/products?category=arts", "Folk painting", "Natural earth"} {
		if !strings.Contains(body, want) {
			t.Errorf("product detail is missing %q", want)
		}
	}
	for _, want := range []string{`id="detail-quantity"`, `hx-include="#detail-quantity"`, `hx-vals='{"intent":"buy"}'`} {
		if !strings.Contains(body, want) {
			t.Errorf("product detail is missing quantity/buy-now contract %q", want)
		}
	}
	if strings.Contains(body, "This exquisite Chanderi Silk Cotton saree") || strings.Contains(body, "Only 3 left") {
		t.Fatal("product detail leaked saree-specific fixture copy")
	}
}

func TestMockOrderDetailsFollowRequestedOrder(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/orders/WC2504216632", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "#WC2504216632") || !strings.Contains(body, "Madhubani Painting - Tree of Life") || !strings.Contains(body, "INR 2,499") {
		t.Fatal("order detail did not render the requested past order")
	}
	if strings.Contains(body, "#WC2504267819") || strings.Contains(body, "Chanderi Silk Cotton Saree - Royal Maroon") {
		t.Fatal("order detail leaked the first order fixture")
	}
}

func TestMockCustomerInteractionAffordances(t *testing.T) {
	for _, route := range []string{"/", "/wishlist"} {
		res := httptest.NewRecorder()
		NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, route, nil))
		if res.Code != http.StatusOK {
			t.Fatalf("route %s status = %d, want 200", route, res.Code)
		}
		body := res.Body.String()
		if strings.Contains(body, `href="#"`) {
			t.Fatalf("route %s contains a dead hash link", route)
		}
		if !strings.Contains(body, `aria-expanded`) || !strings.Contains(body, `mobile-menu-backdrop`) {
			t.Fatalf("route %s is missing mobile navigation affordances", route)
		}
		if !strings.Contains(body, `href="/our-story">Our Story`) {
			t.Fatalf("route %s has an incorrect Our Story destination", route)
		}
	}
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/wishlist", nil))
	if !strings.Contains(res.Body.String(), `id="wishlist-grid"`) || !strings.Contains(res.Body.String(), `hx-target="#wishlist-grid"`) {
		t.Fatal("wishlist cards are missing a focused HTMX grid target")
	}
}

func TestMockPortalRoutesHaveNoDeadGeneratedLinks(t *testing.T) {
	routes := []string{
		"/", "/categories", "/brands", "/brands/mithila-arts", "/products", "/deals", "/search?q=saree",
		"/products/chanderi-royal", "/makers/mithila-arts", "/wishlist", "/cart", "/checkout", "/orders",
		"/tracking", "/payments", "/payments/success", "/payments/pending", "/payments/failed", "/returns",
		"/account", "/faq", "/contact", "/seller",
		"/admin", "/admin/sellers", "/admin/products", "/admin/brands", "/admin/inventory", "/admin/orders",
		"/admin/returns", "/admin/disputes", "/admin/finance", "/admin/payments", "/admin/failed-payments",
		"/admin/support", "/admin/customers", "/admin/reviews", "/admin/marketing", "/admin/analytics",
		"/admin/security", "/admin/roles",
		"/seller-admin", "/seller-admin/products", "/seller-admin/inventory", "/seller-admin/orders",
		"/seller-admin/returns", "/seller-admin/earnings", "/seller-admin/commission", "/seller-admin/analytics",
		"/seller-admin/audit", "/seller-admin/promotions", "/seller-admin/reviews", "/seller-admin/messages",
		"/seller-admin/support", "/seller-admin/settings", "/seller-admin/team",
		"/support-portal", "/support-portal/cases", "/support-portal/customers", "/support-portal/orders",
	}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h := NewMockHandler()
			h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, route))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			if strings.Contains(body, `href="#"`) {
				t.Fatal("route contains a dead hash link")
			}
			if strings.Contains(body, `href="/orders/%23`) {
				t.Fatal("route contains an encoded order hash link")
			}
		})
	}
}

func TestMockPRDScreenRouteMatrix(t *testing.T) {
	routes := []struct {
		name   string
		prefix string
		paths  []string
	}{
		{name: "customer", prefix: "WeeVCrafts", paths: []string{
			"/", "/search?q=saree", "/products?category=sarees", "/categories", "/brands", "/brands/mithila-arts", "/products/chanderi-royal", "/makers/mithila-arts", "/deals", "/about", "/contact", "/faq", "/privacy", "/terms", "/returns", "/shipping",
			"/register", "/login", "/forgot-password", "/reset-password", "/verify-email", "/account/security", "/account", "/account/profile", "/account/addresses", "/account/notifications", "/account/privacy", "/account/reviews", "/orders", "/orders/WC2504267819", "/tracking", "/returns/RET123456/label", "/wishlist", "/cart", "/checkout", "/payments", "/payments/success", "/payments/failed", "/payments/pending",
		}},
		{name: "seller", prefix: "Seller Portal", paths: []string{
			"/seller-admin/onboarding", "/seller-admin/verification", "/seller-admin", "/seller-admin/products", "/seller-admin/product-list", "/seller-admin/add-product", "/seller-admin/edit-product", "/seller-admin/variants", "/seller-admin/media", "/seller-admin/pricing", "/seller-admin/inventory", "/seller-admin/inventory-transactions", "/seller-admin/orders", "/seller-admin/order-details", "/seller-admin/fulfillment", "/seller-admin/returns", "/seller-admin/refund-status", "/seller-admin/promotions", "/seller-admin/analytics", "/seller-admin/commission", "/seller-admin/earnings", "/seller-admin/settings", "/seller-admin/team", "/seller-admin/staff-permissions", "/seller-admin/support", "/seller-admin/audit",
		}},
		{name: "admin", prefix: "Admin Portal", paths: []string{
			"/admin", "/admin/sellers", "/admin/seller-approvals", "/admin/seller-details", "/admin/seller-suspensions", "/admin/products", "/admin/product-details", "/admin/category-management", "/admin/brands", "/admin/customers", "/admin/orders", "/admin/payments", "/admin/failed-payments", "/admin/refunds", "/admin/returns", "/admin/disputes", "/admin/marketing", "/admin/coupons", "/admin/reviews", "/admin/commission-rules", "/admin/settlements", "/admin/finance-reconciliation", "/admin/support", "/admin/audit-logs", "/admin/system-health", "/admin/search-indexing-health", "/admin/worker-queue-health", "/admin/security", "/admin/security-events", "/admin/settings", "/admin/roles",
		}},
	}
	for _, group := range routes {
		for _, route := range group.paths {
			t.Run(group.name+route, func(t *testing.T) {
				res := httptest.NewRecorder()
				h := NewMockHandler()
				h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, route))
				if res.Code != http.StatusOK {
					t.Fatalf("route %s status = %d, want 200", route, res.Code)
				}
				body := res.Body.String()
				if !strings.Contains(body, group.prefix) {
					t.Fatalf("route %s is missing role/page identity %q", route, group.prefix)
				}
				if strings.Contains(body, "Page not found") || strings.Contains(body, `href="#"`) || strings.Contains(body, `/orders/%23`) {
					t.Fatalf("route %s rendered a dead or not-found state", route)
				}
			})
		}
	}
}

func TestMockProductQuantityIsAppliedToCartMutation(t *testing.T) {
	h := NewMockHandler()
	initial := httptest.NewRecorder()
	h.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/products/chanderi-royal", nil))
	cookie := initial.Result().Cookies()[0]

	request := httptest.NewRequest(http.MethodPost, "/ui/cart/chanderi-royal", strings.NewReader("quantity=3"))
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "cart-count")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	cart := httptest.NewRecorder()
	cartRequest := httptest.NewRequest(http.MethodGet, "/cart", nil)
	cartRequest.AddCookie(cookie)
	h.ServeHTTP(cart, cartRequest)
	if !strings.Contains(cart.Body.String(), "6 items from independent Indian makers") {
		t.Fatal("detail quantity did not add three units to the existing cart")
	}
}

func TestMockBuyNowRedirectsToCheckout(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/ui/cart/blue-tote", strings.NewReader("intent=buy&quantity=2"))
	request.Header.Set("HX-Request", "true")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("HX-Redirect") != "/checkout" {
		t.Fatalf("buy now response = %d with redirect %q, want 200 and /checkout", response.Code, response.Header().Get("HX-Redirect"))
	}
}

func TestMockCustomerResponsibilities(t *testing.T) {
	tests := []struct {
		route string
		want  []string
	}{
		{route: "/", want: []string{"Trending now", "Stories behind the craft", "/assets/images/customer/hero-studio.webp"}},
		{route: "/categories", want: []string{"Browse by craft", "Shop by craft", "Gifts that feel personal"}},
		{route: "/brands", want: []string{"Explore brands with a story", "Mithila Arts", "Weaver&#39;s Touch"}},
		{route: "/brands/mithila-arts", want: []string{"WEEVCRAFTS BRAND PROFILE", "Verified maker brand", "Pieces from Mithila Arts"}},
		{route: "/search?q=handloom+sarees", want: []string{"Filter by", "THE HANDLOOM EDIT", "Add to Cart"}},
		{route: "/products/chanderi-royal", want: []string{"Add to cart", "Specifications", "Maker story", "More made-for-you finds"}},
		{route: "/wishlist", want: []string{"items in your wishlist", "Move to cart", "You may also love"}},
		{route: "/cart", want: []string{"Order summary", "Proceed to checkout", "supporting independent makers"}},
		{route: "/checkout", want: []string{"Delivery address", "Payment method", "Place order"}},
		{route: "/orders", want: []string{"Track order", "Past orders", "Orders"}},
		{route: "/tracking", want: []string{"Track your order", "Shipment in transit", "Tracking ID: 149730248756"}},
		{route: "/payments", want: []string{"Saved payment methods", "Recent payment activity", "Failed"}},
		{route: "/payments/success", want: []string{"Payment successful", "Payment captured", "#PAY25603421"}},
		{route: "/payments/pending", want: []string{"Payment pending", "Reconciliation in progress", "#PAY25602912"}},
		{route: "/payments/failed", want: []string{"Payment failed", "Action needed", "#PAY25601432"}},
		{route: "/deals", want: []string{"Handmade deals", "Offers with a purpose", "Limited-time edit"}},
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

func TestMockCustomerSummaryValuesComeFromPreviewData(t *testing.T) {
	categories := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(categories, httptest.NewRequest(http.MethodGet, "/categories", nil))
	if categories.Code != http.StatusOK {
		t.Fatalf("categories status = %d, want 200", categories.Code)
	}
	categoryBody := categories.Body.String()
	if !strings.Contains(categoryBody, "All categories <span>6</span>") || !strings.Contains(categoryBody, "Arts <span>148</span>") {
		t.Fatal("category navigation is missing viewmodel-provided counts")
	}

	payments := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(payments, httptest.NewRequest(http.MethodGet, "/payments", nil))
	if payments.Code != http.StatusOK {
		t.Fatalf("payments status = %d, want 200", payments.Code)
	}
	paymentBody := payments.Body.String()
	for _, want := range []string{"#PAY-01", "INR 10,497", "#PAY-04", "Failed"} {
		if !strings.Contains(paymentBody, want) {
			t.Errorf("payment activity is missing derived value %q", want)
		}
	}
}

func TestMockOrderStatusEmptyStateIsTruthful(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/orders?status=closed", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("orders status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "No orders in this view") || strings.Contains(body, "ORDER PLACED") {
		t.Fatal("empty order status view should render a clear empty state without a blank order card")
	}
}

func TestMockReturnTabsUseTheirOwnDataset(t *testing.T) {
	refunds := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(refunds, httptest.NewRequest(http.MethodGet, "/returns?tab=refunds", nil))
	if refunds.Code != http.StatusOK || !strings.Contains(refunds.Body.String(), "Refund completed") || !strings.Contains(refunds.Body.String(), "View refund details") {
		t.Fatal("refunds tab did not render the refund dataset")
	}
	if strings.Contains(refunds.Body.String(), "Return request RET123456") {
		t.Fatal("refunds tab leaked the active return card")
	}

	closed := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(closed, httptest.NewRequest(http.MethodGet, "/returns?tab=closed", nil))
	if closed.Code != http.StatusOK || !strings.Contains(closed.Body.String(), "No returns in this view") || strings.Contains(closed.Body.String(), "Return request RET123456") {
		t.Fatal("closed returns tab should render a truthful empty state")
	}
}

func TestMockCheckoutPaymentTabsAreAlpineControlled(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/checkout", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("checkout status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, `:class="payment === 'upi' ? 'active' : ''"`) || !strings.Contains(body, `@click="payment='card'"`) {
		t.Fatal("checkout payment methods are missing Alpine state bindings")
	}
	if strings.Contains(body, `<button class="active" type="button" role="tab"`) {
		t.Fatal("checkout payment tabs must not keep a permanent active class")
	}
}

func TestMockProductCardsUseProductRatings(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/products?page=2", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("products status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	start := strings.Index(body, `data-product="terracotta-planter"`)
	if start < 0 {
		t.Fatal("rating fixture product card was not rendered")
	}
	remainder := body[start:]
	end := strings.Index(remainder, "</article>")
	if end < 0 {
		t.Fatal("rating fixture product card is not closed")
	}
	card := remainder[:end]
	if got := strings.Count(card, `aria-hidden="true">star</span>`); got != 4 {
		t.Fatalf("4.4-rated product rendered %d stars, want 4", got)
	}
	if !strings.Contains(card, `aria-label="4.4 out of 5 stars"`) {
		t.Fatal("product rating is missing its accessible label")
	}
}

func TestMockAdminRoutes(t *testing.T) {
	routes := []string{"/admin", "/admin/sellers", "/admin/products", "/admin/brands", "/admin/inventory", "/admin/orders", "/admin/returns", "/admin/disputes", "/admin/finance", "/admin/payments", "/admin/failed-payments", "/admin/support", "/admin/customers", "/admin/reviews", "/admin/marketing", "/admin/analytics", "/admin/security", "/admin/roles"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			h := NewMockHandler()
			req := authenticatedPreviewRequest(t, h, http.MethodGet, route+"?notice=Preview+action+completed")
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

func TestMockAdminAndSellerWorkspacesRenderTheirOwnSurface(t *testing.T) {
	tests := []struct {
		route string
		want  string
	}{
		{route: "/admin/brands", want: "Brands &amp; Categories"},
		{route: "/admin/failed-payments", want: "UPI collect request expired"},
		{route: "/admin/roles", want: "Role Management"},
		{route: "/seller-admin/promotions", want: "Promotions &amp; Offers"},
		{route: "/seller-admin/commission", want: "Commission Reports"},
		{route: "/seller-admin/audit", want: "Activity &amp; Audit"},
	}
	for _, test := range tests {
		res := httptest.NewRecorder()
		h := NewMockHandler()
		h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, test.route))
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), test.want) {
			t.Fatalf("route %s = %d, missing %q", test.route, res.Code, test.want)
		}
	}
}

func TestMockSellerAdminRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/seller-admin/onboarding", "/seller-admin", "/seller-admin/products", "/seller-admin/inventory", "/seller-admin/orders", "/seller-admin/returns", "/seller-admin/earnings", "/seller-admin/commission", "/seller-admin/analytics", "/seller-admin/audit", "/seller-admin/promotions", "/seller-admin/reviews", "/seller-admin/messages", "/seller-admin/support", "/seller-admin/settings", "/seller-admin/team"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, route+"?notice=Preview+action+completed"))
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
	h := NewMockHandler()
	h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodPost, "/seller-admin/orders"))
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.Code)
	}
}

func TestMockRoleFiltersAreApplied(t *testing.T) {
	tests := []struct {
		route string
		want  string
		miss  string
	}{
		{route: "/admin/products?category=sarees", want: "Chanderi Silk Saree", miss: "Ceramic Mug - Blue Floral"},
		{route: "/admin/inventory?stock=low", want: "Kanchipuram Silk Saree", miss: "Tussar Silk Saree"},
		{route: "/admin/orders?type=export", want: "New York, USA", miss: "Bengaluru, KA"},
		{route: "/seller-admin/products?status=approved", want: "Chanderi Silk Cotton Saree - Royal Maroon", miss: "Block Print Cushion Cover - Marigold"},
		{route: "/seller-admin/inventory?status=low-stock", want: "Handpainted Ceramic Mugs", miss: "Carved Wooden Jewellery Box"},
		{route: "/seller-admin/orders?payment=refunded", want: "Meera Krishnan", miss: "#WC2504267831"},
		{route: "/support-portal/customers?type=seller", want: "Weaver&#39;s Touch", miss: "Priya Sharma"},
	}
	for _, test := range tests {
		t.Run(test.route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h := NewMockHandler()
			h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, test.route))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			if !strings.Contains(body, test.want) {
				t.Errorf("filtered response is missing %q", test.want)
			}
			if strings.Contains(body, test.miss) {
				t.Errorf("filtered response still contains %q", test.miss)
			}
		})
	}

	res := httptest.NewRecorder()
	h := NewMockHandler()
	h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, "/support-portal/cases?q=does-not-exist"))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "No support cases") {
		t.Fatalf("empty support search response = %d, expected safe empty state", res.Code)
	}
}

func TestMockRoleSearchesKeepEmptyStatesSafe(t *testing.T) {
	tests := []struct {
		route string
		want  string
	}{
		{route: "/admin/sellers?q=does-not-exist", want: "No sellers match"},
		{route: "/admin/products?q=does-not-exist", want: "No products match"},
		{route: "/admin/inventory?q=does-not-exist", want: "No inventory match"},
		{route: "/admin/orders?q=does-not-exist", want: "No orders match"},
		{route: "/admin/returns?q=does-not-exist", want: "No return requests"},
		{route: "/admin/support?q=does-not-exist", want: "No support cases"},
		{route: "/seller-admin/products?q=does-not-exist", want: "No products match"},
		{route: "/seller-admin/inventory?q=does-not-exist", want: "No inventory match"},
		{route: "/seller-admin/orders?q=does-not-exist", want: "No orders match"},
		{route: "/seller-admin/returns?q=does-not-exist", want: "No return requests"},
		{route: "/support-portal/cases?q=does-not-exist", want: "No support cases"},
		{route: "/support-portal/customers?q=does-not-exist", want: "No customer context"},
		{route: "/support-portal/orders?q=does-not-exist", want: "No order context"},
	}
	for _, test := range tests {
		t.Run(test.route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h := NewMockHandler()
			h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, test.route))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			body := res.Body.String()
			if strings.Contains(body, "rendering failed") || strings.Contains(body, "runtime error") {
				t.Fatal("filtered empty state contains a rendering failure")
			}
			if !strings.Contains(body, test.want) {
				t.Errorf("filtered empty state is missing %q", test.want)
			}
		})
	}
}

func TestMockSupportPortalRoutes(t *testing.T) {
	h := NewMockHandler()
	routes := []string{"/support-portal", "/support-portal/cases", "/support-portal/customers", "/support-portal/orders"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, route+"?notice=Preview+action+completed"))
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

func TestMockFilteredRoleQueuesKeepEmptyStatesSafe(t *testing.T) {
	handler := NewMockHandler()
	cases := []struct {
		route string
		want  string
	}{
		{route: "/admin/orders?q=does-not-exist", want: "No orders match this search."},
		{route: "/admin/returns?q=does-not-exist", want: "No return requests"},
		{route: "/seller-admin/returns?q=does-not-exist", want: "No return requests"},
		{route: "/support-portal/cases?q=does-not-exist", want: "No support cases"},
		{route: "/support-portal/customers?q=does-not-exist", want: "No customer context"},
		{route: "/support-portal/orders?q=does-not-exist", want: "No order context"},
	}
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, authenticatedPreviewRequest(t, handler, http.MethodGet, tc.route))
			if response.Code != http.StatusOK {
				t.Fatalf("expected 200 for filtered empty state, got %d", response.Code)
			}
			if !strings.Contains(response.Body.String(), tc.want) {
				t.Fatalf("expected %q in filtered empty state", tc.want)
			}
		})
	}
}

func TestMockSupportPortalDataBoundary(t *testing.T) {
	res := httptest.NewRecorder()
	h := NewMockHandler()
	h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodGet, "/support-portal/customers"))
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
	h := NewMockHandler()
	h.ServeHTTP(res, authenticatedPreviewRequest(t, h, http.MethodPost, "/support-portal/cases"))
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

func TestMockListingStateAndSafeLinks(t *testing.T) {
	h := NewMockHandler()
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/products?category=sarees&sort=price-desc&brand=handloom", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{`option value="price-desc" selected`, `name="category" value="sarees" checked`, `name="brand" value="handloom" checked`, `Sarees`, `Handloom`, `<span>(2)</span>`, "2 results"} {
		if !strings.Contains(body, want) {
			t.Errorf("listing response is missing %q", want)
		}
	}
	if strings.Contains(body, `href="/orders/%23`) {
		t.Fatal("listing response contains an encoded order hash")
	}

	htmx := httptest.NewRequest(http.MethodGet, "/products?category=Home+%26+Living&sort=price-low", nil)
	htmx.Header.Set("HX-Request", "true")
	htmxResponse := httptest.NewRecorder()
	h.ServeHTTP(htmxResponse, htmx)
	if htmxResponse.Code != http.StatusOK || !strings.Contains(htmxResponse.Body.String(), `id="listing-results"`) {
		t.Fatal("HTMX listing response does not expose the stable results target")
	}
	if got := strings.Count(htmxResponse.Body.String(), `class="product-card"`); got != 4 {
		t.Fatalf("HTMX filtered listing cards = %d, want 4", got)
	}

	orders := httptest.NewRecorder()
	h.ServeHTTP(orders, httptest.NewRequest(http.MethodGet, "/orders", nil))
	if strings.Contains(orders.Body.String(), "/orders/%23") || !strings.Contains(orders.Body.String(), `/orders/WC2504267819`) {
		t.Fatal("orders page does not expose a usable order detail link")
	}
}

func TestMockSearchSuggestions(t *testing.T) {
	h := NewMockHandler()
	header := httptest.NewRecorder()
	h.ServeHTTP(header, httptest.NewRequest(http.MethodGet, "/", nil))
	headerBody := header.Body.String()
	for _, want := range []string{`role="combobox"`, `aria-expanded="false"`, `aria-controls="search-suggestions"`, `hx-disinherit="hx-select hx-push-url"`, `hx-push-url="false"`, `initSearchSuggestions`} {
		if !strings.Contains(headerBody, want) {
			t.Errorf("search input is missing accessible combobox contract %q", want)
		}
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ui/search-suggest?q=saree", nil))
	body := res.Body.String()
	if res.Code != http.StatusOK || !strings.Contains(body, `role="option"`) || !strings.Contains(body, `aria-selected="false"`) || !strings.Contains(body, `id="search-suggestion-`) {
		t.Fatalf("suggestion response = %d, missing accessible options", res.Code)
	}
	if strings.Contains(body, "<html") || !strings.Contains(body, "/products/chanderi-royal") {
		t.Fatal("suggestions should be a focused product fragment")
	}

	empty := httptest.NewRecorder()
	h.ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/ui/search-suggest", nil))
	if strings.TrimSpace(empty.Body.String()) != "" {
		t.Fatal("empty search suggestions should return an empty target fragment")
	}
}

func TestMockOrderTabsFilterOrders(t *testing.T) {
	for _, test := range []struct {
		status string
		want   string
		miss   string
	}{
		{status: "active", want: "#WC2504267819", miss: "#WC2504216632"},
		{status: "delivered", want: "#WC2504216632", miss: "#WC2504097712"},
		{status: "cancelled", want: "#WC2504097712", miss: "#WC2504216632"},
	} {
		t.Run(test.status, func(t *testing.T) {
			res := httptest.NewRecorder()
			NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/orders?status="+test.status, nil))
			body := res.Body.String()
			if res.Code != http.StatusOK || !strings.Contains(body, test.want) {
				t.Fatalf("status = %d, missing filtered order %q", res.Code, test.want)
			}
			if strings.Contains(body, test.miss) {
				t.Fatalf("filtered order page still contains %q", test.miss)
			}
			if !strings.Contains(body, `aria-current="page"`) || !strings.Contains(body, `href="/orders?status=`+test.status+`"`) {
				t.Fatalf("%s tab is not marked current", test.status)
			}
		})
	}
}

func TestMockCartMutationReturnsOOBBadge(t *testing.T) {
	h := NewMockHandler()
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	cookie := first.Result().Cookies()[0]
	request := httptest.NewRequest(http.MethodPost, "/ui/cart/blue-tote", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "cart-count")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `hx-swap-oob="outerHTML"`) || !strings.Contains(body, `id="ui-toast"`) {
		t.Fatal("cart mutation did not return the out-of-band badge and status message")
	}
}

func TestMockHTMXMutationsReturnFragments(t *testing.T) {
	h := NewMockHandler()
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/cart", nil))
	cookie := first.Result().Cookies()[0]

	request := httptest.NewRequest(http.MethodPost, "/ui/cart/blue-tote", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "cart-layout")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "<!doctype html") || !strings.Contains(response.Body.String(), `id="cart-layout"`) {
		t.Fatalf("cart fragment response = %d, expected target fragment", response.Code)
	}

	checkout := httptest.NewRequest(http.MethodPost, "/ui/checkout", nil)
	checkout.Header.Set("HX-Request", "true")
	checkoutResponse := httptest.NewRecorder()
	h.ServeHTTP(checkoutResponse, checkout)
	if checkoutResponse.Code != http.StatusOK || strings.Contains(checkoutResponse.Body.String(), "<!doctype html") || !strings.Contains(checkoutResponse.Body.String(), "Preview order placed") {
		t.Fatalf("checkout fragment response = %d, expected completion fragment", checkoutResponse.Code)
	}
}

func TestMockMoveCartItemToWishlistPreservesSavedState(t *testing.T) {
	h := NewMockHandler()
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/cart", nil))
	cookie := first.Result().Cookies()[0]

	request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/chanderi-royal", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "page")
	request.Header.Set("HX-Current-URL", "http://preview.test/cart")
	request.AddCookie(cookie)
	request.URL.RawQuery = "source=cart"
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "Chanderi Silk Cotton Saree - Royal Maroon") {
		t.Fatal("moved cart item is still rendered in the cart")
	}

	wishlist := httptest.NewRecorder()
	wishlistRequest := httptest.NewRequest(http.MethodGet, "/wishlist", nil)
	wishlistRequest.AddCookie(cookie)
	h.ServeHTTP(wishlist, wishlistRequest)
	if !strings.Contains(wishlist.Body.String(), "Chanderi Silk Cotton Saree - Royal Maroon") {
		t.Fatal("moved cart item is not present in the wishlist")
	}
}

func TestMockMoveCartItemToWishlistReturnsBothBadges(t *testing.T) {
	h := NewMockHandler()
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/cart", nil))
	cookie := first.Result().Cookies()[0]

	request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/chanderi-royal?source=cart", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "cart-layout")
	request.Header.Set("HX-Current-URL", "http://preview.test/cart")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "<!doctype html") || !strings.Contains(body, `id="cart-layout"`) {
		t.Fatal("cart-to-wishlist response is not a focused cart fragment")
	}
	if strings.Count(body, `id="cart-count"`) != 1 || strings.Count(body, `id="wishlist-count"`) != 1 || !strings.Contains(body, `id="ui-toast"`) {
		t.Fatal("cart-to-wishlist response did not update both badges and the toast")
	}
}

func TestMockWishlistRecommendationKeepsArticleTarget(t *testing.T) {
	h := NewMockHandler()
	request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/blue-tote", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "closest article")
	request.Header.Set("HX-Current-URL", "http://preview.test/wishlist")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, `id="wishlist-grid"`) {
		t.Fatal("wishlist recommendation mutation returned the whole wishlist grid")
	}
	if !strings.Contains(body, `class="product-card"`) || !strings.Contains(body, `hx-swap-oob="outerHTML"`) {
		t.Fatal("wishlist recommendation mutation did not return the focused card and badge update")
	}
}

func TestMockEmptyWishlistReflectsSessionState(t *testing.T) {
	h := NewMockHandler()
	initial := httptest.NewRecorder()
	h.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/wishlist", nil))
	cookie := initial.Result().Cookies()[0]
	for _, slug := range []string{"madhubani-tree", "brass-elephant", "chanderi-royal", "wooden-box", "ceramic-mugs", "blue-tote"} {
		request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/"+slug, nil)
		request.Header.Set("HX-Request", "true")
		request.AddCookie(cookie)
		mutation := httptest.NewRecorder()
		h.ServeHTTP(mutation, request)
		if mutation.Code != http.StatusOK {
			t.Fatalf("wishlist toggle %s status = %d, want 200", slug, mutation.Code)
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/wishlist", nil)
	request.AddCookie(cookie)
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "0 items in your wishlist") || !strings.Contains(body, "Your wishlist is empty") {
		t.Fatal("empty wishlist did not reflect the session state")
	}
}

func TestMockRemovingLastWishlistItemReturnsEmptyGrid(t *testing.T) {
	h := NewMockHandler()
	initial := httptest.NewRecorder()
	h.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/wishlist", nil))
	cookie := initial.Result().Cookies()[0]
	for _, slug := range []string{"madhubani-tree", "brass-elephant", "chanderi-royal", "wooden-box", "ceramic-mugs"} {
		request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/"+slug+"?source=wishlist", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", "wishlist-grid")
		request.Header.Set("HX-Current-URL", "http://preview.test/wishlist")
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("wishlist toggle %s status = %d, want 200", slug, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/ui/wishlist/blue-tote?source=wishlist", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "wishlist-grid")
	request.Header.Set("HX-Current-URL", "http://preview.test/wishlist")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("last wishlist removal status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "<!doctype html") || !strings.Contains(body, `id="wishlist-grid"`) || !strings.Contains(body, "Your wishlist is empty") {
		t.Fatal("last wishlist removal did not return the focused empty grid")
	}
	if strings.Count(body, `id="wishlist-count"`) != 1 || !strings.Contains(body, `id="ui-toast"`) {
		t.Fatal("last wishlist removal did not include badge and toast updates")
	}
}

func TestMockUnknownRoute(t *testing.T) {
	res := httptest.NewRecorder()
	NewMockHandler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}

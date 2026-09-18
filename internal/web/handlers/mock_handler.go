package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/wecratfs/commerce/internal/web/viewmodels"
	adminpages "github.com/wecratfs/commerce/web/admin"
	"github.com/wecratfs/commerce/web/assets"
	"github.com/wecratfs/commerce/web/components"
	"github.com/wecratfs/commerce/web/pages"
	selleradminpages "github.com/wecratfs/commerce/web/selleradmin"
	supportportalpages "github.com/wecratfs/commerce/web/supportportal"
)

type MockHandler struct {
	store  *mockStore
	static http.Handler
}

func NewMockHandler() *MockHandler {
	return &MockHandler{store: newMockStore(), static: http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS)))}
}

func (h *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.static.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/logout" {
		h.logout(w, r)
		return
	}
	if r.URL.Path == "/login" && r.Method == http.MethodPost {
		h.login(w, r)
		return
	}
	if notice := r.URL.Query().Get("notice"); notice != "" {
		h.store.recordAction(w, r, notice)
	}
	if isRolePortal(r.URL.Path) {
		allowed := []mockRole{roleCustomer}
		switch {
		case r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/"):
			allowed = []mockRole{roleAdmin, roleSuperAdmin}
		case r.URL.Path == "/seller-admin" || strings.HasPrefix(r.URL.Path, "/seller-admin/"):
			allowed = []mockRole{roleVendor}
		case r.URL.Path == "/support-portal" || strings.HasPrefix(r.URL.Path, "/support-portal/"):
			allowed = []mockRole{roleSupport}
		}
		if !h.requireRoles(w, r, allowed...) {
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, "/ui/") {
		if r.Method == http.MethodGet {
			h.fragment(w, r)
			return
		}
		h.mutation(w, r)
		return
	}
	if r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
		h.admin(w, r)
		return
	}
	if r.URL.Path == "/seller-admin" || strings.HasPrefix(r.URL.Path, "/seller-admin/") {
		h.sellerAdmin(w, r)
		return
	}
	if r.URL.Path == "/support-portal" || strings.HasPrefix(r.URL.Path, "/support-portal/") {
		h.supportPortal(w, r)
		return
	}
	page := h.page(r, w)
	switch {
	case r.URL.Path == "/":
		h.render(w, r, pages.Home(page))
	case r.URL.Path == "/categories":
		h.render(w, r, pages.Categories(page))
	case r.URL.Path == "/brands":
		h.render(w, r, pages.Brands(page))
	case strings.HasPrefix(r.URL.Path, "/brands/"):
		brand, ok := brandBySlug(strings.TrimPrefix(r.URL.Path, "/brands/"))
		if !ok {
			h.notFound(w, r)
			return
		}
		page.Title = brand.Name + " brand"
		page.Description = brand.Description
		h.render(w, r, pages.BrandDetail(page, brand))
	case r.URL.Path == "/products" || r.URL.Path == "/search":
		h.render(w, r, pages.Listing(page))
	case r.URL.Path == "/deals":
		h.render(w, r, pages.Deals(page))
	case strings.HasPrefix(r.URL.Path, "/products/"):
		if _, ok := productBySlug(strings.TrimPrefix(r.URL.Path, "/products/")); !ok {
			h.notFound(w, r)
			return
		}
		h.render(w, r, pages.ProductDetail(page))
	case strings.HasPrefix(r.URL.Path, "/makers/"):
		seller, ok := sellerBySlug(strings.TrimPrefix(r.URL.Path, "/makers/"))
		if !ok {
			h.notFound(w, r)
			return
		}
		page.Title = seller.Name + " maker store"
		page.Description = seller.Description
		h.render(w, r, pages.SellerStore(page, seller))
	case r.URL.Path == "/wishlist":
		h.render(w, r, pages.Wishlist(page))
	case r.URL.Path == "/cart":
		h.render(w, r, pages.Cart(page))
	case r.URL.Path == "/checkout":
		h.render(w, r, pages.Checkout(page))
	case r.URL.Path == "/orders/invoice":
		h.render(w, r, pages.Empty(viewmodels.CustomerPage{Title: "Invoice preview", Notice: "Your invoice preview is ready. No file was downloaded in this UI-only environment."}))
	case r.URL.Path == "/orders":
		h.render(w, r, pages.Orders(page))
	case r.URL.Path == "/tracking":
		h.render(w, r, pages.Tracking(page))
	case strings.HasPrefix(r.URL.Path, "/orders/"):
		h.render(w, r, pages.OrderDetail(page))
	case r.URL.Path == "/returns":
		if tab := r.URL.Query().Get("tab"); tab != "" {
			h.render(w, r, pages.ReturnsFiltered(page, tab))
		} else {
			h.render(w, r, pages.Returns(page))
		}
	case strings.HasPrefix(r.URL.Path, "/returns/"):
		h.render(w, r, pages.Empty(viewmodels.CustomerPage{Title: "Return label preview", Notice: "Your return label preview is ready. Download delivery is disabled in this UI-only environment."}))
	case r.URL.Path == "/account":
		h.render(w, r, pages.Account(page))
	case r.URL.Path == "/login" || r.URL.Path == "/register" || r.URL.Path == "/forgot-password" || r.URL.Path == "/reset-password" || r.URL.Path == "/verify-email":
		h.render(w, r, pages.Auth(page))
	case r.URL.Path == "/account/profile" || r.URL.Path == "/account/addresses" || r.URL.Path == "/account/notifications" || r.URL.Path == "/account/security" || r.URL.Path == "/account/privacy" || r.URL.Path == "/account/reviews":
		h.render(w, r, pages.AccountDetail(page))
	case r.URL.Path == "/payments":
		h.render(w, r, pages.Payments(page))
	case r.URL.Path == "/payments/success":
		h.render(w, r, pages.PaymentStatus(page, "success"))
	case r.URL.Path == "/payments/failed":
		h.render(w, r, pages.PaymentStatus(page, "failed"))
	case r.URL.Path == "/payments/pending":
		h.render(w, r, pages.PaymentStatus(page, "pending"))
	case r.URL.Path == "/faq" || r.URL.Path == "/contact" || r.URL.Path == "/shipping" || r.URL.Path == "/size-guide" || r.URL.Path == "/terms" || r.URL.Path == "/privacy" || r.URL.Path == "/about" || r.URL.Path == "/our-story" || r.URL.Path == "/sustainability" || r.URL.Path == "/press" || r.URL.Path == "/careers":
		h.render(w, r, pages.InfoPage(page))
	case r.URL.Path == "/seller":
		sellerPage := sellerAdminPage("/seller-admin/onboarding", r.URL.Query())
		h.render(w, r, selleradminpages.Onboarding(sellerPage))
	default:
		h.notFound(w, r)
	}
}

func (h *MockHandler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	account, ok := findMockAccount(r.FormValue("email"), r.FormValue("password"))
	if selected := strings.TrimSpace(r.FormValue("role")); selected != "" && selected != string(account.Identity.Role) {
		ok = false
	}
	if !ok {
		page := h.page(r, w)
		page.AuthRole = strings.TrimSpace(r.FormValue("role"))
		page.AuthEmail = strings.TrimSpace(r.FormValue("email"))
		page.Notice = "Sign-in failed. Choose a demo role and use its exact preview credentials."
		h.render(w, r, pages.Auth(page))
		return
	}

	h.store.authenticate(w, r, account.Identity)
	h.redirect(w, r, account.Identity.Destination)
}

func (h *MockHandler) logout(w http.ResponseWriter, r *http.Request) {
	h.store.logout(w, r)
	h.redirect(w, r, "/login?notice=You+have+been+logged+out+of+the+preview")
}

func (h *MockHandler) admin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	page := adminPage(r.URL.Path, r.URL.Query())
	if page.Notice == "" {
		page.Notice = h.store.lastAction(w, r)
	}
	identity, _ := identityForRequest(h, w, r)
	page.UserName = identity.Name
	page.UserEmail = identity.Email
	page.UserRole = roleLabel(identity.Role)
	page.PortalLabel = strings.ToUpper(roleLabel(identity.Role)) + " PORTAL"
	page.IsSuperAdmin = identity.Role == roleSuperAdmin
	if !page.IsSuperAdmin && (page.Active == "security" || page.Workspace == "roles") {
		w.WriteHeader(http.StatusForbidden)
		h.render(w, r, h.accessDeniedPage(r, identity))
		return
	}
	if page.Workspace != "" {
		h.render(w, r, adminpages.Workspace(page))
		return
	}
	switch page.Active {
	case "dashboard":
		h.render(w, r, adminpages.Dashboard(page))
	case "sellers":
		h.render(w, r, adminpages.Sellers(page))
	case "products":
		h.render(w, r, adminpages.Products(page))
	case "inventory":
		h.render(w, r, adminpages.Inventory(page))
	case "orders":
		h.render(w, r, adminpages.Orders(page))
	case "returns":
		h.render(w, r, adminpages.Returns(page))
	case "finance":
		h.render(w, r, adminpages.Finance(page))
	case "support":
		h.render(w, r, adminpages.Support(page))
	case "marketing":
		h.render(w, r, adminpages.Marketing(page))
	case "analytics":
		h.render(w, r, adminpages.Analytics(page))
	case "security":
		h.render(w, r, adminpages.Security(page))
	default:
		h.notFound(w, r)
	}
}

func (h *MockHandler) sellerAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	page := sellerAdminPage(r.URL.Path, r.URL.Query())
	if page.Notice == "" {
		page.Notice = h.store.lastAction(w, r)
	}
	identity, _ := identityForRequest(h, w, r)
	page.UserName = identity.Name
	page.UserEmail = identity.Email
	page.UserRole = "Vendor"
	if page.Workspace != "" {
		h.render(w, r, selleradminpages.Workspace(page))
		return
	}
	switch page.Active {
	case "onboarding":
		h.render(w, r, selleradminpages.Onboarding(page))
	case "dashboard":
		h.render(w, r, selleradminpages.Dashboard(page))
	case "products":
		h.render(w, r, selleradminpages.Products(page))
	case "inventory":
		h.render(w, r, selleradminpages.Inventory(page))
	case "orders":
		h.render(w, r, selleradminpages.Orders(page))
	case "returns":
		h.render(w, r, selleradminpages.Returns(page))
	case "earnings":
		h.render(w, r, selleradminpages.Earnings(page))
	case "analytics":
		h.render(w, r, selleradminpages.Analytics(page))
	case "settings":
		h.render(w, r, selleradminpages.Settings(page))
	case "team":
		h.render(w, r, selleradminpages.Team(page))
	default:
		h.notFound(w, r)
	}
}

func (h *MockHandler) supportPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	page := supportPortalPage(r.URL.Path, r.URL.Query())
	if page.Notice == "" {
		page.Notice = h.store.lastAction(w, r)
	}
	identity, _ := identityForRequest(h, w, r)
	page.UserName = identity.Name
	page.UserEmail = identity.Email
	page.UserRole = "Support Agent"
	switch page.Active {
	case "dashboard":
		h.render(w, r, supportportalpages.Dashboard(page))
	case "cases":
		h.render(w, r, supportportalpages.Cases(page))
	case "customers":
		h.render(w, r, supportportalpages.Customers(page))
	case "orders":
		h.render(w, r, supportportalpages.Orders(page))
	default:
		h.notFound(w, r)
	}
}

// fragment exposes stable HTMX targets for the UI-first fixture. The
// responses are server-rendered templ components; live adapters can replace
// these handlers without changing customer URLs.
func (h *MockHandler) fragment(w http.ResponseWriter, r *http.Request) {
	page := h.page(r, w)
	switch r.URL.Path {
	case "/ui/search-suggest":
		h.render(w, r, pages.SearchSuggestions(page.Products, strings.TrimSpace(page.Query)))
	case "/ui/listing", "/ui/fragments/listing":
		h.render(w, r, pages.Listing(page))
	case "/ui/cart", "/ui/fragments/cart":
		h.render(w, r, pages.Cart(page))
	case "/ui/wishlist", "/ui/fragments/wishlist":
		h.render(w, r, pages.Wishlist(page))
	case "/ui/checkout", "/ui/fragments/checkout":
		h.render(w, r, pages.Checkout(page))
	case "/ui/orders", "/ui/fragments/orders":
		h.render(w, r, pages.Orders(page))
	case "/ui/returns", "/ui/fragments/returns":
		if tab := r.URL.Query().Get("tab"); tab != "" {
			h.render(w, r, pages.ReturnsFiltered(page, tab))
		} else {
			h.render(w, r, pages.Returns(page))
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *MockHandler) mutation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[1] == "cart" {
		h.store.updateCart(w, r, parts[2], parseDelta(r))
		if r.Header.Get("HX-Request") == "true" {
			if r.FormValue("intent") == "buy" {
				w.Header().Set("HX-Redirect", "/checkout")
				return
			}
			if r.Header.Get("HX-Target") == "cart-count" {
				h.render(w, r, components.CartMutation(h.page(r, w)))
				return
			}
			if r.Header.Get("HX-Target") == "cart-layout" {
				h.render(w, r, pages.CartUpdate(h.page(r, w)))
				return
			}
			h.render(w, r, pages.Cart(h.page(r, w)))
			return
		}
		h.redirect(w, r, "/cart")
		return
	}
	if len(parts) == 3 && parts[1] == "wishlist" {
		source := r.FormValue("source")
		if source == "cart" {
			h.store.moveCartToWishlist(w, r, parts[2])
		} else {
			h.store.toggleWishlist(w, r, parts[2])
		}
		product, ok := productBySlug(parts[2])
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if source == "cart" {
			if r.Header.Get("HX-Request") == "true" {
				h.render(w, r, pages.CartUpdate(h.page(r, w)))
				return
			}
		}
		if r.Header.Get("HX-Request") == "true" {
			page := h.page(r, w)
			// SavedProduct mutations explicitly target the wishlist grid. Do not
			// infer that target from the current URL: recommendation cards also
			// render on /wishlist and target only their own article.
			if source == "wishlist" {
				h.render(w, r, pages.WishlistGridMutation(page))
				return
			}
			h.render(w, r, components.WishlistMutation(page, product, false))
			return
		}
		h.redirect(w, r, "/wishlist")
		return
	}
	if r.URL.Path == "/ui/checkout" {
		h.render(w, r, pages.CheckoutComplete(viewmodels.CustomerPage{Title: "Preview order placed", Notice: "This is a UI-only order preview. No payment or shipping request was sent."}))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (h *MockHandler) render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "preview-updated")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "rendering failed", http.StatusInternalServerError)
	}
}

func (h *MockHandler) page(r *http.Request, w http.ResponseWriter) viewmodels.CustomerPage {
	snapshot := h.store.snapshot(w, r)
	// Detail and checkout previews intentionally keep a representative line
	// item even if the cart preview was emptied, so those screens remain safe
	// to render while the cart itself can still show its empty state.
	if len(snapshot.cart) == 0 && (r.URL.Path == "/checkout" || strings.HasPrefix(r.URL.Path, "/orders/")) {
		snapshot.cart = seedCart()
	}
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	if r.URL.Path == "/deals" && category == "" {
		category = "offers"
	}
	sortMode := r.URL.Query().Get("sort")
	products := append([]viewmodels.Product(nil), mockProducts...)
	filterCounts := previewFilterCounts(mockProducts)
	if category == "offers" {
		filtered := make([]viewmodels.Product, 0, len(products))
		for _, product := range products {
			if product.Discount != "" {
				filtered = append(filtered, product)
			}
		}
		products = filtered
	} else if category != "" {
		filtered := make([]viewmodels.Product, 0, len(products))
		for _, product := range products {
			if strings.EqualFold(product.Category, category) || strings.Contains(strings.ToLower(product.Category), strings.ReplaceAll(strings.ToLower(category), "-", " ")) {
				filtered = append(filtered, product)
			}
		}
		products = filtered
	}
	if query != "" {
		filtered := make([]viewmodels.Product, 0, len(products))
		terms := strings.Fields(strings.ToLower(query))
		for _, product := range products {
			searchText := strings.ToLower(product.Name + " " + product.Category + " " + product.Seller + " " + product.Badge)
			matches := true
			for _, term := range terms {
				if !strings.Contains(searchText, term) {
					matches = false
					break
				}
			}
			if matches {
				filtered = append(filtered, product)
			}
		}
		products = filtered
	}
	products = filterProducts(products, r)
	sort.SliceStable(products, func(i, j int) bool {
		switch sortMode {
		case "price-asc":
			return previewMoney(products[i].Price) < previewMoney(products[j].Price)
		case "price-desc":
			return previewMoney(products[i].Price) > previewMoney(products[j].Price)
		case "newest":
			return products[i].Slug > products[j].Slug
		default:
			return false
		}
	})
	pageNumber := 1
	pageCount := 1
	if r.URL.Path == "/products" || r.URL.Path == "/search" || r.URL.Path == "/ui/listing" || r.URL.Path == "/ui/fragments/listing" {
		const pageSize = 8
		pageCount = (len(products) + pageSize - 1) / pageSize
		if pageCount == 0 {
			pageCount = 1
		}
		pageNumber, _ = strconv.Atoi(r.URL.Query().Get("page"))
		if pageNumber < 1 {
			pageNumber = 1
		}
		if pageNumber > pageCount {
			pageNumber = pageCount
		}
		start := (pageNumber - 1) * pageSize
		end := start + pageSize
		if start > len(products) {
			start = len(products)
		}
		if end > len(products) {
			end = len(products)
		}
		products = products[start:end]
	}
	wishlistProducts := make([]viewmodels.Product, 0, len(products))
	for _, product := range mockProducts {
		if snapshot.wishlist[product.Slug] {
			wishlistProducts = append(wishlistProducts, product)
		}
	}
	selectedOrder := viewmodels.Order{}
	if len(snapshot.orders) > 0 {
		selectedOrder = snapshot.orders[0]
	}
	if strings.HasPrefix(r.URL.Path, "/orders/") {
		requestedNumber, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/orders/"))
		if err == nil {
			requestedNumber = strings.TrimPrefix(strings.TrimSpace(requestedNumber), "#")
			for _, order := range snapshot.orders {
				if strings.TrimPrefix(strings.TrimSpace(order.Number), "#") == requestedNumber {
					selectedOrder = order
					break
				}
			}
		}
	}
	itemsTotal, shippingTotal, discountTotal, total := cartTotals(snapshot.cart)
	detailProduct := viewmodels.Product{}
	if strings.HasPrefix(r.URL.Path, "/products/") {
		detailProduct, _ = productBySlug(strings.TrimPrefix(r.URL.Path, "/products/"))
	}
	notice := r.URL.Query().Get("notice")
	if notice == "" && r.URL.Query().Get("coupon") != "" {
		notice = "Coupon preview applied: " + r.URL.Query().Get("coupon")
	}
	return viewmodels.CustomerPage{Route: r.URL.Path, Title: titleFor(r.URL.Path), Query: query, Sort: sortMode, Notice: notice, Category: category, Filters: r.URL.Query(), Brand: "WeeVCrafts", Tagline: "Handmade Today. A Kinder Tomorrow.", Description: "Authentic Indian arts, crafts and sarees, made with care.", Product: detailProduct, SavedProducts: wishlistProducts, Products: products, FilterCounts: filterCounts, Categories: mockCategories, Sellers: mockSellers, Brands: mockBrands, Cart: snapshot.cart, Orders: snapshot.orders, PaymentActivity: previewPaymentActivity(snapshot.orders), SelectedOrder: selectedOrder, Returns: mockReturns(), CartCount: cartCount(snapshot.cart), WishlistCount: len(wishlistProducts), ItemsTotal: itemsTotal, ShippingTotal: shippingTotal, DiscountTotal: discountTotal, Total: total, Page: pageNumber, Pages: pageCount, Mock: true, Authenticated: snapshot.identity.Role != "", SessionRole: string(snapshot.identity.Role), SessionName: snapshot.identity.Name, SessionEmail: snapshot.identity.Email, AuthRole: r.URL.Query().Get("role"), AuthEmail: r.URL.Query().Get("email"), DemoAccounts: demoAccounts(), Now: time.Now()}
}

func previewPaymentActivity(orders []viewmodels.Order) []viewmodels.PaymentActivity {
	activities := make([]viewmodels.PaymentActivity, 0, len(orders))
	for index, order := range orders {
		status := "Captured"
		if strings.EqualFold(strings.TrimSpace(order.Status), "Processing") {
			status = "Pending"
		} else if strings.EqualFold(strings.TrimSpace(order.Status), "Cancelled") {
			status = "Failed"
		}
		activities = append(activities, viewmodels.PaymentActivity{
			Reference:   fmt.Sprintf("#PAY-%02d", index+1),
			Amount:      order.Total,
			Date:        order.Date,
			Method:      order.Payment,
			OrderNumber: order.Number,
			Status:      status,
		})
	}
	return activities
}

func previewFilterCounts(products []viewmodels.Product) map[string]int {
	counts := make(map[string]int, 16)
	for _, product := range products {
		if strings.EqualFold(product.Category, "Sarees") {
			counts["category:sarees"]++
		}
		for _, brand := range []string{"handloom", "silk cotton", "chanderi"} {
			if matchesAny(product.Name+" "+product.Category+" "+product.Badge+" "+product.Seller, []string{brand}) {
				counts["brand:"+brand]++
			}
		}
		for _, rating := range []string{"4.5+", "4.0+"} {
			if matchesRating(product.Rating, []string{rating}) {
				counts["rating:"+rating]++
			}
		}
		for _, priceRange := range []string{"under-1000", "1000-2500", "2500-5000", "over-5000"} {
			if matchesPrice(previewMoney(product.Price), []string{priceRange}) {
				counts["price:"+priceRange]++
			}
		}
		if product.InStock {
			counts["availability:in-stock"]++
		}
		for _, location := range []string{"Madhya Pradesh", "Bihar", "Rajasthan"} {
			if strings.Contains(strings.ToLower(product.Location), strings.ToLower(location)) {
				counts["location:"+location]++
			}
		}
	}
	return counts
}

func filterProducts(products []viewmodels.Product, r *http.Request) []viewmodels.Product {
	values := r.URL.Query()
	brands := values["brand"]
	prices := values["price"]
	ratings := values["rating"]
	locations := values["location"]
	availability := values.Get("availability")
	filtered := make([]viewmodels.Product, 0, len(products))
	for _, product := range products {
		if len(brands) > 0 && !matchesAny(product.Name+" "+product.Category+" "+product.Badge+" "+product.Seller, brands) {
			continue
		}
		price := previewMoney(product.Price)
		if len(prices) > 0 && !matchesPrice(price, prices) {
			continue
		}
		if len(ratings) > 0 && !matchesRating(product.Rating, ratings) {
			continue
		}
		if len(locations) > 0 && !matchesAny(product.Location, locations) {
			continue
		}
		if availability == "in-stock" && !product.InStock {
			continue
		}
		filtered = append(filtered, product)
	}
	if len(brands) == 0 && len(prices) == 0 && len(ratings) == 0 && len(locations) == 0 && availability == "" {
		return products
	}
	return filtered
}

func matchesAny(value string, candidates []string) bool {
	lower := strings.ToLower(value)
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.ReplaceAll(candidate, "-", " "))
		if strings.Contains(lower, candidate) {
			return true
		}
	}
	return false
}

func matchesPrice(price int, ranges []string) bool {
	for _, value := range ranges {
		switch value {
		case "under-1000":
			if price < 1000 {
				return true
			}
		case "1000-2500":
			if price >= 1000 && price <= 2500 {
				return true
			}
		case "2500-5000":
			if price > 2500 && price <= 5000 {
				return true
			}
		case "over-5000":
			if price > 5000 {
				return true
			}
		}
	}
	return false
}

func matchesRating(rating string, ranges []string) bool {
	value, _ := strconv.ParseFloat(rating, 64)
	for _, candidate := range ranges {
		threshold, _ := strconv.ParseFloat(strings.TrimSuffix(candidate, "+"), 64)
		if value >= threshold {
			return true
		}
	}
	return false
}

func previewMoney(value string) int {
	clean := strings.NewReplacer("INR", "", ",", "", " ", "").Replace(value)
	amount, _ := strconv.Atoi(clean)
	return amount
}

func formatPreviewMoney(value int) string {
	digits := strconv.Itoa(value)
	if len(digits) <= 3 {
		return "INR " + digits
	}

	last := digits[len(digits)-3:]
	prefix := digits[:len(digits)-3]
	groups := make([]string, 0, (len(prefix)+1)/2+1)
	for len(prefix) > 2 {
		groups = append([]string{prefix[len(prefix)-2:]}, groups...)
		prefix = prefix[:len(prefix)-2]
	}
	groups = append([]string{prefix}, groups...)
	return fmt.Sprintf("INR %s,%s", strings.Join(groups, ","), last)
}

func cartTotals(items []viewmodels.CartItem) (itemsTotal, shippingTotal, discountTotal, total string) {
	itemsValue, compareValue := 0, 0
	for _, item := range items {
		itemsValue += previewMoney(item.Product.Price) * item.Quantity
		compareValue += previewMoney(item.Product.CompareAt) * item.Quantity
	}
	if len(items) > 0 {
		shippingValue := 100
		shippingTotal = formatPreviewMoney(shippingValue)
	} else {
		shippingTotal = formatPreviewMoney(0)
	}
	discountValue := compareValue - itemsValue
	if discountValue < 0 {
		discountValue = 0
	}
	totalValue := itemsValue + previewMoney(shippingTotal)
	return formatPreviewMoney(itemsValue), shippingTotal, formatPreviewMoney(discountValue), formatPreviewMoney(totalValue)
}

type sessionSnapshot struct {
	identity mockIdentity
	cart     []viewmodels.CartItem
	orders   []viewmodels.Order
	wishlist map[string]bool
}

func (s *mockStore) snapshot(w http.ResponseWriter, r *http.Request) sessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessionUnlocked(w, r)
	return sessionSnapshot{identity: session.Identity, cart: append([]viewmodels.CartItem(nil), session.Cart...), orders: append([]viewmodels.Order(nil), session.Orders...), wishlist: cloneWishlist(session.Wishlist)}
}

func cloneWishlist(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func titleFor(path string) string {
	switch path {
	case "/":
		return "Indian Arts, Crafts and Sarees"
	case "/categories":
		return "Explore Categories"
	case "/brands":
		return "Explore Brands"
	case "/wishlist":
		return "My Wishlist"
	case "/cart":
		return "Your Cart"
	case "/checkout":
		return "Checkout"
	case "/orders":
		return "My Orders"
	case "/tracking":
		return "Track your order"
	case "/deals":
		return "Handmade deals"
	case "/returns":
		return "Customer Returns & Refunds"
	case "/account":
		return "My Account"
	case "/login":
		return "Sign in"
	case "/register":
		return "Create an account"
	case "/forgot-password", "/reset-password", "/verify-email":
		return "Account recovery"
	case "/account/profile":
		return "Profile information"
	case "/account/addresses":
		return "Saved addresses"
	case "/account/notifications":
		return "Notification preferences"
	case "/account/security":
		return "Security and sessions"
	case "/account/privacy":
		return "Privacy preferences"
	case "/account/reviews":
		return "Reviews to write"
	case "/payments":
		return "Payment methods"
	case "/payments/success":
		return "Payment successful"
	case "/payments/failed":
		return "Payment failed"
	case "/payments/pending":
		return "Payment pending"
	case "/faq":
		return "Frequently asked questions"
	case "/contact":
		return "Contact WeeVCrafts"
	case "/shipping":
		return "Shipping information"
	case "/size-guide":
		return "Size guide"
	case "/terms":
		return "Terms and conditions"
	case "/privacy":
		return "Privacy policy"
	case "/about", "/our-story":
		return "Our story"
	case "/sustainability":
		return "Sustainability"
	case "/press":
		return "Press"
	case "/careers":
		return "Careers"
	default:
		return "WeeVCrafts"
	}
}

func (h *MockHandler) notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.render(w, r, pages.Empty(viewmodels.CustomerPage{Title: "Page not found", Notice: "This UI preview route does not exist."}))
}

func (h *MockHandler) redirect(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, http.StatusSeeOther)
}

var _ http.Handler = (*MockHandler)(nil)

package web

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func TestDefaultWorkspaceUsesPersistedAccountRoles(t *testing.T) {
	tests := []struct {
		name  string
		roles []domainidentity.Role
		want  string
	}{
		{name: "customer", roles: []domainidentity.Role{domainidentity.RoleCustomer}, want: "/account/sessions"},
		{name: "seller", roles: []domainidentity.Role{domainidentity.RoleSellerOwner}, want: "/seller"},
		{name: "support", roles: []domainidentity.Role{domainidentity.RoleSupportAgent}, want: "/support"},
		{name: "finance", roles: []domainidentity.Role{domainidentity.RoleFinanceOperator}, want: "/finance"},
		{name: "marketplace admin", roles: []domainidentity.Role{domainidentity.RoleMarketplaceAdmin}, want: "/admin"},
		{name: "admin takes precedence", roles: []domainidentity.Role{domainidentity.RoleSellerOwner, domainidentity.RoleSuperAdmin}, want: "/admin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := defaultWorkspace(test.roles); got != test.want {
				t.Fatalf("defaultWorkspace(%v) = %q, want %q", test.roles, got, test.want)
			}
		})
	}
}

type repository struct {
	products []domain.Product
}

type webCatalogManagementRepository struct {
	product     domaincommerce.ManagedProduct
	updatedWith domaincommerce.ProductDraftInput
}

type webOrderRepository struct {
	detail domaincommerce.OrderDetail
}

type webSellerStaffRepository struct {
	members          []domaincommerce.SellerStaffMember
	addedEmail       string
	addedPermissions []string
	removedUserID    int64
}

func (r *webSellerStaffRepository) ListStaff(context.Context, int64) ([]domaincommerce.SellerStaffMember, error) {
	return r.members, nil
}

func (r *webSellerStaffRepository) AddStaff(_ context.Context, _ int64, email string, permissions []string) (domaincommerce.SellerStaffMember, error) {
	r.addedEmail = email
	r.addedPermissions = permissions
	member := domaincommerce.SellerStaffMember{UserID: 17, Email: email, DisplayName: "Staff Account", Status: "active", Permissions: permissions}
	r.members = append(r.members, member)
	return member, nil
}

func (r *webSellerStaffRepository) RemoveStaff(_ context.Context, _ int64, staffUserID int64) error {
	r.removedUserID = staffUserID
	return nil
}

func (r *webCatalogManagementRepository) EnsureSeller(context.Context, int64, string) (domaincommerce.Seller, error) {
	return domaincommerce.Seller{ID: 1}, nil
}

func (r *webCatalogManagementRepository) GetSeller(context.Context, int64) (domaincommerce.Seller, error) {
	return domaincommerce.Seller{ID: 1, Status: "active", DisplayName: "Test seller"}, nil
}

func (r *webCatalogManagementRepository) CreateDraft(context.Context, int64, domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	return r.product, nil
}

func (r *webCatalogManagementRepository) GetSellerProduct(context.Context, int64, int64) (domaincommerce.ManagedProduct, error) {
	return r.product, nil
}

func (r *webCatalogManagementRepository) UpdateDraft(_ context.Context, _ int64, _ int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	r.updatedWith = input
	return r.product, nil
}

func (r *webCatalogManagementRepository) SubmitProduct(context.Context, int64, int64) error {
	return nil
}

func (r *webCatalogManagementRepository) ApproveProduct(context.Context, int64, int64, bool, string) error {
	return nil
}

func (r *webCatalogManagementRepository) ListSellerProducts(context.Context, int64) ([]domaincommerce.ManagedProduct, error) {
	return []domaincommerce.ManagedProduct{r.product}, nil
}

func (r *webCatalogManagementRepository) ListPendingProducts(context.Context, int64) ([]domaincommerce.ManagedProduct, error) {
	return nil, nil
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

func (r webOrderRepository) CreateOrder(context.Context, int64, int64, string, domaincommerce.AddressInput) (domaincommerce.Order, error) {
	return r.detail.Order, nil
}

func (r webOrderRepository) ListOrders(context.Context, int64, int) ([]domaincommerce.Order, error) {
	return []domaincommerce.Order{r.detail.Order}, nil
}

func (r webOrderRepository) GetOrder(context.Context, int64, string) (domaincommerce.OrderDetail, error) {
	return r.detail, nil
}

func (r webOrderRepository) CancelOrder(context.Context, int64, string) error { return nil }

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

func TestLiveDiscoveryRoutesUseApprovedCatalogueRecords(t *testing.T) {
	products := []domain.Product{
		{Slug: "linen-bag", Brand: "River Studio", Name: "Linen Bag", HasCompareAt: true, CompareAtCents: 2400, PriceCents: 1800},
		{Slug: "clay-cup", Brand: "Clay House", Name: "Clay Cup", PriceCents: 900},
	}
	handler, err := NewHandler(applicationcatalog.NewService(repository{products: products}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/deals", want: "Linen Bag"},
		{path: "/brands", want: "River Studio"},
		{path: "/brands/river-studio", want: "Linen Bag"},
	} {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("unexpected discovery response: status=%d want=%q body=%s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/brands/unknown", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown brand status = %d, want 404", missing.Code)
	}
}

func TestLiveCustomerOrderStatusRoutesUseServerState(t *testing.T) {
	session := ports.SessionRecord{ID: "customer-session", UserID: 42, CSRFToken: "customer-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	detail := domaincommerce.OrderDetail{
		Order:         domaincommerce.Order{OrderNumber: "WC-42", Status: "shipped", TotalCents: 2500, Currency: "INR"},
		PaymentStatus: "captured",
		Fulfillments:  []domaincommerce.OrderFulfillment{{SellerName: "River Studio", Status: "shipped", Carrier: "Shipline", TrackingNumber: "TRACK-42", UpdatedAt: time.Now()}},
	}
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{}), nil, applicationauth.NewSessionService(&webSessionStore{session: session}), nil, nil, applicationcommerce.NewOrderService(webOrderRepository{detail: detail}), nil, nil, false)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/orders/WC-42/payment-status", want: "Payment confirmed"},
		{path: "/orders/WC-42/tracking", want: "TRACK-42"},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("unexpected order status response: status=%d want=%q body=%s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}

func TestNetlifyManualCheckoutDisclosesOfflinePaymentAndTotal(t *testing.T) {
	session := ports.SessionRecord{ID: "checkout-session", UserID: 42, CSRFToken: "checkout-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	cartRepository := &webCartRepository{cart: domaincommerce.Cart{
		ID: 7, ItemCount: 2, SubtotalCents: 9900,
		Items: []domaincommerce.CartItem{{VariantID: 3, ProductName: "Handmade bowl", Quantity: 2, UnitPriceCents: 4950, LineTotalCents: 9900}},
	}}
	handler, err := NewFullHandler(
		applicationcatalog.NewService(repository{}), nil,
		applicationauth.NewSessionService(&webSessionStore{session: session}), nil,
		applicationcommerce.NewCartService(cartRepository),
		applicationcommerce.NewOrderService(webOrderRepository{}), nil, nil, false,
	)
	if err != nil {
		t.Fatalf("create checkout handler: %v", err)
	}
	handler.SetManualCheckout(true)
	request := httptest.NewRequest(http.MethodGet, "/checkout", nil)
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("checkout status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{"Offline payment on delivery", "No online payment is collected", "₹99.00", "₹198.00", "Place order"} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("manual checkout response does not contain %q", expected)
		}
	}
	if strings.Contains(recorder.Body.String(), "configured online provider") {
		t.Fatal("manual-payment checkout disclosed a nonexistent online provider")
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

func TestLivePageUsesLocalHTMXAndFragmentSafeShell(t *testing.T) {
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
	for _, expected := range []string{
		`src="/static/vendor/htmx.min.js?v=2.0.4"`,
		`code: "[45]..", swap: true, error: true`,
		`hx-boost="true"`,
		`hx-target="#page"`,
		`hx-select="#page"`,
		`hx-swap="outerHTML"`,
		`hx-indicator="#page-loading"`,
		`<a class="skip-link" href="#page" hx-boost="false">Skip to content</a>`,
		`<main id="page">`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in live page response", expected)
		}
	}
	for _, forbidden := range []string{"unpkg.com/htmx", "cdn.jsdelivr.net/npm/alpinejs", `hx-target="body"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("live page still contains forbidden dependency/target %q", forbidden)
		}
	}

	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/static/vendor/htmx.min.js", nil))
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "htmx") {
		t.Fatalf("local HTMX asset is not served: status=%d bytes=%d", asset.Code, asset.Body.Len())
	}
}

func TestLiveProductImagesExposeIntrinsicSizing(t *testing.T) {
	product := domain.Product{Slug: "sized-bowl", Name: "Sized Bowl", ImageURL: "/media/products/1"}
	handler, err := NewHandler(applicationcatalog.NewService(repository{products: []domain.Product{product}}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `width="640" height="800"`) || !strings.Contains(body, `sizes="(min-width: 1280px) 25vw`) {
		t.Fatal("product cards are missing intrinsic image sizing")
	}
}

func TestFullPageTemplatesDeclareFragmentShell(t *testing.T) {
	entries, err := fs.ReadDir(assets, "templates")
	if err != nil {
		t.Fatalf("read embedded templates: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "layout.html" || entry.Name() == "fragments.html" || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		path := "templates/" + entry.Name()
		content, readErr := fs.ReadFile(assets, path)
		if readErr != nil {
			t.Fatalf("read embedded template %s: %v", path, readErr)
		}
		body := string(content)
		if !strings.Contains(body, `hx-boost="true" hx-target="#page" hx-select="#page" hx-swap="outerHTML" hx-indicator="#page-loading"`) {
			t.Errorf("%s does not declare the shared HTMX shell", path)
		}
		if !strings.Contains(body, `<main id="page"`) {
			t.Errorf("%s does not declare a selectable page target", path)
		}
		if strings.Contains(body, `hx-target="body"`) {
			t.Errorf("%s overrides the shared fragment target with the document body", path)
		}
	}
}

func TestCategoryNavigationExposesCurrentPage(t *testing.T) {
	product := domain.Product{Slug: "soap", Name: "Soap", CategoryName: "Natural care"}
	handler, err := NewHandler(applicationcatalog.NewService(repository{products: []domain.Product{product}}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/category/natural-care", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `href="/category/natural-care" aria-current="page"`) {
		t.Fatalf("current category link is missing aria-current: %s", body)
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
	for _, expected := range []string{`name="q" value="soap"`, `name="category" value="natural-care"`, `value="price_desc" selected`, `hx-push-url="true"`, `aria-current="page"`, "Page 2", "Previous"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected search state %q in response: %s", expected, body)
		}
	}
}

func TestSearchSuggestionsReturnsCatalogueFragment(t *testing.T) {
	products := []domain.Product{{Slug: "neem-soap", Name: "Neem Soap", Brand: "Earth Studio", PriceCents: 12900}}
	searchService := applicationcatalog.NewSearchService(repository{products: products}, nil)
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{products: products}), nil, nil, nil, nil, nil, searchService, nil, false)
	if err != nil {
		t.Fatalf("create search handler: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/search/suggest?q=soap", nil)
	request.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected suggestions status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{`role="option"`, "/products/neem-soap", "Neem Soap", "Earth Studio", "View all results"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected suggestions response to contain %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "<!doctype html>") || strings.Contains(body, `<main id="page"`) {
		t.Fatalf("suggestions returned a full document instead of a fragment: %s", body)
	}
}

func TestSearchSuggestionsAvoidsBroadQueryForShortInput(t *testing.T) {
	products := []domain.Product{{Slug: "neem-soap", Name: "Neem Soap", Brand: "Earth Studio", PriceCents: 12900}}
	searchService := applicationcatalog.NewSearchService(repository{products: products}, nil)
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{products: products}), nil, nil, nil, nil, nil, searchService, nil, false)
	if err != nil {
		t.Fatalf("create search handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search/suggest?q=s", nil))
	if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != "" {
		t.Fatalf("short suggestion query should return an empty fragment, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

type webSessionStore struct {
	session ports.SessionRecord
}

func (s *webSessionStore) Create(_ context.Context, session ports.SessionRecord) error {
	s.session = session
	return nil
}

func (s *webSessionStore) Get(_ context.Context, id string) (ports.SessionRecord, error) {
	if id != s.session.ID {
		return ports.SessionRecord{}, ports.ErrOrderNotFound
	}
	return s.session, nil
}

func (s *webSessionStore) Delete(_ context.Context, id string) error {
	if id == s.session.ID {
		s.session = ports.SessionRecord{}
	}
	return nil
}

func (s *webSessionStore) DeleteAll(_ context.Context, userID int64) error {
	if userID == s.session.UserID {
		s.session = ports.SessionRecord{}
	}
	return nil
}

func (s *webSessionStore) List(_ context.Context, userID int64) ([]ports.SessionRecord, error) {
	if userID != s.session.UserID {
		return nil, nil
	}
	return []ports.SessionRecord{s.session}, nil
}

type webCartRepository struct {
	cart     domaincommerce.Cart
	wishlist []domaincommerce.WishlistItem
}

func (r *webCartRepository) GetOrCreateCart(context.Context, int64, string) (domaincommerce.Cart, error) {
	return r.cart, nil
}

func (r *webCartRepository) AddCartItem(_ context.Context, _ int64, variantID int64, quantity int) error {
	for index := range r.cart.Items {
		if r.cart.Items[index].VariantID == variantID {
			r.cart.Items[index].Quantity += quantity
			r.recalculate()
			return nil
		}
	}
	r.cart.Items = append(r.cart.Items, domaincommerce.CartItem{VariantID: variantID, ProductName: "Backend product", Quantity: quantity, UnitPriceCents: 1000, LineTotalCents: int64(quantity) * 1000})
	r.recalculate()
	return nil
}

func (r *webCartRepository) UpdateCartItem(_ context.Context, _ int64, variantID int64, quantity int) error {
	for index := range r.cart.Items {
		if r.cart.Items[index].VariantID == variantID {
			r.cart.Items[index].Quantity = quantity
			r.cart.Items[index].LineTotalCents = int64(quantity) * r.cart.Items[index].UnitPriceCents
			r.recalculate()
			return nil
		}
	}
	return nil
}

func (r *webCartRepository) RemoveCartItem(_ context.Context, _ int64, variantID int64) error {
	items := r.cart.Items[:0]
	for _, item := range r.cart.Items {
		if item.VariantID != variantID {
			items = append(items, item)
		}
	}
	r.cart.Items = items
	r.recalculate()
	return nil
}

func (r *webCartRepository) MergeGuestCart(context.Context, int64, string) error { return nil }

func (r *webCartRepository) GetCart(context.Context, int64) (domaincommerce.Cart, error) {
	return r.cart, nil
}

func (r *webCartRepository) AddWishlist(_ context.Context, userID, productID int64) error {
	for _, item := range r.wishlist {
		if item.ProductID == productID {
			return nil
		}
	}
	r.wishlist = append(r.wishlist, domaincommerce.WishlistItem{ProductID: productID, Name: "Saved product"})
	return nil
}

func (r *webCartRepository) MoveCartItemToWishlist(_ context.Context, cartID, userID, variantID, productID int64) error {
	if err := r.AddWishlist(context.Background(), userID, productID); err != nil {
		return err
	}
	return r.RemoveCartItem(context.Background(), cartID, variantID)
}

func (r *webCartRepository) RemoveWishlist(_ context.Context, _ int64, productID int64) error {
	items := r.wishlist[:0]
	for _, item := range r.wishlist {
		if item.ProductID != productID {
			items = append(items, item)
		}
	}
	r.wishlist = items
	return nil
}

func (r *webCartRepository) ListWishlist(context.Context, int64) ([]domaincommerce.WishlistItem, error) {
	return r.wishlist, nil
}

func (r *webCartRepository) recalculate() {
	r.cart.ItemCount = 0
	r.cart.SubtotalCents = 0
	for _, item := range r.cart.Items {
		r.cart.ItemCount += item.Quantity
		r.cart.SubtotalCents += item.LineTotalCents
	}
}

func newCommerceHandlerForTest(t *testing.T, cartRepository *webCartRepository) (*Handler, ports.SessionRecord) {
	t.Helper()
	session := ports.SessionRecord{ID: "web-session", UserID: 42, CSRFToken: "web-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	sessions := applicationauth.NewSessionService(&webSessionStore{session: session})
	handler, err := NewFullHandler(
		applicationcatalog.NewService(repository{}), nil, sessions, nil,
		applicationcommerce.NewCartService(cartRepository), nil, nil, nil, false,
	)
	if err != nil {
		t.Fatalf("create commerce handler: %v", err)
	}
	return handler, session
}

func TestSellerDraftEditRendersManagedValuesAndHTMXContract(t *testing.T) {
	managementRepository := &webCatalogManagementRepository{product: domaincommerce.ManagedProduct{
		ID: 7, Slug: "handmade-soap", BrandSlug: "earth-studio", BrandName: "Earth Studio",
		Name: "Neem Soap", Description: "Handmade botanical soap", SKU: "SOAP-001",
		PriceCents: 12900, CompareAtCents: 14900, HasCompareAt: true, Stock: 10,
		CategorySlug: "natural-care", CategoryName: "Natural Care", DeliveryLabel: "Dispatches in 2–4 days", Status: "draft",
	}}
	session := ports.SessionRecord{ID: "seller-session", UserID: 42, CSRFToken: "seller-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	sessions := applicationauth.NewSessionService(&webSessionStore{session: session})
	handler, err := NewFullHandler(
		applicationcatalog.NewService(repository{}), nil, sessions,
		applicationcommerce.NewCatalogService(managementRepository), nil, nil, nil, nil, false,
	)
	if err != nil {
		t.Fatalf("create seller handler: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/seller/products/7/edit", nil)
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected seller edit status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{
		"Edit product draft", `action="/seller/products/7/edit"`, `name="name" value="Neem Soap"`,
		`name="compare_at_cents"`, `name="delivery_label"`, `hx-target="#page"`,
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("expected seller edit response to contain %q: %s", expected, recorder.Body.String())
		}
	}

	request = commerceRequest(http.MethodPost, "/seller/products/7/edit", url.Values{
		"csrf_token": {session.CSRFToken}, "slug": {"updated-soap"}, "brand_slug": {"earth-studio"},
		"brand_name": {"Earth Studio"}, "category_slug": {"natural-care"}, "category_name": {"Natural Care"},
		"name": {"Updated Soap"}, "description": {"Updated description"}, "price_cents": {"13900"},
		"compare_at_cents": {"15900"}, "sku": {"SOAP-002"}, "initial_stock": {"12"},
		"delivery_label": {"Dispatches in 3–5 days"},
	}, session)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("HX-Redirect") != "/seller" {
		t.Fatalf("expected HTMX seller edit redirect, got %d %q: %s", recorder.Code, recorder.Header().Get("HX-Redirect"), recorder.Body.String())
	}
	if managementRepository.updatedWith.Name != "Updated Soap" || managementRepository.updatedWith.PriceCents != 13900 || managementRepository.updatedWith.InitialStock != 12 {
		t.Fatalf("repository received incomplete draft update: %+v", managementRepository.updatedWith)
	}
}

func TestSellerProductID(t *testing.T) {
	if productID, err := sellerProductID("/seller/products/42/edit"); err != nil || productID != 42 {
		t.Fatalf("expected product id 42, got %d and %v", productID, err)
	}
	if _, err := sellerProductID("/seller/products/not-a-number/edit"); err == nil {
		t.Fatal("expected malformed seller product path to fail")
	}
}

func TestSellerTeamRendersAndMutatesThroughHTMX(t *testing.T) {
	staffRepository := &webSellerStaffRepository{members: []domaincommerce.SellerStaffMember{{
		UserID: 17, Email: "staff@example.com", DisplayName: "Staff Account", Status: "active",
		Permissions: []string{domaincommerce.SellerPermissionProductRead},
	}}}
	session := ports.SessionRecord{ID: "team-session", UserID: 42, CSRFToken: "team-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	sessions := applicationauth.NewSessionService(&webSessionStore{session: session})
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{}), nil, sessions, nil, nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("create team handler: %v", err)
	}
	handler.SetSellerStaffService(applicationcommerce.NewSellerStaffService(staffRepository, nil))

	request := httptest.NewRequest(http.MethodGet, "/seller/team", nil)
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected seller team status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{"Seller team", "staff@example.com", "PRODUCT_READ", `action="/seller/team/add"`, `name="permissions"`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("expected seller team response to contain %q: %s", expected, recorder.Body.String())
		}
	}

	request = commerceRequest(http.MethodPost, "/seller/team/add", url.Values{
		"csrf_token": {session.CSRFToken}, "email": {"new@example.com"},
		"permissions": {domaincommerce.SellerPermissionProductRead, domaincommerce.SellerPermissionOrderRead},
	}, session)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("HX-Redirect") != "/seller/team" {
		t.Fatalf("expected team add redirect, got %d %q: %s", recorder.Code, recorder.Header().Get("HX-Redirect"), recorder.Body.String())
	}
	if staffRepository.addedEmail != "new@example.com" || len(staffRepository.addedPermissions) != 2 {
		t.Fatalf("team add did not reach repository: email=%q permissions=%#v", staffRepository.addedEmail, staffRepository.addedPermissions)
	}
}

func commerceRequest(method, target string, values url.Values, session ports.SessionRecord) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
	request.Header.Set("HX-Request", "true")
	return request
}

func TestCartMutationReturnsBackendFragmentAndOOBState(t *testing.T) {
	cartRepository := &webCartRepository{cart: domaincommerce.Cart{ID: 7, Items: []domaincommerce.CartItem{{VariantID: 11, ProductSlug: "real-product", ProductName: "Real product", SKU: "REAL-11", Quantity: 1, UnitPriceCents: 1250, LineTotalCents: 1250}}}}
	cartRepository.recalculate()
	handler, session := newCommerceHandlerForTest(t, cartRepository)

	request := commerceRequest(http.MethodPost, "/cart/update", url.Values{
		"csrf_token": {session.CSRFToken}, "variant_id": {"11"}, "quantity": {"3"},
	}, session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fragment status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{`id="cart-content"`, `value="3"`, `hx-swap-oob="outerHTML"`, "Cart (3)", "Cart quantity updated.", `hx-target="#cart-content"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected cart fragment to contain %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "<!doctype html>") || strings.Contains(body, `<main id="page"`) {
		t.Fatalf("cart mutation returned a full document instead of a fragment: %s", body)
	}
}

func TestCartAddReturnsBackendCountAndFeedback(t *testing.T) {
	cartRepository := &webCartRepository{cart: domaincommerce.Cart{ID: 8}}
	handler, session := newCommerceHandlerForTest(t, cartRepository)

	request := commerceRequest(http.MethodPost, "/cart/add", url.Values{
		"csrf_token": {session.CSRFToken}, "variant_id": {"12"}, "quantity": {"2"},
	}, session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fragment status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{`id="cart-feedback"`, "Added to your cart.", `id="cart-count"`, "Cart (2)", `id="live-toast"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected cart add response to contain %q: %s", expected, body)
		}
	}
}

func TestCartMoveToWishlistReturnsUpdatedCartFragment(t *testing.T) {
	cartRepository := &webCartRepository{cart: domaincommerce.Cart{ID: 9, Items: []domaincommerce.CartItem{{
		VariantID: 12, ProductID: 44, ProductName: "Handmade bowl", Quantity: 1, UnitPriceCents: 2400, LineTotalCents: 2400,
	}}}}
	cartRepository.recalculate()
	handler, session := newCommerceHandlerForTest(t, cartRepository)

	request := commerceRequest(http.MethodPost, "/cart/move-to-wishlist", url.Values{
		"csrf_token": {session.CSRFToken}, "variant_id": {"12"}, "product_id": {"44"},
	}, session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected move fragment status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{`id="cart-content"`, "Your cart is empty", "Moved to your wishlist.", `hx-swap-oob="outerHTML"`, "Cart (0)"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected move response to contain %q: %s", expected, body)
		}
	}
	if len(cartRepository.wishlist) != 1 || cartRepository.wishlist[0].ProductID != 44 {
		t.Fatalf("expected product 44 in wishlist, got %#v", cartRepository.wishlist)
	}
	if strings.Contains(body, "<!doctype html>") {
		t.Fatalf("move response returned a full document instead of a fragment: %s", body)
	}
}

func TestWishlistAddReturnsSmallHTMXFeedbackFragment(t *testing.T) {
	handler, session := newCommerceHandlerForTest(t, &webCartRepository{})

	request := commerceRequest(http.MethodPost, "/wishlist/add", url.Values{
		"csrf_token": {session.CSRFToken}, "product_id": {"9"},
	}, session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fragment status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if body != "<span>Saved to your wishlist.</span>" {
		t.Fatalf("wishlist add returned an unexpected payload: %s", body)
	}
	if strings.Contains(body, "<!doctype html>") || strings.Contains(body, "wishlist-content") {
		t.Fatalf("wishlist add returned a page fragment instead of feedback: %s", body)
	}
}

func TestWishlistRemoveReturnsUpdatedBackendFragment(t *testing.T) {
	cartRepository := &webCartRepository{wishlist: []domaincommerce.WishlistItem{
		{ProductID: 1, Slug: "keep-me", Name: "Keep me", Brand: "Maker one", PriceCents: 1000},
		{ProductID: 2, Slug: "remove-me", Name: "Remove me", Brand: "Maker two", PriceCents: 1200},
	}}
	handler, session := newCommerceHandlerForTest(t, cartRepository)

	request := commerceRequest(http.MethodPost, "/wishlist/remove", url.Values{
		"csrf_token": {session.CSRFToken}, "product_id": {"2"},
	}, session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fragment status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{`id="wishlist-content"`, "Keep me", "Removed from your wishlist.", `hx-swap-oob="innerHTML"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected wishlist fragment to contain %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "Remove me") {
		t.Fatalf("removed wishlist record leaked into response: %s", body)
	}
}

func TestProductCardAndDetailUseFragmentSafeCartContract(t *testing.T) {
	product := domain.Product{ID: 9, VariantID: 12, Slug: "real-product", Name: "Real product", Available: 4}
	handler, err := NewHandler(applicationcatalog.NewService(repository{products: []domain.Product{product}}))
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}

	for _, target := range []string{"/", "/products/real-product"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s status 200, got %d", target, recorder.Code)
		}
		body := recorder.Body.String()
		for _, expected := range []string{`hx-target="#live-toast"`, `hx-select="#cart-feedback"`, `hx-indicator="#page-loading"`} {
			if !strings.Contains(body, expected) {
				t.Fatalf("%s is missing %q: %s", target, expected, body)
			}
		}
	}
}

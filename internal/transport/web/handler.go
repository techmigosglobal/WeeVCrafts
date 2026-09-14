package web

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"errors"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	applicationprivacy "github.com/wecratfs/commerce/internal/application/privacy"
	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	domainprivacy "github.com/wecratfs/commerce/internal/domain/privacy"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

//go:embed templates/*.html templates/partials/*.html static/*
var assets embed.FS

type Handler struct {
	catalog             *applicationcatalog.Service
	auth                *applicationauth.Service
	roleAdminService    *applicationauth.RoleAdminService
	sessions            *applicationauth.SessionService
	commerceCatalog     *applicationcommerce.CatalogService
	sellerStaff         *applicationcommerce.SellerStaffService
	inventoryService    *applicationcommerce.InventoryService
	sellerOrderService  *applicationcommerce.SellerOrderService
	sellerReportService *applicationcommerce.SellerReportService
	sellerAdminService  *applicationcommerce.SellerAdminService
	auditService        *applicationcommerce.AuditService
	adminOperations     *applicationcommerce.AdminOperationsService
	brandDirectory      *applicationcommerce.BrandDirectoryService
	customerDirectory   *applicationauth.CustomerDirectoryService
	financeService      *applicationcommerce.FinanceService
	supportService      *applicationcommerce.SupportService
	returnService       *applicationcommerce.ReturnService
	cartService         *applicationcommerce.CartService
	orderService        *applicationcommerce.OrderService
	paymentService      *applicationpayment.OrderService
	refundService       *applicationpayment.RefundService
	recoveryService     *applicationauth.RecoveryService
	paymentPublicKey    string
	searchService       *applicationcatalog.SearchService
	privacyService      *applicationprivacy.Service
	mediaService        *applicationcommerce.MediaService
	secureCookies       bool
	templates           *template.Template
	static              http.Handler
}

type pageData struct {
	Brand              string
	Tagline            string
	Description        string
	Title              string
	Category           string
	SearchQuery        string
	SearchCategory     string
	SearchSort         string
	SearchPage         int
	SearchHasNext      bool
	SearchHasPrevious  bool
	SearchNextURL      string
	SearchPreviousURL  string
	SearchFacets       []ports.SearchFacet
	Products           []domain.Product
	Brands             []BrandSummary
	BrandName          string
	BrandSlug          string
	Product            domain.Product
	HasProduct         bool
	Error              string
	Authenticated      bool
	AccountUserID      int64
	CSRFToken          string
	Sessions           []ports.SessionRecord
	FormEmail          string
	FormName           string
	Cart               domaincommerce.Cart
	HasCart            bool
	ManagedProducts    []domaincommerce.ManagedProduct
	ManagedProduct     domaincommerce.ManagedProduct
	HasManagedProduct  bool
	StaffMembers       []domaincommerce.SellerStaffMember
	Inventory          []domaincommerce.InventoryItem
	SellerOrders       []domaincommerce.SellerOrder
	SellerReport       domaincommerce.SellerReport
	SellerApplications []domaincommerce.SellerAdminEntry
	AuditEntries       []domaincommerce.AuditEntry
	RoleAssignments    []domainidentity.RoleAssignment
	AdminOrders        []domaincommerce.AdminOrder
	AdminBrands        []domaincommerce.AdminBrand
	AdminCustomers     []domainidentity.AdminCustomer
	FinanceEntries     []domaincommerce.FinanceEntry
	FinanceView        string
	SupportTickets     []domaincommerce.SupportTicket
	SupportAgent       bool
	SupportSeller      bool
	Returns            []domaincommerce.ReturnRequest
	ReturnRequest      domaincommerce.ReturnRequest
	HasReturnRequest   bool
	ReturnEnabled      bool
	SellerProfile      domaincommerce.Seller
	HasSellerProfile   bool
	PendingProducts    []domaincommerce.ManagedProduct
	Orders             []domaincommerce.Order
	OrderDetail        domaincommerce.OrderDetail
	HasOrderDetail     bool
	PaymentIntent      domaincommerce.PaymentIntent
	HasPayment         bool
	PaymentPublicKey   string
	RefundKey          string
	ActionToken        string
	RecoveryEnabled    bool
	Wishlist           []domaincommerce.WishlistItem
	Notice             string
	PrivacyCenter      domainprivacy.Center
}

type productCardData struct {
	Product   domain.Product
	CSRFToken string
}

func NewHandler(catalogService *applicationcatalog.Service) (*Handler, error) {
	return newHandler(catalogService, nil, nil, nil, nil, nil, nil, nil, nil, false)
}

func NewHandlerWithAuth(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, secureCookies bool) (*Handler, error) {
	return newHandler(catalogService, authService, sessionService, nil, nil, nil, nil, nil, nil, secureCookies)
}

func NewFullHandler(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, secureCookies bool) (*Handler, error) {
	return newHandler(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, nil, searchService, privacyService, secureCookies)
}

func NewFullHandlerWithPayment(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, secureCookies bool) (*Handler, error) {
	return newHandler(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentService, searchService, privacyService, secureCookies)
}

func NewFullHandlerWithPaymentConfig(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, secureCookies bool, paymentPublicKey string) (*Handler, error) {
	handler, err := NewFullHandlerWithPayment(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentService, searchService, privacyService, secureCookies)
	if err != nil {
		return nil, err
	}
	handler.paymentPublicKey = paymentPublicKey
	return handler, nil
}

func NewFullHandlerWithPaymentConfigAndMedia(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, secureCookies bool, paymentPublicKey string) (*Handler, error) {
	handler, err := NewFullHandlerWithPaymentConfig(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentService, searchService, privacyService, secureCookies, paymentPublicKey)
	if err != nil {
		return nil, err
	}
	handler.mediaService = mediaService
	return handler, nil
}

func NewFullHandlerWithPaymentConfigAndMediaAndRefund(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, refundService *applicationpayment.RefundService, secureCookies bool, paymentPublicKey string) (*Handler, error) {
	handler, err := NewFullHandlerWithPaymentConfigAndMedia(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentService, searchService, privacyService, mediaService, secureCookies, paymentPublicKey)
	if err != nil {
		return nil, err
	}
	handler.refundService = refundService
	return handler, nil
}

func NewFullHandlerWithPaymentConfigMediaRefundAndRecovery(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, refundService *applicationpayment.RefundService, recoveryService *applicationauth.RecoveryService, secureCookies bool, paymentPublicKey string) (*Handler, error) {
	handler, err := NewFullHandlerWithPaymentConfigAndMediaAndRefund(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentService, searchService, privacyService, mediaService, refundService, secureCookies, paymentPublicKey)
	if err != nil {
		return nil, err
	}
	handler.recoveryService = recoveryService
	return handler, nil
}

func newHandler(catalogService *applicationcatalog.Service, authService *applicationauth.Service, sessionService *applicationauth.SessionService, commerceCatalog *applicationcommerce.CatalogService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, paymentService *applicationpayment.OrderService, searchService *applicationcatalog.SearchService, privacyService *applicationprivacy.Service, secureCookies bool) (*Handler, error) {
	templates, err := template.New("pages").Funcs(template.FuncMap{
		"money": func(cents int64) string {
			return "₹" + strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
		},
		"rating": func(value float64) string {
			return strconv.FormatFloat(value, 'f', 1, 64)
		},
		"date": func(value time.Time) string {
			return value.Local().Format("2 Jan 2006, 15:04")
		},
		"urlquery": url.QueryEscape,
		"productCard": func(product domain.Product, csrfToken string) productCardData {
			return productCardData{Product: product, CSRFToken: csrfToken}
		},
		"hasPermission": func(permissions []string, wanted string) bool {
			for _, permission := range permissions {
				if permission == wanted {
					return true
				}
			}
			return false
		},
	}).ParseFS(assets, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	return &Handler{
		catalog:         catalogService,
		auth:            authService,
		sessions:        sessionService,
		commerceCatalog: commerceCatalog,
		cartService:     cartService,
		orderService:    orderService,
		paymentService:  paymentService,
		searchService:   searchService,
		privacyService:  privacyService,
		secureCookies:   secureCookies,
		templates:       templates,
		static:          http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/static/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.static.ServeHTTP(w, r)
		return
	}

	switch {
	case r.URL.Path == "/login":
		h.login(w, r)
	case r.URL.Path == "/register":
		h.register(w, r)
	case r.URL.Path == "/forgot-password":
		h.forgotPassword(w, r)
	case r.URL.Path == "/reset-password":
		h.resetPassword(w, r)
	case r.URL.Path == "/verify-email":
		h.verifyEmail(w, r)
	case r.URL.Path == "/account/verify-email":
		h.requestEmailVerification(w, r)
	case r.URL.Path == "/logout":
		h.logout(w, r)
	case r.URL.Path == "/account/sessions":
		h.sessionsPage(w, r)
	case r.URL.Path == "/privacy":
		h.privacyPage(w, r)
	case r.URL.Path == "/privacy/center":
		h.privacyCenter(w, r)
	case r.URL.Path == "/privacy/consent":
		h.privacyConsent(w, r)
	case r.URL.Path == "/privacy/requests":
		h.privacyRequest(w, r)
	case r.URL.Path == "/privacy/delete":
		h.privacyDelete(w, r)
	case r.URL.Path == "/cart":
		h.cartPage(w, r)
	case r.URL.Path == "/cart/add":
		h.cartAdd(w, r)
	case r.URL.Path == "/cart/update":
		h.cartUpdate(w, r)
	case r.URL.Path == "/cart/remove":
		h.cartRemove(w, r)
	case r.URL.Path == "/cart/move-to-wishlist":
		h.cartMoveToWishlist(w, r)
	case r.URL.Path == "/wishlist":
		h.wishlistPage(w, r)
	case r.URL.Path == "/wishlist/add":
		h.wishlistAdd(w, r)
	case r.URL.Path == "/wishlist/remove":
		h.wishlistRemove(w, r)
	case r.URL.Path == "/checkout":
		h.checkout(w, r)
	case r.URL.Path == "/orders":
		h.ordersPage(w, r)
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/payment-status"):
		h.orderPaymentStatus(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/payment-status"))
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/tracking"):
		h.orderTracking(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/tracking"))
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/payment"):
		h.orderPayment(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/payment"))
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/refund"):
		h.orderRefund(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/refund"))
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/return"):
		h.orderReturn(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/return"))
	case strings.HasPrefix(r.URL.Path, "/orders/") && strings.HasSuffix(r.URL.Path, "/cancel"):
		h.orderCancel(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/orders/"), "/cancel"))
	case strings.HasPrefix(r.URL.Path, "/orders/"):
		h.orderPage(w, r, strings.TrimPrefix(r.URL.Path, "/orders/"))
	case strings.HasPrefix(r.URL.Path, "/media/products/"):
		h.publicMedia(w, r, strings.TrimPrefix(r.URL.Path, "/media/products/"))
	case r.URL.Path == "/seller":
		h.sellerPage(w, r)
	case r.URL.Path == "/seller/onboard":
		h.sellerOnboard(w, r)
	case r.URL.Path == "/seller/products/new":
		h.sellerNewProduct(w, r)
	case r.URL.Path == "/seller/products":
		h.sellerProducts(w, r)
	case r.URL.Path == "/seller/products/submit":
		h.sellerSubmitProduct(w, r)
	case r.URL.Path == "/seller/team":
		h.sellerTeam(w, r)
	case r.URL.Path == "/seller/team/add":
		h.sellerTeamAdd(w, r)
	case r.URL.Path == "/seller/team/remove":
		h.sellerTeamRemove(w, r)
	case r.URL.Path == "/seller/inventory":
		h.sellerInventory(w, r)
	case r.URL.Path == "/seller/inventory/adjust":
		h.sellerInventoryAdjust(w, r)
	case r.URL.Path == "/seller/orders":
		h.sellerOrders(w, r)
	case r.URL.Path == "/seller/analytics":
		h.sellerAnalytics(w, r)
	case r.URL.Path == "/seller/audit":
		h.sellerAudit(w, r)
	case r.URL.Path == "/seller/orders/fulfill":
		h.sellerOrderFulfill(w, r)
	case r.URL.Path == "/admin/sellers":
		h.adminSellers(w, r)
	case r.URL.Path == "/admin/sellers/status":
		h.adminSellerStatus(w, r)
	case r.URL.Path == "/admin/audit" || r.URL.Path == "/admin/audit-logs":
		h.adminAudit(w, r)
	case r.URL.Path == "/admin/roles":
		h.adminRoles(w, r)
	case r.URL.Path == "/admin/roles/update":
		h.adminRoleUpdate(w, r)
	case r.URL.Path == "/admin/orders":
		h.adminOrders(w, r)
	case r.URL.Path == "/admin/brands":
		h.adminBrands(w, r)
	case r.URL.Path == "/admin/customers":
		h.adminCustomers(w, r)
	case r.URL.Path == "/finance" || r.URL.Path == "/admin/payments" || r.URL.Path == "/admin/failed-payments" || r.URL.Path == "/admin/refunds" || r.URL.Path == "/admin/finance-reconciliation":
		h.financePage(w, r)
	case r.URL.Path == "/support":
		h.supportPage(w, r)
	case r.URL.Path == "/support/create":
		h.supportCreate(w, r)
	case r.URL.Path == "/support/status":
		h.supportStatus(w, r)
	case r.URL.Path == "/seller/support":
		h.supportPage(w, r)
	case r.URL.Path == "/seller/support/create":
		h.supportCreate(w, r)
	case strings.HasPrefix(r.URL.Path, "/seller/products/") && strings.HasSuffix(r.URL.Path, "/edit"):
		h.sellerEditProduct(w, r)
	case r.URL.Path == "/admin/products":
		h.adminProducts(w, r)
	case r.URL.Path == "/admin/products/review":
		h.adminReviewProduct(w, r)
	case r.URL.Path == "/admin/returns":
		h.adminReturns(w, r)
	case r.URL.Path == "/admin/returns/status":
		h.adminReturnStatus(w, r)
	case r.URL.Path == "/":
		h.home(w, r, "")
	case r.URL.Path == "/products":
		h.home(w, r, r.URL.Query().Get("category"))
	case r.URL.Path == "/search":
		h.search(w, r)
	case r.URL.Path == "/search/suggest":
		h.searchSuggestions(w, r)
	case r.URL.Path == "/deals":
		h.deals(w, r)
	case r.URL.Path == "/brands":
		h.brands(w, r)
	case strings.HasPrefix(r.URL.Path, "/brands/"):
		h.brand(w, r, strings.TrimPrefix(r.URL.Path, "/brands/"))
	case r.URL.Path == "/shipping":
		h.policy(w, "Shipping", "Shipping details will be published with the first approved catalogue and fulfilment workflow.")
	case r.URL.Path == "/returns":
		h.policy(w, "Returns", "Returns details will be published before checkout is enabled. No order workflow is active in this preview.")
	case r.URL.Path == "/contact":
		h.policy(w, "Contact", "Contact details will be published when the seller and customer-care workflow is connected.")
	case strings.HasPrefix(r.URL.Path, "/category/"):
		h.home(w, r, strings.TrimPrefix(r.URL.Path, "/category/"))
	case strings.HasPrefix(r.URL.Path, "/products/"):
		h.product(w, r, strings.TrimPrefix(r.URL.Path, "/products/"))
	default:
		h.notFound(w, r)
	}
}

// SetSellerStaffService attaches the seller-owner team-management workflow
// without changing the existing handler constructors used by isolated tests.
func (h *Handler) SetSellerStaffService(service *applicationcommerce.SellerStaffService) {
	h.sellerStaff = service
}

// SetInventoryService attaches the live seller stock workflow without
// changing the existing constructors used by isolated web tests.
func (h *Handler) SetInventoryService(service *applicationcommerce.InventoryService) {
	h.inventoryService = service
}

// SetSellerOrderService attaches seller-scoped fulfilment without changing
// the existing handler constructors used by isolated web tests.
func (h *Handler) SetSellerOrderService(service *applicationcommerce.SellerOrderService) {
	h.sellerOrderService = service
}

// SetSellerReportService attaches seller-scoped operational reporting without
// changing the existing handler constructors used by isolated web tests.
func (h *Handler) SetSellerReportService(service *applicationcommerce.SellerReportService) {
	h.sellerReportService = service
}

// SetSellerAdminService attaches marketplace seller approval and suspension
// workflows without changing constructors used by isolated web tests.
func (h *Handler) SetSellerAdminService(service *applicationcommerce.SellerAdminService) {
	h.sellerAdminService = service
}

// SetAuditService attaches the role-scoped audit-log read workflow without
// changing constructors used by isolated web tests.
func (h *Handler) SetAuditService(service *applicationcommerce.AuditService) {
	h.auditService = service
}

// SetRoleAdminService attaches the Super Admin role-assignment workflow
// without changing constructors used by isolated web tests.
func (h *Handler) SetRoleAdminService(service *applicationauth.RoleAdminService) {
	h.roleAdminService = service
}

// SetAdminOperationsService attaches the masked marketplace order queue
// without changing constructors used by isolated web tests.
func (h *Handler) SetAdminOperationsService(service *applicationcommerce.AdminOperationsService) {
	h.adminOperations = service
}

// SetBrandDirectoryService attaches the role-scoped admin brand directory
// without changing the existing handler constructors used by isolated tests.
func (h *Handler) SetBrandDirectoryService(service *applicationcommerce.BrandDirectoryService) {
	h.brandDirectory = service
}

// SetCustomerDirectoryService attaches the masked, role-scoped customer
// directory without changing the existing handler constructors used by
// isolated tests.
func (h *Handler) SetCustomerDirectoryService(service *applicationauth.CustomerDirectoryService) {
	h.customerDirectory = service
}

// SetFinanceService attaches the read-only finance workspace without changing
// the existing handler constructors used by isolated web tests.
func (h *Handler) SetFinanceService(service *applicationcommerce.FinanceService) {
	h.financeService = service
}

// SetSupportService attaches the masked customer/support workflow without
// changing the existing handler constructors used by isolated web tests.
func (h *Handler) SetSupportService(service *applicationcommerce.SupportService) {
	h.supportService = service
}

// SetReturnService attaches customer return requests and the operations
// review queue without changing constructors used by isolated web tests.
func (h *Handler) SetReturnService(service *applicationcommerce.ReturnService) {
	h.returnService = service
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request, category string) {
	products, err := h.catalog.List(r.Context(), category, 24)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Catalog unavailable", "We could not reach the product catalogue. Please try again when the local PostgreSQL service is ready.")
		return
	}
	data := pageData{
		Brand:       identity.Name,
		Tagline:     identity.Tagline,
		Description: identity.Description,
		Title:       identity.Name + " · " + identity.Tagline,
		Category:    category,
		Products:    products,
	}
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.decorateSession(r, &data)
	h.render(w, http.StatusOK, "home", data)
}

func (h *Handler) product(w http.ResponseWriter, r *http.Request, slug string) {
	if slug == "" || strings.Contains(slug, "/") {
		h.notFound(w, r)
		return
	}
	product, err := h.catalog.Get(r.Context(), slug)
	if errors.Is(err, applicationcatalog.ErrProductNotFound) {
		h.notFound(w, r)
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Product unavailable", "We could not load this product from PostgreSQL.")
		return
	}
	data := pageData{
		Brand:       identity.Name,
		Tagline:     identity.Tagline,
		Description: identity.Description,
		Title:       product.Name + " · " + identity.Name,
		Product:     product,
		HasProduct:  true,
	}
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "product", data)
}

func (h *Handler) notFound(w http.ResponseWriter, _ *http.Request) {
	h.renderError(w, http.StatusNotFound, "Page not found", "This page does not exist or the product is not publicly available.")
}

func (h *Handler) renderError(w http.ResponseWriter, status int, title, message string) {
	h.render(w, status, "error", pageData{
		Brand:       identity.Name,
		Tagline:     identity.Tagline,
		Description: identity.Description,
		Title:       title + " · " + identity.Name,
		Error:       message,
	})
}

func (h *Handler) policy(w http.ResponseWriter, title, message string) {
	data := pageData{
		Brand:       identity.Name,
		Tagline:     identity.Tagline,
		Description: identity.Description,
		Title:       title + " · " + identity.Name,
		Error:       message,
	}
	h.render(w, http.StatusOK, "policy", data)
}

func (h *Handler) render(w http.ResponseWriter, status int, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		return
	}
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || h.sessions == nil {
		h.renderError(w, http.StatusNotFound, "Page not found", "Authentication is not enabled in this runtime.")
		return
	}
	if r.Method == http.MethodGet {
		h.renderAuth(w, r, "login", http.StatusOK, "", "")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.validAnonymousCSRF(w, r) {
		h.renderAuth(w, r, "login", http.StatusForbidden, r.FormValue("email"), "The form expired. Please try again.")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderAuth(w, r, "login", http.StatusBadRequest, "", "We could not read that form.")
		return
	}
	user, err := h.auth.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		h.renderAuth(w, r, "login", http.StatusUnauthorized, r.FormValue("email"), "Email or password is incorrect.")
		return
	}
	session, err := h.sessions.Start(r.Context(), user, r.UserAgent(), clientIP(r), applicationauth.DefaultSessionTTL)
	if err != nil {
		h.renderAuth(w, r, "login", http.StatusServiceUnavailable, r.FormValue("email"), "We could not start a secure session. Please try again.")
		return
	}
	h.setSessionCookie(w, session)
	h.mergeGuestCart(r.Context(), user.ID, r)
	h.clearCSRFCookie(w)
	h.redirect(w, r, "/account/sessions")
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || h.sessions == nil {
		h.renderError(w, http.StatusNotFound, "Page not found", "Authentication is not enabled in this runtime.")
		return
	}
	if r.Method == http.MethodGet {
		h.renderAuth(w, r, "register", http.StatusOK, "", "")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.validAnonymousCSRF(w, r) {
		h.renderAuth(w, r, "register", http.StatusForbidden, r.FormValue("email"), "The form expired. Please try again.")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderAuth(w, r, "register", http.StatusBadRequest, "", "We could not read that form.")
		return
	}
	user, err := h.auth.Register(r.Context(), applicationauth.RegisterInput{
		Email:       r.FormValue("email"),
		DisplayName: r.FormValue("display_name"),
		Password:    r.FormValue("password"),
	})
	if err != nil {
		message := "Please check your name, email, and password. Use at least 12 characters."
		if errors.Is(err, ports.ErrUserAlreadyExists) {
			message = "An account with that email already exists."
		}
		h.renderAuth(w, r, "register", http.StatusUnprocessableEntity, r.FormValue("email"), message)
		return
	}
	session, err := h.sessions.Start(r.Context(), user, r.UserAgent(), clientIP(r), applicationauth.DefaultSessionTTL)
	if err != nil {
		h.renderAuth(w, r, "register", http.StatusServiceUnavailable, r.FormValue("email"), "Your account was created, but we could not start a secure session. Please try logging in.")
		return
	}
	h.setSessionCookie(w, session)
	h.mergeGuestCart(r.Context(), user.ID, r)
	h.clearCSRFCookie(w)
	h.redirect(w, r, "/account/sessions")
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.sessions == nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	_ = h.sessions.Revoke(r.Context(), session.ID)
	h.clearSessionCookie(w)
	h.redirect(w, r, "/")
}

func (h *Handler) sessionsPage(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if r.Method == http.MethodPost {
		if !h.validSessionCSRF(r, session) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if err := h.sessions.RevokeAll(r.Context(), session.UserID); err != nil {
			h.renderError(w, http.StatusServiceUnavailable, "Sessions unavailable", "We could not revoke the active sessions.")
			return
		}
		h.clearSessionCookie(w)
		h.redirect(w, r, "/")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	sessions, err := h.sessions.List(r.Context(), session.UserID)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Sessions unavailable", "We could not load your active sessions.")
		return
	}
	h.render(w, http.StatusOK, "sessions", pageData{
		Brand:           identity.Name,
		Tagline:         identity.Tagline,
		Description:     identity.Description,
		Title:           "Your sessions · " + identity.Name,
		Authenticated:   true,
		AccountUserID:   session.UserID,
		CSRFToken:       session.CSRFToken,
		Sessions:        sessions,
		Notice:          verificationNotice(r),
		RecoveryEnabled: h.recoveryService != nil,
	})
}

func verificationNotice(r *http.Request) string {
	if r.URL.Query().Get("verification") == "sent" {
		return "If delivery is configured, a verification link has been sent to your account email."
	}
	return ""
}

func (h *Handler) renderAuth(w http.ResponseWriter, r *http.Request, mode string, status int, email, formError string) {
	csrf := h.ensureCSRFCookie(w, r)
	name := ""
	if r.Method == http.MethodPost {
		name = r.FormValue("display_name")
	}
	data := pageData{
		Brand:       identity.Name,
		Tagline:     identity.Tagline,
		Description: identity.Description,
		Title:       titleMode(mode) + " · " + identity.Name,
		CSRFToken:   csrf,
		FormEmail:   email,
		FormName:    name,
		Error:       formError,
	}
	h.render(w, status, mode, data)
}

func (h *Handler) decorateSession(r *http.Request, data *pageData) {
	session, ok := h.sessionFromRequest(r)
	if !ok {
		return
	}
	data.Authenticated = true
	data.AccountUserID = session.UserID
	data.CSRFToken = session.CSRFToken
}

func (h *Handler) sessionFromRequest(r *http.Request) (ports.SessionRecord, bool) {
	if h.sessions == nil {
		return ports.SessionRecord{}, false
	}
	cookie, err := r.Cookie("wecratfs_session")
	if err != nil {
		return ports.SessionRecord{}, false
	}
	session, err := h.sessions.Get(r.Context(), cookie.Value)
	if err != nil {
		return ports.SessionRecord{}, false
	}
	return session, true
}

func (h *Handler) validSessionCSRF(r *http.Request, session ports.SessionRecord) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return csrfEqual(session.CSRFToken, r.FormValue("csrf_token"))
}

func (h *Handler) validAnonymousCSRF(w http.ResponseWriter, r *http.Request) bool {
	token := h.ensureCSRFCookie(w, r)
	return csrfEqual(token, r.FormValue("csrf_token"))
}

func (h *Handler) ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie("wecratfs_csrf"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	token, err := randomCSRFToken()
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_csrf", Value: token, Path: "/", Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: 3600})
	return token
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, session ports.SessionRecord) {
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_session", Value: session.ID, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt, MaxAge: int(time.Until(session.ExpiresAt).Seconds())})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_session", Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func (h *Handler) clearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_csrf", Value: "", Path: "/", Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, location string) {
	if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func csrfEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func randomCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func titleMode(mode string) string {
	if mode == "" {
		return ""
	}
	return strings.ToUpper(mode[:1]) + mode[1:]
}

package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/wecratfs/commerce/internal/web/viewmodels"
	"github.com/wecratfs/commerce/web/assets"
	"github.com/wecratfs/commerce/web/components"
	"github.com/wecratfs/commerce/web/pages"
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
	if strings.HasPrefix(r.URL.Path, "/ui/") {
		if r.Method == http.MethodGet {
			h.fragment(w, r)
			return
		}
		h.mutation(w, r)
		return
	}
	page := h.page(r, w)
	switch {
	case r.URL.Path == "/":
		h.render(w, r, pages.Home(page))
	case r.URL.Path == "/categories":
		h.render(w, r, pages.Categories(page))
	case r.URL.Path == "/products" || r.URL.Path == "/search":
		h.render(w, r, pages.Listing(page))
	case strings.HasPrefix(r.URL.Path, "/products/"):
		if _, ok := productBySlug(strings.TrimPrefix(r.URL.Path, "/products/")); !ok {
			h.notFound(w, r)
			return
		}
		h.render(w, r, pages.ProductDetail(page))
	case strings.HasPrefix(r.URL.Path, "/makers/"):
		if _, ok := sellerBySlug(strings.TrimPrefix(r.URL.Path, "/makers/")); !ok {
			h.notFound(w, r)
			return
		}
		h.render(w, r, pages.SellerStore(page))
	case r.URL.Path == "/wishlist":
		h.render(w, r, pages.Wishlist(page))
	case r.URL.Path == "/cart":
		h.render(w, r, pages.Cart(page))
	case r.URL.Path == "/checkout":
		h.render(w, r, pages.Checkout(page))
	case r.URL.Path == "/orders":
		h.render(w, r, pages.Orders(page))
	case strings.HasPrefix(r.URL.Path, "/orders/"):
		h.render(w, r, pages.OrderDetail(page))
	case r.URL.Path == "/returns":
		h.render(w, r, pages.Returns(page))
	case r.URL.Path == "/account":
		h.render(w, r, pages.Account(page))
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
		h.render(w, r, pages.Returns(page))
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
			h.render(w, r, pages.Cart(h.page(r, w)))
			return
		}
		h.redirect(w, r, "/cart")
		return
	}
	if len(parts) == 3 && parts[1] == "wishlist" {
		h.store.toggleWishlist(w, r, parts[2])
		product, ok := productBySlug(parts[2])
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			h.render(w, r, components.ProductCard(product, true))
			return
		}
		h.redirect(w, r, "/wishlist")
		return
	}
	if r.URL.Path == "/ui/checkout" {
		h.render(w, r, pages.Empty(viewmodels.CustomerPage{Title: "Preview order placed", Notice: "This is a UI-only order preview. No payment or shipping request was sent."}))
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
	products := append([]viewmodels.Product(nil), mockProducts...)
	if category != "" && category != "offers" {
		filtered := products[:0]
		for _, product := range products {
			if strings.EqualFold(product.Category, category) || strings.Contains(strings.ToLower(product.Category), strings.ReplaceAll(strings.ToLower(category), "-", " ")) {
				filtered = append(filtered, product)
			}
		}
		if len(filtered) > 0 {
			products = filtered
		}
	}
	if query != "" {
		filtered := products[:0]
		for _, product := range products {
			if strings.Contains(strings.ToLower(product.Name+" "+product.Category+" "+product.Seller), strings.ToLower(query)) {
				filtered = append(filtered, product)
			}
		}
		if len(filtered) > 0 {
			products = filtered
		}
	}
	wishlistProducts := make([]viewmodels.Product, 0, len(products))
	for _, product := range products {
		if snapshot.wishlist[product.Slug] {
			wishlistProducts = append(wishlistProducts, product)
		}
	}
	if len(wishlistProducts) == 0 {
		wishlistProducts = products
	}
	return viewmodels.CustomerPage{Route: r.URL.Path, Title: titleFor(r.URL.Path), Query: query, Category: category, Brand: "WeeVCrafts", Tagline: "Handmade Today. A Kinder Tomorrow.", Description: "Authentic Indian arts, crafts and sarees, made with care.", Products: products, Categories: mockCategories, Sellers: mockSellers, Cart: snapshot.cart, Orders: snapshot.orders, Returns: mockReturns(), CartCount: cartCount(snapshot.cart), WishlistCount: len(wishlistProducts), Mock: true, Authenticated: true, Now: time.Now()}
}

type sessionSnapshot struct {
	cart     []viewmodels.CartItem
	orders   []viewmodels.Order
	wishlist map[string]bool
}

func (s *mockStore) snapshot(w http.ResponseWriter, r *http.Request) sessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessionUnlocked(w, r)
	return sessionSnapshot{cart: append([]viewmodels.CartItem(nil), session.Cart...), orders: append([]viewmodels.Order(nil), session.Orders...), wishlist: cloneWishlist(session.Wishlist)}
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
	case "/wishlist":
		return "My Wishlist"
	case "/cart":
		return "Your Cart"
	case "/checkout":
		return "Checkout"
	case "/orders":
		return "My Orders"
	case "/returns":
		return "Customer Returns & Refunds"
	case "/account":
		return "My Account"
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

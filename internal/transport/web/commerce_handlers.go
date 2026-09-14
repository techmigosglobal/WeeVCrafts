package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) cartPage(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil {
		h.notFound(w, r)
		return
	}
	userID, guestHash := h.cartOwner(w, r)
	cart, err := h.cartService.GetOrCreate(r.Context(), userID, guestHash)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Cart unavailable", "We could not load your saved cart.")
		return
	}
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Your cart · " + identity.Name, Cart: cart, HasCart: true}
	h.decorateSession(r, &data)
	if data.CSRFToken == "" {
		data.CSRFToken = h.ensureCSRFCookie(w, r)
	}
	h.render(w, http.StatusOK, "cart", data)
}

func (h *Handler) cartAdd(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, authenticated := h.sessionFromRequest(r)
	if !h.validMutationCSRF(w, r, session, authenticated) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	variantID, err := strconv.ParseInt(r.FormValue("variant_id"), 10, 64)
	quantity, quantityErr := strconv.ParseInt(r.FormValue("quantity"), 10, 64)
	if err != nil || quantityErr != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Cart update failed", "That product selection is invalid.")
		return
	}
	userID, guestHash := h.cartOwner(w, r)
	if authenticated {
		userID, guestHash = session.UserID, ""
	}
	cart, err := h.cartService.Add(r.Context(), userID, guestHash, variantID, quantity)
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not add that product to your cart.")
		return
	}
	if isHTMXRequest(r) {
		h.render(w, http.StatusOK, "cart-add-response", pageData{
			Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
			Title: "Cart · " + identity.Name, Cart: cart, HasCart: true,
			CSRFToken: mutationCSRFToken(w, r, session, authenticated), Notice: "Added to your cart.",
		})
		return
	}
	h.redirect(w, r, "/cart")
}

func (h *Handler) cartUpdate(w http.ResponseWriter, r *http.Request) {
	h.cartMutation(w, r, "update")
}

func (h *Handler) cartRemove(w http.ResponseWriter, r *http.Request) {
	h.cartMutation(w, r, "remove")
}

func (h *Handler) cartMoveToWishlist(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, authenticated := h.sessionFromRequest(r)
	if !authenticated {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validMutationCSRF(w, r, session, true) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	variantID, variantErr := strconv.ParseInt(r.FormValue("variant_id"), 10, 64)
	productID, productErr := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	if variantErr != nil || productErr != nil || variantID <= 0 || productID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Wishlist update failed", "That cart item is invalid.")
		return
	}
	cart, err := h.cartService.MoveToWishlist(r.Context(), session.UserID, variantID, productID)
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not move that product to your wishlist.")
		return
	}
	if isHTMXRequest(r) {
		h.render(w, http.StatusOK, "cart-mutation", pageData{
			Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
			Title: "Cart · " + identity.Name, Cart: cart, HasCart: true,
			CSRFToken: session.CSRFToken, Notice: "Moved to your wishlist.",
		})
		return
	}
	h.redirect(w, r, "/cart")
}

func (h *Handler) cartMutation(w http.ResponseWriter, r *http.Request, action string) {
	if h.cartService == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, authenticated := h.sessionFromRequest(r)
	if !h.validMutationCSRF(w, r, session, authenticated) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	variantID, err := strconv.ParseInt(r.FormValue("variant_id"), 10, 64)
	if err != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Cart update failed", "That product selection is invalid.")
		return
	}
	userID, guestHash := h.cartOwner(w, r)
	if authenticated {
		userID, guestHash = session.UserID, ""
	}
	var cart domaincommerce.Cart
	if action == "remove" {
		cart, err = h.cartService.Remove(r.Context(), userID, guestHash, variantID)
	} else {
		quantity, parseErr := strconv.ParseInt(r.FormValue("quantity"), 10, 64)
		if parseErr != nil {
			h.renderError(w, http.StatusUnprocessableEntity, "Cart update failed", "Quantity is invalid.")
			return
		}
		cart, err = h.cartService.Update(r.Context(), userID, guestHash, variantID, quantity)
	}
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not update your cart.")
		return
	}
	if isHTMXRequest(r) {
		h.render(w, http.StatusOK, "cart-mutation", pageData{
			Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
			Title: "Cart · " + identity.Name, Cart: cart, HasCart: true,
			CSRFToken: mutationCSRFToken(w, r, session, authenticated),
			Notice:    map[string]string{"remove": "Removed from your cart.", "update": "Cart quantity updated."}[action],
		})
		return
	}
	h.redirect(w, r, "/cart")
}

func (h *Handler) wishlistPage(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	items, err := h.cartService.ListWishlist(r.Context(), session.UserID)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Wishlist unavailable", "We could not load your saved products.")
		return
	}
	h.render(w, http.StatusOK, "wishlist", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Wishlist · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, Wishlist: items,
	})
}

func (h *Handler) wishlistAdd(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	productID, err := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Wishlist update failed", "That product identifier is invalid.")
		return
	}
	if err := h.cartService.AddWishlist(r.Context(), session.UserID, productID); err != nil {
		h.renderCommerceError(w, r, err, "We could not save that product to your wishlist.")
		return
	}
	if isHTMXRequest(r) {
		h.render(w, http.StatusOK, "wishlist-add-response", pageData{
			Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
			Title: "Wishlist · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
			CSRFToken: session.CSRFToken, Notice: "Saved to your wishlist.",
		})
		return
	}
	h.redirect(w, r, "/wishlist")
}

func (h *Handler) wishlistRemove(w http.ResponseWriter, r *http.Request) {
	if h.cartService == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	productID, err := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Wishlist update failed", "That product identifier is invalid.")
		return
	}
	if err := h.cartService.RemoveWishlist(r.Context(), session.UserID, productID); err != nil {
		h.renderCommerceError(w, r, err, "We could not remove that product from your wishlist.")
		return
	}
	if isHTMXRequest(r) {
		items, err := h.cartService.ListWishlist(r.Context(), session.UserID)
		if err != nil {
			h.renderError(w, http.StatusServiceUnavailable, "Wishlist unavailable", "We could not reload your saved products.")
			return
		}
		h.render(w, http.StatusOK, "wishlist-mutation", pageData{
			Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
			Title: "Wishlist · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
			CSRFToken: session.CSRFToken, Wishlist: items, Notice: "Removed from your wishlist.",
		})
		return
	}
	h.redirect(w, r, "/wishlist")
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	if h.orderService == nil || h.cartService == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	cart, err := h.cartService.GetOrCreate(r.Context(), session.UserID, "")
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Cart is empty", "Add a product before starting checkout.")
		return
	}
	if r.Method == http.MethodGet {
		h.render(w, http.StatusOK, "checkout", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Checkout · " + identity.Name, Cart: cart, HasCart: true, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, FormEmail: newIdempotencyKey()})
		return
	}
	if r.Method != http.MethodPost || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = r.FormValue("idempotency_key")
	}
	order, err := h.orderService.Create(r.Context(), session.UserID, cart.ID, key, domaincommerce.AddressInput{
		RecipientName: r.FormValue("recipient_name"),
		Line1:         r.FormValue("line1"),
		Line2:         r.FormValue("line2"),
		City:          r.FormValue("city"),
		State:         r.FormValue("state"),
		PostalCode:    r.FormValue("postal_code"),
		CountryCode:   r.FormValue("country_code"),
	})
	if err != nil {
		if errors.Is(err, ports.ErrIdempotencyConflict) {
			h.renderError(w, http.StatusConflict, "Checkout request already used", "Please refresh checkout and use a new request key for a different address or cart.")
			return
		}
		h.renderCommerceError(w, r, err, "We could not create this order. Please review your address and stock.")
		return
	}
	h.redirect(w, r, "/orders/"+order.OrderNumber)
}

func (h *Handler) ordersPage(w http.ResponseWriter, r *http.Request) {
	if h.orderService == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	orders, err := h.orderService.List(r.Context(), session.UserID, 20)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Orders unavailable", "We could not load your orders.")
		return
	}
	h.render(w, http.StatusOK, "orders", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Your orders · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, Orders: orders})
}

func (h *Handler) orderPage(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if h.orderService == nil || strings.Contains(orderNumber, "/") || orderNumber == "" {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	detail, err := h.orderService.Get(r.Context(), session.UserID, orderNumber)
	if errors.Is(err, ports.ErrOrderNotFound) {
		h.notFound(w, r)
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Order unavailable", "We could not load this order.")
		return
	}
	data := pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: detail.OrderNumber + " · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, OrderDetail: detail, HasOrderDetail: true, RefundKey: newIdempotencyKey(), ReturnEnabled: h.returnService != nil}
	if h.returnService != nil {
		if request, requestErr := h.returnService.Get(r.Context(), session.UserID, orderNumber); requestErr == nil {
			data.ReturnRequest = request
			data.HasReturnRequest = true
		}
	}
	h.render(w, http.StatusOK, "order", data)
}

func (h *Handler) orderRefund(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if h.refundService == nil || h.orderService == nil || r.Method != http.MethodPost || orderNumber == "" || strings.Contains(orderNumber, "/") {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Refund request invalid", "We could not read the refund request.")
		return
	}
	requestKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if requestKey == "" {
		requestKey = strings.TrimSpace(r.FormValue("idempotency_key"))
	}
	refund, err := h.refundService.Refund(r.Context(), session.UserID, orderNumber, requestKey, r.FormValue("reason"))
	if err != nil {
		switch {
		case errors.Is(err, applicationpayment.ErrRefundInvalidRequest):
			h.renderError(w, http.StatusUnprocessableEntity, "Refund request invalid", "Please provide a valid refund request and try again.")
		case errors.Is(err, ports.ErrRefundNotFound):
			h.notFound(w, r)
		case errors.Is(err, ports.ErrRefundState):
			h.renderError(w, http.StatusConflict, "Refund unavailable", "This order is not currently refundable.")
		case errors.Is(err, applicationpayment.ErrRefundCreationFailed), errors.Is(err, applicationpayment.ErrRefundResponseMismatch):
			h.renderError(w, http.StatusBadGateway, "Refund provider unavailable", "The payment provider could not complete this refund.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Refund unavailable", "We could not complete this refund.")
		}
		return
	}
	_ = refund
	h.redirect(w, r, "/orders/"+orderNumber)
}

func (h *Handler) orderPayment(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if h.paymentService == nil || h.orderService == nil || r.Method != http.MethodPost || orderNumber == "" || strings.Contains(orderNumber, "/") {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	intent, err := h.paymentService.Create(r.Context(), session.UserID, orderNumber)
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrPaymentNotFound):
			h.notFound(w, r)
		case errors.Is(err, ports.ErrPaymentState):
			h.renderError(w, http.StatusConflict, "Payment unavailable", "This order is no longer available for payment.")
		case errors.Is(err, applicationpayment.ErrPaymentCreationFailed):
			h.renderError(w, http.StatusBadGateway, "Payment provider unavailable", "The payment provider could not prepare this order.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Payment unavailable", "We could not prepare this payment order.")
		}
		return
	}
	detail, err := h.orderService.Get(r.Context(), session.UserID, orderNumber)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Order unavailable", "We could not reload this order.")
		return
	}
	h.render(w, http.StatusOK, "order", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: detail.OrderNumber + " · " + identity.Name, Authenticated: true,
		AccountUserID: session.UserID, CSRFToken: session.CSRFToken, OrderDetail: detail,
		HasOrderDetail: true, PaymentIntent: intent, HasPayment: true, PaymentPublicKey: h.paymentPublicKey,
	})
}

func (h *Handler) orderCancel(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if h.orderService == nil || r.Method != http.MethodPost || orderNumber == "" || strings.Contains(orderNumber, "/") {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err := h.orderService.Cancel(r.Context(), session.UserID, orderNumber); err != nil {
		if errors.Is(err, ports.ErrOrderNotFound) {
			h.notFound(w, r)
			return
		}
		if errors.Is(err, ports.ErrInvalidState) {
			h.renderError(w, http.StatusConflict, "Order cannot be cancelled", "This order has already moved beyond pending payment.")
			return
		}
		h.renderError(w, http.StatusServiceUnavailable, "Order cancellation failed", "We could not release this order reservation.")
		return
	}
	h.redirect(w, r, "/orders/"+orderNumber)
}

func (h *Handler) sellerPage(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	profile, profileErr := h.commerceCatalog.SellerProfile(r.Context(), session.UserID)
	if profileErr != nil && !errors.Is(profileErr, ports.ErrSellerNotFound) {
		h.renderError(w, http.StatusServiceUnavailable, "Seller workspace unavailable", "We could not load your seller profile.")
		return
	}
	products, err := h.commerceCatalog.SellerProducts(r.Context(), session.UserID)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Seller workspace unavailable", "We could not load your seller records.")
		return
	}
	h.render(w, http.StatusOK, "seller", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Seller workspace · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, ManagedProducts: products, SellerProfile: profile, HasSellerProfile: profileErr == nil})
}

func (h *Handler) sellerOnboard(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if _, err := h.commerceCatalog.EnsureSeller(r.Context(), session.UserID, r.FormValue("display_name")); err != nil {
		h.renderCommerceError(w, r, err, "We could not start your seller profile.")
		return
	}
	h.redirect(w, r, "/seller")
}

func (h *Handler) sellerNewProduct(w http.ResponseWriter, r *http.Request) {
	session, ok := h.sessionFromRequest(r)
	if h.commerceCatalog == nil || !ok {
		h.redirect(w, r, "/login")
		return
	}
	profile, err := h.commerceCatalog.SellerProfile(r.Context(), session.UserID)
	if err != nil || profile.Status != "active" {
		h.renderError(w, http.StatusForbidden, "Seller tools unavailable", "Your seller application must be approved before you can create products.")
		return
	}
	h.render(w, http.StatusOK, "seller-product", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "New product · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken})
}

func (h *Handler) sellerTeam(w http.ResponseWriter, r *http.Request) {
	if h.sellerStaff == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	staff, err := h.sellerStaff.List(r.Context(), session.UserID)
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not load your seller team.")
		return
	}
	h.render(w, http.StatusOK, "seller-team", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Seller team · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, StaffMembers: staff,
	})
}

func (h *Handler) sellerTeamAdd(w http.ResponseWriter, r *http.Request) {
	if h.sellerStaff == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if _, err := h.sellerStaff.Add(r.Context(), session.UserID, r.FormValue("email"), r.Form["permissions"]); err != nil {
		h.renderCommerceError(w, r, err, "We could not add that staff account.")
		return
	}
	h.redirect(w, r, "/seller/team")
}

func (h *Handler) sellerTeamRemove(w http.ResponseWriter, r *http.Request) {
	if h.sellerStaff == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	staffUserID, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	if err != nil || staffUserID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Staff request invalid", "Choose a valid staff account.")
		return
	}
	if err := h.sellerStaff.Remove(r.Context(), session.UserID, staffUserID); err != nil {
		h.renderCommerceError(w, r, err, "We could not remove that staff account.")
		return
	}
	h.redirect(w, r, "/seller/team")
}

func (h *Handler) sellerEditProduct(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	productID, err := sellerProductID(r.URL.Path)
	if err != nil {
		h.notFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		product, getErr := h.commerceCatalog.GetSellerProduct(r.Context(), session.UserID, productID)
		if errors.Is(getErr, ports.ErrForbidden) {
			h.notFound(w, r)
			return
		}
		if getErr != nil {
			h.renderError(w, http.StatusServiceUnavailable, "Seller product unavailable", "We could not load this draft.")
			return
		}
		h.render(w, http.StatusOK, "seller-product", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Edit product · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, ManagedProduct: product, HasManagedProduct: true})
		return
	}
	if r.Method != http.MethodPost || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	price, priceErr := strconv.ParseInt(r.FormValue("price_cents"), 10, 64)
	stock, stockErr := strconv.Atoi(r.FormValue("initial_stock"))
	if priceErr != nil || stockErr != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Product details invalid", "Price and stock must be numeric.")
		return
	}
	var compareAt *int64
	if raw := strings.TrimSpace(r.FormValue("compare_at_cents")); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			h.renderError(w, http.StatusUnprocessableEntity, "Product details invalid", "Compare-at price must be numeric.")
			return
		}
		compareAt = &value
	}
	if _, updateErr := h.commerceCatalog.UpdateDraft(r.Context(), session.UserID, productID, domaincommerce.ProductDraftInput{
		Slug: r.FormValue("slug"), BrandSlug: r.FormValue("brand_slug"), BrandName: r.FormValue("brand_name"), CategorySlug: r.FormValue("category_slug"), CategoryName: r.FormValue("category_name"), Name: r.FormValue("name"), Description: r.FormValue("description"), PriceCents: price, CompareAtCents: compareAt, SKU: r.FormValue("sku"), InitialStock: stock, DeliveryLabel: r.FormValue("delivery_label"),
	}); updateErr != nil {
		h.renderCommerceError(w, r, updateErr, "We could not update that product draft.")
		return
	}
	h.redirect(w, r, "/seller")
}

func sellerProductID(path string) (int64, error) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(path, "/seller/products/"), "/edit")
	return strconv.ParseInt(strings.Trim(trimmed, "/"), 10, 64)
}

func (h *Handler) sellerProducts(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	if !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	price, err := strconv.ParseInt(r.FormValue("price_cents"), 10, 64)
	stock, stockErr := strconv.Atoi(r.FormValue("initial_stock"))
	if err != nil || stockErr != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Product details invalid", "Price and stock must be numeric.")
		return
	}
	var compareAt *int64
	if raw := strings.TrimSpace(r.FormValue("compare_at_cents")); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			h.renderError(w, http.StatusUnprocessableEntity, "Product details invalid", "Compare-at price must be numeric.")
			return
		}
		compareAt = &value
	}
	product, err := h.commerceCatalog.CreateDraft(r.Context(), session.UserID, domaincommerce.ProductDraftInput{
		Slug: r.FormValue("slug"), BrandSlug: r.FormValue("brand_slug"), BrandName: r.FormValue("brand_name"), CategorySlug: r.FormValue("category_slug"), CategoryName: r.FormValue("category_name"), Name: r.FormValue("name"), Description: r.FormValue("description"), PriceCents: price, CompareAtCents: compareAt, SKU: r.FormValue("sku"), InitialStock: stock, DeliveryLabel: r.FormValue("delivery_label"),
	})
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not create that product draft.")
		return
	}
	h.redirect(w, r, "/seller")
	_ = product
}

func (h *Handler) sellerSubmitProduct(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	productID, err := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	if err != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Product submission invalid", "That product identifier is invalid.")
		return
	}
	if err := h.commerceCatalog.SubmitProduct(r.Context(), session.UserID, productID); err != nil {
		h.renderCommerceError(w, r, err, "We could not submit this product for review.")
		return
	}
	h.redirect(w, r, "/seller")
}

func (h *Handler) adminProducts(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	products, err := h.commerceCatalog.PendingProducts(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		w.WriteHeader(http.StatusForbidden)
		h.renderError(w, http.StatusForbidden, "Access denied", "This workspace is restricted to marketplace administrators.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Admin workspace unavailable", "We could not load moderation records.")
		return
	}
	h.render(w, http.StatusOK, "admin-products", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Catalog moderation · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, PendingProducts: products})
}

func (h *Handler) adminReviewProduct(w http.ResponseWriter, r *http.Request) {
	if h.commerceCatalog == nil || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	productID, err := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	approved := r.FormValue("decision") == "approve"
	if err != nil {
		h.renderError(w, http.StatusUnprocessableEntity, "Review invalid", "That product identifier is invalid.")
		return
	}
	if err := h.commerceCatalog.ApproveProduct(r.Context(), session.UserID, productID, approved, r.FormValue("reason")); err != nil {
		h.renderCommerceError(w, r, err, "We could not save the moderation decision.")
		return
	}
	h.redirect(w, r, "/admin/products")
}

func (h *Handler) validMutationCSRF(w http.ResponseWriter, r *http.Request, session ports.SessionRecord, authenticated bool) bool {
	if authenticated {
		return h.validSessionCSRF(r, session)
	}
	return h.validAnonymousCSRF(w, r)
}

func (h *Handler) cartOwner(w http.ResponseWriter, r *http.Request) (int64, string) {
	if session, ok := h.sessionFromRequest(r); ok {
		return session.UserID, ""
	}
	cookie, err := r.Cookie("wecratfs_guest_cart")
	if err != nil || cookie.Value == "" {
		token, tokenErr := randomCSRFToken()
		if tokenErr != nil {
			return 0, ""
		}
		http.SetCookie(w, &http.Cookie{Name: "wecratfs_guest_cart", Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int((30 * 24 * time.Hour).Seconds())})
		return 0, hashGuestToken(token)
	}
	return 0, hashGuestToken(cookie.Value)
}

func (h *Handler) mergeGuestCart(ctx context.Context, userID int64, r *http.Request) {
	if h.cartService == nil {
		return
	}
	cookie, err := r.Cookie("wecratfs_guest_cart")
	if err != nil || cookie.Value == "" {
		return
	}
	_ = h.cartService.MergeGuest(ctx, userID, hashGuestToken(cookie.Value))
}

func (h *Handler) renderCommerceError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	status := http.StatusUnprocessableEntity
	message := fallback
	switch {
	case errors.Is(err, ports.ErrForbidden):
		status = http.StatusForbidden
		message = "You do not have permission to perform that action."
	case errors.Is(err, ports.ErrInsufficientStock):
		message = "Some items do not have enough available stock."
	case errors.Is(err, applicationcommerce.ErrInvalidQuantity), errors.Is(err, applicationcommerce.ErrInvalidAddress):
		message = "Please check the submitted details."
	case errors.Is(err, applicationcommerce.ErrInvalidInventoryAdjustment):
		message = "Enter a non-zero stock adjustment and a reason between 3 and 500 characters."
	case errors.Is(err, applicationcommerce.ErrInvalidSellerFulfillment):
		message = "Choose the next fulfilment status and provide valid tracking details before shipping."
	}
	h.renderError(w, status, "Request could not be completed", message)
}

func hashGuestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func newIdempotencyKey() string {
	token, err := randomCSRFToken()
	if err != nil {
		return "checkout-key-fallback-000000"
	}
	return token
}

func isHTMXRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

func mutationCSRFToken(w http.ResponseWriter, r *http.Request, session ports.SessionRecord, authenticated bool) string {
	if authenticated {
		return session.CSRFToken
	}
	return htmxAnonymousCSRFToken(w, r)
}

func htmxAnonymousCSRFToken(w http.ResponseWriter, r *http.Request) string {
	// validAnonymousCSRF has already ensured this cookie exists for the
	// mutation. Reading it here keeps the fragment forms usable for the next
	// guest mutation without introducing a second token source.
	if cookie, err := r.Cookie("wecratfs_csrf"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

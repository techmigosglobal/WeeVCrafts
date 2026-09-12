package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	applicationprivacy "github.com/wecratfs/commerce/internal/application/privacy"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type Handler struct {
	catalog       *applicationcatalog.Service
	search        *applicationcatalog.SearchService
	auth          *applicationauth.Service
	sessions      *applicationauth.SessionService
	cart          *applicationcommerce.CartService
	orders        *applicationcommerce.OrderService
	privacy       *applicationprivacy.Service
	media         *applicationcommerce.MediaService
	payment       *applicationpayment.OrderService
	refunds       *applicationpayment.RefundService
	recovery      *applicationauth.RecoveryService
	secureCookies bool
}

func NewHandler(catalogService *applicationcatalog.Service) *Handler {
	return &Handler{catalog: catalogService}
}

func NewHandlerWithSearch(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService) *Handler {
	return &Handler{catalog: catalogService, search: searchService}
}

func NewFullHandler(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService) *Handler {
	return &Handler{catalog: catalogService, search: searchService, auth: authService, sessions: sessionService, cart: cartService, orders: orderService}
}

func NewCompleteHandler(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service) *Handler {
	return &Handler{catalog: catalogService, search: searchService, auth: authService, sessions: sessionService, cart: cartService, orders: orderService, privacy: privacyService}
}

func NewCompleteHandlerWithConfig(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service, secureCookies bool) *Handler {
	handler := NewCompleteHandler(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService)
	handler.secureCookies = secureCookies
	return handler
}

func NewCompleteHandlerWithMedia(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, secureCookies bool) *Handler {
	handler := NewCompleteHandlerWithConfig(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService, secureCookies)
	handler.media = mediaService
	return handler
}

func NewCompleteHandlerWithMediaAndPayment(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, paymentService *applicationpayment.OrderService, secureCookies bool) *Handler {
	handler := NewCompleteHandlerWithMedia(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService, mediaService, secureCookies)
	handler.payment = paymentService
	return handler
}

func NewCompleteHandlerWithMediaPaymentAndRefund(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, paymentService *applicationpayment.OrderService, refundService *applicationpayment.RefundService, secureCookies bool) *Handler {
	handler := NewCompleteHandlerWithMediaAndPayment(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService, mediaService, paymentService, secureCookies)
	handler.refunds = refundService
	return handler
}

func NewCompleteHandlerWithMediaPaymentRefundAndRecovery(catalogService *applicationcatalog.Service, searchService *applicationcatalog.SearchService, authService *applicationauth.Service, sessionService *applicationauth.SessionService, cartService *applicationcommerce.CartService, orderService *applicationcommerce.OrderService, privacyService *applicationprivacy.Service, mediaService *applicationcommerce.MediaService, paymentService *applicationpayment.OrderService, refundService *applicationpayment.RefundService, recoveryService *applicationauth.RecoveryService, secureCookies bool) *Handler {
	handler := NewCompleteHandlerWithMediaPaymentAndRefund(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService, mediaService, paymentService, refundService, secureCookies)
	handler.recovery = recoveryService
	return handler
}

func (h *Handler) Products(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is supported.")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/products")
	if path == "" || path == "/" {
		products, err := h.catalog.List(r.Context(), r.URL.Query().Get("category"), 24)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "PRODUCTS_UNAVAILABLE", "Products could not be loaded.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": products, "source": "postgresql"})
		return
	}

	slug := strings.TrimPrefix(path, "/")
	if strings.Contains(slug, "/") || slug == "" {
		writeError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found.")
		return
	}
	product, err := h.catalog.Get(r.Context(), slug)
	if errors.Is(err, applicationcatalog.ErrProductNotFound) {
		writeError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PRODUCT_UNAVAILABLE", "Product could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": product, "source": "postgresql"})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || h.search == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is supported.")
		return
	}
	limit := 24
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			limit = parsed
		}
	}
	offset := 0
	if value := r.URL.Query().Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			offset = parsed
		}
	}
	products, total, facets, err := h.search.SearchWithOptionsAndFacets(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("category"), r.URL.Query().Get("sort"), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SEARCH_UNAVAILABLE", "Search could not be completed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": products, "total": total, "facets": facets, "source": "postgresql"})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.auth == nil || h.sessions == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported.")
		return
	}
	var input struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := h.auth.Register(r.Context(), applicationauth.RegisterInput{Email: input.Email, DisplayName: input.DisplayName, Password: input.Password})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "REGISTRATION_INVALID", "Registration details are invalid.")
		return
	}
	session, err := h.sessions.Start(r.Context(), user, r.UserAgent(), "", applicationauth.DefaultSessionTTL)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE", "A secure session could not be started.")
		return
	}
	h.setSessionCookie(w, session)
	h.mergeGuestCart(r.Context(), w, r, user.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"data": user, "csrf_token": session.CSRFToken})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.auth == nil || h.sessions == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported.")
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := h.auth.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect.")
		return
	}
	session, err := h.sessions.Start(r.Context(), user, r.UserAgent(), "", applicationauth.DefaultSessionTTL)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE", "A secure session could not be started.")
		return
	}
	h.setSessionCookie(w, session)
	h.mergeGuestCart(r.Context(), w, r, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"data": user, "csrf_token": session.CSRFToken})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	_ = h.sessions.Revoke(r.Context(), session.ID)
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "signed_out"}})
}

func (h *Handler) Cart(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodGet || !ok || h.cart == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sign in to access the cart API.")
		return
	}
	cart, err := h.cart.GetOrCreate(r.Context(), session.UserID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CART_UNAVAILABLE", "Cart could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

func (h *Handler) CartAdd(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.cart == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		VariantID int64 `json:"variant_id"`
		Quantity  int64 `json:"quantity"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	cart, err := h.cart.Add(r.Context(), session.UserID, "", input.VariantID, input.Quantity)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "CART_UPDATE_FAILED", "Cart item could not be added.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.cart == nil || h.orders == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		CartID         int64                       `json:"cart_id"`
		IdempotencyKey string                      `json:"idempotency_key"`
		Address        domaincommerce.AddressInput `json:"address"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); headerKey != "" {
		input.IdempotencyKey = headerKey
	}
	order, err := h.orders.Create(r.Context(), session.UserID, input.CartID, input.IdempotencyKey, input.Address)
	if err != nil {
		if errors.Is(err, ports.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "This idempotency key was already used for a different checkout request.")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "CHECKOUT_FAILED", "Order could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": order})
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodGet || !ok || h.orders == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sign in to access orders.")
		return
	}
	orders, err := h.orders.List(r.Context(), session.UserID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORDERS_UNAVAILABLE", "Orders could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": orders})
}

func (h *Handler) Order(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/payment") {
		h.PaymentOrder(w, r)
		return
	}
	if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/refund") {
		h.RefundOrder(w, r)
		return
	}
	if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/cancel") {
		h.CancelOrder(w, r)
		return
	}
	session, ok := h.session(r)
	if r.Method != http.MethodGet || !ok || h.orders == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sign in to access orders.")
		return
	}
	orderNumber := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")
	detail, err := h.orders.Get(r.Context(), session.UserID, orderNumber)
	if errors.Is(err, ports.ErrOrderNotFound) {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORDER_UNAVAILABLE", "Order could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": detail})
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.orders == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	path := strings.TrimSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/cancel")
	orderNumber := strings.TrimPrefix(path, "/api/v1/orders/")
	if orderNumber == "" || strings.Contains(orderNumber, "/") {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order not found.")
		return
	}
	if err := h.orders.Cancel(r.Context(), session.UserID, orderNumber); err != nil {
		if errors.Is(err, ports.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order not found.")
			return
		}
		if errors.Is(err, ports.ErrInvalidState) {
			writeError(w, http.StatusConflict, "ORDER_NOT_CANCELABLE", "This order can no longer be cancelled.")
			return
		}
		writeError(w, http.StatusInternalServerError, "ORDER_CANCEL_FAILED", "The order could not be cancelled.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handler) PaymentOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.payment == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/orders/"), "/payment")
	path = strings.Trim(path, "/")
	if path == "" || strings.Contains(path, "/") {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order not found.")
		return
	}
	intent, err := h.payment.Create(r.Context(), session.UserID, path)
	if errors.Is(err, ports.ErrPaymentNotFound) {
		writeError(w, http.StatusNotFound, "PAYMENT_ORDER_NOT_FOUND", "Payment order not found.")
		return
	}
	if errors.Is(err, ports.ErrPaymentState) {
		writeError(w, http.StatusConflict, "PAYMENT_NOT_PAYABLE", "This order is not available for payment.")
		return
	}
	if errors.Is(err, applicationpayment.ErrPaymentCreationFailed) {
		writeError(w, http.StatusBadGateway, "PAYMENT_PROVIDER_UNAVAILABLE", "The payment provider could not prepare this order.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PAYMENT_ORDER_FAILED", "The payment order could not be prepared.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": intent})
}

func (h *Handler) PrivacyCenter(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodGet || !ok || h.privacy == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sign in to access privacy controls.")
		return
	}
	center, err := h.privacy.Center(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PRIVACY_UNAVAILABLE", "Privacy records could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": center, "policy_version": "privacy-2026-09"})
}

func (h *Handler) PrivacyConsent(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.privacy == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		Purpose string `json:"purpose"`
		Granted bool   `json:"granted"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := h.privacy.SetConsent(r.Context(), session.UserID, input.Purpose, input.Granted); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "CONSENT_INVALID", "Consent choice is invalid.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (h *Handler) PrivacyRequest(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.privacy == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		Type    string `json:"type"`
		Details string `json:"details"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	request, err := h.privacy.Request(r.Context(), session.UserID, input.Type, input.Details)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "PRIVACY_REQUEST_INVALID", "Privacy request is invalid.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": request})
}

func (h *Handler) PrivacyDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.privacy == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		RequestID int64 `json:"request_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := h.privacy.ExecuteDeletion(r.Context(), session.UserID, input.RequestID); err != nil {
		if errors.Is(err, ports.ErrPrivacyRequestNotFound) {
			writeError(w, http.StatusNotFound, "PRIVACY_REQUEST_NOT_FOUND", "Deletion request not found.")
			return
		}
		if errors.Is(err, ports.ErrPrivacyRequestState) {
			writeError(w, http.StatusConflict, "PRIVACY_REQUEST_NOT_EXECUTABLE", "That deletion request is no longer executable.")
			return
		}
		writeError(w, http.StatusInternalServerError, "PRIVACY_DELETE_FAILED", "Eligible account data could not be deleted.")
		return
	}
	if h.sessions != nil {
		_ = h.sessions.RevokeAll(r.Context(), session.UserID)
	}
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) session(r *http.Request) (ports.SessionRecord, bool) {
	if h.sessions == nil {
		return ports.SessionRecord{}, false
	}
	cookie, err := r.Cookie("wecratfs_session")
	if err != nil {
		return ports.SessionRecord{}, false
	}
	record, err := h.sessions.Get(r.Context(), cookie.Value)
	return record, err == nil
}

func (h *Handler) validCSRF(r *http.Request, session ports.SessionRecord) bool {
	return session.CSRFToken != "" && subtle.ConstantTimeCompare([]byte(session.CSRFToken), []byte(r.Header.Get("X-CSRF-Token"))) == 1
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "The JSON request is invalid.")
		return false
	}
	return true
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, session ports.SessionRecord) {
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_session", Value: session.ID, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt, MaxAge: int(time.Until(session.ExpiresAt).Seconds())})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_session", Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func (h *Handler) mergeGuestCart(ctx context.Context, w http.ResponseWriter, r *http.Request, userID int64) {
	if h.cart == nil {
		return
	}
	cookie, err := r.Cookie("wecratfs_guest_cart")
	if err != nil || cookie.Value == "" {
		return
	}
	digest := sha256.Sum256([]byte(cookie.Value))
	if err := h.cart.MergeGuest(ctx, userID, hex.EncodeToString(digest[:])); err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "wecratfs_guest_cart", Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	requestID := w.Header().Get("X-Request-ID")
	errorBody := map[string]string{
		"code":    code,
		"message": message,
	}
	if requestID != "" {
		errorBody["request_id"] = requestID
	}
	writeJSON(w, status, map[string]any{
		"error": errorBody,
	})
}

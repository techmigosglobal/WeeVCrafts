package web

import (
	"errors"
	"net/http"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) orderPaymentStatus(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	detail, session, ok := h.customerOrderDetail(w, r, orderNumber)
	if !ok {
		return
	}
	h.render(w, http.StatusOK, "payment-status", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Payment status · " + detail.OrderNumber, Authenticated: true,
		AccountUserID: session.UserID, CSRFToken: session.CSRFToken, OrderDetail: detail, HasOrderDetail: true,
	})
}

func (h *Handler) orderTracking(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	detail, session, ok := h.customerOrderDetail(w, r, orderNumber)
	if !ok {
		return
	}
	h.render(w, http.StatusOK, "tracking", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Shipment tracking · " + detail.OrderNumber, Authenticated: true,
		AccountUserID: session.UserID, CSRFToken: session.CSRFToken, OrderDetail: detail, HasOrderDetail: true,
	})
}

func (h *Handler) customerOrderDetail(w http.ResponseWriter, r *http.Request, orderNumber string) (domaincommerce.OrderDetail, ports.SessionRecord, bool) {
	if h.orderService == nil || strings.TrimSpace(orderNumber) == "" || strings.Contains(orderNumber, "/") {
		h.notFound(w, r)
		return domaincommerce.OrderDetail{}, ports.SessionRecord{}, false
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return domaincommerce.OrderDetail{}, ports.SessionRecord{}, false
	}
	detail, err := h.orderService.Get(r.Context(), session.UserID, orderNumber)
	if errors.Is(err, ports.ErrOrderNotFound) {
		h.notFound(w, r)
		return domaincommerce.OrderDetail{}, ports.SessionRecord{}, false
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Order unavailable", "We could not load this order.")
		return domaincommerce.OrderDetail{}, ports.SessionRecord{}, false
	}
	return detail, session, true
}

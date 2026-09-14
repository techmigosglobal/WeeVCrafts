package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) sellerOrders(w http.ResponseWriter, r *http.Request) {
	if h.sellerOrderService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	orders, err := h.sellerOrderService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Orders access denied", "Your seller account does not have order-read permission.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Orders unavailable", "We could not load the live seller order queue.")
		return
	}
	h.render(w, http.StatusOK, "seller-orders", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Seller orders · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, SellerOrders: orders,
	})
}

func (h *Handler) sellerOrderFulfill(w http.ResponseWriter, r *http.Request) {
	if h.sellerOrderService == nil || r.Method != http.MethodPost {
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
	sellerID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("seller_id")), 10, 64)
	if err != nil || sellerID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Fulfilment request invalid", "Choose a valid seller order.")
		return
	}
	err = h.sellerOrderService.UpdateFulfillment(r.Context(), session.UserID, sellerID, r.FormValue("order_number"), r.FormValue("status"), r.FormValue("carrier"), r.FormValue("tracking_number"), r.FormValue("note"))
	if err != nil {
		switch {
		case errors.Is(err, applicationcommerce.ErrInvalidSellerFulfillment):
			h.renderError(w, http.StatusUnprocessableEntity, "Fulfilment details invalid", "Choose the next available status and provide tracking details before shipping.")
		case errors.Is(err, ports.ErrForbidden):
			h.renderError(w, http.StatusForbidden, "Fulfilment access denied", "Your seller account does not have order-fulfilment permission.")
		case errors.Is(err, ports.ErrOrderNotFound):
			h.renderError(w, http.StatusNotFound, "Seller order unavailable", "That order is not part of your seller workspace.")
		case errors.Is(err, ports.ErrInvalidState):
			h.renderError(w, http.StatusConflict, "Fulfilment state changed", "Reload the seller order queue before applying another transition.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Fulfilment unavailable", "We could not save that fulfilment update.")
		}
		return
	}
	h.redirect(w, r, "/seller/orders")
}

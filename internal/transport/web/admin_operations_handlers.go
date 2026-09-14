package web

import (
	"errors"
	"net/http"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminOrders(w http.ResponseWriter, r *http.Request) {
	if h.adminOperations == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	orders, err := h.adminOperations.ListOrders(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Order administration restricted", "Only marketplace, operations, and super administrators can inspect marketplace orders.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Order queue unavailable", "We could not load the live marketplace order queue.")
		return
	}
	h.render(w, http.StatusOK, "admin-orders", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Marketplace orders · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, AdminOrders: orders,
	})
}

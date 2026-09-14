package web

import (
	"errors"
	"net/http"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminCustomers(w http.ResponseWriter, r *http.Request) {
	if h.customerDirectory == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	customers, err := h.customerDirectory.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Customer administration restricted", "Only marketplace and Super Admin roles can inspect the customer directory.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Customer directory unavailable", "We could not load the live customer directory.")
		return
	}
	h.render(w, http.StatusOK, "admin-customers", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Customers · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, AdminCustomers: customers,
	})
}

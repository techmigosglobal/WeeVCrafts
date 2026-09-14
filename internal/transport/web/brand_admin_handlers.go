package web

import (
	"errors"
	"net/http"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminBrands(w http.ResponseWriter, r *http.Request) {
	if h.brandDirectory == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	brands, err := h.brandDirectory.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Brand administration restricted", "Only marketplace and Super Admin roles can inspect the brand directory.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Brand directory unavailable", "We could not load the live brand directory.")
		return
	}
	h.render(w, http.StatusOK, "admin-brands", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Brands · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, AdminBrands: brands,
	})
}

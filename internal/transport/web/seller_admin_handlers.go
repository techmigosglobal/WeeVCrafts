package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminSellers(w http.ResponseWriter, r *http.Request) {
	if h.sellerAdminService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	sellers, err := h.sellerAdminService.List(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, ports.ErrForbidden) {
			h.renderError(w, http.StatusForbidden, "Seller administration restricted", "Only marketplace and super administrators can review seller accounts.")
			return
		}
		h.renderError(w, http.StatusServiceUnavailable, "Seller administration unavailable", "We could not load the seller review queue.")
		return
	}
	h.render(w, http.StatusOK, "admin-sellers", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Seller administration · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, SellerApplications: sellers})
}

func (h *Handler) adminSellerStatus(w http.ResponseWriter, r *http.Request) {
	if h.sellerAdminService == nil || r.Method != http.MethodPost {
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
	sellerID, err := strconv.ParseInt(r.FormValue("seller_id"), 10, 64)
	if err != nil || sellerID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Seller decision invalid", "Choose a valid seller application.")
		return
	}
	if err := h.sellerAdminService.UpdateStatus(r.Context(), session.UserID, sellerID, r.FormValue("status"), r.FormValue("reason")); err != nil {
		switch {
		case errors.Is(err, ports.ErrForbidden):
			h.renderError(w, http.StatusForbidden, "Seller administration restricted", "Only marketplace and super administrators can change seller status.")
		case errors.Is(err, ports.ErrSellerNotFound):
			h.renderError(w, http.StatusNotFound, "Seller not found", "That seller application is no longer available.")
		case errors.Is(err, ports.ErrSellerState):
			h.renderError(w, http.StatusConflict, "Seller state changed", "Reload the seller queue before applying another decision.")
		default:
			h.renderError(w, http.StatusUnprocessableEntity, "Seller decision invalid", "Choose an allowed status and provide a short reason.")
		}
		return
	}
	h.redirect(w, r, "/admin/sellers")
}

package web

import (
	"errors"
	"net/http"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminAudit(w http.ResponseWriter, r *http.Request) {
	if h.auditService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	entries, err := h.auditService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Audit access restricted", "Only marketplace and super administrators can inspect privileged audit records.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Audit log unavailable", "We could not load the privileged activity log.")
		return
	}
	h.render(w, http.StatusOK, "audit", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Audit log · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, AuditEntries: entries,
	})
}

func (h *Handler) sellerAudit(w http.ResponseWriter, r *http.Request) {
	if h.auditService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	entries, err := h.auditService.ListSeller(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Seller activity restricted", "Only the seller owner can inspect activity for this seller workspace.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Seller activity unavailable", "We could not load the seller activity log.")
		return
	}
	h.render(w, http.StatusOK, "seller-audit", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Seller activity · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, AuditEntries: entries,
	})
}

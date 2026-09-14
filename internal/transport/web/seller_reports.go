package web

import (
	"errors"
	"net/http"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) sellerAnalytics(w http.ResponseWriter, r *http.Request) {
	if h.sellerReportService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	report, err := h.sellerReportService.Summary(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Seller reports restricted", "Only the seller owner or a staff member with report access can view this workspace.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Seller reports unavailable", "We could not load the live seller order summary.")
		return
	}
	h.render(w, http.StatusOK, "seller-analytics", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Seller analytics · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, SellerReport: report,
	})
}

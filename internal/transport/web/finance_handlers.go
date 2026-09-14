package web

import (
	"errors"
	"net/http"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) financePage(w http.ResponseWriter, r *http.Request) {
	if h.financeService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	entries, err := h.financeService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Finance access denied", "This workspace is restricted to finance operators, marketplace administrators, and super administrators.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Finance unavailable", "We could not load the live payment and refund records.")
		return
	}
	view := "payments"
	title := "Payments and refunds"
	switch r.URL.Path {
	case "/admin/failed-payments":
		view = "failed-payments"
		title = "Failed payments"
		entries = filterFinanceEntries(entries, func(entry domaincommerce.FinanceEntry) bool { return strings.EqualFold(entry.PaymentStatus, "failed") })
	case "/admin/refunds":
		view = "refunds"
		title = "Refunds"
		entries = filterFinanceEntries(entries, func(entry domaincommerce.FinanceEntry) bool { return entry.RefundStatus != "" })
	case "/admin/finance-reconciliation":
		view = "reconciliation"
		title = "Finance reconciliation"
	}
	h.render(w, http.StatusOK, "finance", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: title + " · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, FinanceEntries: entries, FinanceView: view,
	})
}

func filterFinanceEntries(entries []domaincommerce.FinanceEntry, keep func(domaincommerce.FinanceEntry) bool) []domaincommerce.FinanceEntry {
	filtered := make([]domaincommerce.FinanceEntry, 0, len(entries))
	for _, entry := range entries {
		if keep(entry) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

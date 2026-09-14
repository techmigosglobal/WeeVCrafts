package web

import (
	"errors"
	"net/http"
	"strconv"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) sellerInventory(w http.ResponseWriter, r *http.Request) {
	if h.inventoryService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	items, err := h.inventoryService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Inventory access denied", "Your seller account does not have inventory-read permission.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Inventory unavailable", "We could not load the live stock records.")
		return
	}
	h.render(w, http.StatusOK, "seller-inventory", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Inventory · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, Inventory: items,
	})
}

func (h *Handler) sellerInventoryAdjust(w http.ResponseWriter, r *http.Request) {
	if h.inventoryService == nil || r.Method != http.MethodPost {
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
	variantID, variantErr := strconv.ParseInt(r.FormValue("variant_id"), 10, 64)
	delta, deltaErr := strconv.Atoi(r.FormValue("delta"))
	if variantErr != nil || deltaErr != nil {
		h.renderCommerceError(w, r, applicationcommerce.ErrInvalidInventoryAdjustment, "The stock adjustment is invalid.")
		return
	}
	err := h.inventoryService.Adjust(r.Context(), session.UserID, domaincommerce.InventoryAdjustment{
		VariantID: variantID,
		Delta:     delta,
		Reason:    r.FormValue("reason"),
	})
	if err != nil {
		h.renderCommerceError(w, r, err, "We could not save that stock adjustment.")
		return
	}
	h.redirect(w, r, "/seller/inventory")
}

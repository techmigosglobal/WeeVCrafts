package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) Wishlist(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodGet || !ok || h.cart == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sign in to access the wishlist.")
		return
	}
	items, err := h.cart.ListWishlist(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "WISHLIST_UNAVAILABLE", "Wishlist could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) WishlistAdd(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.cart == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		ProductID int64 `json:"product_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := h.cart.AddWishlist(r.Context(), session.UserID, input.ProductID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "WISHLIST_UPDATE_FAILED", "Product could not be added to the wishlist.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (h *Handler) WishlistRemove(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodDelete || !ok || h.cart == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	productID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/v1/wishlist/"), 10, 64)
	if err != nil || productID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "WISHLIST_UPDATE_FAILED", "Product identifier is invalid.")
		return
	}
	if err := h.cart.RemoveWishlist(r.Context(), session.UserID, productID); err != nil {
		if errors.Is(err, ports.ErrForbidden) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication is required.")
			return
		}
		writeError(w, http.StatusInternalServerError, "WISHLIST_UPDATE_FAILED", "Product could not be removed from the wishlist.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

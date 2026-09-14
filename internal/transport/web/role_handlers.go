package web

import (
	"errors"
	"net/http"
	"strconv"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) adminRoles(w http.ResponseWriter, r *http.Request) {
	if h.roleAdminService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	assignments, err := h.roleAdminService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Role administration restricted", "Only Super Admin can change operational role assignments.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Role administration unavailable", "We could not load the live role assignments.")
		return
	}
	h.render(w, http.StatusOK, "roles", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Role management · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, RoleAssignments: assignments,
	})
}

func (h *Handler) adminRoleUpdate(w http.ResponseWriter, r *http.Request) {
	if h.roleAdminService == nil || r.Method != http.MethodPost {
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
	userID, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Role change invalid", "Choose a valid user.")
		return
	}
	grant := r.FormValue("action") == "grant"
	if r.FormValue("action") != "grant" && r.FormValue("action") != "revoke" {
		h.renderError(w, http.StatusUnprocessableEntity, "Role change invalid", "Choose grant or revoke.")
		return
	}
	err = h.roleAdminService.Update(r.Context(), session.UserID, userID, r.FormValue("role"), grant, r.FormValue("reason"))
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrForbidden):
			h.renderError(w, http.StatusForbidden, "Role administration restricted", "Only Super Admin can change operational roles.")
		case errors.Is(err, ports.ErrNotFound):
			h.renderError(w, http.StatusNotFound, "User not found", "That user account is no longer available.")
		case errors.Is(err, ports.ErrRoleState), errors.Is(err, applicationauth.ErrInvalidRoleAssignment):
			h.renderError(w, http.StatusConflict, "Role change rejected", "The role assignment is invalid, unchanged, or protected.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Role change unavailable", "We could not save the role assignment.")
		}
		return
	}
	h.redirect(w, r, "/admin/roles")
}

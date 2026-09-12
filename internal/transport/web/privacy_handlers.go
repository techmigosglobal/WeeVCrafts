package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) privacyPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "privacy", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Privacy · " + identity.Name})
}

func (h *Handler) privacyCenter(w http.ResponseWriter, r *http.Request) {
	if h.privacyService == nil {
		h.notFound(w, r)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	center, err := h.privacyService.Center(r.Context(), session.UserID)
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Privacy center unavailable", "We could not load your privacy records.")
		return
	}
	h.render(w, http.StatusOK, "privacy-center", pageData{Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description, Title: "Privacy center · " + identity.Name, Authenticated: true, AccountUserID: session.UserID, CSRFToken: session.CSRFToken, PrivacyCenter: center})
}

func (h *Handler) privacyConsent(w http.ResponseWriter, r *http.Request) {
	session, ok := h.sessionFromRequest(r)
	if h.privacyService == nil || r.Method != http.MethodPost || !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	granted := r.FormValue("granted") == "true"
	if err := h.privacyService.SetConsent(r.Context(), session.UserID, r.FormValue("purpose"), granted); err != nil {
		h.renderCommerceError(w, r, err, "We could not update that consent choice.")
		return
	}
	h.redirect(w, r, "/privacy/center")
}

func (h *Handler) privacyRequest(w http.ResponseWriter, r *http.Request) {
	session, ok := h.sessionFromRequest(r)
	if h.privacyService == nil || r.Method != http.MethodPost || !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if _, err := h.privacyService.Request(r.Context(), session.UserID, r.FormValue("request_type"), r.FormValue("details")); err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, ports.ErrForbidden) {
			status = http.StatusForbidden
		}
		h.renderError(w, status, "Privacy request invalid", "Please choose a supported request type and keep the details concise.")
		return
	}
	h.redirect(w, r, "/privacy/center")
}

func (h *Handler) privacyDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := h.sessionFromRequest(r)
	if h.privacyService == nil || r.Method != http.MethodPost || !ok || !h.validSessionCSRF(r, session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	requestID, err := strconv.ParseInt(r.FormValue("request_id"), 10, 64)
	if err != nil || requestID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Deletion request invalid", "Please submit a valid deletion request.")
		return
	}
	if err := h.privacyService.ExecuteDeletion(r.Context(), session.UserID, requestID); err != nil {
		if errors.Is(err, ports.ErrPrivacyRequestNotFound) {
			h.renderError(w, http.StatusNotFound, "Deletion request not found", "That privacy request is not available for this account.")
			return
		}
		if errors.Is(err, ports.ErrPrivacyRequestState) {
			h.renderError(w, http.StatusConflict, "Deletion request unavailable", "That deletion request has already been completed or cannot be executed.")
			return
		}
		h.renderError(w, http.StatusServiceUnavailable, "Deletion unavailable", "Eligible account data could not be deleted.")
		return
	}
	_ = h.sessions.RevokeAll(r.Context(), session.UserID)
	h.clearSessionCookie(w)
	h.redirect(w, r, "/login")
}

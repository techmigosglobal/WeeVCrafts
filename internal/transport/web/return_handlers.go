package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	"github.com/wecratfs/commerce/internal/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) orderReturn(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if h.returnService == nil || r.Method != http.MethodPost || orderNumber == "" || strings.Contains(orderNumber, "/") {
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
	if _, err := h.returnService.Create(r.Context(), session.UserID, orderNumber, r.FormValue("reason")); err != nil {
		switch {
		case errors.Is(err, applicationcommerce.ErrInvalidReturnRequest):
			h.renderError(w, http.StatusUnprocessableEntity, "Return request invalid", "Tell us what went wrong in at least 10 characters.")
		case errors.Is(err, ports.ErrReturnNotFound):
			h.renderError(w, http.StatusNotFound, "Return unavailable", "This order is not available for a return request.")
		case errors.Is(err, ports.ErrReturnState):
			h.renderError(w, http.StatusConflict, "Return already requested", "A return request already exists for this order.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Return unavailable", "We could not save the return request.")
		}
		return
	}
	h.redirect(w, r, "/orders/"+orderNumber)
}

func (h *Handler) adminReturns(w http.ResponseWriter, r *http.Request) {
	if h.returnService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	requests, err := h.returnService.List(r.Context(), session.UserID)
	if errors.Is(err, ports.ErrForbidden) {
		h.renderError(w, http.StatusForbidden, "Returns access denied", "This queue is restricted to operations and marketplace administrators.")
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Returns unavailable", "We could not load the live return queue.")
		return
	}
	h.render(w, http.StatusOK, "admin-returns", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Returns queue · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, Returns: requests,
	})
}

func (h *Handler) adminReturnStatus(w http.ResponseWriter, r *http.Request) {
	if h.returnService == nil || r.Method != http.MethodPost {
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
	requestID, err := strconv.ParseInt(r.FormValue("request_id"), 10, 64)
	if err != nil || requestID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Return update invalid", "Choose a valid return request.")
		return
	}
	if err := h.returnService.Update(r.Context(), session.UserID, requestID, r.FormValue("status"), r.FormValue("reason")); err != nil {
		switch {
		case errors.Is(err, applicationcommerce.ErrInvalidReturnRequest):
			h.renderError(w, http.StatusUnprocessableEntity, "Return update invalid", "Choose an available return status.")
		case errors.Is(err, ports.ErrForbidden):
			h.renderError(w, http.StatusForbidden, "Returns access denied", "You do not have permission to update returns.")
		case errors.Is(err, ports.ErrReturnState):
			h.renderError(w, http.StatusConflict, "Return state changed", "Reload the queue before applying another status update.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Return update unavailable", "We could not save that return status.")
		}
		return
	}
	h.redirect(w, r, "/admin/returns")
}

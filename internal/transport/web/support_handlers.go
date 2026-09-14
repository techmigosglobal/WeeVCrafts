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

func (h *Handler) supportPage(w http.ResponseWriter, r *http.Request) {
	if h.supportService == nil || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.sessionFromRequest(r)
	if !ok {
		h.redirect(w, r, "/login")
		return
	}
	seller := r.URL.Path == "/seller/support"
	var tickets []domaincommerce.SupportTicket
	var err error
	agent := false
	if seller {
		tickets, err = h.supportService.ListSeller(r.Context(), session.UserID)
	} else {
		tickets, err = h.supportService.ListAgent(r.Context(), session.UserID)
		agent = err == nil
		if errors.Is(err, ports.ErrForbidden) {
			tickets, err = h.supportService.ListCustomer(r.Context(), session.UserID)
		}
	}
	if errors.Is(err, ports.ErrForbidden) {
		message := "Sign in as a customer or support agent to use this workspace."
		if seller {
			message = "Only an approved seller owner can use seller support."
		}
		h.renderError(w, http.StatusForbidden, "Support access denied", message)
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Support unavailable", "We could not load the live support requests.")
		return
	}
	h.render(w, http.StatusOK, "support", pageData{
		Brand: identity.Name, Tagline: identity.Tagline, Description: identity.Description,
		Title: "Support · " + identity.Name, Authenticated: true, AccountUserID: session.UserID,
		CSRFToken: session.CSRFToken, SupportTickets: tickets, SupportAgent: agent, SupportSeller: seller,
	})
}

func (h *Handler) supportCreate(w http.ResponseWriter, r *http.Request) {
	if h.supportService == nil || r.Method != http.MethodPost {
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
	seller := r.URL.Path == "/seller/support/create"
	var createErr error
	if seller {
		_, createErr = h.supportService.CreateSeller(r.Context(), session.UserID, r.FormValue("order_number"), r.FormValue("subject"), r.FormValue("message"))
	} else {
		_, createErr = h.supportService.Create(r.Context(), session.UserID, r.FormValue("order_number"), r.FormValue("subject"), r.FormValue("message"))
	}
	if createErr != nil {
		switch {
		case errors.Is(createErr, applicationcommerce.ErrInvalidSupportTicket):
			h.renderError(w, http.StatusUnprocessableEntity, "Support request invalid", "Use a subject of 4–160 characters and a message of at least 10 characters.")
		case errors.Is(createErr, ports.ErrOrderNotFound):
			h.renderError(w, http.StatusNotFound, "Order unavailable", "That order does not belong to your account.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Support unavailable", "We could not create the support request.")
		}
		return
	}
	if seller {
		h.redirect(w, r, "/seller/support")
	} else {
		h.redirect(w, r, "/support")
	}
}

func (h *Handler) supportStatus(w http.ResponseWriter, r *http.Request) {
	if h.supportService == nil || r.Method != http.MethodPost {
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
	ticketID, err := strconv.ParseInt(r.FormValue("ticket_id"), 10, 64)
	if err != nil || ticketID <= 0 {
		h.renderError(w, http.StatusUnprocessableEntity, "Support update invalid", "Choose a valid support ticket.")
		return
	}
	if err := h.supportService.Update(r.Context(), session.UserID, ticketID, r.FormValue("status"), r.FormValue("note")); err != nil {
		switch {
		case errors.Is(err, applicationcommerce.ErrInvalidSupportTicket):
			h.renderError(w, http.StatusUnprocessableEntity, "Support update invalid", "Choose a supported status and keep the note concise.")
		case errors.Is(err, ports.ErrForbidden):
			h.renderError(w, http.StatusForbidden, "Support access denied", "Only support agents and super administrators can update tickets.")
		case errors.Is(err, ports.ErrSupportNotFound):
			h.renderError(w, http.StatusNotFound, "Ticket unavailable", "That support ticket no longer exists.")
		case errors.Is(err, ports.ErrSupportState):
			h.renderError(w, http.StatusConflict, "Support state changed", "Reload the queue before applying another status update.")
		default:
			h.renderError(w, http.StatusServiceUnavailable, "Support unavailable", "We could not save that support update.")
		}
		return
	}
	h.redirect(w, r, "/support")
}

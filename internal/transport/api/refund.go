package api

import (
	"errors"
	"net/http"
	"strings"

	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	if h.refunds == nil {
		writeError(w, http.StatusServiceUnavailable, "REFUNDS_UNAVAILABLE", "Refunds are not configured.")
		return
	}
	var input struct {
		Reason         string `json:"reason"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); headerKey != "" {
		input.IdempotencyKey = headerKey
	}
	path := strings.TrimSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/refund")
	orderNumber := strings.Trim(strings.TrimPrefix(path, "/api/v1/orders/"), "/")
	if orderNumber == "" || strings.Contains(orderNumber, "/") {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order not found.")
		return
	}
	refund, err := h.refunds.Refund(r.Context(), session.UserID, orderNumber, input.IdempotencyKey, input.Reason)
	if errors.Is(err, applicationpayment.ErrRefundInvalidRequest) {
		writeError(w, http.StatusUnprocessableEntity, "REFUND_INVALID", "A valid idempotency key and refund reason are required.")
		return
	}
	if errors.Is(err, ports.ErrRefundNotFound) {
		writeError(w, http.StatusNotFound, "REFUND_NOT_FOUND", "Refundable order not found.")
		return
	}
	if errors.Is(err, ports.ErrRefundState) {
		writeError(w, http.StatusConflict, "REFUND_NOT_ALLOWED", "This order is not currently refundable.")
		return
	}
	if errors.Is(err, applicationpayment.ErrRefundCreationFailed) || errors.Is(err, applicationpayment.ErrRefundResponseMismatch) {
		writeError(w, http.StatusBadGateway, "REFUND_PROVIDER_FAILED", "The payment provider could not complete this refund.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "REFUND_FAILED", "The refund could not be completed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": refund})
}

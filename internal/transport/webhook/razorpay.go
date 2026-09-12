package webhook

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
)

type Handler struct{ payments *applicationpayment.Service }

func NewRazorpayHandler(payments *applicationpayment.Service) *Handler {
	return &Handler{payments: payments}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.payments == nil {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	processed, err := h.payments.HandleWebhook(r.Context(), body, r.Header.Get("X-Razorpay-Signature"))
	if errors.Is(err, applicationpayment.ErrWebhookRejected) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "webhook rejected"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "webhook processing failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"processed": processed})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

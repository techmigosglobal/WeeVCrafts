package api

import (
	"errors"
	"net/http"
	"strings"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.recovery == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported.")
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := h.recovery.RequestPasswordReset(r.Context(), input.Email); err != nil {
		switch {
		case errors.Is(err, applicationauth.ErrRecoveryInvalidRequest):
			writeError(w, http.StatusUnprocessableEntity, "RECOVERY_INVALID", "Enter a valid email address.")
		case errors.Is(err, applicationauth.ErrRecoveryUnavailable):
			writeError(w, http.StatusServiceUnavailable, "RECOVERY_UNAVAILABLE", "Recovery email delivery is not configured.")
		default:
			writeError(w, http.StatusServiceUnavailable, "RECOVERY_UNAVAILABLE", "Recovery could not be started.")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.recovery == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported.")
		return
	}
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	userID, err := h.recovery.ResetPassword(r.Context(), strings.TrimSpace(input.Token), input.Password)
	if err != nil {
		switch {
		case errors.Is(err, applicationauth.ErrRecoveryInvalidRequest):
			writeError(w, http.StatusUnprocessableEntity, "RESET_INVALID", "The reset token or password is invalid.")
		case errors.Is(err, ports.ErrAuthTokenInvalid):
			writeError(w, http.StatusUnprocessableEntity, "RESET_EXPIRED", "That reset link is invalid or expired.")
		default:
			writeError(w, http.StatusInternalServerError, "RESET_FAILED", "The password could not be reset.")
		}
		return
	}
	if h.sessions != nil {
		_ = h.sessions.RevokeAll(r.Context(), userID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

func (h *Handler) EmailVerificationRequest(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.recovery == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	if err := h.recovery.RequestEmailVerification(r.Context(), session.UserID); err != nil {
		if errors.Is(err, applicationauth.ErrRecoveryInvalidRequest) {
			writeError(w, http.StatusUnprocessableEntity, "VERIFICATION_INVALID", "The verification request is invalid.")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "VERIFICATION_UNAVAILABLE", "Verification email delivery is not configured.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) EmailVerificationConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || h.recovery == nil {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported.")
		return
	}
	var input struct {
		Token string `json:"token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if _, err := h.recovery.VerifyEmail(r.Context(), strings.TrimSpace(input.Token)); err != nil {
		switch {
		case errors.Is(err, applicationauth.ErrRecoveryInvalidRequest):
			writeError(w, http.StatusUnprocessableEntity, "VERIFICATION_INVALID", "The verification token is invalid.")
		case errors.Is(err, ports.ErrAuthTokenInvalid):
			writeError(w, http.StatusUnprocessableEntity, "VERIFICATION_EXPIRED", "That verification link is invalid or expired.")
		default:
			writeError(w, http.StatusInternalServerError, "VERIFICATION_FAILED", "The email could not be verified.")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "email_verified"})
}

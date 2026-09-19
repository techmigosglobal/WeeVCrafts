package api

import (
	"errors"
	"io"
	"mime"
	"net/http"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) MediaUploadURL(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.media == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		ProductID   int64  `json:"product_id"`
		ContentType string `json:"content_type"`
		ByteSize    int64  `json:"byte_size"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	media, err := h.media.PrepareUpload(r.Context(), session.UserID, input.ProductID, input.ContentType, input.ByteSize)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "MEDIA_UPLOAD_INVALID", "Media upload request is invalid.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": media})
}

// MediaUpload receives small first-party uploads for Netlify's database-backed
// media adapter. It intentionally requires both the session cookie and CSRF
// token even though the upload URL itself contains an opaque object key.
func (h *Handler) MediaUpload(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPut || !ok || h.media == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	objectKey := r.URL.Query().Get("object_key")
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || objectKey == "" {
		writeError(w, http.StatusBadRequest, "INVALID_UPLOAD", "The media upload request is invalid.")
		return
	}
	maxBytes := h.media.MaxUploadBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "MEDIA_TOO_LARGE", "The image exceeds the upload size limit.")
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_UPLOAD", "The media upload request is invalid.")
		}
		return
	}
	if err := h.media.UploadObject(r.Context(), session.UserID, objectKey, contentType, body); err != nil {
		h.writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MediaFinalize(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.media == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		MediaID int64 `json:"media_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	media, err := h.media.FinalizeUpload(r.Context(), session.UserID, input.MediaID)
	if err != nil {
		h.writeMediaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": media})
}

func (h *Handler) MediaDelete(w http.ResponseWriter, r *http.Request) {
	session, ok := h.session(r)
	if r.Method != http.MethodPost || !ok || h.media == nil || !h.validCSRF(r, session) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Authentication and CSRF validation are required.")
		return
	}
	var input struct {
		MediaID int64 `json:"media_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := h.media.DeleteUpload(r.Context(), session.UserID, input.MediaID); err != nil {
		h.writeMediaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

func (h *Handler) writeMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "This media does not belong to the authenticated seller.")
	case errors.Is(err, applicationcommerce.ErrMediaNotReady):
		writeError(w, http.StatusConflict, "MEDIA_NOT_READY", "The media upload is not in a finalizable state.")
	case errors.Is(err, ports.ErrInvalidState):
		writeError(w, http.StatusConflict, "MEDIA_NOT_READY", "The media upload is not in a finalizable state.")
	case errors.Is(err, applicationcommerce.ErrInvalidMedia):
		writeError(w, http.StatusUnprocessableEntity, "MEDIA_INVALID", "The stored object did not match the validated upload.")
	default:
		writeError(w, http.StatusInternalServerError, "MEDIA_UNAVAILABLE", "Media lifecycle processing failed.")
	}
}

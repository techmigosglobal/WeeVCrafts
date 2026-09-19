package web

import (
	"errors"
	"net/http"
	"strconv"

	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func (h *Handler) publicMedia(w http.ResponseWriter, r *http.Request, mediaID string) {
	if r.Method != http.MethodGet || h.mediaService == nil {
		h.notFound(w, r)
		return
	}
	id, err := strconv.ParseInt(mediaID, 10, 64)
	if err != nil || id <= 0 {
		h.notFound(w, r)
		return
	}
	body, contentType, readErr := h.mediaService.ReadPublic(r.Context(), id)
	if readErr == nil {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}
	if !errors.Is(readErr, applicationcommerce.ErrMediaInlineUnsupported) {
		if errors.Is(readErr, ports.ErrNotFound) {
			h.notFound(w, r)
			return
		}
		h.renderError(w, http.StatusServiceUnavailable, "Media unavailable", "This product image is temporarily unavailable.")
		return
	}
	url, err := h.mediaService.ResolvePublicURL(r.Context(), id)
	if errors.Is(err, ports.ErrNotFound) {
		h.notFound(w, r)
		return
	}
	if err != nil {
		h.renderError(w, http.StatusServiceUnavailable, "Media unavailable", "This product image is temporarily unavailable.")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	http.Redirect(w, r, url, http.StatusFound)
}

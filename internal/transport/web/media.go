package web

import (
	"errors"
	"net/http"
	"strconv"

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

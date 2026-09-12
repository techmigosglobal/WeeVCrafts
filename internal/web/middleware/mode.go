package middleware

import (
	"net/http"
	"strings"
)

type Mode string

const (
	ModeMock Mode = "mock"
	ModeLive Mode = "live"
)

func ParseMode(value string) Mode {
	if strings.EqualFold(strings.TrimSpace(value), string(ModeLive)) {
		return ModeLive
	}
	return ModeMock
}

func PreviewHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}


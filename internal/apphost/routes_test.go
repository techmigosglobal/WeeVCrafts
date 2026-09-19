package apphost

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteAliasesMapPreviewWorkspacePathsToLiveWorkspaces(t *testing.T) {
	tests := map[string]string{
		"/seller-admin":                 "/seller",
		"/seller-admin/products/new":    "/seller/products/new",
		"/seller-admin/orders":          "/seller/orders",
		"/support-portal":               "/support",
		"/support-portal/create":        "/support/create",
		"/admin":                        "/admin/sellers",
		"/account":                      "/account/sessions",
		"/products/real-catalogue-item": "/products/real-catalogue-item",
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(r.URL.Path))
			})
			recorder := httptest.NewRecorder()
			routeAliases(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, input, nil))
			if recorder.Body.String() != expected {
				t.Fatalf("alias target = %q, want %q", recorder.Body.String(), expected)
			}
		})
	}
}

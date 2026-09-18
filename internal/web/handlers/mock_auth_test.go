package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMockLoginAccountsAndDestinations(t *testing.T) {
	for _, account := range mockAccounts {
		t.Run(string(account.Identity.Role), func(t *testing.T) {
			h := NewMockHandler()
			form := url.Values{"email": {account.Identity.Email}, "password": {account.Password}, "role": {string(account.Identity.Role)}}
			request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			h.ServeHTTP(response, request)
			if response.Code != http.StatusSeeOther {
				t.Fatalf("login status = %d, want 303", response.Code)
			}
			if got := response.Header().Get("Location"); got != account.Identity.Destination {
				t.Fatalf("login destination = %q, want %q", got, account.Identity.Destination)
			}
		})
	}
}

func TestMockLoginRejectsInvalidCredentials(t *testing.T) {
	h := NewMockHandler()
	form := url.Values{"email": {"admin@demo.weevcrafts.in"}, "password": {"wrong-password"}, "role": {string(roleAdmin)}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Sign-in failed") {
		t.Fatalf("invalid login = %d, want a visible failure state", response.Code)
	}
}

func TestMockPortalRoutesRequireAuthentication(t *testing.T) {
	for _, route := range []string{"/admin", "/seller-admin", "/support-portal"} {
		t.Run(route, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewMockHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
			if response.Code != http.StatusSeeOther || !strings.HasPrefix(response.Header().Get("Location"), "/login") {
				t.Fatalf("unauthenticated %s = %d %q, want login redirect", route, response.Code, response.Header().Get("Location"))
			}
		})
	}
}

func TestMockPortalRolesAreIsolated(t *testing.T) {
	h := NewMockHandler()
	vendorCookie := previewLogin(t, h, roleVendor)
	adminRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminRequest.AddCookie(vendorCookie)
	adminResponse := httptest.NewRecorder()
	h.ServeHTTP(adminResponse, adminRequest)
	if adminResponse.Code != http.StatusForbidden || !strings.Contains(adminResponse.Body.String(), "Access restricted") {
		t.Fatalf("vendor admin access = %d, want a clear 403 denial", adminResponse.Code)
	}

	supportCookie := previewLogin(t, h, roleSupport)
	sellerRequest := httptest.NewRequest(http.MethodGet, "/seller-admin", nil)
	sellerRequest.AddCookie(supportCookie)
	sellerResponse := httptest.NewRecorder()
	h.ServeHTTP(sellerResponse, sellerRequest)
	if sellerResponse.Code != http.StatusForbidden || !strings.Contains(sellerResponse.Body.String(), "Access restricted") {
		t.Fatalf("support seller access = %d, want a clear 403 denial", sellerResponse.Code)
	}
}

func TestMockSessionIdentityAndLogout(t *testing.T) {
	h := NewMockHandler()
	cookie := previewLogin(t, h, roleSupport)

	portalRequest := httptest.NewRequest(http.MethodGet, "/support-portal", nil)
	portalRequest.AddCookie(cookie)
	portalResponse := httptest.NewRecorder()
	h.ServeHTTP(portalResponse, portalRequest)
	if portalResponse.Code != http.StatusOK || !strings.Contains(portalResponse.Body.String(), "Aditi Rao") || !strings.Contains(portalResponse.Body.String(), "Support Agent") {
		t.Fatalf("support identity was not rendered from the session")
	}

	logoutRequest := httptest.NewRequest(http.MethodGet, "/logout", nil)
	logoutRequest.AddCookie(cookie)
	logoutResponse := httptest.NewRecorder()
	h.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusSeeOther || logoutResponse.Header().Get("Location") != "/login?notice=You+have+been+logged+out+of+the+preview" {
		t.Fatalf("logout = %d %q, want login redirect", logoutResponse.Code, logoutResponse.Header().Get("Location"))
	}

	afterLogout := httptest.NewRecorder()
	portalRequest = httptest.NewRequest(http.MethodGet, "/support-portal", nil)
	portalRequest.AddCookie(cookie)
	h.ServeHTTP(afterLogout, portalRequest)
	if afterLogout.Code != http.StatusSeeOther {
		t.Fatalf("portal after logout = %d, want 303", afterLogout.Code)
	}
}

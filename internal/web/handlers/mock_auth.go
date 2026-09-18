package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/wecratfs/commerce/internal/web/viewmodels"
	"github.com/wecratfs/commerce/web/pages"
)

type mockRole string

const (
	roleCustomer   mockRole = "customer"
	roleVendor     mockRole = "vendor"
	roleAdmin      mockRole = "marketplace_admin"
	roleSuperAdmin mockRole = "super_admin"
	roleSupport    mockRole = "support_agent"
)

type mockIdentity struct {
	Role                     mockRole
	Name, Email, Destination string
}

type mockAccount struct {
	Identity mockIdentity
	Password string
	Label    string
}

var mockAccounts = []mockAccount{
	{Identity: mockIdentity{Role: roleCustomer, Name: "Ananya Sharma", Email: "customer@demo.weevcrafts.in", Destination: "/account"}, Password: "Customer@123", Label: "Customer"},
	{Identity: mockIdentity{Role: roleVendor, Name: "Priya Sharma", Email: "vendor@demo.weevcrafts.in", Destination: "/seller-admin"}, Password: "Vendor@123", Label: "Vendor / Seller"},
	{Identity: mockIdentity{Role: roleAdmin, Name: "Rohan Mehta", Email: "admin@demo.weevcrafts.in", Destination: "/admin"}, Password: "Admin@123", Label: "Marketplace Admin"},
	{Identity: mockIdentity{Role: roleSuperAdmin, Name: "Rohan Mehta", Email: "superadmin@demo.weevcrafts.in", Destination: "/admin"}, Password: "SuperAdmin@123", Label: "Super Admin"},
	{Identity: mockIdentity{Role: roleSupport, Name: "Aditi Rao", Email: "support@demo.weevcrafts.in", Destination: "/support-portal"}, Password: "Support@123", Label: "Support Agent"},
}

func demoAccounts() []viewmodels.DemoAccount {
	accounts := make([]viewmodels.DemoAccount, 0, len(mockAccounts))
	for _, account := range mockAccounts {
		accounts = append(accounts, viewmodels.DemoAccount{
			Role:        string(account.Identity.Role),
			Label:       account.Label,
			Email:       account.Identity.Email,
			Password:    account.Password,
			Name:        account.Identity.Name,
			Destination: account.Identity.Destination,
		})
	}
	return accounts
}

func findMockAccount(email, password string) (mockAccount, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, account := range mockAccounts {
		if account.Identity.Email == email && account.Password == password {
			return account, true
		}
	}
	return mockAccount{}, false
}

func findMockAccountByRole(role string) (mockAccount, bool) {
	role = strings.TrimSpace(role)
	for _, account := range mockAccounts {
		if string(account.Identity.Role) == role {
			return account, true
		}
	}
	return mockAccount{}, false
}

func identityForRequest(h *MockHandler, w http.ResponseWriter, r *http.Request) (mockIdentity, bool) {
	return h.store.identity(w, r)
}

func (h *MockHandler) requireRoles(w http.ResponseWriter, r *http.Request, allowed ...mockRole) bool {
	identity, authenticated := identityForRequest(h, w, r)
	if !authenticated {
		next := r.URL.RequestURI()
		h.redirect(w, r, "/login?next="+url.QueryEscape(next))
		return false
	}
	for _, role := range allowed {
		if identity.Role == role {
			return true
		}
	}

	w.WriteHeader(http.StatusForbidden)
	h.render(w, r, h.accessDeniedPage(r, identity))
	return false
}

func (h *MockHandler) accessDeniedPage(r *http.Request, identity mockIdentity) templ.Component {
	return pages.AccessDenied(viewmodels.CustomerPage{
		Route:         "/access-denied",
		Title:         "Access restricted",
		Notice:        "Your " + roleLabel(identity.Role) + " preview account cannot open this workspace.",
		Description:   "This UI-only preview keeps each role in its own workspace.",
		Authenticated: true,
		SessionRole:   string(identity.Role),
		SessionName:   identity.Name,
		SessionEmail:  identity.Email,
		DemoAccounts:  demoAccounts(),
	})
}

func roleLabel(role mockRole) string {
	switch role {
	case roleCustomer:
		return "Customer"
	case roleVendor:
		return "Vendor"
	case roleAdmin:
		return "Marketplace Admin"
	case roleSuperAdmin:
		return "Super Admin"
	case roleSupport:
		return "Support Agent"
	default:
		return "preview"
	}
}

func isRolePortal(path string) bool {
	return path == "/admin" || strings.HasPrefix(path, "/admin/") || path == "/seller-admin" || strings.HasPrefix(path, "/seller-admin/") || path == "/support-portal" || strings.HasPrefix(path, "/support-portal/")
}

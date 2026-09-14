package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type operationRoles struct {
	roles map[int64]map[domainidentity.Role]bool
}

func (r operationRoles) HasAnyRole(_ context.Context, userID int64, roles ...domainidentity.Role) (bool, error) {
	for _, role := range roles {
		if r.roles[userID][role] {
			return true, nil
		}
	}
	return false, nil
}

type webSellerOrderRepository struct{}

type webSellerAdminRepository struct{}

func (webSellerAdminRepository) ListSellerApplications(context.Context, int64) ([]domaincommerce.SellerAdminEntry, error) {
	return []domaincommerce.SellerAdminEntry{{ID: 7, DisplayName: "Clay Studio", OwnerName: "Maker", OwnerEmail: "maker@example.com", Status: "pending", CreatedAt: time.Now()}}, nil
}
func (webSellerAdminRepository) UpdateSellerStatus(context.Context, int64, int64, string, string) error {
	return nil
}

type webAuditRepository struct{}

func (webAuditRepository) ListAuditEntries(context.Context, int64, int) ([]domaincommerce.AuditEntry, error) {
	return []domaincommerce.AuditEntry{{ID: 1, ActorID: "42", Action: "seller.status_updated", ResourceType: "seller", ResourceID: "7", RequestID: "req-7", Metadata: `{"status":"active"}`, CreatedAt: time.Now()}}, nil
}
func (webAuditRepository) ListSellerAuditEntries(context.Context, int64, int) ([]domaincommerce.AuditEntry, error) {
	return []domaincommerce.AuditEntry{{ID: 2, ActorID: "42", Action: "seller.inventory_adjusted", ResourceType: "inventory_variant", ResourceID: "9", RequestID: "req-9", Metadata: `{"delta":2}`, CreatedAt: time.Now()}}, nil
}

type webRoleAdminRepository struct{}

func (webRoleAdminRepository) ListRoleAssignments(context.Context, int64) ([]domainidentity.RoleAssignment, error) {
	return []domainidentity.RoleAssignment{{ID: 8, Email: "staff@example.com", DisplayName: "Staff", Status: "active", Roles: []string{"customer"}}}, nil
}
func (webRoleAdminRepository) UpdateRoleAssignment(context.Context, int64, int64, domainidentity.Role, bool, string) error {
	return nil
}

type webAdminOperationsRepository struct{}

type webBrandDirectoryRepository struct{}

func (webBrandDirectoryRepository) ListBrandDirectory(context.Context, int64) ([]domaincommerce.AdminBrand, error) {
	return []domaincommerce.AdminBrand{{Slug: "clay-house", Name: "Clay House", ProductCount: 2, CategoryCount: 1}}, nil
}

type webCustomerDirectoryRepository struct{}

func (webCustomerDirectoryRepository) ListCustomers(context.Context, int64, int) ([]domainidentity.AdminCustomer, error) {
	return []domainidentity.AdminCustomer{{ID: 9, DisplayName: "Aditi Rao", EmailMasked: "a***@example.com", Status: "active", OrderCount: 3, CreatedAt: time.Now()}}, nil
}

func (webAdminOperationsRepository) ListAdminOrders(context.Context, int64, int) ([]domaincommerce.AdminOrder, error) {
	return []domaincommerce.AdminOrder{{OrderNumber: "WC-42", Status: "paid", CustomerLabel: "Maker Customer", CustomerEmailMasked: "m***@example.com", PaymentStatus: "captured", FulfillmentStatus: "processing", TotalCents: 2500, Currency: "INR", CreatedAt: time.Now()}}, nil
}

func (webSellerOrderRepository) ListSellerOrders(context.Context, int64) ([]domaincommerce.SellerOrder, error) {
	return []domaincommerce.SellerOrder{{ID: 9, SellerID: 4, OrderNumber: "WC-9", OverallStatus: "paid", FulfillmentStatus: "pending", Currency: "INR", SellerSubtotalCents: 1250, CreatedAt: time.Now(), Items: []domaincommerce.SellerOrderItem{{ProductName: "Real bowl", SKU: "BOWL-9", Quantity: 1, LineTotalCents: 1250}}}}, nil
}

func (webSellerOrderRepository) UpdateSellerFulfillment(context.Context, int64, int64, string, string, string, string, string) error {
	return nil
}

type webFinanceRepository struct{}

func (webFinanceRepository) ListFinanceEntries(context.Context, int) ([]domaincommerce.FinanceEntry, error) {
	return []domaincommerce.FinanceEntry{{OrderNumber: "WC-9", AmountCents: 1250, Currency: "INR", PaymentStatus: "captured", OrderStatus: "paid", CreatedAt: time.Now()}}, nil
}

type webSupportRepository struct{}

func (webSupportRepository) CreateSupportTicket(context.Context, int64, string, string, string) (domaincommerce.SupportTicket, error) {
	return domaincommerce.SupportTicket{}, nil
}
func (webSupportRepository) ListCustomerSupportTickets(context.Context, int64) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-9", Subject: "A real question", Message: "A real support request.", Status: "open"}}, nil
}
func (webSupportRepository) ListSellerSupportTickets(context.Context, int64) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-9", Subject: "A seller question", Message: "A real seller support request.", Status: "open"}}, nil
}
func (webSupportRepository) ListSupportTickets(context.Context, int) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-9", Subject: "A real question", Message: "A real support request.", Status: "open", CustomerLabel: "Customer A***", CustomerEmailMasked: "a***@example.com"}}, nil
}
func (webSupportRepository) UpdateSupportTicket(context.Context, int64, int64, string, string) error {
	return nil
}

func TestLiveOperationsPagesRenderTheirDataContracts(t *testing.T) {
	session := ports.SessionRecord{ID: "operations-session", UserID: 42, CSRFToken: "operations-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	sessions := applicationauth.NewSessionService(&webSessionStore{session: session})
	handler, err := NewFullHandler(applicationcatalog.NewService(repository{}), nil, sessions, nil, nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("create operations handler: %v", err)
	}
	handler.SetSellerOrderService(applicationcommerce.NewSellerOrderService(webSellerOrderRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleSellerOwner: true}}}))
	handler.SetSellerReportService(applicationcommerce.NewSellerReportService(webSellerOrderRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleSellerOwner: true}}}))
	handler.SetSellerAdminService(applicationcommerce.NewSellerAdminService(webSellerAdminRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleMarketplaceAdmin: true}}}))
	handler.SetAuditService(applicationcommerce.NewAuditService(webAuditRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleMarketplaceAdmin: true, domainidentity.RoleSellerOwner: true}}}))
	handler.SetRoleAdminService(applicationauth.NewRoleAdminService(webRoleAdminRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleSuperAdmin: true}}}))
	handler.SetAdminOperationsService(applicationcommerce.NewAdminOperationsService(webAdminOperationsRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleMarketplaceAdmin: true}}}))
	handler.SetBrandDirectoryService(applicationcommerce.NewBrandDirectoryService(webBrandDirectoryRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleMarketplaceAdmin: true}}}))
	handler.SetCustomerDirectoryService(applicationauth.NewCustomerDirectoryService(webCustomerDirectoryRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleMarketplaceAdmin: true}}}))
	handler.SetFinanceService(applicationcommerce.NewFinanceService(webFinanceRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleFinanceOperator: true}}}))
	handler.SetSupportService(applicationcommerce.NewSupportService(webSupportRepository{}, operationRoles{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleSupportAgent: true, domainidentity.RoleSellerOwner: true}}}))

	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/seller/orders", want: "Seller orders"},
		{path: "/seller/analytics", want: "Seller analytics"},
		{path: "/admin/sellers", want: "Seller applications"},
		{path: "/admin/audit", want: "Audit log"},
		{path: "/seller/audit", want: "Seller activity"},
		{path: "/admin/roles", want: "Role management"},
		{path: "/admin/orders", want: "Marketplace orders"},
		{path: "/admin/brands", want: "Brands and categories"},
		{path: "/admin/customers", want: "Customer directory"},
		{path: "/finance", want: "Payments and refunds"},
		{path: "/admin/payments", want: "Payments and refunds"},
		{path: "/admin/failed-payments", want: "Failed payments"},
		{path: "/admin/refunds", want: "Refunds"},
		{path: "/admin/finance-reconciliation", want: "Finance reconciliation"},
		{path: "/support", want: "Support queue"},
		{path: "/seller/support", want: "Seller support"},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: session.ID})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("page %s = %d, want 200 with %q: %s", test.path, recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}

package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type sellerReportRepository struct {
	orders []domaincommerce.SellerOrder
}

func (r sellerReportRepository) ListSellerOrders(context.Context, int64) ([]domaincommerce.SellerOrder, error) {
	return r.orders, nil
}

func (sellerReportRepository) UpdateSellerFulfillment(context.Context, int64, int64, string, string, string, string, string) error {
	return nil
}

func TestSellerReportSummaryUsesSellerOrderProjection(t *testing.T) {
	service := NewSellerReportService(sellerReportRepository{orders: []domaincommerce.SellerOrder{
		{Currency: "INR", SellerSubtotalCents: 2500, FulfillmentStatus: "processing", Items: []domaincommerce.SellerOrderItem{{Quantity: 2}}},
		{Currency: "INR", SellerSubtotalCents: 1200, FulfillmentStatus: "delivered", Items: []domaincommerce.SellerOrderItem{{Quantity: 1}, {Quantity: 3}}},
	}}, roleChecker{roles: map[int64]map[domainidentity.Role]bool{42: {domainidentity.RoleSellerOwner: true}}})

	report, err := service.Summary(context.Background(), 42)
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
	if report.OrderCount != 2 || report.UnitsSold != 6 || report.GrossSalesCents != 3700 || report.Currency != "INR" || report.ProcessingOrders != 1 || report.DeliveredOrders != 1 {
		t.Fatalf("unexpected seller report: %#v", report)
	}
}

func TestSellerReportRequiresReportPermissionForStaff(t *testing.T) {
	checker := roleChecker{
		roles:       map[int64]map[domainidentity.Role]bool{17: {domainidentity.RoleSellerStaff: true}},
		permissions: map[int64]map[string]bool{17: {domaincommerce.SellerPermissionReportRead: true}},
	}
	service := NewSellerReportService(sellerReportRepository{}, checker)
	if _, err := service.Summary(context.Background(), 17); err != nil {
		t.Fatalf("report-enabled staff member was denied: %v", err)
	}
	if _, err := service.Summary(context.Background(), 18); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("staff member without report permission returned %v", err)
	}
}

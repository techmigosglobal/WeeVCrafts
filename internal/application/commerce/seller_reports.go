package commerce

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

// SellerReportService exposes seller-owned operational metrics without
// pretending that order totals are settled payouts. Commission and settlement
// reporting belongs to the future seller-finance ledger.
type SellerReportService struct {
	repository  ports.SellerOrderRepository
	roles       ports.RoleChecker
	permissions ports.SellerPermissionChecker
}

func NewSellerReportService(repository ports.SellerOrderRepository, roles ports.RoleChecker) *SellerReportService {
	service := &SellerReportService{repository: repository, roles: roles}
	if checker, ok := roles.(ports.SellerPermissionChecker); ok {
		service.permissions = checker
	}
	return service
}

func (s *SellerReportService) Summary(ctx context.Context, userID int64) (domaincommerce.SellerReport, error) {
	if !s.reportAllowed(ctx, userID) {
		return domaincommerce.SellerReport{}, ports.ErrForbidden
	}
	orders, err := s.repository.ListSellerOrders(ctx, userID)
	if err != nil {
		return domaincommerce.SellerReport{}, err
	}

	var report domaincommerce.SellerReport
	for _, order := range orders {
		report.OrderCount++
		report.GrossSalesCents += order.SellerSubtotalCents
		if report.Currency == "" {
			report.Currency = order.Currency
		} else if order.Currency != "" && report.Currency != order.Currency {
			report.MixedCurrencies = true
		}
		for _, item := range order.Items {
			report.UnitsSold += item.Quantity
		}
		switch order.FulfillmentStatus {
		case "pending":
			report.PendingOrders++
		case "processing":
			report.ProcessingOrders++
		case "shipped":
			report.ShippedOrders++
		case "delivered":
			report.DeliveredOrders++
		}
	}
	if report.MixedCurrencies {
		report.Currency = "Multiple currencies"
	}
	return report, nil
}

func (s *SellerReportService) reportAllowed(ctx context.Context, userID int64) bool {
	if userID <= 0 || s.repository == nil {
		return false
	}
	if s.roles != nil {
		allowed, err := s.roles.HasAnyRole(ctx, userID, domainidentity.RoleSellerOwner)
		if err == nil && allowed {
			return true
		}
	}
	if s.permissions == nil {
		return false
	}
	allowed, err := s.permissions.HasSellerPermission(ctx, userID, domaincommerce.SellerPermissionReportRead)
	return err == nil && allowed
}

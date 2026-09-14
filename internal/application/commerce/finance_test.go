package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type financeRepository struct{}

func (financeRepository) ListFinanceEntries(context.Context, int) ([]domaincommerce.FinanceEntry, error) {
	return []domaincommerce.FinanceEntry{{OrderNumber: "WC-1", AmountCents: 1200}}, nil
}

func TestFinanceServiceAllowsFinanceButNotSellerOrCustomer(t *testing.T) {
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		20: {domainidentity.RoleFinanceOperator: true},
		21: {domainidentity.RoleSellerOwner: true},
		22: {domainidentity.RoleMarketplaceAdmin: true},
	}}
	service := NewFinanceService(financeRepository{}, checker)
	entries, err := service.List(context.Background(), 20)
	if err != nil || len(entries) != 1 {
		t.Fatalf("finance operator denied: entries=%+v err=%v", entries, err)
	}
	if _, err := service.List(context.Background(), 21); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("seller could view finance records: %v", err)
	}
	if entries, err := service.List(context.Background(), 22); err != nil || len(entries) != 1 {
		t.Fatalf("marketplace admin denied finance records: entries=%+v err=%v", entries, err)
	}
}

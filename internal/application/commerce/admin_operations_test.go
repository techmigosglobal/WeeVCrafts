package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type adminOperationsRepository struct{}

func (adminOperationsRepository) ListAdminOrders(context.Context, int64, int) ([]domaincommerce.AdminOrder, error) {
	return []domaincommerce.AdminOrder{{OrderNumber: "WC-42", CustomerEmailMasked: "c***@example.com"}}, nil
}

type adminOperationsRoles struct{ allowed bool }

func (r adminOperationsRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

func TestAdminOperationsServiceScopesOrderReads(t *testing.T) {
	service := NewAdminOperationsService(adminOperationsRepository{}, adminOperationsRoles{})
	if _, err := service.ListOrders(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected order denial, got %v", err)
	}
	service = NewAdminOperationsService(adminOperationsRepository{}, adminOperationsRoles{allowed: true})
	orders, err := service.ListOrders(context.Background(), 42)
	if err != nil || len(orders) != 1 || orders[0].CustomerEmailMasked != "c***@example.com" {
		t.Fatalf("authorized order read = %#v, %v", orders, err)
	}
}

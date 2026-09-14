package auth

import (
	"context"
	"errors"
	"testing"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type customerDirectoryRoles struct{ allowed bool }

func (r customerDirectoryRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

type customerDirectoryRepository struct{}

func (customerDirectoryRepository) ListCustomers(context.Context, int64, int) ([]domainidentity.AdminCustomer, error) {
	return []domainidentity.AdminCustomer{{DisplayName: "Customer"}}, nil
}

func TestCustomerDirectoryRequiresMarketplaceScope(t *testing.T) {
	service := NewCustomerDirectoryService(customerDirectoryRepository{}, customerDirectoryRoles{})
	if _, err := service.List(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected forbidden without marketplace scope, got %v", err)
	}
	service = NewCustomerDirectoryService(customerDirectoryRepository{}, customerDirectoryRoles{allowed: true})
	customers, err := service.List(context.Background(), 42)
	if err != nil || len(customers) != 1 {
		t.Fatalf("expected one authorized customer, got customers=%v err=%v", customers, err)
	}
}

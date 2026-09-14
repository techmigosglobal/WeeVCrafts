package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type sellerAdminRepository struct{}

func (sellerAdminRepository) ListSellerApplications(context.Context, int64) ([]domaincommerce.SellerAdminEntry, error) {
	return []domaincommerce.SellerAdminEntry{{ID: 7, Status: "pending"}}, nil
}
func (sellerAdminRepository) UpdateSellerStatus(context.Context, int64, int64, string, string) error {
	return nil
}

type sellerAdminRoles struct{ allowed bool }

func (r sellerAdminRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

func TestSellerAdminServiceRequiresMarketplaceRole(t *testing.T) {
	service := NewSellerAdminService(sellerAdminRepository{}, sellerAdminRoles{})
	if _, err := service.List(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected list denial, got %v", err)
	}
	if err := service.UpdateStatus(context.Background(), 42, 7, "active", "approved"); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected update denial, got %v", err)
	}
}

func TestSellerAdminServiceValidatesStatusAndReason(t *testing.T) {
	service := NewSellerAdminService(sellerAdminRepository{}, sellerAdminRoles{allowed: true})
	if err := service.UpdateStatus(context.Background(), 42, 7, "pending", ""); !errors.Is(err, ErrInvalidSellerStatus) {
		t.Fatalf("expected invalid status, got %v", err)
	}
	if err := service.UpdateStatus(context.Background(), 42, 7, "active", string(make([]byte, 1001))); !errors.Is(err, ErrInvalidSellerStatus) {
		t.Fatalf("expected oversized reason, got %v", err)
	}
	if err := service.UpdateStatus(context.Background(), 42, 7, "active", "approved after review"); err != nil {
		t.Fatalf("valid status update failed: %v", err)
	}
}

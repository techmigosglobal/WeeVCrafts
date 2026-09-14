package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type auditRepository struct{}

func (auditRepository) ListAuditEntries(context.Context, int64, int) ([]domaincommerce.AuditEntry, error) {
	return []domaincommerce.AuditEntry{{ID: 1, Action: "seller.status_updated"}}, nil
}
func (auditRepository) ListSellerAuditEntries(context.Context, int64, int) ([]domaincommerce.AuditEntry, error) {
	return []domaincommerce.AuditEntry{{ID: 2, Action: "seller.inventory_adjusted"}}, nil
}

type auditRoles struct{ allowed bool }

func (r auditRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

func TestAuditServiceRestrictsAuditReads(t *testing.T) {
	service := NewAuditService(auditRepository{}, auditRoles{})
	if _, err := service.List(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected audit denial, got %v", err)
	}
	service = NewAuditService(auditRepository{}, auditRoles{allowed: true})
	entries, err := service.List(context.Background(), 42)
	if err != nil || len(entries) != 1 {
		t.Fatalf("authorized audit read = %#v, %v", entries, err)
	}
}

func TestAuditServiceRestrictsSellerActivityToSellerOwners(t *testing.T) {
	service := NewAuditService(auditRepository{}, sellerAuditRoles{allowed: false})
	if _, err := service.ListSeller(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected seller audit denial, got %v", err)
	}
	service = NewAuditService(auditRepository{}, sellerAuditRoles{allowed: true})
	entries, err := service.ListSeller(context.Background(), 42)
	if err != nil || len(entries) != 1 || entries[0].Action != "seller.inventory_adjusted" {
		t.Fatalf("authorized seller audit read = %#v, %v", entries, err)
	}
}

type sellerAuditRoles struct{ allowed bool }

func (r sellerAuditRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

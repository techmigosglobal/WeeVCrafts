package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type brandDirectoryRoles struct{ allowed bool }

func (r brandDirectoryRoles) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.allowed, nil
}

type brandDirectoryRepository struct{}

func (brandDirectoryRepository) ListBrandDirectory(context.Context, int64) ([]domaincommerce.AdminBrand, error) {
	return []domaincommerce.AdminBrand{{Name: "Clay House"}}, nil
}

func TestBrandDirectoryRequiresMarketplaceScope(t *testing.T) {
	service := NewBrandDirectoryService(brandDirectoryRepository{}, brandDirectoryRoles{})
	if _, err := service.List(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected forbidden without marketplace scope, got %v", err)
	}
	service = NewBrandDirectoryService(brandDirectoryRepository{}, brandDirectoryRoles{allowed: true})
	brands, err := service.List(context.Background(), 42)
	if err != nil || len(brands) != 1 {
		t.Fatalf("expected one authorized brand, got brands=%v err=%v", brands, err)
	}
}

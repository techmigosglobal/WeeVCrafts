package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type inventoryRepository struct {
	adjustment domaincommerce.InventoryAdjustment
}

func (r *inventoryRepository) ListSellerInventory(context.Context, int64) ([]domaincommerce.InventoryItem, error) {
	return []domaincommerce.InventoryItem{{VariantID: 7, AvailableQuantity: 4}}, nil
}

func (r *inventoryRepository) AdjustSellerInventory(_ context.Context, _ int64, adjustment domaincommerce.InventoryAdjustment) error {
	r.adjustment = adjustment
	return nil
}

func TestInventoryServiceSeparatesReadAndWritePermissions(t *testing.T) {
	repository := &inventoryRepository{}
	checker := roleChecker{
		roles:       map[int64]map[domainidentity.Role]bool{10: {domainidentity.RoleSellerOwner: true}, 11: {domainidentity.RoleSellerStaff: true}},
		permissions: map[int64]map[string]bool{11: {domaincommerce.SellerPermissionInventoryRead: true}},
	}
	service := NewInventoryService(repository, checker)
	if items, err := service.List(context.Background(), 11); err != nil || len(items) != 1 {
		t.Fatalf("inventory reader was denied: items=%+v err=%v", items, err)
	}
	if err := service.Adjust(context.Background(), 11, domaincommerce.InventoryAdjustment{VariantID: 7, Delta: 3, Reason: "received stock"}); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("inventory reader adjusted stock: %v", err)
	}
	checker.permissions[11][domaincommerce.SellerPermissionInventoryWrite] = true
	if err := service.Adjust(context.Background(), 11, domaincommerce.InventoryAdjustment{VariantID: 7, Delta: 3, Reason: "received stock"}); err != nil {
		t.Fatalf("inventory writer was denied: %v", err)
	}
	if repository.adjustment.Delta != 3 || repository.adjustment.Reason != "received stock" {
		t.Fatalf("adjustment was not passed through: %+v", repository.adjustment)
	}
}

func TestInventoryServiceRejectsUnsafeAdjustments(t *testing.T) {
	service := NewInventoryService(&inventoryRepository{}, roleChecker{roles: map[int64]map[domainidentity.Role]bool{10: {domainidentity.RoleSellerOwner: true}}})
	for _, adjustment := range []domaincommerce.InventoryAdjustment{
		{VariantID: 0, Delta: 1, Reason: "count"},
		{VariantID: 7, Delta: 0, Reason: "count"},
		{VariantID: 7, Delta: 1, Reason: "no"},
	} {
		if err := service.Adjust(context.Background(), 10, adjustment); !errors.Is(err, ErrInvalidInventoryAdjustment) {
			t.Fatalf("adjustment %+v returned %v", adjustment, err)
		}
	}
}

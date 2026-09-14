package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type catalogRepository struct{}

type roleChecker struct {
	roles       map[int64]map[domainidentity.Role]bool
	permissions map[int64]map[string]bool
}

func (r roleChecker) HasAnyRole(_ context.Context, userID int64, roles ...domainidentity.Role) (bool, error) {
	for _, role := range roles {
		if r.roles[userID][role] {
			return true, nil
		}
	}
	return false, nil
}

func (r roleChecker) HasSellerPermission(_ context.Context, userID int64, permission string) (bool, error) {
	return r.permissions[userID][permission], nil
}

func (catalogRepository) EnsureSeller(context.Context, int64, string) (domaincommerce.Seller, error) {
	return domaincommerce.Seller{ID: 1}, nil
}
func (catalogRepository) GetSeller(context.Context, int64) (domaincommerce.Seller, error) {
	return domaincommerce.Seller{ID: 1, Status: "active"}, nil
}
func (catalogRepository) CreateDraft(context.Context, int64, domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	return domaincommerce.ManagedProduct{ID: 1}, nil
}
func (catalogRepository) GetSellerProduct(context.Context, int64, int64) (domaincommerce.ManagedProduct, error) {
	return domaincommerce.ManagedProduct{ID: 1, Status: "draft"}, nil
}
func (catalogRepository) UpdateDraft(context.Context, int64, int64, domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	return domaincommerce.ManagedProduct{ID: 1, Status: "draft"}, nil
}
func (catalogRepository) SubmitProduct(context.Context, int64, int64) error { return nil }
func (catalogRepository) ApproveProduct(context.Context, int64, int64, bool, string) error {
	return nil
}
func (catalogRepository) ListSellerProducts(context.Context, int64) ([]domaincommerce.ManagedProduct, error) {
	return nil, nil
}

func TestCatalogServiceAppliesRolePolicyBeforeRepository(t *testing.T) {
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		10: {domainidentity.RoleCustomer: true},
		11: {domainidentity.RoleSellerOwner: true},
		13: {domainidentity.RoleSellerStaff: true},
		12: {domainidentity.RoleMarketplaceAdmin: true},
	}, permissions: map[int64]map[string]bool{13: {domaincommerce.SellerPermissionProductWrite: true}}}
	service := NewCatalogServiceWithRoles(catalogRepository{}, checker)

	if _, err := service.CreateDraft(context.Background(), 10, validDraft()); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("customer created a draft without seller role: %v", err)
	}
	if _, err := service.CreateDraft(context.Background(), 11, validDraft()); err != nil {
		t.Fatalf("seller owner was denied draft creation: %v", err)
	}
	if _, err := service.CreateDraft(context.Background(), 13, validDraft()); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("product writer seeded stock without inventory permission: %v", err)
	}
	if err := service.ApproveProduct(context.Background(), 11, 1, true, ""); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("seller owner approved a product: %v", err)
	}
	if err := service.ApproveProduct(context.Background(), 12, 1, true, "reviewed"); err != nil {
		t.Fatalf("marketplace admin was denied approval: %v", err)
	}
}

func validDraft() domaincommerce.ProductDraftInput {
	return domaincommerce.ProductDraftInput{
		Slug: "handmade-soap", BrandSlug: "earth-studio", BrandName: "Earth Studio",
		CategorySlug: "natural-care", CategoryName: "Natural Care", Name: "Neem Soap",
		Description: "Handmade botanical soap", PriceCents: 12900, SKU: "SOAP-001", InitialStock: 10,
	}
}
func (catalogRepository) ListPendingProducts(context.Context, int64) ([]domaincommerce.ManagedProduct, error) {
	return nil, nil
}

func TestCatalogServiceRejectsInvalidSlugsAndPrices(t *testing.T) {
	service := NewCatalogService(catalogRepository{})
	_, err := service.CreateDraft(context.Background(), 1, domaincommerce.ProductDraftInput{Name: "Soap", Slug: "Not Valid", BrandSlug: "brand", BrandName: "Brand", CategorySlug: "care", CategoryName: "Care", Description: "details", PriceCents: 100, SKU: "SKU", InitialStock: 1})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected invalid product, got %v", err)
	}
	compareAt := int64(99)
	_, err = service.CreateDraft(context.Background(), 1, domaincommerce.ProductDraftInput{Name: "Soap", Slug: "soap", BrandSlug: "brand", BrandName: "Brand", CategorySlug: "care", CategoryName: "Care", Description: "details", PriceCents: 100, CompareAtCents: &compareAt, SKU: "SKU", InitialStock: 1})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected invalid compare-at price, got %v", err)
	}
}

func TestCatalogServiceProtectsSellerDraftEditing(t *testing.T) {
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		11: {domainidentity.RoleSellerOwner: true},
		12: {domainidentity.RoleSellerStaff: true},
	}, permissions: map[int64]map[string]bool{
		12: {domaincommerce.SellerPermissionProductRead: true, domaincommerce.SellerPermissionProductWrite: true},
	}}
	service := NewCatalogServiceWithRoles(catalogRepository{}, checker)

	if _, err := service.GetSellerProduct(context.Background(), 12, 1); err != nil {
		t.Fatalf("seller staff with product-read permission was denied: %v", err)
	}
	if _, err := service.UpdateDraft(context.Background(), 12, 1, validDraft()); err != nil {
		t.Fatalf("seller staff with product-write permission was denied: %v", err)
	}
	checker.permissions[12][domaincommerce.SellerPermissionProductWrite] = false
	if _, err := service.UpdateDraft(context.Background(), 12, 1, validDraft()); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("seller staff without product-write permission edited a draft: %v", err)
	}
	if _, err := service.UpdateDraft(context.Background(), 11, 1, validDraft()); err != nil {
		t.Fatalf("seller owner was denied draft edit: %v", err)
	}
	if _, err := service.UpdateDraft(context.Background(), 11, 1, domaincommerce.ProductDraftInput{Slug: "invalid slug"}); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("invalid draft edit returned %v", err)
	}
}

func TestCatalogServiceRequiresActorForPendingProducts(t *testing.T) {
	service := NewCatalogService(catalogRepository{})
	if _, err := service.PendingProducts(context.Background(), 0); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected forbidden pending-product read, got %v", err)
	}
}

type orderRepository struct{}

func (orderRepository) CreateOrder(context.Context, int64, int64, string, domaincommerce.AddressInput) (domaincommerce.Order, error) {
	return domaincommerce.Order{}, nil
}
func (orderRepository) ListOrders(context.Context, int64, int) ([]domaincommerce.Order, error) {
	return nil, nil
}
func (orderRepository) GetOrder(context.Context, int64, string) (domaincommerce.OrderDetail, error) {
	return domaincommerce.OrderDetail{}, nil
}
func (orderRepository) CancelOrder(context.Context, int64, string) error { return nil }

func TestOrderServiceRequiresStrongIdempotencyKeyAndAddress(t *testing.T) {
	service := NewOrderService(orderRepository{})
	_, err := service.Create(context.Background(), 1, 1, "short", domaincommerce.AddressInput{RecipientName: "A", Line1: "B", City: "C", State: "D", PostalCode: "E"})
	if !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected key validation, got %v", err)
	}
	_, err = service.Create(context.Background(), 1, 1, "idempotency-key-1234", domaincommerce.AddressInput{RecipientName: "", Line1: "B", City: "C", State: "D", PostalCode: "E"})
	if !errors.Is(err, ErrInvalidAddress) {
		t.Fatalf("expected address validation, got %v", err)
	}
}

func TestCartServiceRequiresAuthenticatedWishlistOwner(t *testing.T) {
	service := NewCartService(nil)
	if err := service.AddWishlist(context.Background(), 0, 1); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected forbidden wishlist write, got %v", err)
	}
	if _, err := service.ListWishlist(context.Background(), 0); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected forbidden wishlist read, got %v", err)
	}
}

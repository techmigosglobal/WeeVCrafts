package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidProduct = errors.New("product details are invalid")

type CatalogService struct {
	repository  ports.CatalogManagementRepository
	roles       ports.RoleChecker
	permissions ports.SellerPermissionChecker
}

func NewCatalogService(repository ports.CatalogManagementRepository) *CatalogService {
	return &CatalogService{repository: repository}
}

// NewCatalogServiceWithRoles is the production constructor. The legacy
// constructor remains available for small isolated unit fixtures that do not
// model identity storage; the composition root always supplies a RoleChecker.
func NewCatalogServiceWithRoles(repository ports.CatalogManagementRepository, roles ports.RoleChecker) *CatalogService {
	service := &CatalogService{repository: repository, roles: roles}
	if checker, ok := roles.(ports.SellerPermissionChecker); ok {
		service.permissions = checker
	}
	return service
}

func (s *CatalogService) EnsureSeller(ctx context.Context, ownerUserID int64, displayName string) (domaincommerce.Seller, error) {
	displayName = strings.TrimSpace(displayName)
	if ownerUserID <= 0 || len(displayName) < 2 || len(displayName) > 120 || !s.allowed(ctx, ownerUserID, domainidentity.RoleCustomer, domainidentity.RoleSellerOwner) {
		return domaincommerce.Seller{}, ports.ErrForbidden
	}
	return s.repository.EnsureSeller(ctx, ownerUserID, displayName)
}

func (s *CatalogService) SellerProfile(ctx context.Context, userID int64) (domaincommerce.Seller, error) {
	if userID <= 0 || s.repository == nil {
		return domaincommerce.Seller{}, ports.ErrForbidden
	}
	return s.repository.GetSeller(ctx, userID)
}

func (s *CatalogService) CreateDraft(ctx context.Context, ownerUserID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	if !s.sellerAllowed(ctx, ownerUserID, domaincommerce.SellerPermissionProductWrite) {
		return domaincommerce.ManagedProduct{}, ports.ErrForbidden
	}
	input, err := normalizeDraft(input)
	if err != nil || ownerUserID <= 0 {
		return domaincommerce.ManagedProduct{}, ErrInvalidProduct
	}
	if input.InitialStock > 0 && !s.sellerAllowed(ctx, ownerUserID, domaincommerce.SellerPermissionInventoryWrite) {
		return domaincommerce.ManagedProduct{}, ports.ErrForbidden
	}
	product, err := s.repository.CreateDraft(ctx, ownerUserID, input)
	if err != nil {
		return domaincommerce.ManagedProduct{}, fmt.Errorf("create product draft: %w", err)
	}
	return product, nil
}

func (s *CatalogService) GetSellerProduct(ctx context.Context, ownerUserID, productID int64) (domaincommerce.ManagedProduct, error) {
	if productID <= 0 || !s.sellerAllowed(ctx, ownerUserID, domaincommerce.SellerPermissionProductRead) {
		return domaincommerce.ManagedProduct{}, ports.ErrForbidden
	}
	return s.repository.GetSellerProduct(ctx, ownerUserID, productID)
}

func (s *CatalogService) UpdateDraft(ctx context.Context, ownerUserID, productID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	if productID <= 0 || !s.sellerAllowed(ctx, ownerUserID, domaincommerce.SellerPermissionProductWrite) {
		return domaincommerce.ManagedProduct{}, ports.ErrForbidden
	}
	var err error
	input, err = normalizeDraft(input)
	if err != nil {
		return domaincommerce.ManagedProduct{}, err
	}
	product, err := s.repository.UpdateDraft(ctx, ownerUserID, productID, input)
	if err != nil {
		return domaincommerce.ManagedProduct{}, fmt.Errorf("update product draft: %w", err)
	}
	return product, nil
}

func (s *CatalogService) SubmitProduct(ctx context.Context, ownerUserID, productID int64) error {
	if productID <= 0 || !s.sellerAllowed(ctx, ownerUserID, domaincommerce.SellerPermissionProductWrite) {
		return ports.ErrForbidden
	}
	return s.repository.SubmitProduct(ctx, ownerUserID, productID)
}

func (s *CatalogService) ApproveProduct(ctx context.Context, actorID, productID int64, approved bool, reason string) error {
	if productID <= 0 || !s.allowed(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin) {
		return ports.ErrForbidden
	}
	return s.repository.ApproveProduct(ctx, actorID, productID, approved, strings.TrimSpace(reason))
}

func (s *CatalogService) SellerProducts(ctx context.Context, ownerUserID int64) ([]domaincommerce.ManagedProduct, error) {
	if !s.allowed(ctx, ownerUserID, domainidentity.RoleCustomer, domainidentity.RoleSellerOwner) && !s.hasSellerPermission(ctx, ownerUserID, domaincommerce.SellerPermissionProductRead) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListSellerProducts(ctx, ownerUserID)
}

func (s *CatalogService) PendingProducts(ctx context.Context, actorID int64) ([]domaincommerce.ManagedProduct, error) {
	if !s.allowed(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListPendingProducts(ctx, actorID)
}

func (s *CatalogService) allowed(ctx context.Context, userID int64, roles ...domainidentity.Role) bool {
	if userID <= 0 {
		return false
	}
	if s.roles == nil {
		return true
	}
	allowed, err := s.roles.HasAnyRole(ctx, userID, roles...)
	return err == nil && allowed
}

func (s *CatalogService) sellerAllowed(ctx context.Context, userID int64, permission string) bool {
	return s.allowed(ctx, userID, domainidentity.RoleSellerOwner) || s.hasSellerPermission(ctx, userID, permission)
}

func (s *CatalogService) hasSellerPermission(ctx context.Context, userID int64, permission string) bool {
	if userID <= 0 || s.permissions == nil {
		return false
	}
	allowed, err := s.permissions.HasSellerPermission(ctx, userID, permission)
	return err == nil && allowed
}

func validSlug(value string) bool {
	if len(value) < 2 || len(value) > 120 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func normalizeDraft(input domaincommerce.ProductDraftInput) (domaincommerce.ProductDraftInput, error) {
	input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
	input.BrandSlug = strings.TrimSpace(strings.ToLower(input.BrandSlug))
	input.BrandName = strings.TrimSpace(input.BrandName)
	input.CategorySlug = strings.TrimSpace(strings.ToLower(input.CategorySlug))
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.SKU = strings.TrimSpace(strings.ToUpper(input.SKU))
	input.DeliveryLabel = strings.TrimSpace(input.DeliveryLabel)
	if !validSlug(input.Slug) || !validSlug(input.BrandSlug) || !validSlug(input.CategorySlug) || input.BrandName == "" || input.CategoryName == "" || input.Name == "" || input.Description == "" || input.SKU == "" || input.PriceCents <= 0 || input.InitialStock < 0 || input.InitialStock > 100000 {
		return domaincommerce.ProductDraftInput{}, ErrInvalidProduct
	}
	if input.CompareAtCents != nil && *input.CompareAtCents < input.PriceCents {
		return domaincommerce.ProductDraftInput{}, ErrInvalidProduct
	}
	if input.DeliveryLabel == "" {
		input.DeliveryLabel = "Dispatch details at checkout"
	}
	return input, nil
}

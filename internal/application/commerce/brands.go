package commerce

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type BrandDirectoryService struct {
	repository ports.BrandDirectoryRepository
	roles      ports.RoleChecker
}

func NewBrandDirectoryService(repository ports.BrandDirectoryRepository, roles ports.RoleChecker) *BrandDirectoryService {
	return &BrandDirectoryService{repository: repository, roles: roles}
}

func (s *BrandDirectoryService) List(ctx context.Context, actorID int64) ([]domaincommerce.AdminBrand, error) {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	if err != nil || !ok {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListBrandDirectory(ctx, actorID)
}

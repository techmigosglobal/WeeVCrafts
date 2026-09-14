package auth

import (
	"context"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type CustomerDirectoryService struct {
	repository ports.CustomerDirectoryRepository
	roles      ports.RoleChecker
}

func NewCustomerDirectoryService(repository ports.CustomerDirectoryRepository, roles ports.RoleChecker) *CustomerDirectoryService {
	return &CustomerDirectoryService{repository: repository, roles: roles}
}

func (s *CustomerDirectoryService) List(ctx context.Context, actorID int64) ([]domainidentity.AdminCustomer, error) {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	if err != nil || !ok {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListCustomers(ctx, actorID, 200)
}

package commerce

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type AdminOperationsService struct {
	repository ports.AdminOperationsRepository
	roles      ports.RoleChecker
}

func NewAdminOperationsService(repository ports.AdminOperationsRepository, roles ports.RoleChecker) *AdminOperationsService {
	return &AdminOperationsService{repository: repository, roles: roles}
}

func (s *AdminOperationsService) ListOrders(ctx context.Context, actorID int64) ([]domaincommerce.AdminOrder, error) {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleOperations, domainidentity.RoleSuperAdmin)
	if err != nil || !ok {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListAdminOrders(ctx, actorID, 200)
}

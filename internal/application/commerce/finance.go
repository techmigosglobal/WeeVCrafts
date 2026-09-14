package commerce

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type FinanceService struct {
	repository ports.FinanceRepository
	roles      ports.RoleChecker
}

func NewFinanceService(repository ports.FinanceRepository, roles ports.RoleChecker) *FinanceService {
	return &FinanceService{repository: repository, roles: roles}
}

func (s *FinanceService) List(ctx context.Context, actorID int64) ([]domaincommerce.FinanceEntry, error) {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	allowed, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleFinanceOperator, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	if err != nil || !allowed {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListFinanceEntries(ctx, 200)
}

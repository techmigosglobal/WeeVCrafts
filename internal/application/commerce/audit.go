package commerce

import (
	"context"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type AuditService struct {
	repository ports.AuditRepository
	roles      ports.RoleChecker
}

func NewAuditService(repository ports.AuditRepository, roles ports.RoleChecker) *AuditService {
	return &AuditService{repository: repository, roles: roles}
}

func (s *AuditService) List(ctx context.Context, actorID int64) ([]domaincommerce.AuditEntry, error) {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	if err != nil || !ok {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListAuditEntries(ctx, actorID, 200)
}

func (s *AuditService) ListSeller(ctx context.Context, ownerUserID int64) ([]domaincommerce.AuditEntry, error) {
	if ownerUserID <= 0 || s.repository == nil || s.roles == nil {
		return nil, ports.ErrForbidden
	}
	ok, err := s.roles.HasAnyRole(ctx, ownerUserID, domainidentity.RoleSellerOwner)
	if err != nil || !ok {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListSellerAuditEntries(ctx, ownerUserID, 200)
}

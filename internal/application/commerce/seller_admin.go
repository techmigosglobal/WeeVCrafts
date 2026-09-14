package commerce

import (
	"context"
	"errors"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidSellerStatus = errors.New("seller status request is invalid")

type SellerAdminService struct {
	repository ports.SellerAdministrationRepository
	roles      ports.RoleChecker
}

func NewSellerAdminService(repository ports.SellerAdministrationRepository, roles ports.RoleChecker) *SellerAdminService {
	return &SellerAdminService{repository: repository, roles: roles}
}

func (s *SellerAdminService) List(ctx context.Context, actorID int64) ([]domaincommerce.SellerAdminEntry, error) {
	if !s.allowed(ctx, actorID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListSellerApplications(ctx, actorID)
}

func (s *SellerAdminService) UpdateStatus(ctx context.Context, actorID, sellerID int64, status, reason string) error {
	if !s.allowed(ctx, actorID) {
		return ports.ErrForbidden
	}
	status = strings.ToLower(strings.TrimSpace(status))
	reason = strings.TrimSpace(reason)
	if sellerID <= 0 || (status != "active" && status != "suspended" && status != "closed") || len(reason) < 4 || len(reason) > 1000 {
		return ErrInvalidSellerStatus
	}
	return s.repository.UpdateSellerStatus(ctx, actorID, sellerID, status, reason)
}

func (s *SellerAdminService) allowed(ctx context.Context, actorID int64) bool {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return false
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	return err == nil && ok
}

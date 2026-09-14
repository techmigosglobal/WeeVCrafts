package auth

import (
	"context"
	"errors"
	"strings"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidRoleAssignment = errors.New("role assignment request is invalid")

var RoleAdminAssignableRoles = []domainidentity.Role{
	domainidentity.RoleCustomer,
	domainidentity.RoleSellerOwner,
	domainidentity.RoleSellerStaff,
	domainidentity.RoleMarketplaceAdmin,
	domainidentity.RoleSupportAgent,
	domainidentity.RoleFinanceOperator,
	domainidentity.RoleOperations,
}

type RoleAdminService struct {
	repository ports.RoleAdministrationRepository
	roles      ports.RoleChecker
}

func NewRoleAdminService(repository ports.RoleAdministrationRepository, roles ports.RoleChecker) *RoleAdminService {
	return &RoleAdminService{repository: repository, roles: roles}
}

func (s *RoleAdminService) List(ctx context.Context, actorID int64) ([]domainidentity.RoleAssignment, error) {
	if !s.allowed(ctx, actorID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListRoleAssignments(ctx, actorID)
}

func (s *RoleAdminService) Update(ctx context.Context, actorID, userID int64, role string, grant bool, reason string) error {
	if !s.allowed(ctx, actorID) {
		return ports.ErrForbidden
	}
	roleValue := domainidentity.Role(strings.ToLower(strings.TrimSpace(role)))
	reason = strings.TrimSpace(reason)
	if actorID <= 0 || userID <= 0 || len(reason) < 4 || len(reason) > 1000 || !assignableRole(roleValue) {
		return ErrInvalidRoleAssignment
	}
	if actorID == userID && roleValue == domainidentity.RoleSuperAdmin {
		return ports.ErrRoleState
	}
	return s.repository.UpdateRoleAssignment(ctx, actorID, userID, roleValue, grant, reason)
}

func (s *RoleAdminService) allowed(ctx context.Context, actorID int64) bool {
	if actorID <= 0 || s.repository == nil || s.roles == nil {
		return false
	}
	ok, err := s.roles.HasAnyRole(ctx, actorID, domainidentity.RoleSuperAdmin)
	return err == nil && ok
}

func assignableRole(role domainidentity.Role) bool {
	for _, allowed := range RoleAdminAssignableRoles {
		if role == allowed {
			return true
		}
	}
	return false
}

package auth

import (
	"context"
	"errors"
	"testing"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type roleAdminRepository struct{}

func (roleAdminRepository) ListRoleAssignments(context.Context, int64) ([]domainidentity.RoleAssignment, error) {
	return []domainidentity.RoleAssignment{{ID: 9, Roles: []string{"customer"}}}, nil
}
func (roleAdminRepository) UpdateRoleAssignment(context.Context, int64, int64, domainidentity.Role, bool, string) error {
	return nil
}

type roleAdminChecker struct{ super bool }

func (r roleAdminChecker) HasAnyRole(context.Context, int64, ...domainidentity.Role) (bool, error) {
	return r.super, nil
}

func TestRoleAdminServiceRequiresSuperAdmin(t *testing.T) {
	service := NewRoleAdminService(roleAdminRepository{}, roleAdminChecker{})
	if _, err := service.List(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected role list denial, got %v", err)
	}
}

func TestRoleAdminServiceValidatesAssignableRoleAndReason(t *testing.T) {
	service := NewRoleAdminService(roleAdminRepository{}, roleAdminChecker{super: true})
	if err := service.Update(context.Background(), 42, 9, "super_admin", true, "promote"); !errors.Is(err, ErrInvalidRoleAssignment) {
		t.Fatalf("expected super-admin assignment denial, got %v", err)
	}
	if err := service.Update(context.Background(), 42, 9, "finance_operator", true, "no"); !errors.Is(err, ErrInvalidRoleAssignment) {
		t.Fatalf("expected short reason denial, got %v", err)
	}
	if err := service.Update(context.Background(), 42, 9, "finance_operator", true, "finance responsibility"); err != nil {
		t.Fatalf("valid role grant failed: %v", err)
	}
}

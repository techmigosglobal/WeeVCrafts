package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type sellerStaffRepository struct {
	addedEmail       string
	addedPermissions []string
	removedUserID    int64
}

func (r *sellerStaffRepository) ListStaff(context.Context, int64) ([]domaincommerce.SellerStaffMember, error) {
	return []domaincommerce.SellerStaffMember{{UserID: 7, Email: "staff@example.com", Status: "active"}}, nil
}

func (r *sellerStaffRepository) AddStaff(_ context.Context, _ int64, email string, permissions []string) (domaincommerce.SellerStaffMember, error) {
	r.addedEmail = email
	r.addedPermissions = permissions
	return domaincommerce.SellerStaffMember{UserID: 7, Email: email, Permissions: permissions, Status: "active"}, nil
}

func (r *sellerStaffRepository) RemoveStaff(_ context.Context, _ int64, staffUserID int64) error {
	r.removedUserID = staffUserID
	return nil
}

func TestSellerStaffServiceScopesOwnerManagementAndNormalizesPermissions(t *testing.T) {
	repository := &sellerStaffRepository{}
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		10: {domainidentity.RoleSellerOwner: true},
		11: {domainidentity.RoleSellerStaff: true},
	}}
	service := NewSellerStaffService(repository, checker)

	member, err := service.Add(context.Background(), 10, " Staff@Example.com ", []string{"ORDER_READ", "PRODUCT_WRITE", "ORDER_READ"})
	if err != nil {
		t.Fatalf("owner could not add staff: %v", err)
	}
	if member.Email != "staff@example.com" || repository.addedEmail != "staff@example.com" {
		t.Fatalf("staff email was not normalized: member=%+v repo=%q", member, repository.addedEmail)
	}
	if got := repository.addedPermissions; len(got) != 2 || got[0] != "ORDER_READ" || got[1] != "PRODUCT_WRITE" {
		t.Fatalf("permissions were not deduplicated/sorted: %#v", got)
	}
	if _, err := service.Add(context.Background(), 11, "staff@example.com", []string{"PRODUCT_READ"}); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("seller staff managed a team: %v", err)
	}
	if _, err := service.Add(context.Background(), 10, "not-an-email", []string{"PRODUCT_READ"}); !errors.Is(err, ErrInvalidStaffEmail) {
		t.Fatalf("invalid staff email returned %v", err)
	}
	if _, err := service.Add(context.Background(), 10, "staff@example.com", []string{"SECURITY_ADMIN"}); !errors.Is(err, ErrInvalidStaffPermissions) {
		t.Fatalf("invalid staff permission returned %v", err)
	}
	if err := service.Remove(context.Background(), 10, 10); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("owner removed itself: %v", err)
	}
	if err := service.Remove(context.Background(), 10, 7); err != nil || repository.removedUserID != 7 {
		t.Fatalf("owner could not remove staff: err=%v id=%d", err, repository.removedUserID)
	}
}

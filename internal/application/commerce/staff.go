package commerce

import (
	"context"
	"errors"
	"net/mail"
	"sort"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var (
	ErrInvalidStaffEmail       = errors.New("staff email is invalid")
	ErrInvalidStaffPermissions = errors.New("staff permissions are invalid")
)

type SellerStaffService struct {
	repository ports.SellerStaffRepository
	roles      ports.RoleChecker
}

func NewSellerStaffService(repository ports.SellerStaffRepository, roles ports.RoleChecker) *SellerStaffService {
	return &SellerStaffService{repository: repository, roles: roles}
}

func (s *SellerStaffService) List(ctx context.Context, ownerUserID int64) ([]domaincommerce.SellerStaffMember, error) {
	if !s.ownerAllowed(ctx, ownerUserID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListStaff(ctx, ownerUserID)
}

func (s *SellerStaffService) Add(ctx context.Context, ownerUserID int64, email string, permissions []string) (domaincommerce.SellerStaffMember, error) {
	if !s.ownerAllowed(ctx, ownerUserID) {
		return domaincommerce.SellerStaffMember{}, ports.ErrForbidden
	}
	email, err := normalizeStaffEmail(email)
	if err != nil {
		return domaincommerce.SellerStaffMember{}, err
	}
	permissions, err = normalizeStaffPermissions(permissions)
	if err != nil {
		return domaincommerce.SellerStaffMember{}, err
	}
	return s.repository.AddStaff(ctx, ownerUserID, email, permissions)
}

func (s *SellerStaffService) Remove(ctx context.Context, ownerUserID, staffUserID int64) error {
	if staffUserID <= 0 || staffUserID == ownerUserID || !s.ownerAllowed(ctx, ownerUserID) {
		return ports.ErrForbidden
	}
	return s.repository.RemoveStaff(ctx, ownerUserID, staffUserID)
}

func (s *SellerStaffService) ownerAllowed(ctx context.Context, userID int64) bool {
	if userID <= 0 || s.repository == nil {
		return false
	}
	if s.roles == nil {
		return true
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, domainidentity.RoleSellerOwner)
	return err == nil && ok
}

func normalizeStaffEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return "", ErrInvalidStaffEmail
	}
	return value, nil
}

func normalizeStaffPermissions(values []string) ([]string, error) {
	allowed := make(map[string]struct{}, len(domaincommerce.SellerPermissions))
	for _, permission := range domaincommerce.SellerPermissions {
		allowed[permission] = struct{}{}
	}
	unique := make(map[string]struct{}, len(values))
	for _, permission := range values {
		permission = strings.TrimSpace(strings.ToUpper(permission))
		if _, ok := allowed[permission]; !ok {
			return nil, ErrInvalidStaffPermissions
		}
		unique[permission] = struct{}{}
	}
	if len(unique) == 0 {
		return nil, ErrInvalidStaffPermissions
	}
	permissions := make([]string, 0, len(unique))
	for permission := range unique {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions, nil
}

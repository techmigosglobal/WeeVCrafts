package commerce

import (
	"context"
	"errors"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidReturnRequest = errors.New("return request is invalid")

type ReturnService struct {
	repository ports.ReturnRepository
	roles      ports.RoleChecker
}

func NewReturnService(repository ports.ReturnRepository, roles ports.RoleChecker) *ReturnService {
	return &ReturnService{repository: repository, roles: roles}
}

func (s *ReturnService) Create(ctx context.Context, userID int64, orderNumber, reason string) (domaincommerce.ReturnRequest, error) {
	if !s.hasRole(ctx, userID, domainidentity.RoleCustomer) {
		return domaincommerce.ReturnRequest{}, ports.ErrForbidden
	}
	orderNumber = strings.TrimSpace(orderNumber)
	reason = strings.TrimSpace(reason)
	if orderNumber == "" || len(orderNumber) > 120 || len(reason) < 10 || len(reason) > 1000 {
		return domaincommerce.ReturnRequest{}, ErrInvalidReturnRequest
	}
	return s.repository.CreateReturnRequest(ctx, userID, orderNumber, reason)
}

func (s *ReturnService) Get(ctx context.Context, userID int64, orderNumber string) (domaincommerce.ReturnRequest, error) {
	if !s.hasRole(ctx, userID, domainidentity.RoleCustomer) || strings.TrimSpace(orderNumber) == "" {
		return domaincommerce.ReturnRequest{}, ports.ErrForbidden
	}
	return s.repository.GetReturnRequest(ctx, userID, strings.TrimSpace(orderNumber))
}

func (s *ReturnService) List(ctx context.Context, actorID int64) ([]domaincommerce.ReturnRequest, error) {
	if !s.operationsAllowed(ctx, actorID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListReturnRequests(ctx, actorID)
}

func (s *ReturnService) Update(ctx context.Context, actorID, requestID int64, status, reason string) error {
	if !s.operationsAllowed(ctx, actorID) {
		return ports.ErrForbidden
	}
	status = strings.TrimSpace(strings.ToLower(status))
	if requestID <= 0 || (status != "approved" && status != "rejected" && status != "received") {
		return ErrInvalidReturnRequest
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 1000 {
		return ErrInvalidReturnRequest
	}
	return s.repository.UpdateReturnRequest(ctx, actorID, requestID, status, reason)
}

func (s *ReturnService) hasRole(ctx context.Context, userID int64, role domainidentity.Role) bool {
	if userID <= 0 || s.repository == nil {
		return false
	}
	if s.roles == nil {
		return true
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, role)
	return err == nil && ok
}

func (s *ReturnService) operationsAllowed(ctx context.Context, userID int64) bool {
	if userID <= 0 || s.repository == nil {
		return false
	}
	if s.roles == nil {
		return true
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, domainidentity.RoleOperations, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	return err == nil && ok
}

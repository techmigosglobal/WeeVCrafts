package commerce

import (
	"context"
	"errors"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidSupportTicket = errors.New("support ticket is invalid")

type SupportService struct {
	repository ports.SupportRepository
	roles      ports.RoleChecker
}

func NewSupportService(repository ports.SupportRepository, roles ports.RoleChecker) *SupportService {
	return &SupportService{repository: repository, roles: roles}
}

func (s *SupportService) Create(ctx context.Context, userID int64, orderNumber, subject, message string) (domaincommerce.SupportTicket, error) {
	if !s.hasRole(ctx, userID, domainidentity.RoleCustomer) {
		return domaincommerce.SupportTicket{}, ports.ErrForbidden
	}
	orderNumber = strings.TrimSpace(orderNumber)
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if err := validateSupportTicket(orderNumber, subject, message); err != nil {
		return domaincommerce.SupportTicket{}, err
	}
	return s.repository.CreateSupportTicket(ctx, userID, orderNumber, subject, message)
}

func (s *SupportService) CreateSeller(ctx context.Context, userID int64, orderNumber, subject, message string) (domaincommerce.SupportTicket, error) {
	if !s.sellerAllowed(ctx, userID) {
		return domaincommerce.SupportTicket{}, ports.ErrForbidden
	}
	orderNumber = strings.TrimSpace(orderNumber)
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if err := validateSupportTicket(orderNumber, subject, message); err != nil {
		return domaincommerce.SupportTicket{}, err
	}
	return s.repository.CreateSupportTicket(ctx, userID, orderNumber, subject, message)
}

func validateSupportTicket(orderNumber, subject, message string) error {
	if len(orderNumber) > 120 || len(subject) < 4 || len(subject) > 160 || len(message) < 10 || len(message) > 4000 {
		return ErrInvalidSupportTicket
	}
	return nil
}

func (s *SupportService) ListCustomer(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error) {
	if !s.hasRole(ctx, userID, domainidentity.RoleCustomer) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListCustomerSupportTickets(ctx, userID)
}

func (s *SupportService) ListSeller(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error) {
	if !s.sellerAllowed(ctx, userID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListSellerSupportTickets(ctx, userID)
}

func (s *SupportService) ListAgent(ctx context.Context, actorID int64) ([]domaincommerce.SupportTicket, error) {
	if !s.agentAllowed(ctx, actorID) {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListSupportTickets(ctx, 200)
}

func (s *SupportService) Update(ctx context.Context, actorID, ticketID int64, status, note string) error {
	if !s.agentAllowed(ctx, actorID) {
		return ports.ErrForbidden
	}
	status = strings.ToLower(strings.TrimSpace(status))
	note = strings.TrimSpace(note)
	if ticketID <= 0 || (status != "open" && status != "in_progress" && status != "waiting_customer" && status != "resolved" && status != "closed") || len(note) > 1000 {
		return ErrInvalidSupportTicket
	}
	return s.repository.UpdateSupportTicket(ctx, actorID, ticketID, status, note)
}

func (s *SupportService) hasRole(ctx context.Context, userID int64, role domainidentity.Role) bool {
	if userID <= 0 || s.repository == nil || s.roles == nil {
		return false
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, role)
	return err == nil && ok
}

func (s *SupportService) agentAllowed(ctx context.Context, userID int64) bool {
	if userID <= 0 || s.repository == nil || s.roles == nil {
		return false
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, domainidentity.RoleSupportAgent, domainidentity.RoleSuperAdmin)
	return err == nil && ok
}

func (s *SupportService) sellerAllowed(ctx context.Context, userID int64) bool {
	if userID <= 0 || s.repository == nil || s.roles == nil {
		return false
	}
	ok, err := s.roles.HasAnyRole(ctx, userID, domainidentity.RoleSellerOwner)
	return err == nil && ok
}

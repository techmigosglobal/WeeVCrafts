package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type supportRepository struct{}

func (supportRepository) CreateSupportTicket(context.Context, int64, string, string, string) (domaincommerce.SupportTicket, error) {
	return domaincommerce.SupportTicket{TicketNumber: "SUP-1"}, nil
}
func (supportRepository) ListCustomerSupportTickets(context.Context, int64) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-1"}}, nil
}
func (supportRepository) ListSellerSupportTickets(context.Context, int64) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-1"}}, nil
}
func (supportRepository) ListSupportTickets(context.Context, int) ([]domaincommerce.SupportTicket, error) {
	return []domaincommerce.SupportTicket{{TicketNumber: "SUP-1"}}, nil
}
func (supportRepository) UpdateSupportTicket(context.Context, int64, int64, string, string) error {
	return nil
}

func TestSupportServiceSeparatesCustomerAndAgentViews(t *testing.T) {
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		30: {domainidentity.RoleCustomer: true},
		31: {domainidentity.RoleSupportAgent: true},
	}}
	service := NewSupportService(supportRepository{}, checker)
	if _, err := service.Create(context.Background(), 30, "WC-1", "Damaged item", "The item arrived damaged and needs help."); err != nil {
		t.Fatalf("customer could not create ticket: %v", err)
	}
	if _, err := service.ListAgent(context.Background(), 30); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("customer could view support queue: %v", err)
	}
	if tickets, err := service.ListAgent(context.Background(), 31); err != nil || len(tickets) != 1 {
		t.Fatalf("support agent denied queue: tickets=%+v err=%v", tickets, err)
	}
}

func TestSupportServiceValidatesTicketContent(t *testing.T) {
	service := NewSupportService(supportRepository{}, roleChecker{roles: map[int64]map[domainidentity.Role]bool{30: {domainidentity.RoleCustomer: true}}})
	if _, err := service.Create(context.Background(), 30, "", "bad", "short"); !errors.Is(err, ErrInvalidSupportTicket) {
		t.Fatalf("invalid ticket returned %v", err)
	}
}

func TestSupportServiceProvidesSellerOwnedSupport(t *testing.T) {
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		41: {domainidentity.RoleSellerOwner: true},
		42: {domainidentity.RoleCustomer: true},
	}}
	service := NewSupportService(supportRepository{}, checker)
	if _, err := service.CreateSeller(context.Background(), 41, "WC-41", "Catalogue review", "Please explain the review status for this product."); err != nil {
		t.Fatalf("seller could not create a support ticket: %v", err)
	}
	if tickets, err := service.ListSeller(context.Background(), 41); err != nil || len(tickets) != 1 {
		t.Fatalf("seller could not list owned support tickets: tickets=%+v err=%v", tickets, err)
	}
	if _, err := service.ListSeller(context.Background(), 42); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("customer accessed seller support: %v", err)
	}
}

package commerce

import (
	"context"
	"errors"
	"testing"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type returnRepository struct {
	createdOrder string
	updated      struct {
		id     int64
		status string
	}
}

func (r *returnRepository) CreateReturnRequest(_ context.Context, _ int64, orderNumber, _ string) (domaincommerce.ReturnRequest, error) {
	r.createdOrder = orderNumber
	return domaincommerce.ReturnRequest{ID: 3, OrderNumber: orderNumber, Status: "pending"}, nil
}

func (r *returnRepository) GetReturnRequest(context.Context, int64, string) (domaincommerce.ReturnRequest, error) {
	return domaincommerce.ReturnRequest{}, ports.ErrReturnNotFound
}

func (r *returnRepository) ListReturnRequests(context.Context, int64) ([]domaincommerce.ReturnRequest, error) {
	return []domaincommerce.ReturnRequest{{ID: 3, Status: "pending"}}, nil
}

func (r *returnRepository) UpdateReturnRequest(_ context.Context, _ int64, requestID int64, status, _ string) error {
	r.updated.id, r.updated.status = requestID, status
	return nil
}

func TestReturnServiceSeparatesCustomerCreateFromOperationsUpdate(t *testing.T) {
	repository := &returnRepository{}
	checker := roleChecker{roles: map[int64]map[domainidentity.Role]bool{
		10: {domainidentity.RoleCustomer: true},
		20: {domainidentity.RoleOperations: true},
	}}
	service := NewReturnService(repository, checker)
	if _, err := service.Create(context.Background(), 10, "WC-123", "The item arrived damaged"); err != nil {
		t.Fatalf("customer return request failed: %v", err)
	}
	if repository.createdOrder != "WC-123" {
		t.Fatalf("unexpected order number: %q", repository.createdOrder)
	}
	if err := service.Update(context.Background(), 10, 3, "approved", "no"); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("customer updated operations queue: %v", err)
	}
	if err := service.Update(context.Background(), 20, 3, "approved", "inspection queued"); err != nil {
		t.Fatalf("operations update failed: %v", err)
	}
	if repository.updated.id != 3 || repository.updated.status != "approved" {
		t.Fatalf("unexpected update: %+v", repository.updated)
	}
}

func TestReturnServiceValidatesReasonAndState(t *testing.T) {
	service := NewReturnService(&returnRepository{}, roleChecker{roles: map[int64]map[domainidentity.Role]bool{10: {domainidentity.RoleCustomer: true}}})
	if _, err := service.Create(context.Background(), 10, "WC-123", "short"); !errors.Is(err, ErrInvalidReturnRequest) {
		t.Fatalf("short reason returned %v", err)
	}
	if err := service.Update(context.Background(), 10, 1, "received", ""); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("customer received update returned %v", err)
	}
}

package payment

import (
	"context"
	"errors"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrPaymentCreationFailed = errors.New("payment order could not be created")

type OrderService struct {
	gateway    ports.PaymentGateway
	repository ports.PaymentOrderRepository
}

func NewOrderService(gateway ports.PaymentGateway, repository ports.PaymentOrderRepository) *OrderService {
	return &OrderService{gateway: gateway, repository: repository}
}

// Create prepares a provider order from the server-side order total. The
// browser never supplies or decides the amount or currency.
func (s *OrderService) Create(ctx context.Context, userID int64, orderNumber string) (domaincommerce.PaymentIntent, error) {
	if userID <= 0 || orderNumber == "" || s.gateway == nil || s.repository == nil {
		return domaincommerce.PaymentIntent{}, ports.ErrPaymentNotFound
	}
	record, err := s.repository.GetPaymentOrder(ctx, userID, orderNumber)
	if err != nil {
		return domaincommerce.PaymentIntent{}, err
	}
	if record.ProviderOrderID != "" {
		return mapPaymentIntent(record), nil
	}
	if record.OrderStatus != "pending_payment" || record.PaymentStatus != "created" || record.AmountCents <= 0 || record.Currency == "" {
		return domaincommerce.PaymentIntent{}, ports.ErrPaymentState
	}
	providerOrder, err := s.gateway.CreateOrder(ctx, ports.PaymentOrderRequest{
		Reference:   record.OrderNumber,
		AmountCents: record.AmountCents,
		Currency:    record.Currency,
	})
	if err != nil || providerOrder.ProviderOrderID == "" || providerOrder.AmountCents != record.AmountCents || providerOrder.Currency != record.Currency {
		return domaincommerce.PaymentIntent{}, ErrPaymentCreationFailed
	}
	attached, err := s.repository.AttachProviderOrder(ctx, userID, orderNumber, providerOrder)
	if err != nil {
		return domaincommerce.PaymentIntent{}, err
	}
	return mapPaymentIntent(attached), nil
}

func mapPaymentIntent(record ports.PaymentOrderRecord) domaincommerce.PaymentIntent {
	return domaincommerce.PaymentIntent{
		OrderID:         record.OrderID,
		OrderNumber:     record.OrderNumber,
		Provider:        record.Provider,
		ProviderOrderID: record.ProviderOrderID,
		Status:          record.PaymentStatus,
		AmountCents:     record.AmountCents,
		Currency:        record.Currency,
	}
}

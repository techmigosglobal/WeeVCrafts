package commerce

import (
	"context"
	"errors"
	"strings"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidAddress = errors.New("shipping address is invalid")

type OrderService struct {
	repository ports.OrderRepository
}

func NewOrderService(repository ports.OrderRepository) *OrderService {
	return &OrderService{repository: repository}
}

func (s *OrderService) Create(ctx context.Context, userID, cartID int64, idempotencyKey string, address domaincommerce.AddressInput) (domaincommerce.Order, error) {
	if userID <= 0 || cartID <= 0 || len(strings.TrimSpace(idempotencyKey)) < 16 || len(idempotencyKey) > 200 {
		return domaincommerce.Order{}, ports.ErrForbidden
	}
	if strings.TrimSpace(address.RecipientName) == "" || strings.TrimSpace(address.Line1) == "" || strings.TrimSpace(address.City) == "" || strings.TrimSpace(address.State) == "" || strings.TrimSpace(address.PostalCode) == "" {
		return domaincommerce.Order{}, ErrInvalidAddress
	}
	if address.CountryCode == "" {
		address.CountryCode = "IN"
	}
	return s.repository.CreateOrder(ctx, userID, cartID, idempotencyKey, address)
}

func (s *OrderService) List(ctx context.Context, userID int64, limit int) ([]domaincommerce.Order, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repository.ListOrders(ctx, userID, limit)
}

func (s *OrderService) Get(ctx context.Context, userID int64, orderNumber string) (domaincommerce.OrderDetail, error) {
	if userID <= 0 || strings.TrimSpace(orderNumber) == "" {
		return domaincommerce.OrderDetail{}, ports.ErrOrderNotFound
	}
	return s.repository.GetOrder(ctx, userID, orderNumber)
}

func (s *OrderService) Cancel(ctx context.Context, userID int64, orderNumber string) error {
	if userID <= 0 || strings.TrimSpace(orderNumber) == "" {
		return ports.ErrOrderNotFound
	}
	return s.repository.CancelOrder(ctx, userID, orderNumber)
}
